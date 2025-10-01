package models

import (
	"strings"
	"time"
)

// DefaultUsername is used when no explicit operator account is supplied.
const DefaultUsername = "admin"

// NormaliseUsername lowercases and defaults empty usernames to the administrator account.
func NormaliseUsername(username string) string {
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return DefaultUsername
	}
	return strings.ToLower(trimmed)
}

// LedgerType represents the category of a ledger entry.
type LedgerType string

const (
	LedgerTypeIP        LedgerType = "ips"
	LedgerTypeDevice    LedgerType = "devices"
	LedgerTypePersonnel LedgerType = "personnel"
	LedgerTypeSystem    LedgerType = "systems"
)

// AllLedgerTypes lists the supported ledgers in a stable order.
var AllLedgerTypes = []LedgerType{LedgerTypeIP, LedgerTypeDevice, LedgerTypePersonnel, LedgerTypeSystem}

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

// User represents an authenticated operator of the system.
type User struct {
	Username  string                 `json:"username"`
	Devices   map[string]*UserDevice `json:"devices"`
	Roles     []string               `json:"roles"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// UserDevice represents a registered device bound to a user.
type UserDevice struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	PublicKey      []byte    `json:"-"`
	FingerprintSum string    `json:"fingerprint_sum"`
	BoundIP        string    `json:"bound_ip,omitempty"`
	AdminSignature string    `json:"admin_signature,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Enrollment describes an in-flight enrollment challenge.
type Enrollment struct {
	Username   string
	DeviceID   string
	DeviceName string
	Nonce      string
	PublicKey  []byte
	CreatedAt  time.Time
}

// LoginChallenge stores a nonce waiting to be signed by a device.
type LoginChallenge struct {
	Username  string
	DeviceID  string
	Nonce     string
	CreatedAt time.Time
}
