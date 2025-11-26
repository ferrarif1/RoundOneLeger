package models

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	mrand "math/rand"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
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
	// ErrWorkspaceNotFound indicates the requested collaborative workspace does not exist.
	ErrWorkspaceNotFound = errors.New("workspace_not_found")
	// ErrWorkspaceParentInvalid indicates that a workspace parent assignment is invalid.
	ErrWorkspaceParentInvalid = errors.New("workspace_parent_invalid")
	// ErrWorkspaceKindUnsupported indicates the requested operation is not allowed for the workspace kind.
	ErrWorkspaceKindUnsupported = errors.New("workspace_kind_unsupported")
	// ErrWorkspaceVersionConflict indicates that the workspace has been modified since the last fetch.
	ErrWorkspaceVersionConflict = errors.New("workspace_version_conflict")
	// ErrUserExists indicates the username is already registered.
	ErrUserExists = errors.New("user_exists")
	// ErrUserNotFound indicates the requested user cannot be located.
	ErrUserNotFound = errors.New("user_not_found")
	// ErrInvalidCredentials indicates username or password validation failed.
	ErrInvalidCredentials = errors.New("invalid_credentials")
	// ErrUserDeleteLastAdmin prevents removal of the final administrator account.
	ErrUserDeleteLastAdmin = errors.New("cannot_delete_last_admin")
	// ErrUsernameInvalid indicates the provided username is empty or malformed.
	ErrUsernameInvalid = errors.New("username_invalid")
	// ErrPasswordTooShort indicates the provided password is shorter than the minimum length.
	ErrPasswordTooShort = errors.New("password_too_short")
	// ErrPasswordTooWeak indicates the provided password lacks the required complexity.
	ErrPasswordTooWeak = errors.New("password_too_weak")
	// ErrPasswordHashInvalid indicates that a configured password hash cannot be parsed.
	ErrPasswordHashInvalid = errors.New("password_hash_invalid")
)

var (
	seededRand   = mrand.New(mrand.NewSource(time.Now().UnixNano()))
	seededRandMu sync.Mutex
	idAlphabet   = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

const (
	defaultAdminUsername     = "hzdsz_admin"
	defaultAdminPasswordHash = "120000:okGdNKgWgWak3qj9s5FsOA:/GkUWT7QKUDXD6i5kXAq9L9S87meVZPZ2nU6IWY9tNk"
	passwordSaltBytes        = 16
	passwordKeyBytes         = 32
	passwordIterations       = 120_000

	adminPasswordEnv     = "LEDGER_ADMIN_PASSWORD"
	adminPasswordHashEnv = "LEDGER_ADMIN_PASSWORD_HASH"
)

// LedgerStore coordinates all mutable state for the in-memory implementation.
type LedgerStore struct {
	mu sync.RWMutex

	entries             map[LedgerType][]LedgerEntry
	workspaces          map[string]*Workspace
	workspaceOrder      []string
	workspaceChildren   map[string][]string
	allow               map[string]*IPAllowlistEntry
	audits              []*AuditLogEntry
	profiles            map[string]IdentityProfile
	approvals           map[string]*IdentityApproval
	approvalOrder       []*IdentityApproval
	approvalByApplicant map[string]*IdentityApproval

	loginChallenges map[string]*LoginChallenge

	users      map[string]*User
	userByName map[string]*User
	userOrder  []string

	history historyStack
}

// SnapshotVersion represents the current serialization format for persisted snapshots.
const SnapshotVersion = 2

// Snapshot captures all persisted state required to rebuild the in-memory store.
type Snapshot struct {
	Version        int                          `json:"version"`
	Entries        map[LedgerType][]LedgerEntry `json:"entries"`
	Workspaces     []*Workspace                 `json:"workspaces"`
	WorkspaceOrder []string                     `json:"workspace_order,omitempty"`
	Allowlist      []*IPAllowlistEntry          `json:"allowlist"`
	Audits         []*AuditLogEntry             `json:"audits"`
	Users          []*User                      `json:"users"`
	UserOrder      []string                     `json:"user_order,omitempty"`
	Profiles       []IdentityProfile            `json:"profiles,omitempty"`
	Approvals      []*IdentityApproval          `json:"approvals,omitempty"`
}

// OverviewStats summarizes ledger contents for the overview page.
type OverviewStats struct {
	Ledgers       []LedgerOverview  `json:"ledgers"`
	TagTop        []TagCount        `json:"tag_top"`
	Relationships RelationshipStats `json:"relationships"`
	Recent        []RecentEntry     `json:"recent"`
}

// LedgerOverview aggregates basic counts and freshness for a ledger type.
type LedgerOverview struct {
	Type        LedgerType `json:"type"`
	Count       int        `json:"count"`
	LastUpdated time.Time  `json:"last_updated,omitempty"`
}

// TagCount captures how frequently a tag appears.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// RelationshipStats reports link counts between ledger entries.
type RelationshipStats struct {
	Total    int                `json:"total"`
	ByLedger map[LedgerType]int `json:"by_ledger"`
}

// RecentEntry highlights the latest updates across all ledgers.
type RecentEntry struct {
	ID        string     `json:"id"`
	Type      LedgerType `json:"type"`
	Name      string     `json:"name"`
	UpdatedAt time.Time  `json:"updated_at"`
	Tags      []string   `json:"tags,omitempty"`
}

// ExportSnapshot returns a deep copy of the current store suitable for persistence.
func (s *LedgerStore) ExportSnapshot() *Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := &Snapshot{
		Version: SnapshotVersion,
		Entries: make(map[LedgerType][]LedgerEntry, len(s.entries)),
	}

	for typ, list := range s.entries {
		snapshot.Entries[typ] = cloneEntrySlice(list)
	}

	snapshot.WorkspaceOrder = append([]string{}, s.workspaceOrder...)
	snapshot.Workspaces = make([]*Workspace, 0, len(s.workspaces))
	seen := make(map[string]struct{}, len(s.workspaces))
	for _, id := range s.workspaceOrder {
		if workspace, ok := s.workspaces[id]; ok {
			snapshot.Workspaces = append(snapshot.Workspaces, workspace.Clone())
			seen[id] = struct{}{}
		}
	}
	for id, workspace := range s.workspaces {
		if _, ok := seen[id]; ok {
			continue
		}
		snapshot.Workspaces = append(snapshot.Workspaces, workspace.Clone())
	}

	snapshot.Allowlist = make([]*IPAllowlistEntry, 0, len(s.allow))
	for _, entry := range s.allow {
		copy := *entry
		snapshot.Allowlist = append(snapshot.Allowlist, &copy)
	}
	sort.Slice(snapshot.Allowlist, func(i, j int) bool {
		return snapshot.Allowlist[i].CreatedAt.Before(snapshot.Allowlist[j].CreatedAt)
	})

	snapshot.Audits = make([]*AuditLogEntry, len(s.audits))
	for i, audit := range s.audits {
		if audit == nil {
			continue
		}
		copy := *audit
		snapshot.Audits[i] = &copy
	}

	snapshot.UserOrder = append([]string{}, s.userOrder...)
	snapshot.Users = make([]*User, 0, len(s.userOrder))
	for _, id := range s.userOrder {
		if user, ok := s.users[id]; ok {
			copy := *user
			snapshot.Users = append(snapshot.Users, &copy)
		}
	}
	if len(snapshot.Users) < len(s.users) {
		seen := make(map[string]struct{}, len(snapshot.UserOrder))
		for _, id := range snapshot.UserOrder {
			seen[id] = struct{}{}
		}
		for id, user := range s.users {
			if _, ok := seen[id]; ok {
				continue
			}
			copy := *user
			snapshot.Users = append(snapshot.Users, &copy)
			snapshot.UserOrder = append(snapshot.UserOrder, id)
		}
	}

	snapshot.Profiles = make([]IdentityProfile, 0, len(s.profiles))
	for _, profile := range s.profiles {
		copy := profile
		copy.Roles = append([]string{}, profile.Roles...)
		snapshot.Profiles = append(snapshot.Profiles, copy)
	}
	sort.Slice(snapshot.Profiles, func(i, j int) bool { return snapshot.Profiles[i].DID < snapshot.Profiles[j].DID })

	snapshot.Approvals = make([]*IdentityApproval, 0, len(s.approvalOrder))
	for _, approval := range s.approvalOrder {
		snapshot.Approvals = append(snapshot.Approvals, approval.Clone())
	}

	return snapshot
}

