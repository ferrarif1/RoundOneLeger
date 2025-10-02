package models

import "time"

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

// WorkspaceRow stores user-entered cell values keyed by column ID.
type WorkspaceRow struct {
	ID        string            `json:"id"`
	Cells     map[string]string `json:"cells"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Workspace represents a flexible spreadsheet-style ledger combined with rich text content.
type Workspace struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
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
