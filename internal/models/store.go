package models

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
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
	// ErrEnrollmentNotFound indicates there is no pending enrollment for the provided identifiers.
	ErrEnrollmentNotFound = errors.New("enrollment challenge not found")
	// ErrLoginChallengeNotFound indicates there is no nonce to fulfil.
	ErrLoginChallengeNotFound = errors.New("login challenge not found")
	// ErrFingerprintMismatch occurs when the submitted fingerprint differs from the enrolled copy.
	ErrFingerprintMismatch = errors.New("fingerprint mismatch")
	// ErrSignatureInvalid indicates that signature verification failed.
	ErrSignatureInvalid = errors.New("signature invalid")
	// ErrIPNotAllowed indicates the source IP is not within the allowlist.
	ErrIPNotAllowed = errors.New("ip not allowed")
	// ErrIPInvalid indicates the provided IP could not be parsed.
	ErrIPInvalid = errors.New("ip invalid")
	// ErrDeviceIPMismatch indicates the login IP differs from the enrolled binding.
	ErrDeviceIPMismatch = errors.New("device ip mismatch")
	// ErrAdminSignatureRequired indicates an administrator signature was not supplied.
	ErrAdminSignatureRequired = errors.New("admin signature required")
	// ErrAdminSignatureInvalid indicates the administrator signature could not be verified.
	ErrAdminSignatureInvalid = errors.New("admin signature invalid")
	// ErrAdminKeyNotConfigured indicates the administrator public key has not been provided.
	ErrAdminKeyNotConfigured = errors.New("admin signing key not configured")
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
	users   map[string]*User

	pendingEnrollments map[string]*Enrollment
	loginChallenges    map[string]*LoginChallenge

	history historyStack

	fingerprintSecret []byte
	adminPublicKey    ed25519.PublicKey
}

// NewLedgerStore constructs a ledger store with an optional fingerprint secret.
func NewLedgerStore(secret []byte) *LedgerStore {
	if len(secret) == 0 {
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			secret = []byte("default-secret")
		}
	}
	store := &LedgerStore{
		entries:            make(map[LedgerType][]LedgerEntry),
		allow:              make(map[string]*IPAllowlistEntry),
		users:              make(map[string]*User),
		pendingEnrollments: make(map[string]*Enrollment),
		loginChallenges:    make(map[string]*LoginChallenge),
		fingerprintSecret:  secret,
	}
	store.history.limit = 11
	store.history.Reset(store.snapshotLocked())
	return store
}

// SetAdminPublicKey configures the administrator's signing key used to approve devices.
func (s *LedgerStore) SetAdminPublicKey(key []byte) error {
	if len(key) != ed25519.PublicKeySize {
		return fmt.Errorf("admin public key must be %d bytes", ed25519.PublicKeySize)
	}
	copied := make([]byte, len(key))
	copy(copied, key)
	s.mu.Lock()
	s.adminPublicKey = ed25519.PublicKey(copied)
	s.mu.Unlock()
	return nil
}

// AdminPublicKey returns a defensive copy of the configured administrator key.
func (s *LedgerStore) AdminPublicKey() ed25519.PublicKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.adminPublicKey) == 0 {
		return nil
	}
	copied := make([]byte, len(s.adminPublicKey))
	copy(copied, s.adminPublicKey)
	return ed25519.PublicKey(copied)
}