func (s *LedgerStore) WriteSnapshotJSON(w io.Writer) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	buf := bufio.NewWriterSize(w, 256*1024)
	writeString := func(value string) error {
		_, err := buf.WriteString(value)
		return err
	}
	writeJSON := func(v any) error {
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		_, err = buf.Write(data)
		return err
	}
	if err := writeString(`{"version":`); err != nil {
		return err
	}
	if err := writeJSON(SnapshotVersion); err != nil {
		return err
	}
	if err := writeString(`,"entries":{`); err != nil {
		return err
	}
	writtenEntries := false
	for _, typ := range AllLedgerTypes {
		entries, ok := s.entries[typ]
		if !ok {
			continue
		}
		if writtenEntries {
			if err := writeString(","); err != nil {
				return err
			}
		} else {
			writtenEntries = true
		}
		if err := writeJSON(string(typ)); err != nil {
			return err
		}
		if err := writeString(":"); err != nil {
			return err
		}
		if err := writeString("["); err != nil {
			return err
		}
		for i, entry := range entries {
			if i > 0 {
				if err := writeString(","); err != nil {
					return err
				}
			}
			if err := writeJSON(entry); err != nil {
				return err
			}
		}
		if err := writeString("]"); err != nil {
			return err
		}
	}

	if err := writeString("}"); err != nil {
		return err
	}
	if err := writeString(`,"workspace_order":`); err != nil {
		return err
	}
	if err := writeJSON(s.workspaceOrder); err != nil {
		return err
	}
	if err := writeString(`,"workspaces":[`); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(s.workspaces))
	firstWorkspace := true
	for _, id := range s.workspaceOrder {
		workspace, ok := s.workspaces[id]
		if !ok || workspace == nil {
			continue
		}
		if !firstWorkspace {
			if err := writeString(","); err != nil {
				return err
			}
		}
		if err := writeJSON(workspace); err != nil {
			return err
		}
		firstWorkspace = false
		seen[id] = struct{}{}
	}
	for id, workspace := range s.workspaces {
		if workspace == nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		if !firstWorkspace {
			if err := writeString(","); err != nil {
				return err
			}
		}
		if err := writeJSON(workspace); err != nil {
			return err
		}
		firstWorkspace = false
	}
	if err := writeString("]"); err != nil {
		return err
	}
	allowlist := make([]*IPAllowlistEntry, 0, len(s.allow))
	for _, entry := range s.allow {
		if entry == nil {
			continue
		}
		copy := *entry
		allowlist = append(allowlist, &copy)
	}
	sort.Slice(allowlist, func(i, j int) bool {
		return allowlist[i].CreatedAt.Before(allowlist[j].CreatedAt)
	})
	if err := writeString(`,"allowlist":`); err != nil {
		return err
	}
	if err := writeJSON(allowlist); err != nil {
		return err
	}
	if err := writeString(`,"audits":`); err != nil {
		return err
	}
	audits := make([]*AuditLogEntry, len(s.audits))
	for i, audit := range s.audits {
		if audit == nil {
			continue
		}
		copy := *audit
		audits[i] = &copy
	}
	if err := writeJSON(audits); err != nil {
		return err
	}
	if err := writeString(`,"users":`); err != nil {
		return err
	}
	users := make([]*User, 0, len(s.userOrder))
	for _, id := range s.userOrder {
		if user, ok := s.users[id]; ok && user != nil {
			clone := *user
			users = append(users, &clone)
		}
	}
	if len(users) < len(s.users) {
		seenUsers := make(map[string]struct{}, len(s.userOrder))
		for _, id := range s.userOrder {
			seenUsers[id] = struct{}{}
		}
		for id, user := range s.users {
			if _, ok := seenUsers[id]; ok || user == nil {
				continue
			}
			clone := *user
			users = append(users, &clone)
		}
	}
	if err := writeJSON(users); err != nil {
		return err
	}
	if err := writeString(`,"user_order":`); err != nil {
		return err
	}
	if err := writeJSON(s.userOrder); err != nil {
		return err
	}
	profiles := make([]IdentityProfile, 0, len(s.profiles))
	for _, profile := range s.profiles {
		copy := profile
		copy.Roles = append([]string{}, profile.Roles...)
		profiles = append(profiles, copy)
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].DID < profiles[j].DID })
	if err := writeString(`,"profiles":`); err != nil {
		return err
	}
	if err := writeJSON(profiles); err != nil {
		return err
	}
	approvals := make([]*IdentityApproval, 0, len(s.approvalOrder))
	for _, approval := range s.approvalOrder {
		if approval == nil {
			continue
		}
		approvals = append(approvals, approval.Clone())
	}
	if len(approvals) == 0 {
		for _, approval := range s.approvals {
			if approval == nil {
				continue
			}
			approvals = append(approvals, approval.Clone())
		}
	}
	if err := writeString(`,"approvals":`); err != nil {
		return err
	}
	if err := writeJSON(approvals); err != nil {
		return err
	}
	if err := writeString("}"); err != nil {
		return err
	}
	return buf.Flush()
}

