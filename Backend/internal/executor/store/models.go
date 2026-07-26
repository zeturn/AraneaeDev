package store

import "time"

type ExecutionRecord struct {
	RunID      string    `gorm:"primaryKey;size:36"`
	TaskID     string    `gorm:"size:36;index"`
	Status     string    `gorm:"size:32;not null"`
	Output     string    `gorm:"type:text"`
	ExitCode   int       `gorm:"not null;default:0"`
	CreatedAt  time.Time `gorm:"not null"`
	FinishedAt *time.Time
}

// SourceFetchState is executor-local operational state for a source task. It
// keeps HTTP validators and retry timing separate from the task definition so
// any RSS or API task can use conditional requests and failure backoff.
type SourceFetchState struct {
	TaskID              string    `gorm:"primaryKey;size:36"`
	ETag                string    `gorm:"size:1024"`
	LastModified        string    `gorm:"size:1024"`
	ConsecutiveFailures int       `gorm:"not null;default:0"`
	NextAttemptAt       time.Time `gorm:"index"`
	LastStatus          int       `gorm:"not null;default:0"`
	LastError           string    `gorm:"type:text"`
	UpdatedAt           time.Time `gorm:"not null"`
}
