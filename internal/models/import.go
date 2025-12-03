package models

import "time"

type ImportTaskStatus string

const (
	ImportPending ImportTaskStatus = "pending"
	ImportRunning ImportTaskStatus = "running"
	ImportSuccess ImportTaskStatus = "success"
	ImportFailed  ImportTaskStatus = "failed"
)

type ImportTask struct {
	ID          string           `json:"id" db:"id"`
	TableID     string           `json:"tableId" db:"table_id"`
	Status      ImportTaskStatus `json:"status" db:"status"`
	Progress    int              `json:"progress" db:"progress"`
	Error       string           `json:"error,omitempty" db:"error"`
	PayloadPath string           `json:"payloadPath,omitempty" db:"payload_path"`
	CreatedAt   time.Time        `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time        `json:"updatedAt" db:"updated_at"`
}
