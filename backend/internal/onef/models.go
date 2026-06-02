// Package onef integrates with the external "Первая форма" (1F) HR system.
// Responsibilities: poll the 1F user feed on a schedule, reconcile workers
// against our local users table (matched by one_f_user_id), and grant
// derived management roles based on the IsManager / Otdel signal.
package onef

import (
	"time"

	"github.com/google/uuid"
)

// OneFUser is the JSON shape returned by GET /publications/action/get1FUsers.
// Field names match the 1F response verbatim (Pascal-case, mixed Russian/English semantics).
type OneFUser struct {
	UserID      int64  `json:"UserID"`
	FirstName   string `json:"FirstName"`
	LastName    string `json:"LastName"`
	DisplayName string `json:"DisplayName"`
	Email       string `json:"Email"`
	PhoneNumber string `json:"PhoneNumber"`
	BirthDate   string `json:"BirthDate"` // "2000-09-25" (ISO)
	DateHired   string `json:"DateHired"` // "26.02.2024" (dd.mm.yyyy)
	Department  string `json:"Department"`
	Otdel       string `json:"Otdel"`
	Dolzhnost   string `json:"Dolzhnost"`
	Manager     int64  `json:"Manager"`   // 1F UserID of supervisor; 0 = none
	IsManager   int    `json:"IsManager"` // 0 | 1
}

// SyncRun is one polling attempt recorded in onef_sync_runs.
type SyncRun struct {
	ID            uuid.UUID  `json:"id"`
	StartedAt     time.Time  `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	TriggeredBy   *uuid.UUID `json:"triggered_by,omitempty"`
	TriggerKind   string     `json:"trigger_kind"`
	Status        string     `json:"status"`
	FetchedCount  int        `json:"fetched_count"`
	CreatedCount  int        `json:"created_count"`
	UpdatedCount  int        `json:"updated_count"`
	SkippedCount  int        `json:"skipped_count"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
	DurationMS    *int       `json:"duration_ms,omitempty"`
}

// SyncResult is the in-memory tally returned by Service.RunSync.
type SyncResult struct {
	RunID        uuid.UUID `json:"run_id"`
	Status       string    `json:"status"`
	FetchedCount int       `json:"fetched_count"`
	CreatedCount int       `json:"created_count"`
	UpdatedCount int       `json:"updated_count"`
	SkippedCount int       `json:"skipped_count"`
	DurationMS   int       `json:"duration_ms"`
	Error        string    `json:"error,omitempty"`
}

const (
	TriggerCron   = "cron"
	TriggerManual = "manual"

	StatusRunning = "running"
	StatusSuccess = "success"
	StatusFailed  = "failed"
)
