package models

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	mrand "math/rand"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	// ErrEntryNotFound is returned when a ledger entry cannot be located.
	ErrEntryNotFound = errors.New("ledger entry not found")
	// ErrUndoUnavailable indicates there is no further history to rewind.
	ErrUndoUnavailable = errors.New("no undo steps available")
	// ErrRedoUnavailable indicates there is no further forward history.
	ErrRedoUnavailable = errors.New("no redo steps available")
	// ErrLoginChallengeNotFound indicates there is no nonce to fulfil.
	ErrLoginChallengeNotFound = errors.New("login challenge not found")
	// ErrSignatureInvalid indicates that signature verification failed.
	ErrSignatureInvalid = errors.New("signature invalid")
	// ErrIPNotAllowed indicates the source IP is not within the allowlist.
	ErrIPNotAllowed = errors.New("ip not allowed")
)

var (
	seededRand   = mrand.New(mrand.NewSource(time.Now().UnixNano()))
	seededRandMu sync.Mutex
	idAlphabet   = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

// LedgerStore coordinates all mutable state for the in-memory implementation.
type LedgerStore struct {
	mu sync.RWMutex

	entries map[LedgerType][]LedgerEntry
	allow   map[string]*IPAllowlistEntry
	audits  []*AuditLogEntry

	loginChallenges map[string]*LoginChallenge

	history historyStack
}

// NewLedgerStore constructs a ledger store.
func NewLedgerStore() *LedgerStore {
	store := &LedgerStore{
		entries:         make(map[LedgerType][]LedgerEntry),
		allow:           make(map[string]*IPAllowlistEntry),
		loginChallenges: make(map[string]*LoginChallenge),
	}
	store.history.limit = 11
	store.history.Reset(store.snapshotLocked())
	return store
}

// GenerateID creates a pseudo-random identifier string.
func GenerateID(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, randomIDToken(18))
}

func randomIDToken(length int) string {
	seededRandMu.Lock()
	defer seededRandMu.Unlock()
	b := make([]rune, length)
	for i := range b {
		b[i] = idAlphabet[seededRand.Intn(len(idAlphabet))]
	}
	return string(b)
}

func cloneEntrySlice(entries []LedgerEntry) []LedgerEntry {
	cloned := make([]LedgerEntry, len(entries))
	for i, e := range entries {
		cloned[i] = e.Clone()
	}
	return cloned
}

// recordSnapshotLocked must be called with the mutex locked.
func (s *LedgerStore) snapshotLocked() storeSnapshot {
	snapshot := storeSnapshot{entries: make(map[LedgerType][]LedgerEntry)}
	for _, typ := range AllLedgerTypes {
		if items, ok := s.entries[typ]; ok {
			snapshot.entries[typ] = cloneEntrySlice(items)
		}
	}
	return snapshot
}

func (s *LedgerStore) commitLocked() {
	snapshot := s.snapshotLocked()
	if len(s.history.states) == 0 {
		s.history.Reset(snapshot)
		return
	}
	s.history.Push(snapshot)
}

// ListEntries returns ordered entries for a ledger.
func (s *LedgerStore) ListEntries(typ LedgerType) []LedgerEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := s.entries[typ]
	cloned := make([]LedgerEntry, len(entries))
	for i, e := range entries {
		cloned[i] = e.Clone()
	}
	sort.Slice(cloned, func(i, j int) bool { return cloned[i].Order < cloned[j].Order })
	return cloned
}

// GetEntry retrieves an entry by ID.
func (s *LedgerStore) GetEntry(typ LedgerType, id string) (LedgerEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.entries[typ] {
		if e.ID == id {
			return e.Clone(), nil
		}
	}
	return LedgerEntry{}, ErrEntryNotFound
}

// CreateEntry appends a new entry to the ledger.
func (s *LedgerStore) CreateEntry(typ LedgerType, entry LedgerEntry, actor string) (LedgerEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry.ID = GenerateID(string(typ))
	entry.CreatedAt = time.Now().UTC()
	entry.UpdatedAt = entry.CreatedAt
	entry.Order = len(s.entries[typ])
	entry.Tags = normaliseStrings(entry.Tags)
	s.entries[typ] = append(s.entries[typ], entry.Clone())
	s.appendAuditLocked(actor, fmt.Sprintf("create_%s", typ), entry.ID)
	s.commitLocked()
	return entry, nil
}

