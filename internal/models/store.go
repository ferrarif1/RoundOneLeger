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
	// ErrIdentityNotApproved indicates that the SDID identity lacks the required administrator certification.
	ErrIdentityNotApproved = errors.New("identity_not_approved")
	// ErrApprovalAlreadyPending is returned when an approval request already exists for an identity.
	ErrApprovalAlreadyPending = errors.New("approval_pending")
	// ErrApprovalNotFound indicates the approval request cannot be located.
	ErrApprovalNotFound = errors.New("approval_not_found")
	// ErrApprovalAlreadyCompleted indicates the approval request has already been signed off.
	ErrApprovalAlreadyCompleted = errors.New("approval_already_completed")
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

	entries             map[LedgerType][]LedgerEntry
	allow               map[string]*IPAllowlistEntry
	audits              []*AuditLogEntry
	profiles            map[string]IdentityProfile
	approvals           map[string]*IdentityApproval
	approvalOrder       []*IdentityApproval
	approvalByApplicant map[string]*IdentityApproval

	loginChallenges map[string]*LoginChallenge

	history historyStack
}

// IdentityProfile stores metadata about a SDID identity that has interacted with the system.
type IdentityProfile struct {
	DID      string
	Label    string
	Roles    []string
	Admin    bool
	Approved bool
	Updated  time.Time
}

// IdentityApproval captures an approval workflow between an applicant and an administrator.
type IdentityApproval struct {
	ID                string     `json:"id"`
	ApplicantDid      string     `json:"applicantDid"`
	ApplicantLabel    string     `json:"applicantLabel"`
	ApplicantRoles    []string   `json:"applicantRoles"`
	CreatedAt         time.Time  `json:"createdAt"`
	Status            string     `json:"status"`
	ApprovedAt        *time.Time `json:"approvedAt,omitempty"`
	ApproverDid       string     `json:"approverDid,omitempty"`
	ApproverLabel     string     `json:"approverLabel,omitempty"`
	ApproverRoles     []string   `json:"approverRoles,omitempty"`
	RequestChallenge  string     `json:"requestChallenge,omitempty"`
	RequestSignature  string     `json:"requestSignature,omitempty"`
	RequestCanonical  string     `json:"requestCanonical,omitempty"`
	ApprovalChallenge string     `json:"approvalChallenge,omitempty"`
	ApprovalSignature string     `json:"approvalSignature,omitempty"`
}

// Clone returns a copy of the approval entry.
func (a *IdentityApproval) Clone() *IdentityApproval {
	if a == nil {
		return nil
	}
	clone := *a
	if a.ApplicantRoles != nil {
		clone.ApplicantRoles = append([]string{}, a.ApplicantRoles...)
	}
	if a.ApproverRoles != nil {
		clone.ApproverRoles = append([]string{}, a.ApproverRoles...)
	}
	if a.ApprovedAt != nil {
		approvedAt := *a.ApprovedAt
		clone.ApprovedAt = &approvedAt
	}
	return &clone
}

const (
	// ApprovalStatusPending represents an approval request awaiting administrator action.
	ApprovalStatusPending = "pending"
	// ApprovalStatusApproved indicates the request has been certified by an administrator.
	ApprovalStatusApproved = "approved"
	// ApprovalStatusMissing indicates that no approval request exists for the identity.
	ApprovalStatusMissing = "missing"
)