// ImportSnapshot replaces the in-memory state using the provided snapshot payload.
func (s *LedgerStore) ImportSnapshot(snapshot *Snapshot) error {
	if snapshot == nil {
		return errors.New("empty_snapshot")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = make(map[LedgerType][]LedgerEntry, len(snapshot.Entries))
	for typ, list := range snapshot.Entries {
		s.entries[typ] = cloneEntrySlice(list)
	}

	s.workspaces = make(map[string]*Workspace)
	s.workspaceChildren = make(map[string][]string)

	workspaceSource := make(map[string]*Workspace, len(snapshot.Workspaces))
	for _, ws := range snapshot.Workspaces {
		if ws == nil || strings.TrimSpace(ws.ID) == "" {
			continue
		}
		workspaceSource[ws.ID] = ws
	}

	order := snapshot.WorkspaceOrder
	if len(order) == 0 {
		order = make([]string, 0, len(workspaceSource))
		for id := range workspaceSource {
			order = append(order, id)
		}
		sort.Strings(order)
	}
	s.workspaceOrder = make([]string, 0, len(order))
	seen := make(map[string]struct{}, len(order))
	for _, id := range order {
		ws, ok := workspaceSource[id]
		if !ok {
			continue
		}
		clone := ws.Clone()
		s.workspaces[clone.ID] = clone
		s.workspaceOrder = append(s.workspaceOrder, clone.ID)
		if parent := strings.TrimSpace(clone.ParentID); parent != "" {
			s.workspaceChildren[parent] = append(s.workspaceChildren[parent], clone.ID)
		}
		seen[id] = struct{}{}
	}
	for id, ws := range workspaceSource {
		if _, ok := seen[id]; ok {
			continue
		}
		clone := ws.Clone()
		s.workspaces[clone.ID] = clone
		s.workspaceOrder = append(s.workspaceOrder, clone.ID)
		if parent := strings.TrimSpace(clone.ParentID); parent != "" {
			s.workspaceChildren[parent] = append(s.workspaceChildren[parent], clone.ID)
		}
	}

	s.allow = make(map[string]*IPAllowlistEntry, len(snapshot.Allowlist))
	for _, entry := range snapshot.Allowlist {
		if entry == nil || strings.TrimSpace(entry.ID) == "" {
			continue
		}
		copy := *entry
		s.allow[copy.ID] = &copy
	}

	s.audits = make([]*AuditLogEntry, 0, len(snapshot.Audits))
	for _, audit := range snapshot.Audits {
		if audit == nil {
			continue
		}
		copy := *audit
		s.audits = append(s.audits, &copy)
	}

	s.users = make(map[string]*User)
	s.userByName = make(map[string]*User)

	userSource := make(map[string]*User, len(snapshot.Users))
	for _, user := range snapshot.Users {
		if user == nil || strings.TrimSpace(user.ID) == "" {
			continue
		}
		copy := *user
		userSource[copy.ID] = &copy
	}

	userOrder := snapshot.UserOrder
	if len(userOrder) == 0 {
		userOrder = make([]string, 0, len(userSource))
		for id := range userSource {
			userOrder = append(userOrder, id)
		}
		sort.Strings(userOrder)
	}

	s.userOrder = make([]string, 0, len(userOrder))
	for _, id := range userOrder {
		user, ok := userSource[id]
		if !ok {
			continue
		}
		s.users[id] = user
		s.userOrder = append(s.userOrder, id)
		s.userByName[normalizeUsername(user.Username)] = user
		delete(userSource, id)
	}
	for id, user := range userSource {
		s.users[id] = user
		s.userOrder = append(s.userOrder, id)
		s.userByName[normalizeUsername(user.Username)] = user
	}

	s.profiles = make(map[string]IdentityProfile, len(snapshot.Profiles))
	for _, profile := range snapshot.Profiles {
		trimmed := strings.TrimSpace(profile.DID)
		if trimmed == "" {
			continue
		}
		copy := profile
		copy.DID = trimmed
		copy.Roles = append([]string{}, profile.Roles...)
		s.profiles[trimmed] = copy
	}

	s.approvals = make(map[string]*IdentityApproval)
	s.approvalByApplicant = make(map[string]*IdentityApproval)
	s.approvalOrder = make([]*IdentityApproval, 0, len(snapshot.Approvals))
	for _, approval := range snapshot.Approvals {
		if approval == nil || strings.TrimSpace(approval.ID) == "" {
			continue
		}
		clone := approval.Clone()
		s.approvals[clone.ID] = clone
		s.approvalOrder = append(s.approvalOrder, clone)
		if did := strings.TrimSpace(clone.ApplicantDid); did != "" {
			s.approvalByApplicant[did] = clone
		}
	}

	s.loginChallenges = make(map[string]*LoginChallenge)

	if err := s.ensureDefaultAdminLocked(); err != nil {
		return err
	}

	s.history.Reset(s.snapshotLocked())
	return nil
}

// ImportSnapshotMerge merges snapshot data into current state (ID-based replace + append).
func (s *LedgerStore) ImportSnapshotMerge(snapshot *Snapshot) error {
	if snapshot == nil {
		return errors.New("empty_snapshot")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// Merge ledger entries
	for typ, list := range snapshot.Entries {
		existing := s.entries[typ]
		index := make(map[string]LedgerEntry, len(existing))
		for _, e := range existing {
			index[e.ID] = e
		}
		for _, incoming := range list {
			index[incoming.ID] = incoming
		}
		merged := make([]LedgerEntry, 0, len(index))
		// keep existing order; append new ones
		seen := make(map[string]struct{})
		for _, e := range existing {
			if updated, ok := index[e.ID]; ok {
				merged = append(merged, updated)
				seen[e.ID] = struct{}{}
			}
		}
		for _, e := range index {
			if _, ok := seen[e.ID]; !ok {
				merged = append(merged, e)
			}
		}
		sort.Slice(merged, func(i, j int) bool { return merged[i].Order < merged[j].Order })
		s.entries[typ] = merged
	}

	// Merge workspaces by ID
	for _, ws := range snapshot.Workspaces {
		if ws == nil || strings.TrimSpace(ws.ID) == "" {
			continue
		}
		id := strings.TrimSpace(ws.ID)
		clone := ws.Clone()
		if existing, ok := s.workspaces[id]; ok {
			if clone.Version <= existing.Version {
				clone.Version = existing.Version + 1
			}
		}
		s.workspaces[id] = clone
	}
	// Merge workspace order (preserve existing order, append new ids)
	existingOrder := make(map[string]struct{}, len(s.workspaceOrder))
	for _, id := range s.workspaceOrder {
		existingOrder[id] = struct{}{}
	}
	for _, id := range snapshot.WorkspaceOrder {
		if strings.TrimSpace(id) != "" {
			if _, ok := existingOrder[id]; !ok {
				s.workspaceOrder = append(s.workspaceOrder, id)
				existingOrder[id] = struct{}{}
			}
		}
	}
	// rebuild children
	s.workspaceChildren = make(map[string][]string)
	for _, ws := range s.workspaces {
		parent := strings.TrimSpace(ws.ParentID)
		s.workspaceChildren[parent] = append(s.workspaceChildren[parent], ws.ID)
	}

	// Allowlist merge
	for _, entry := range snapshot.Allowlist {
		if entry == nil || strings.TrimSpace(entry.ID) == "" {
			continue
		}
		copy := *entry
		s.allow[copy.ID] = &copy
	}

	// Audits append
	for _, audit := range snapshot.Audits {
		if audit == nil {
			continue
		}
		copy := *audit
		s.audits = append(s.audits, &copy)
	}

	// Users merge
	for _, user := range snapshot.Users {
		if user == nil || strings.TrimSpace(user.ID) == "" {
			continue
		}
		copy := *user
		s.users[copy.ID] = &copy
		s.userByName[normalizeUsername(copy.Username)] = &copy
	}
	existingUserOrder := make(map[string]struct{}, len(s.userOrder))
	for _, id := range s.userOrder {
		existingUserOrder[id] = struct{}{}
	}
	for _, id := range snapshot.UserOrder {
		if _, ok := existingUserOrder[id]; !ok {
			s.userOrder = append(s.userOrder, id)
		}
	}

	// Profiles merge
	for _, profile := range snapshot.Profiles {
		trimmed := strings.TrimSpace(profile.DID)
		if trimmed == "" {
			continue
		}
		copy := profile
		copy.DID = trimmed
		copy.Roles = append([]string{}, profile.Roles...)
		s.profiles[trimmed] = copy
	}

	// Approvals merge
	for _, approval := range snapshot.Approvals {
		if approval == nil || strings.TrimSpace(approval.ID) == "" {
			continue
		}
		clone := approval.Clone()
		s.approvals[clone.ID] = clone
		s.approvalByApplicant[strings.TrimSpace(clone.ApplicantDid)] = clone
	}
	s.approvalOrder = make([]*IdentityApproval, 0, len(s.approvals))
	for _, approval := range s.approvals {
		if approval != nil {
			s.approvalOrder = append(s.approvalOrder, approval)
		}
	}
	sort.Slice(s.approvalOrder, func(i, j int) bool { return s.approvalOrder[i].CreatedAt.Before(s.approvalOrder[j].CreatedAt) })

	s.history.Reset(s.snapshotLocked())
	return nil
}

// SaveTo persists a snapshot.json file atomically in dir.
func (s *LedgerStore) SaveTo(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return errors.New("empty_dir")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(dir, "snapshot.tmp")
	file := filepath.Join(dir, "snapshot.json")
	if err := func() error {
		fh, err := os.Create(tmp)
		if err != nil {
			return err
		}
		defer fh.Close()
		if err := s.WriteSnapshotJSON(fh); err != nil {
			return err
		}
		return fh.Sync()
	}(); err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp) }()
	if err := os.Rename(tmp, file); err != nil {
		if removeErr := os.Remove(file); removeErr != nil && !os.IsNotExist(removeErr) {
			return err
		}
		if retryErr := os.Rename(tmp, file); retryErr != nil {
			return retryErr
		}
	}
	return nil
}