// UpdateEntry modifies the entry with matching ID.
func (s *LedgerStore) UpdateEntry(typ LedgerType, id string, updates LedgerEntry, actor string) (LedgerEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.entries[typ]
	for i, e := range items {
		if e.ID == id {
			updated := e.Clone()
			if updates.Name != "" {
				updated.Name = updates.Name
			}
			if updates.Description != "" {
				updated.Description = updates.Description
			}
			if updates.Attributes != nil {
				updated.Attributes = make(map[string]string, len(updates.Attributes))
				for k, v := range updates.Attributes {
					updated.Attributes[k] = v
				}
			}
			if updates.Tags != nil {
				updated.Tags = normaliseStrings(updates.Tags)
			}
			if updates.Links != nil {
				updated.Links = make(map[LedgerType][]string, len(updates.Links))
				for lt, ids := range updates.Links {
					updated.Links[lt] = append([]string{}, ids...)
				}
			}
			updated.UpdatedAt = time.Now().UTC()
			items[i] = updated
			s.entries[typ] = items
			s.appendAuditLocked(actor, fmt.Sprintf("update_%s", typ), id)
			s.commitLocked()
			return updated.Clone(), nil
		}
	}
	return LedgerEntry{}, ErrEntryNotFound
}

// DeleteEntry removes an entry and compacts ordering.
func (s *LedgerStore) DeleteEntry(typ LedgerType, id string, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.entries[typ]
	for i, e := range items {
		if e.ID == id {
			items = append(items[:i], items[i+1:]...)
			for idx := range items {
				items[idx].Order = idx
				items[idx].UpdatedAt = time.Now().UTC()
			}
			s.entries[typ] = items
			s.appendAuditLocked(actor, fmt.Sprintf("delete_%s", typ), id)
			s.commitLocked()
			return nil
		}
	}
	return ErrEntryNotFound
}

// ReorderEntries sets the ordering based on provided IDs. IDs not listed retain current order at end.
func (s *LedgerStore) ReorderEntries(typ LedgerType, orderedIDs []string, actor string) ([]LedgerEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.entries[typ]
	if len(items) == 0 {
		return nil, nil
	}
	index := make(map[string]LedgerEntry, len(items))
	for _, item := range items {
		index[item.ID] = item
	}

	result := make([]LedgerEntry, 0, len(items))
	seen := make(map[string]struct{}, len(orderedIDs))
	for _, id := range orderedIDs {
		if entry, ok := index[id]; ok {
			seen[id] = struct{}{}
			result = append(result, entry)
		}
	}
	for _, item := range items {
		if _, ok := seen[item.ID]; ok {
			continue
		}
		result = append(result, item)
	}
	for i := range result {
		result[i].Order = i
		result[i].UpdatedAt = time.Now().UTC()
	}
	s.entries[typ] = result
	s.appendAuditLocked(actor, fmt.Sprintf("reorder_%s", typ), strings.Join(orderedIDs, ","))
	s.commitLocked()
	out := make([]LedgerEntry, len(result))
	for i, item := range result {
		out[i] = item.Clone()
	}
	return out, nil
}

// ReplaceEntries overwrites the ledger with provided entries.
func (s *LedgerStore) ReplaceEntries(typ LedgerType, entries []LedgerEntry, actor string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	normalized := make([]LedgerEntry, len(entries))
	for i, entry := range entries {
		entry.Order = i
		entry.Tags = normaliseStrings(entry.Tags)
		entry.CreatedAt = entry.CreatedAt.UTC()
		entry.UpdatedAt = time.Now().UTC()
		normalized[i] = entry.Clone()
	}
	s.entries[typ] = normalized
	s.appendAuditLocked(actor, fmt.Sprintf("replace_%s", typ), fmt.Sprintf("count=%d", len(entries)))
	s.commitLocked()
}

// Undo reverts the store to the previous snapshot.
func (s *LedgerStore) Undo() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, err := s.history.Undo()
	if err != nil {
		return err
	}
	s.entries = cloneSnapshot(snapshot)
	return nil
}

// Redo reapplies the next snapshot from history.
func (s *LedgerStore) Redo() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, err := s.history.Redo()
	if err != nil {
		return err
	}
	s.entries = cloneSnapshot(snapshot)
	return nil
}

// CanUndo reports whether history contains a previous snapshot.
func (s *LedgerStore) CanUndo() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.history.CanUndo()
}

// CanRedo reports whether history contains a forward snapshot.
func (s *LedgerStore) CanRedo() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.history.CanRedo()
}

// HistoryDepth returns the counts of undo and redo steps currently available.
func (s *LedgerStore) HistoryDepth() (int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	undo := 0
	if len(s.history.states) > 0 {
		undo = len(s.history.states) - 1
	}
	redo := len(s.history.future)
	return undo, redo
}

func cloneSnapshot(snapshot storeSnapshot) map[LedgerType][]LedgerEntry {
	cloned := make(map[LedgerType][]LedgerEntry, len(snapshot.entries))
	for typ, items := range snapshot.entries {
		cloned[typ] = cloneEntrySlice(items)
	}
	return cloned
}

