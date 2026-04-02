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
