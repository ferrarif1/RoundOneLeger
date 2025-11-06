package models

import (
	"strings"
	"time"
)

// LedgerType represents the category of a ledger entry.
type LedgerType string

const (
	LedgerTypeIP        LedgerType = "ips"
	LedgerTypePersonnel LedgerType = "personnel"
	LedgerTypeSystem    LedgerType = "systems"
)

// AllLedgerTypes lists the supported ledgers in a stable order.
var AllLedgerTypes = []LedgerType{LedgerTypeIP, LedgerTypePersonnel, LedgerTypeSystem}

// LedgerEntry describes a single item within a ledger.
type LedgerEntry struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description,omitempty"`
	Attributes  map[string]string       `json:"attributes,omitempty"`
	Tags        []string                `json:"tags,omitempty"`
	Links       map[LedgerType][]string `json:"links,omitempty"`
	Order       int                     `json:"order"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}

// Clone returns a deep copy of the entry.
func (e LedgerEntry) Clone() LedgerEntry {
	clone := e
	if e.Attributes != nil {
		clone.Attributes = make(map[string]string, len(e.Attributes))
		for k, v := range e.Attributes {
			clone.Attributes[k] = v
		}
	}
	if e.Tags != nil {
		clone.Tags = append([]string{}, e.Tags...)
	}
	if e.Links != nil {
		clone.Links = make(map[LedgerType][]string, len(e.Links))
		for t, ids := range e.Links {
			clone.Links[t] = append([]string{}, ids...)
		}
	}
	return clone
}

// WorkspaceColumn describes a dynamic column within a collaborative sheet.
type WorkspaceColumn struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Width int    `json:"width,omitempty"`
}

// WorkspaceKind describes the layout style of a workspace entry.
type WorkspaceKind string

const (
	// WorkspaceKindSheet represents a spreadsheet-style workspace with rows and columns.
	WorkspaceKindSheet WorkspaceKind = "sheet"
	// WorkspaceKindDocument represents a freeform rich-text document.
	WorkspaceKindDocument WorkspaceKind = "document"
	// WorkspaceKindFolder groups other workspaces without storing content directly.
	WorkspaceKindFolder WorkspaceKind = "folder"
)

// WorkspaceRow stores user-entered cell values keyed by column ID.
type WorkspaceRow struct {
	ID        string            `json:"id"`
	Cells     map[string]string `json:"cells"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Workspace represents a flexible workspace that can behave as a sheet, document, or folder.
type Workspace struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Kind      WorkspaceKind     `json:"kind"`
	ParentID  string            `json:"parent_id,omitempty"`
	Columns   []WorkspaceColumn `json:"columns"`
	Rows      []WorkspaceRow    `json:"rows"`
	Document  string            `json:"document,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Clone returns a deep copy of the workspace structure for safe sharing across callers.
func (w *Workspace) Clone() *Workspace {
	if w == nil {
		return nil
	}
	clone := *w
	clone.Kind = NormalizeWorkspaceKind(w.Kind)
	clone.ParentID = strings.TrimSpace(w.ParentID)
	if len(w.Columns) > 0 {
		clone.Columns = append([]WorkspaceColumn{}, w.Columns...)
	}
	if len(w.Rows) > 0 {
		clone.Rows = make([]WorkspaceRow, len(w.Rows))
		for i, row := range w.Rows {
			clonedRow := row
			if row.Cells != nil {
				clonedRow.Cells = make(map[string]string, len(row.Cells))
				for key, value := range row.Cells {
					clonedRow.Cells[key] = value
				}
			}
			clone.Rows[i] = clonedRow
		}
	}
	return &clone
}

// NormalizeWorkspaceKind coerces unknown values to the default sheet type.
func NormalizeWorkspaceKind(kind WorkspaceKind) WorkspaceKind {
	switch kind {
	case WorkspaceKindDocument, WorkspaceKindFolder:
		return kind
	case WorkspaceKindSheet:
		return kind
	default:
		return WorkspaceKindSheet
	}
}

// ParseWorkspaceKind converts free-form input into a WorkspaceKind.
func ParseWorkspaceKind(value string) WorkspaceKind {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(WorkspaceKindDocument):
		return WorkspaceKindDocument
	case string(WorkspaceKindFolder):
		return WorkspaceKindFolder
	case string(WorkspaceKindSheet):
		return WorkspaceKindSheet
	default:
		return WorkspaceKindSheet
	}
}

// WorkspaceKindSupportsTable reports whether the workspace type accepts tabular data.
func WorkspaceKindSupportsTable(kind WorkspaceKind) bool {
	return NormalizeWorkspaceKind(kind) == WorkspaceKindSheet
}

// WorkspaceKindSupportsDocument reports whether the workspace type stores document content.
func WorkspaceKindSupportsDocument(kind WorkspaceKind) bool {
	normalized := NormalizeWorkspaceKind(kind)
	return normalized == WorkspaceKindSheet || normalized == WorkspaceKindDocument
}

// IPAllowlistEntry represents a single CIDR or address allowed to access the system.
type IPAllowlistEntry struct {
	ID          string    `json:"id"`
	Label       string    `json:"label"`
	CIDR        string    `json:"cidr"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AuditLogEntry represents a tamper evident log item.
type AuditLogEntry struct {
	ID        string    `json:"id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	Hash      string    `json:"hash"`
	PrevHash  string    `json:"prev_hash"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginChallenge stores a nonce waiting to be signed by an SDID wallet.
type LoginChallenge struct {
	Nonce     string    `json:"nonce"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// User represents an operator who can access the admin console.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Admin        bool      `json:"admin"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Clone returns a copy of the user omitting the password hash for safe sharing.
func (u *User) Clone() *User {
	if u == nil {
		return nil
	}
	clone := *u
	clone.PasswordHash = ""
	return &clone
}