// NewLedgerStore constructs a ledger store.
func NewLedgerStore() *LedgerStore {
	store := &LedgerStore{
		entries:             make(map[LedgerType][]LedgerEntry),
		allow:               make(map[string]*IPAllowlistEntry),
		profiles:            make(map[string]IdentityProfile),
		approvals:           make(map[string]*IdentityApproval),
		approvalByApplicant: make(map[string]*IdentityApproval),
		loginChallenges:     make(map[string]*LoginChallenge),
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

// UpdateIdentityProfile upserts metadata about an identity interacting with the system.
func (s *LedgerStore) UpdateIdentityProfile(did, label string, roles []string, admin, approved bool) IdentityProfile {
	did = strings.TrimSpace(did)
	if did == "" {
		return IdentityProfile{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	normalisedRoles := normaliseStrings(roles)
	profile := IdentityProfile{
		DID:      did,
		Label:    strings.TrimSpace(label),
		Roles:    append([]string{}, normalisedRoles...),
		Admin:    admin,
		Approved: approved || admin,
		Updated:  time.Now().UTC(),
	}
	if existing, ok := s.profiles[did]; ok {
		if profile.Label == "" {
			profile.Label = existing.Label
		}
		if len(profile.Roles) == 0 {
			profile.Roles = append([]string{}, existing.Roles...)
		}
		profile.Admin = profile.Admin || existing.Admin
		profile.Approved = profile.Approved || existing.Approved || existing.Admin
	}
	s.profiles[did] = profile
	return profile
}

// IdentityProfileByDID returns a copy of the stored profile for the DID.
func (s *LedgerStore) IdentityProfileByDID(did string) IdentityProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profile, ok := s.profiles[strings.TrimSpace(did)]
	if !ok {
		return IdentityProfile{}
	}
	profileCopy := profile
	profileCopy.Roles = append([]string{}, profile.Roles...)
	return profileCopy
}

// IdentityIsAdmin reports whether the DID is recognised as an administrator.
func (s *LedgerStore) IdentityIsAdmin(did string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profile, ok := s.profiles[strings.TrimSpace(did)]
	return ok && profile.Admin
}

// IdentityApproved reports whether the DID has been approved (either by admin role or certification).
func (s *LedgerStore) IdentityApproved(did string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	trimmed := strings.TrimSpace(did)
	if trimmed == "" {
		return false
	}
	if profile, ok := s.profiles[trimmed]; ok {
		if profile.Admin || profile.Approved {
			return true
		}
	}
	if approval, ok := s.approvalByApplicant[trimmed]; ok {
		return approval.Status == ApprovalStatusApproved
	}
	return false
}

// LatestApprovalForApplicant returns the most recent approval request for a DID.
func (s *LedgerStore) LatestApprovalForApplicant(did string) *IdentityApproval {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if approval, ok := s.approvalByApplicant[strings.TrimSpace(did)]; ok {
		return approval.Clone()
	}
	return nil
}

// SubmitApproval records a pending approval request for an identity.
func (s *LedgerStore) SubmitApproval(did, label string, roles []string, challenge, signature, canonical string) (*IdentityApproval, error) {
	did = strings.TrimSpace(did)
	if did == "" {
		return nil, ErrApprovalNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.approvalByApplicant[did]; ok {
		switch existing.Status {
		case ApprovalStatusPending:
			return existing.Clone(), ErrApprovalAlreadyPending
		case ApprovalStatusApproved:
			return existing.Clone(), ErrApprovalAlreadyCompleted
		}
	}
	now := time.Now().UTC()
	approval := &IdentityApproval{
		ID:               GenerateID("approval"),
		ApplicantDid:     did,
		ApplicantLabel:   strings.TrimSpace(label),
		ApplicantRoles:   normaliseStrings(roles),
		CreatedAt:        now,
		Status:           ApprovalStatusPending,
		RequestChallenge: strings.TrimSpace(challenge),
		RequestSignature: strings.TrimSpace(signature),
		RequestCanonical: strings.TrimSpace(canonical),
	}
	s.approvals[approval.ID] = approval
	s.approvalOrder = append(s.approvalOrder, approval)
	s.approvalByApplicant[did] = approval
	profile := s.profiles[did]
	profile.DID = did
	if approval.ApplicantLabel != "" {
		profile.Label = approval.ApplicantLabel
	}
	if len(approval.ApplicantRoles) > 0 {
		profile.Roles = append([]string{}, approval.ApplicantRoles...)
	}
	profile.Approved = false
	profile.Updated = now
	s.profiles[did] = profile
	return approval.Clone(), nil
}

// ApproveRequest finalises an approval entry with administrator details.
func (s *LedgerStore) ApproveRequest(id string, approver IdentityProfile, challenge, signature string) (*IdentityApproval, error) {
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	approval, ok := s.approvals[id]
	if !ok {
		return nil, ErrApprovalNotFound
	}
	if approval.Status == ApprovalStatusApproved {
		return approval.Clone(), ErrApprovalAlreadyCompleted
	}
	now := time.Now().UTC()
	approval.Status = ApprovalStatusApproved
	approval.ApproverDid = strings.TrimSpace(approver.DID)
	approval.ApproverLabel = strings.TrimSpace(approver.Label)
	approval.ApproverRoles = normaliseStrings(approver.Roles)
	approval.ApprovalChallenge = strings.TrimSpace(challenge)
	approval.ApprovalSignature = strings.TrimSpace(signature)
	approval.ApprovedAt = &now
	s.approvalByApplicant[approval.ApplicantDid] = approval
	profile := s.profiles[approval.ApplicantDid]
	profile.DID = approval.ApplicantDid
	if approval.ApplicantLabel != "" {
		profile.Label = approval.ApplicantLabel
	}
	if len(approval.ApplicantRoles) > 0 {
		profile.Roles = append([]string{}, approval.ApplicantRoles...)
	}
	profile.Approved = true
	profile.Updated = now
	s.profiles[approval.ApplicantDid] = profile
	return approval.Clone(), nil
}

// ListApprovals returns approval requests ordered by status then recency.
func (s *LedgerStore) ListApprovals() []*IdentityApproval {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*IdentityApproval, 0, len(s.approvalOrder))
	for _, approval := range s.approvalOrder {
		out = append(out, approval.Clone())
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Status == out[j].Status {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		if out[i].Status == ApprovalStatusPending {
			return true
		}
		if out[j].Status == ApprovalStatusPending {
			return false
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

// ApprovalByID retrieves an approval request by identifier.
func (s *LedgerStore) ApprovalByID(id string) (*IdentityApproval, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return nil, ErrApprovalNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	approval, ok := s.approvals[trimmed]
	if !ok {
		return nil, ErrApprovalNotFound
	}
	return approval.Clone(), nil
}

// IdentityApprovalState returns the approval status and latest request for a DID.
func (s *LedgerStore) IdentityApprovalState(did string) (string, *IdentityApproval) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	trimmed := strings.TrimSpace(did)
	if trimmed == "" {
		return ApprovalStatusMissing, nil
	}
	if profile, ok := s.profiles[trimmed]; ok {
		if profile.Admin || profile.Approved {
			return ApprovalStatusApproved, nil
		}
	}
	if approval, ok := s.approvalByApplicant[trimmed]; ok {
		return approval.Status, approval.Clone()
	}
	return ApprovalStatusMissing, nil
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
	return fmt.Sprintf("Sign in to RoundOneLeger with nonce %s", nonce)
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