// AppendAllowlist inserts or updates an allowlist entry.
func (s *LedgerStore) AppendAllowlist(entry *IPAllowlistEntry, actor string) (*IPAllowlistEntry, error) {
	if entry == nil {
		return nil, errors.New("allowlist entry cannot be nil")
	}
	if _, _, err := net.ParseCIDR(entry.CIDR); err != nil {
		if ip := net.ParseIP(entry.CIDR); ip == nil {
			return nil, fmt.Errorf("invalid CIDR or IP: %w", err)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if entry.ID == "" {
		entry.ID = GenerateID("allow")
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now
	copied := *entry
	s.allow[copied.ID] = &copied
	s.appendAuditLocked(actor, "allowlist_upsert", copied.ID)
	return &copied, nil
}

// RemoveAllowlist deletes an entry.
func (s *LedgerStore) RemoveAllowlist(id string, actor string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.allow[id]; ok {
		delete(s.allow, id)
		s.appendAuditLocked(actor, "allowlist_delete", id)
		return true
	}
	return false
}

// ListAllowlist returns allow entries ordered by creation time.
func (s *LedgerStore) ListAllowlist() []*IPAllowlistEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*IPAllowlistEntry, 0, len(s.allow))
	for _, entry := range s.allow {
		copy := *entry
		out = append(out, &copy)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// IsIPAllowed checks whether ipStr is within the allowlist. Empty allowlist permits all.
func (s *LedgerStore) IsIPAllowed(ipStr string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.allow) == 0 {
		return true
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, entry := range s.allow {
		if _, network, err := net.ParseCIDR(entry.CIDR); err == nil {
			if network.Contains(ip) {
				return true
			}
			continue
		}
		if ip.Equal(net.ParseIP(entry.CIDR)) {
			return true
		}
	}
	return false
}

// RecordLogin appends an audit entry for a successful SDID login.
func (s *LedgerStore) RecordLogin(actor string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appendAuditLocked(strings.TrimSpace(actor), "login", "")
}

// AppendAudit adds an audit log entry.
func (s *LedgerStore) appendAuditLocked(actor, action, details string) {
	entry := &AuditLogEntry{
		ID:        GenerateID("audit"),
		Actor:     actor,
		Action:    action,
		Details:   details,
		CreatedAt: time.Now().UTC(),
	}
	if n := len(s.audits); n > 0 {
		entry.PrevHash = s.audits[n-1].Hash
	}
	entry.Hash = computeAuditHash(entry)
	s.audits = append(s.audits, entry)
}

// ListAudits returns stored audit entries.
func (s *LedgerStore) ListAudits() []*AuditLogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*AuditLogEntry, len(s.audits))
	for i, entry := range s.audits {
		copy := *entry
		out[i] = &copy
	}
	return out
}

func computeAuditHash(entry *AuditLogEntry) string {
	payload := fmt.Sprintf("%s|%s|%s|%s", entry.PrevHash, entry.Action, entry.Details, entry.CreatedAt.Format(time.RFC3339Nano))
	sum := sha256.Sum256([]byte(payload))
	return base64.RawStdEncoding.EncodeToString(sum[:])
}

// VerifyAuditChain recomputes the hashes and ensures the chain remains valid.
func (s *LedgerStore) VerifyAuditChain() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prev := ""
	for _, entry := range s.audits {
		expected := computeAuditHash(&AuditLogEntry{
			PrevHash:  prev,
			Action:    entry.Action,
			Details:   entry.Details,
			CreatedAt: entry.CreatedAt,
		})
		if entry.Hash != expected {
			return false
		}
		prev = entry.Hash
	}
	return true
}

func loginMessage(nonce string) string {
	return fmt.Sprintf("Sign in to RoundOne Ledger with nonce %s", nonce)
}

// CreateLoginChallenge issues a one-time nonce for SDID authentication.
func (s *LedgerStore) CreateLoginChallenge() *LoginChallenge {
	challenge := &LoginChallenge{
		Nonce:     GenerateID("nonce"),
		CreatedAt: time.Now().UTC(),
	}
	challenge.Message = loginMessage(challenge.Nonce)
	s.mu.Lock()
	s.loginChallenges[challenge.Nonce] = challenge
	s.mu.Unlock()
	return challenge
}

// ConsumeLoginChallenge removes the stored challenge for the supplied nonce.
func (s *LedgerStore) ConsumeLoginChallenge(nonce string) (*LoginChallenge, error) {
	trimmed := strings.TrimSpace(nonce)
	if trimmed == "" {
		return nil, ErrLoginChallengeNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	challenge, ok := s.loginChallenges[trimmed]
	if !ok {
		return nil, ErrLoginChallengeNotFound
	}
	delete(s.loginChallenges, trimmed)
	return challenge, nil
}

func normaliseStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}