// LoadFrom restores store state from snapshot.json in dir when present.
func (s *LedgerStore) LoadFrom(dir string) error {
	path := filepath.Join(dir, "snapshot.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}
	return s.ImportSnapshot(&snap)
}

// SaveToWithRetention persists the current state and a timestamped backup, pruning old files.
func (s *LedgerStore) SaveToWithRetention(dir string, retention int) error {
	if err := s.SaveTo(dir); err != nil {
		return err
	}
	if retention <= 0 {
		return nil
	}
	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z07-00")
	backup := filepath.Join(dir, "snapshot-"+ts+".json")
	if err := func() error {
		fh, err := os.Create(backup)
		if err != nil {
			return err
		}
		defer fh.Close()
		if err := s.WriteSnapshotJSON(fh); err != nil {
			return err
		}
		return fh.Sync()
	}(); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	backups := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "snapshot-") && strings.HasSuffix(name, ".json") {
			backups = append(backups, filepath.Join(dir, name))
		}
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i] > backups[j] })
	if len(backups) > retention {
		for _, stale := range backups[retention:] {
			_ = os.Remove(stale)
		}
	}
	return nil
}

// LoadFromDatabase restores state from the latest snapshot row and reports whether one was found.
func (s *LedgerStore) LoadFromDatabase(db *sql.DB) (bool, error) {
	if db == nil {
		return false, errors.New("database_not_configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ensureSnapshotTable(ctx, db); err != nil {
		return false, err
	}
	var payload []byte
	err := db.QueryRowContext(ctx, `SELECT payload FROM snapshots ORDER BY created_at DESC, id DESC LIMIT 1`).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var snap Snapshot
	if err := json.Unmarshal(payload, &snap); err != nil {
		return false, err
	}
	return true, s.ImportSnapshot(&snap)
}

// SaveToDatabaseWithRetention writes a snapshot row and prunes older ones.
func (s *LedgerStore) SaveToDatabaseWithRetention(db *sql.DB, retention int) error {
	if db == nil {
		return errors.New("database_not_configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ensureSnapshotTable(ctx, db); err != nil {
		return err
	}
	snapshot := s.ExportSnapshot()
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO snapshots (payload) VALUES ($1)`, payload); err != nil {
		_ = tx.Rollback()
		return err
	}
	if retention > 0 {
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM snapshots
			WHERE id NOT IN (
				SELECT id FROM snapshots ORDER BY created_at DESC, id DESC LIMIT $1
			)`, retention); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func ensureSnapshotTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS snapshots (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			payload JSONB NOT NULL
		)
	`)
	return err
}

// WriteBinary persists arbitrary data inside an assets directory under dir.
func (s *LedgerStore) WriteBinary(dir, name string, data []byte) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return "", errors.New("empty_dir")
	}
	if name == "" {
		name = GenerateID("asset")
	}
	assetsDir := filepath.Join(dir, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return "", err
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(filepath.Base(name), ext)
	filename := base + ext
	target := filepath.Join(assetsDir, filename)
	if err := os.WriteFile(target, data, fs.FileMode(0o644)); err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join("assets", filename)), nil
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
		workspaces:          make(map[string]*Workspace),
		workspaceChildren:   make(map[string][]string),
		allow:               make(map[string]*IPAllowlistEntry),
		profiles:            make(map[string]IdentityProfile),
		approvals:           make(map[string]*IdentityApproval),
		approvalByApplicant: make(map[string]*IdentityApproval),
		loginChallenges:     make(map[string]*LoginChallenge),
		users:               make(map[string]*User),
		userByName:          make(map[string]*User),
	}
	store.history.limit = 11
	store.history.Reset(store.snapshotLocked())
	if err := store.ensureDefaultAdmin(); err != nil {
		panic(fmt.Sprintf("failed to seed default admin: %v", err))
	}
	return store
}

func (s *LedgerStore) ensureDefaultAdmin() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ensureDefaultAdminLocked()
}

// ensureDefaultAdminLocked seeds the default admin while assuming the mutex is already held.
func (s *LedgerStore) ensureDefaultAdminLocked() error {
	normalized := normalizeUsername(defaultAdminUsername)
	if normalized == "" {
		return ErrUsernameInvalid
	}
	hash, err := resolveDefaultAdminPasswordHash()
	if err != nil {
		return err
	}
	if existing, exists := s.userByName[normalized]; exists {
		if existing.PasswordHash != hash {
			existing.PasswordHash = hash
			existing.UpdatedAt = time.Now().UTC()
			s.appendAuditLocked("system", "user_seed_reset", existing.ID)
		}
		return nil
	}
	now := time.Now().UTC()
	user := &User{
		ID:           GenerateID("user"),
		Username:     defaultAdminUsername,
		Admin:        true,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.users[user.ID] = user
	s.userByName[normalized] = user
	s.userOrder = append(s.userOrder, user.ID)
	s.appendAuditLocked("system", "user_seed", user.ID)
	return nil
}

// DefaultAdminActive reports whether the seeded default admin still exists.
func (s *LedgerStore) DefaultAdminActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	normalized := normalizeUsername(defaultAdminUsername)
	_, exists := s.userByName[normalized]
	return exists
}

// ListUsers returns all registered users in creation order.
func (s *LedgerStore) ListUsers() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*User, 0, len(s.userOrder))
	for _, id := range s.userOrder {
		if user, ok := s.users[id]; ok {
			out = append(out, user.Clone())
		}
	}
	return out
}

// CreateUser registers a new operator account.
func (s *LedgerStore) CreateUser(username, password string, admin bool, actor string) (*User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, ErrUsernameInvalid
	}
	normalized := normalizeUsername(username)
	if normalized == "" {
		return nil, ErrUsernameInvalid
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.userByName[normalized]; exists {
		return nil, ErrUserExists
	}
	user := &User{
		ID:           GenerateID("user"),
		Username:     username,
		Admin:        admin,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.users[user.ID] = user
	s.userByName[normalized] = user
	s.userOrder = append(s.userOrder, user.ID)
	s.appendAuditLocked(strings.TrimSpace(actor), "user_create", user.ID)
	return user.Clone(), nil
}

// DeleteUser removes the specified user unless it would orphan the system without administrators.
func (s *LedgerStore) DeleteUser(id string, actor string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return ErrUserNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[trimmed]
	if !ok {
		return ErrUserNotFound
	}
	if user.Admin {
		adminCount := 0
		for _, candidate := range s.users {
			if candidate != nil && candidate.Admin {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return ErrUserDeleteLastAdmin
		}
	}
	delete(s.userByName, normalizeUsername(user.Username))
	delete(s.users, trimmed)
	filtered := s.userOrder[:0]
	for _, existing := range s.userOrder {
		if existing != trimmed {
			filtered = append(filtered, existing)
		}
	}
	s.userOrder = filtered
	s.appendAuditLocked(strings.TrimSpace(actor), "user_delete", trimmed)
	return nil
}

// ChangePassword updates the password for the specified user when the current password matches.
func (s *LedgerStore) ChangePassword(username, oldPassword, newPassword, actor string) error {
	normalized := normalizeUsername(username)
	if normalized == "" {
		return ErrUsernameInvalid
	}
	oldPassword = strings.TrimSpace(oldPassword)
	newPassword = strings.TrimSpace(newPassword)
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.userByName[normalized]
	if !ok {
		return ErrUserNotFound
	}
	if !verifyPassword(user.PasswordHash, oldPassword) {
		return ErrInvalidCredentials
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hash
	user.UpdatedAt = time.Now().UTC()
	s.appendAuditLocked(strings.TrimSpace(actor), "user_password_change", user.ID)
	return nil
}

// AuthenticateUser validates a username/password combination.
func (s *LedgerStore) AuthenticateUser(username, password string) (*User, error) {
	normalized := normalizeUsername(username)
	if normalized == "" {
		return nil, ErrInvalidCredentials
	}
	s.mu.RLock()
	user, ok := s.userByName[normalized]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrInvalidCredentials
	}
	if !verifyPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}
	return user.Clone(), nil
}

// IsUserAdmin reports whether the provided username maps to an administrator.
func (s *LedgerStore) IsUserAdmin(username string) bool {
	normalized := normalizeUsername(username)
	if normalized == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.userByName[normalized]
	if !ok {
		return false
	}
	return user.Admin
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

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func validatePassword(password string) error {
	if len(password) < 10 {
		return ErrPasswordTooShort
	}
	if strings.TrimSpace(password) == "" {
		return ErrPasswordTooShort
	}
	var hasUpper, hasLower, hasNumber, hasSymbol bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasNumber = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}
	if !hasUpper || !hasLower || !hasNumber || !hasSymbol {
		return ErrPasswordTooWeak
	}
	return nil
}

func resolveDefaultAdminPasswordHash() (string, error) {
	if hash := strings.TrimSpace(os.Getenv(adminPasswordHashEnv)); hash != "" {
		if !isSupportedPasswordHash(hash) {
			return "", ErrPasswordHashInvalid
		}
		return hash, nil
	}
	if password := strings.TrimSpace(os.Getenv(adminPasswordEnv)); password != "" {
		if err := validatePassword(password); err != nil {
			return "", err
		}
		hash, err := hashPassword(password)
		if err != nil {
			return "", err
		}
		return hash, nil
	}
	return defaultAdminPasswordHash, nil
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	derived := pbkdf2Key([]byte(password), salt, passwordIterations, passwordKeyBytes)
	if len(derived) == 0 {
		return "", errors.New("derive_failed")
	}
	return fmt.Sprintf(
		"%d:%s:%s",
		passwordIterations,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(derived),
	), nil
}

func verifyPassword(hash, password string) bool {
	parts := strings.Split(hash, ":")
	switch len(parts) {
	case 3:
		iterations, err := strconv.Atoi(parts[0])
		if err != nil || iterations <= 0 {
			return false
		}
		salt, err := base64.RawStdEncoding.DecodeString(parts[1])
		if err != nil {
			return false
		}
		digest, err := base64.RawStdEncoding.DecodeString(parts[2])
		if err != nil {
			return false
		}
		derived := pbkdf2Key([]byte(password), salt, iterations, len(digest))
		if len(derived) != len(digest) {
			return false
		}
		return subtle.ConstantTimeCompare(derived, digest) == 1
	case 2:
		// Backwards-compatibility for legacy SHA-256 salted hashes.
		salt, err := base64.RawStdEncoding.DecodeString(parts[0])
		if err != nil {
			return false
		}
		digest, err := base64.RawStdEncoding.DecodeString(parts[1])
		if err != nil {
			return false
		}
		sum := sha256.Sum256(append(salt, []byte(password)...))
		return subtle.ConstantTimeCompare(sum[:], digest) == 1
	default:
		return false
	}
}

func pbkdf2Key(password, salt []byte, iterations, length int) []byte {
	if iterations <= 0 || length <= 0 {
		return nil
	}
	hashLen := sha256.Size
	blocks := (length + hashLen - 1) / hashLen
	derived := make([]byte, blocks*hashLen)
	mac := hmac.New(sha256.New, password)
	var counter [4]byte
	for i := 1; i <= blocks; i++ {
		mac.Reset()
		mac.Write(salt)
		counter[0] = byte(i >> 24)
		counter[1] = byte(i >> 16)
		counter[2] = byte(i >> 8)
		counter[3] = byte(i)
		mac.Write(counter[:])
		u := mac.Sum(nil)
		block := make([]byte, len(u))
		copy(block, u)
		for j := 1; j < iterations; j++ {
			mac.Reset()
			mac.Write(u)
			u = mac.Sum(nil)
			for k := 0; k < len(block); k++ {
				block[k] ^= u[k]
			}
		}
		offset := (i - 1) * hashLen
		copy(derived[offset:], block)
	}
	return derived[:length]
}

func isSupportedPasswordHash(hash string) bool {
	parts := strings.Split(hash, ":")
	switch len(parts) {
	case 3:
		if _, err := strconv.Atoi(parts[0]); err != nil {
			return false
		}
		if _, err := base64.RawStdEncoding.DecodeString(parts[1]); err != nil {
			return false
		}
		if _, err := base64.RawStdEncoding.DecodeString(parts[2]); err != nil {
			return false
		}
		return true
	case 2:
		if _, err := base64.RawStdEncoding.DecodeString(parts[0]); err != nil {
			return false
		}
		if _, err := base64.RawStdEncoding.DecodeString(parts[1]); err != nil {
			return false
		}
		return true
	default:
		return false
	}
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

// ListEntriesPaged returns a slice for the requested page and total count.
func (s *LedgerStore) ListEntriesPaged(typ LedgerType, page, pageSize int) ([]LedgerEntry, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := s.entries[typ]
	total := len(entries)
	if pageSize <= 0 {
		return nil, total
	}
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if start >= total {
		return []LedgerEntry{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	slice := entries[start:end]
	cloned := make([]LedgerEntry, len(slice))
	for i, e := range slice {
		cloned[i] = e.Clone()
	}
	sort.Slice(cloned, func(i, j int) bool { return cloned[i].Order < cloned[j].Order })
	return cloned, total
}

// OverviewStats aggregates counts, tags, links, and recents for dashboards.
func (s *LedgerStore) OverviewStats() OverviewStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := OverviewStats{
		Relationships: RelationshipStats{
			ByLedger: make(map[LedgerType]int),
		},
	}

	tagCounts := make(map[string]int)
	recents := make([]RecentEntry, 0, 16)

	for _, typ := range AllLedgerTypes {
		entries := s.entries[typ]
		overview := LedgerOverview{Type: typ, Count: len(entries)}
		var newest time.Time

		for _, entry := range entries {
			if entry.UpdatedAt.After(newest) {
				newest = entry.UpdatedAt
			}
			for _, tag := range entry.Tags {
				normalized := strings.ToLower(strings.TrimSpace(tag))
				if normalized == "" {
					continue
				}
				tagCounts[normalized]++
			}
			for linkType, ids := range entry.Links {
				stats.Relationships.ByLedger[linkType] += len(ids)
				stats.Relationships.Total += len(ids)
			}
			recents = append(recents, RecentEntry{
				ID:        entry.ID,
				Type:      typ,
				Name:      entry.Name,
				UpdatedAt: entry.UpdatedAt,
				Tags:      append([]string{}, entry.Tags...),
			})
		}

		if !newest.IsZero() {
			overview.LastUpdated = newest
		}
		stats.Ledgers = append(stats.Ledgers, overview)
	}

	tagList := make([]TagCount, 0, len(tagCounts))
	for tag, count := range tagCounts {
		tagList = append(tagList, TagCount{Tag: tag, Count: count})
	}
	sort.Slice(tagList, func(i, j int) bool {
		if tagList[i].Count == tagList[j].Count {
			return tagList[i].Tag < tagList[j].Tag
		}
		return tagList[i].Count > tagList[j].Count
	})
	if len(tagList) > 12 {
		tagList = tagList[:12]
	}
	stats.TagTop = tagList

	sort.Slice(recents, func(i, j int) bool {
		return recents[i].UpdatedAt.After(recents[j].UpdatedAt)
	})
	if len(recents) > 10 {
		recents = recents[:10]
	}
	stats.Recent = recents

	return stats
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

// AppendEntries appends entries with new IDs and timestamps.
func (s *LedgerStore) AppendEntries(typ LedgerType, entries []LedgerEntry, actor string) []LedgerEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	start := len(s.entries[typ])
	now := time.Now().UTC()
	added := make([]LedgerEntry, len(entries))
	for i, entry := range entries {
		entry.ID = GenerateID(string(typ))
		entry.Order = start + i
		entry.Tags = normaliseStrings(entry.Tags)
		entry.CreatedAt = now
		entry.UpdatedAt = now
		added[i] = entry.Clone()
		s.entries[typ] = append(s.entries[typ], added[i])
	}
	s.appendAuditLocked(actor, fmt.Sprintf("append_%s", typ), fmt.Sprintf("count=%d", len(entries)))
	s.commitLocked()
	return added
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

// WorkspaceUpdate contains optional updates applied to a workspace.
type WorkspaceUpdate struct {
	Name            string
	Document        string
	Columns         []WorkspaceColumn
	Rows            []WorkspaceRow
	ExpectedVersion int
	SetName         bool
	SetDocument     bool
	SetColumns      bool
	SetRows         bool
	ParentID        string
	SetParent       bool
}

// WorkspaceReorder defines a sibling ordering update under a parent.
type WorkspaceReorder struct {
	ParentID   string
	OrderedIDs []string
}

// ListWorkspaces returns the collaborative workspaces in creation order.
func (s *LedgerStore) ListWorkspaces() []*Workspace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Workspace, 0, len(s.workspaceOrder))
	for _, id := range s.workspaceOrder {
		if workspace, ok := s.workspaces[id]; ok {
			out = append(out, workspace.Clone())
		}
	}
	return out
}

// GetWorkspace retrieves a workspace by ID.
func (s *LedgerStore) GetWorkspace(id string) (*Workspace, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	workspace, ok := s.workspaces[strings.TrimSpace(id)]
	if !ok {
		return nil, ErrWorkspaceNotFound
	}
	return workspace.Clone(), nil
}

// CreateWorkspace adds a new collaborative workspace to the store.
func (s *LedgerStore) CreateWorkspace(name string, kind WorkspaceKind, parentID string, columns []WorkspaceColumn, rows []WorkspaceRow, document string, actor string) (*Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	normalizedKind := NormalizeWorkspaceKind(kind)
	parent := strings.TrimSpace(parentID)
	if err := s.validateWorkspaceParentLocked(parent, ""); err != nil {
		return nil, err
	}
	workspace := &Workspace{
		ID:        GenerateID("ws"),
		Name:      sanitizeWorkspaceName(name),
		Kind:      normalizedKind,
		ParentID:  parent,
		Version:   1,
		Document:  strings.TrimSpace(document),
		CreatedAt: now,
		UpdatedAt: now,
	}

	switch normalizedKind {
	case WorkspaceKindSheet:
		normalizedColumns := normalizeWorkspaceColumns(columns)
		if len(normalizedColumns) == 0 {
			normalizedColumns = []WorkspaceColumn{}
		}
		normalizedRows := normalizeWorkspaceRows(rows, normalizedColumns, now)
		workspace.Columns = normalizedColumns
		workspace.Rows = normalizedRows
	case WorkspaceKindDocument:
		workspace.Columns = []WorkspaceColumn{}
		workspace.Rows = []WorkspaceRow{}
		workspace.Document = strings.TrimSpace(document)
	case WorkspaceKindFolder:
		workspace.Columns = []WorkspaceColumn{}
		workspace.Rows = []WorkspaceRow{}
		workspace.Document = ""
	}

	s.workspaces[workspace.ID] = workspace
	s.workspaceOrder = append(s.workspaceOrder, workspace.ID)
	s.addWorkspaceChildLocked(parent, workspace.ID)
	s.appendAuditLocked(actor, "workspace_create", workspace.ID)
	return workspace.Clone(), nil
}

// UpdateWorkspace applies the provided updates to an existing workspace.
func (s *LedgerStore) UpdateWorkspace(id string, update WorkspaceUpdate, actor string) (*Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspace, ok := s.workspaces[strings.TrimSpace(id)]
	if !ok {
		return nil, ErrWorkspaceNotFound
	}
	if update.ExpectedVersion > 0 && workspace.Version != update.ExpectedVersion {
		return nil, ErrWorkspaceVersionConflict
	}

	now := time.Now().UTC()
	workspace.Kind = NormalizeWorkspaceKind(workspace.Kind)

	if update.SetName {
		workspace.Name = sanitizeWorkspaceName(update.Name)
	}
	if update.SetDocument {
		if !WorkspaceKindSupportsDocument(workspace.Kind) {
			return nil, ErrWorkspaceKindUnsupported
		}
		workspace.Document = strings.TrimSpace(update.Document)
	}
	if update.SetColumns {
		if !WorkspaceKindSupportsTable(workspace.Kind) {
			return nil, ErrWorkspaceKindUnsupported
		}
		normalized := normalizeWorkspaceColumns(update.Columns)
		workspace.Columns = normalized
		workspace.Rows = normalizeWorkspaceRows(workspace.Rows, normalized, now)
	}
	if update.SetRows {
		if !WorkspaceKindSupportsTable(workspace.Kind) {
			return nil, ErrWorkspaceKindUnsupported
		}
		workspace.Rows = normalizeWorkspaceRows(update.Rows, workspace.Columns, now)
	}
	if update.SetParent {
		newParent := strings.TrimSpace(update.ParentID)
		if err := s.validateWorkspaceParentLocked(newParent, workspace.ID); err != nil {
			return nil, err
		}
		if newParent != workspace.ParentID {
			s.removeWorkspaceChildLocked(workspace.ParentID, workspace.ID)
			workspace.ParentID = newParent
			s.addWorkspaceChildLocked(newParent, workspace.ID)
		}
	}

	workspace.Version++
	workspace.UpdatedAt = now
	s.workspaces[workspace.ID] = workspace
	s.appendAuditLocked(actor, "workspace_update", workspace.ID)
	return workspace.Clone(), nil
}

// ReorderWorkspaces updates the order of workspaces under a parent (empty for root).
func (s *LedgerStore) ReorderWorkspaces(parentID string, orderedIDs []string, actor string) error {
	parent := strings.TrimSpace(parentID)
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate parent
	if parent != "" {
		parentWS, ok := s.workspaces[parent]
		if !ok || NormalizeWorkspaceKind(parentWS.Kind) != WorkspaceKindFolder {
			return ErrWorkspaceParentInvalid
		}
	}

	// Collect children under parent according to current order.
	expected := make([]string, 0)
	for _, id := range s.workspaceOrder {
		ws, ok := s.workspaces[id]
		if !ok {
			continue
		}
		if strings.TrimSpace(ws.ParentID) == parent {
			expected = append(expected, id)
		}
	}
	if len(expected) != len(orderedIDs) {
		return ErrWorkspaceParentInvalid
	}
	childSet := make(map[string]struct{}, len(expected))
	for _, id := range expected {
		childSet[id] = struct{}{}
	}
	for _, id := range orderedIDs {
		if _, ok := childSet[id]; !ok {
			return ErrWorkspaceParentInvalid
		}
	}

	// Rewrite workspaceOrder preserving other items.
	newOrder := make([]string, 0, len(s.workspaceOrder))
	inserted := false
	for _, id := range s.workspaceOrder {
		if _, ok := childSet[id]; ok {
			if !inserted {
				newOrder = append(newOrder, orderedIDs...)
				inserted = true
			}
			continue
		}
		newOrder = append(newOrder, id)
	}
	s.workspaceOrder = newOrder
	if parent != "" {
		s.workspaceChildren[parent] = append([]string{}, orderedIDs...)
	}
	s.appendAuditLocked(actor, "workspace_reorder", parent)
	return nil
}

// DeleteWorkspace removes a workspace and its data.
func (s *LedgerStore) DeleteWorkspace(id string, actor string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return ErrWorkspaceNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.workspaces[trimmed]; !ok {
		return ErrWorkspaceNotFound
	}
	idsToRemove := make([]string, 0, 1)
	s.collectWorkspaceDescendantsLocked(trimmed, &idsToRemove)
	removalSet := make(map[string]struct{}, len(idsToRemove))
	for _, removeID := range idsToRemove {
		removalSet[removeID] = struct{}{}
		ws := s.workspaces[removeID]
		if ws != nil {
			s.removeWorkspaceChildLocked(ws.ParentID, removeID)
		}
		delete(s.workspaceChildren, removeID)
		delete(s.workspaces, removeID)
	}
	filtered := s.workspaceOrder[:0]
	for _, existing := range s.workspaceOrder {
		if _, skip := removalSet[existing]; skip {
			continue
		}
		filtered = append(filtered, existing)
	}
	s.workspaceOrder = filtered
	s.appendAuditLocked(actor, "workspace_delete", trimmed)
	return nil
}

// ReplaceWorkspaceData overwrites the table content with provided headers and rows.
func (s *LedgerStore) ReplaceWorkspaceData(id string, headers []string, records [][]string, actor string, expectedVersion int) (*Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspace, ok := s.workspaces[strings.TrimSpace(id)]
	if !ok {
		return nil, ErrWorkspaceNotFound
	}
	if !WorkspaceKindSupportsTable(workspace.Kind) {
		return nil, ErrWorkspaceKindUnsupported
	}
	if expectedVersion > 0 && workspace.Version != expectedVersion {
		return nil, ErrWorkspaceVersionConflict
	}

	now := time.Now().UTC()
	normalizedHeaders := sanitizeHeaders(headers, records)
	columns := make([]WorkspaceColumn, len(normalizedHeaders))
	for i, title := range normalizedHeaders {
		columns[i] = WorkspaceColumn{ID: GenerateID("col"), Title: title}
	}

	rows := make([]WorkspaceRow, 0, len(records))
	for _, record := range records {
		cells := make(map[string]string, len(columns))
		for idx, column := range columns {
			if idx < len(record) {
				cells[column.ID] = strings.TrimSpace(record[idx])
			} else {
				cells[column.ID] = ""
			}
		}
		rows = append(rows, WorkspaceRow{
			ID:        GenerateID("row"),
			Cells:     cells,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	workspace.Columns = columns
	workspace.Rows = rows
	workspace.Version++
	workspace.UpdatedAt = now

	s.workspaces[workspace.ID] = workspace
	s.appendAuditLocked(actor, "workspace_import", workspace.ID)
	return workspace.Clone(), nil
}

// AppendWorkspaceData appends rows to a sheet without deleting existing data.
func (s *LedgerStore) AppendWorkspaceData(id string, headers []string, records [][]string, actor string, expectedVersion int) (*Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspace, ok := s.workspaces[strings.TrimSpace(id)]
	if !ok {
		return nil, ErrWorkspaceNotFound
	}
	if !WorkspaceKindSupportsTable(workspace.Kind) {
		return nil, ErrWorkspaceKindUnsupported
	}
	if expectedVersion > 0 && workspace.Version != expectedVersion {
		return nil, ErrWorkspaceVersionConflict
	}

	now := time.Now().UTC()
	normalizedHeaders := sanitizeHeaders(headers, records)

	// Map existing columns by normalized title.
	titleToID := make(map[string]string)
	for _, col := range workspace.Columns {
		key := strings.ToLower(strings.TrimSpace(col.Title))
		if key != "" {
			titleToID[key] = col.ID
		}
	}

	columns := append([]WorkspaceColumn{}, workspace.Columns...)
	for _, title := range normalizedHeaders {
		key := strings.ToLower(strings.TrimSpace(title))
		if key == "" {
			continue
		}
		if _, exists := titleToID[key]; exists {
			continue
		}
		id := GenerateID("col")
		titleToID[key] = id
		columns = append(columns, WorkspaceColumn{ID: id, Title: title})
	}

	rows := make([]WorkspaceRow, 0, len(records))
	for _, record := range records {
		cells := make(map[string]string, len(columns))
		for idx, column := range columns {
			val := ""
			// populate from matching header if present
			for hIdx, header := range normalizedHeaders {
				if idx >= len(columns) {
					break
				}
				if strings.EqualFold(header, column.Title) {
					if hIdx < len(record) {
						val = strings.TrimSpace(record[hIdx])
					}
					break
				}
			}
			cells[column.ID] = val
		}
		rows = append(rows, WorkspaceRow{
			ID:        GenerateID("row"),
			Cells:     cells,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	workspace.Columns = columns
	workspace.Rows = append(workspace.Rows, rows...)
	workspace.Version++
	workspace.UpdatedAt = now
	s.workspaces[workspace.ID] = workspace
	s.appendAuditLocked(actor, "workspace_import_append", workspace.ID)
	return workspace.Clone(), nil
}

// ReplaceWorkspaceDocument overwrites a document workspace's content.
func (s *LedgerStore) ReplaceWorkspaceDocument(id string, document string, actor string, expectedVersion int) (*Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workspace, ok := s.workspaces[strings.TrimSpace(id)]
	if !ok {
		return nil, ErrWorkspaceNotFound
	}
	if !WorkspaceKindSupportsDocument(workspace.Kind) {
		return nil, ErrWorkspaceKindUnsupported
	}
	if expectedVersion > 0 && workspace.Version != expectedVersion {
		return nil, ErrWorkspaceVersionConflict
	}

	workspace.Document = strings.TrimSpace(document)
	workspace.Version++
	workspace.UpdatedAt = time.Now().UTC()
	s.workspaces[workspace.ID] = workspace
	s.appendAuditLocked(actor, "workspace_document_import", workspace.ID)
	return workspace.Clone(), nil
}

func sanitizeWorkspaceName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed != "" {
		return trimmed
	}
	return "未命名台账"
}

func (s *LedgerStore) addWorkspaceChildLocked(parentID, childID string) {
	parent := strings.TrimSpace(parentID)
	s.workspaceChildren[parent] = append(s.workspaceChildren[parent], childID)
}

func (s *LedgerStore) removeWorkspaceChildLocked(parentID, childID string) {
	parent := strings.TrimSpace(parentID)
	children := s.workspaceChildren[parent]
	if len(children) == 0 {
		return
	}
	filtered := make([]string, 0, len(children))
	for _, existing := range children {
		if existing == childID {
			continue
		}
		filtered = append(filtered, existing)
	}
	if len(filtered) == 0 {
		delete(s.workspaceChildren, parent)
		return
	}
	s.workspaceChildren[parent] = filtered
}

func (s *LedgerStore) collectWorkspaceDescendantsLocked(id string, acc *[]string) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return
	}
	*acc = append(*acc, trimmed)
	for _, child := range s.workspaceChildren[trimmed] {
		s.collectWorkspaceDescendantsLocked(child, acc)
	}
}

func (s *LedgerStore) validateWorkspaceParentLocked(parentID, childID string) error {
	parent := strings.TrimSpace(parentID)
	if parent == "" {
		return nil
	}
	if childID != "" && parent == childID {
		return ErrWorkspaceParentInvalid
	}
	parentWorkspace, ok := s.workspaces[parent]
	if !ok {
		return ErrWorkspaceParentInvalid
	}
	if NormalizeWorkspaceKind(parentWorkspace.Kind) != WorkspaceKindFolder {
		return ErrWorkspaceParentInvalid
	}
	if childID != "" && s.isDescendantLocked(childID, parent) {
		return ErrWorkspaceParentInvalid
	}
	return nil
}

func (s *LedgerStore) isDescendantLocked(rootID, candidate string) bool {
	if rootID == "" || candidate == "" {
		return false
	}
	for _, child := range s.workspaceChildren[rootID] {
		if child == candidate || s.isDescendantLocked(child, candidate) {
			return true
		}
	}
	return false
}

func normalizeWorkspaceColumns(input []WorkspaceColumn) []WorkspaceColumn {
	if len(input) == 0 {
		return []WorkspaceColumn{}
	}
	out := make([]WorkspaceColumn, 0, len(input))
	seenIDs := make(map[string]struct{}, len(input))
	for idx, col := range input {
		title := strings.TrimSpace(col.Title)
		if title == "" {
			title = fmt.Sprintf("列%d", idx+1)
		}
		id := strings.TrimSpace(col.ID)
		for id == "" || hasID(seenIDs, id) {
			id = GenerateID("col")
		}
		seenIDs[id] = struct{}{}
		width := col.Width
		if width < 0 {
			width = 0
		}
		out = append(out, WorkspaceColumn{ID: id, Title: title, Width: width})
	}
	return out
}

func normalizeWorkspaceRows(input []WorkspaceRow, columns []WorkspaceColumn, now time.Time) []WorkspaceRow {
	if len(columns) == 0 || len(input) == 0 {
		return []WorkspaceRow{}
	}
	columnIDs := make([]string, len(columns))
	for i, col := range columns {
		columnIDs[i] = col.ID
	}
	out := make([]WorkspaceRow, 0, len(input))
	seenIDs := make(map[string]struct{}, len(input))
	for _, row := range input {
		id := strings.TrimSpace(row.ID)
		for id == "" || hasID(seenIDs, id) {
			id = GenerateID("row")
		}
		seenIDs[id] = struct{}{}
		cells := make(map[string]string, len(columnIDs))
		for _, colID := range columnIDs {
			var value string
			if row.Cells != nil {
				value = strings.TrimSpace(row.Cells[colID])
			}
			cells[colID] = value
		}
		created := row.CreatedAt
		if created.IsZero() {
			created = now
		}
		out = append(out, WorkspaceRow{
			ID:        id,
			Cells:     cells,
			CreatedAt: created,
			UpdatedAt: now,
		})
	}
	return out
}

func sanitizeHeaders(headers []string, records [][]string) []string {
	maxColumns := len(headers)
	for _, record := range records {
		if len(record) > maxColumns {
			maxColumns = len(record)
		}
	}
	if maxColumns == 0 {
		return []string{}
	}
	out := make([]string, maxColumns)
	used := make(map[string]int, maxColumns)
	for i := 0; i < maxColumns; i++ {
		title := ""
		if i < len(headers) {
			title = strings.TrimSpace(headers[i])
		}
		if title == "" {
			title = fmt.Sprintf("列%d", i+1)
		}
		if count := used[title]; count > 0 {
			title = fmt.Sprintf("%s (%d)", title, count+1)
		}
		used[title]++
		out[i] = title
	}
	return out
}

func hasID(seen map[string]struct{}, id string) bool {
	_, ok := seen[id]
	return ok
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
