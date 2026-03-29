package common

import "time"

type User struct {
	ID           string    `gorm:"primaryKey;size:36" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         string    `gorm:"size:32;not null" json:"role"`
	CreatedAt    time.Time `gorm:"not null" json:"created_at"`
}

type Project struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	CreatedBy string    `gorm:"size:36;not null" json:"created_by"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
}

type ArtifactVersion struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	ProjectID   string    `gorm:"index;size:36;not null" json:"project_id"`
	FileName    string    `gorm:"size:255;not null" json:"file_name"`
	StoragePath string    `gorm:"size:512;not null" json:"storage_path"`
	SHA256      string    `gorm:"size:64;not null" json:"sha256"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
}

type Task struct {
	ID           string    `gorm:"primaryKey;size:36" json:"id"`
	Name         string    `gorm:"size:128;not null" json:"name"`
	ProjectID    string    `gorm:"index;size:36;not null" json:"project_id"`
	VersionID    string    `gorm:"index;size:36;not null" json:"version_id"`
	EntryCommand string    `gorm:"size:512;not null" json:"entry_command"`
	CronExpr     string    `gorm:"size:128" json:"cron_expr"`
	NodeQueue    string    `gorm:"size:64;not null" json:"node_queue"`
	Enabled      bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedBy    string    `gorm:"size:36;not null" json:"created_by"`
	CreatedAt    time.Time `gorm:"not null" json:"created_at"`
}

type TaskRun struct {
	ID            string     `gorm:"primaryKey;size:36" json:"id"`
	TaskID        string     `gorm:"index;size:36;not null" json:"task_id"`
	ScheduleID    string     `gorm:"index;size:36" json:"schedule_id"`
	ChainID       string     `gorm:"index;size:36" json:"chain_id"`
	ChainIndex    int        `gorm:"not null;default:0" json:"chain_index"`
	ChainTotal    int        `gorm:"not null;default:0" json:"chain_total"`
	TriggerSource string     `gorm:"size:32;not null" json:"trigger_source"`
	Status        string     `gorm:"size:32;not null" json:"status"`
	Output        string     `gorm:"type:text" json:"output"`
	ExitCode      int        `gorm:"not null;default:0" json:"exit_code"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	CorrelationID string     `gorm:"size:64" json:"correlation_id"`
	CreatedAt     time.Time  `gorm:"not null" json:"created_at"`
}

type Schedule struct {
	ID              string     `gorm:"primaryKey;size:36" json:"id"`
	Name            string     `gorm:"size:128;not null" json:"name"`
	Description     string     `gorm:"size:512" json:"description"`
	TaskID          string     `gorm:"index;size:36" json:"task_id"`
	ProjectID       string     `gorm:"index;size:36;not null" json:"project_id"`
	VersionID       string     `gorm:"index;size:36;not null" json:"version_id"`
	EntryCommand    string     `gorm:"size:512;not null" json:"entry_command"`
	CronExpr        string     `gorm:"size:128;not null" json:"cron_expr"`
	NodeQueue       string     `gorm:"size:64;not null" json:"node_queue"`
	Mode            string     `gorm:"size:32;not null;default:recurring" json:"mode"`
	OrderJSON       string     `gorm:"type:text" json:"order"`
	Enabled         bool       `gorm:"not null;default:true" json:"enabled"`
	CreatedBy       string     `gorm:"size:36;not null" json:"created_by"`
	LastTriggeredAt *time.Time `json:"last_triggered_at"`
	CreatedAt       time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"not null" json:"updated_at"`
}

func AutoMigrateModels() []interface{} {
	return []interface{}{
		&User{},
		&Project{},
		&ArtifactVersion{},
		&Task{},
		&Schedule{},
		&TaskRun{},
	}
}