// helper to produce map key for pending enrollment login challenge.
func enrollmentKey(username, deviceID string) string {
	return fmt.Sprintf("%s:%s", NormaliseUsername(username), deviceID)
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

func (s *LedgerStore) adminApprovalMessage(username, deviceID string, publicKey []byte) []byte {
	payload := fmt.Sprintf("%s:%s:%s", NormaliseUsername(username), deviceID, base64.RawStdEncoding.EncodeToString(publicKey))
	return []byte(payload)
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

func normaliseIP(ipStr string) (string, error) {
	trimmed := strings.TrimSpace(ipStr)
	if trimmed == "" {
		return "", ErrIPInvalid
	}
	parsed := net.ParseIP(trimmed)
	if parsed == nil {
		return "", ErrIPInvalid
	}
	return parsed.String(), nil
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

// UpsertUser ensures a user exists.
func (s *LedgerStore) UpsertUser(username string) *User {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getOrCreateUserLocked(username)
}

func (s *LedgerStore) getOrCreateUserLocked(username string) *User {
	username = NormaliseUsername(username)
	if user, ok := s.users[username]; ok {
		return user
	}
	user := &User{
		Username:  username,
		Devices:   make(map[string]*UserDevice),
		Roles:     []string{"admin"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	s.users[username] = user
	return user
}

// StartEnrollment registers a new enrollment challenge.
func (s *LedgerStore) StartEnrollment(username, deviceName string, publicKey []byte) (*Enrollment, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key must be %d bytes", ed25519.PublicKeySize)
	}
	user := s.UpsertUser(username)
	s.mu.Lock()
	defer s.mu.Unlock()
	deviceID := GenerateID("device")
	nonce := GenerateID("nonce")
	enrollment := &Enrollment{
		Username:   user.Username,
		DeviceID:   deviceID,
		DeviceName: deviceName,
		Nonce:      nonce,
		PublicKey:  append([]byte(nil), publicKey...),
		CreatedAt:  time.Now().UTC(),
	}
	s.pendingEnrollments[enrollmentKey(user.Username, deviceID)] = enrollment
	return enrollment, nil
}

// CompleteEnrollment verifies the device and administrator signatures, storing the device and binding the originating IP.
func (s *LedgerStore) CompleteEnrollment(username, deviceID, nonce string, signature, adminSignature []byte, fingerprint, ip string) (*UserDevice, error) {
	key := enrollmentKey(username, deviceID)
	s.mu.Lock()
	enrollment, ok := s.pendingEnrollments[key]
	if !ok || enrollment.Nonce != nonce {
		s.mu.Unlock()
		return nil, ErrEnrollmentNotFound
	}
	delete(s.pendingEnrollments, key)
	s.mu.Unlock()

	if len(signature) != ed25519.SignatureSize {
		return nil, ErrSignatureInvalid
	}
	if !ed25519.Verify(ed25519.PublicKey(enrollment.PublicKey), []byte(enrollment.Nonce), signature) {
		return nil, ErrSignatureInvalid
	}
	if len(adminSignature) == 0 {
		return nil, ErrAdminSignatureRequired
	}
	if len(adminSignature) != ed25519.SignatureSize {
		return nil, ErrAdminSignatureInvalid
	}
	adminKey := s.AdminPublicKey()
	if len(adminKey) == 0 {
		return nil, ErrAdminKeyNotConfigured
	}
	if !ed25519.Verify(adminKey, s.adminApprovalMessage(enrollment.Username, deviceID, enrollment.PublicKey), adminSignature) {
		return nil, ErrAdminSignatureInvalid
	}
	fingerprintSum := s.hashFingerprint(fingerprint)

	if ip == "" {
		return nil, ErrIPInvalid
	}
	normalisedIP, err := normaliseIP(ip)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.getOrCreateUserLocked(enrollment.Username)
	device := &UserDevice{
		ID:             deviceID,
		Name:           enrollment.DeviceName,
		PublicKey:      append([]byte(nil), enrollment.PublicKey...),
		FingerprintSum: fingerprintSum,
		BoundIP:        normalisedIP,
		AdminSignature: base64.RawStdEncoding.EncodeToString(adminSignature),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	user.Devices[deviceID] = device
	user.UpdatedAt = time.Now().UTC()
	s.appendAuditLocked(enrollment.Username, "device_enrolled", deviceID)
	return device, nil
}

// RequestLoginNonce generates a login nonce for the given device.
func (s *LedgerStore) RequestLoginNonce(username, deviceID string) (*LoginChallenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	username = NormaliseUsername(username)
	user, ok := s.users[username]
	if !ok {
		return nil, fmt.Errorf("user %s not found", username)
	}
	if _, ok := user.Devices[deviceID]; !ok {
		return nil, fmt.Errorf("device %s not enrolled", deviceID)
	}
	challenge := &LoginChallenge{
		Username:  username,
		DeviceID:  deviceID,
		Nonce:     GenerateID("nonce"),
		CreatedAt: time.Now().UTC(),
	}
	s.loginChallenges[enrollmentKey(username, deviceID)] = challenge
	return challenge, nil
}

// ValidateLogin verifies the submitted signature, administrator approval, fingerprint, and IP allowlist.
func (s *LedgerStore) ValidateLogin(username, deviceID, nonce string, signature []byte, fingerprint, ip string) (*User, error) {
	username = NormaliseUsername(username)
	key := enrollmentKey(username, deviceID)
	s.mu.Lock()
	challenge, ok := s.loginChallenges[key]
	if !ok || challenge.Nonce != nonce {
		s.mu.Unlock()
		return nil, ErrLoginChallengeNotFound
	}
	delete(s.loginChallenges, key)
	user, ok := s.users[username]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("user %s not found", username)
	}
	device, ok := user.Devices[deviceID]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("device %s not enrolled", deviceID)
	}
	s.mu.Unlock()

	if !ed25519.Verify(ed25519.PublicKey(device.PublicKey), []byte(challenge.Nonce), signature) {
		return nil, ErrSignatureInvalid
	}
	adminKey := s.AdminPublicKey()
	if len(adminKey) == 0 {
		return nil, ErrAdminKeyNotConfigured
	}
	if device.AdminSignature == "" {
		return nil, ErrAdminSignatureRequired
	}
	adminSig, err := base64.RawStdEncoding.DecodeString(device.AdminSignature)
	if err != nil {
		return nil, ErrAdminSignatureInvalid
	}
	if len(adminSig) != ed25519.SignatureSize {
		return nil, ErrAdminSignatureInvalid
	}
	if !ed25519.Verify(adminKey, s.adminApprovalMessage(user.Username, deviceID, device.PublicKey), adminSig) {
		return nil, ErrAdminSignatureInvalid
	}
	if s.hashFingerprint(fingerprint) != device.FingerprintSum {
		return nil, ErrFingerprintMismatch
	}
	normalisedIP, err := normaliseIP(ip)
	if err != nil {
		return nil, err
	}
	if !s.IsIPAllowed(normalisedIP) {
		return nil, ErrIPNotAllowed
	}
	if device.BoundIP != "" && device.BoundIP != normalisedIP {
		return nil, ErrDeviceIPMismatch
	}
	s.appendAuditLocked(user.Username, "login", deviceID)
	return user, nil
}

func (s *LedgerStore) hashFingerprint(value string) string {
	mac := hmac.New(sha256.New, s.fingerprintSecret)
	mac.Write([]byte(value))
	return base64.RawStdEncoding.EncodeToString(mac.Sum(nil))
}

// GetUser returns a user by username.
func (s *LedgerStore) GetUser(username string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[strings.ToLower(username)]
	if !ok {
		return nil, false
	}
	copy := *user
	copy.Devices = make(map[string]*UserDevice, len(user.Devices))
	for id, device := range user.Devices {
		dup := *device
		dup.PublicKey = append([]byte(nil), device.PublicKey...)
		dup.BoundIP = device.BoundIP
		copy.Devices[id] = &dup
	}
	copy.Roles = append([]string{}, user.Roles...)
	return &copy, true
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
