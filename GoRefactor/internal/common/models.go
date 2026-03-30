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

type Node struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Name             string    `gorm:"size:128;not null" json:"name"`
	Description      string    `gorm:"size:512" json:"description"`
	Status           string    `gorm:"size:32;not null;default:active" json:"status"`
	IPAddress        string    `gorm:"index;size:64;not null" json:"ip_address"`
	Port             int       `gorm:"not null;default:4280" json:"port"`
	GRPCPort         int       `gorm:"not null;default:9190" json:"grpc_port"`
	RPCURL           string    `gorm:"size:255" json:"rpc_url"`
	CeleryQueue      string    `gorm:"size:64;not null;default:default" json:"celery_queue"`
	IsEnabled        bool      `gorm:"not null;default:true" json:"is_enabled"`
	LastActiveTime   time.Time `gorm:"not null" json:"last_active_time"`
	HDID             string    `gorm:"size:64" json:"HDID"`
	CPUInfo          string    `gorm:"type:text" json:"cpu_info"`
	MemoryInfo       string    `gorm:"type:text" json:"memory_info"`
	CapabilitiesJSON string    `gorm:"type:text" json:"-"`
	CreatedBy        string    `gorm:"size:36" json:"created_by"`
	CreatedAt        time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt        time.Time `gorm:"not null" json:"updated_at"`
}

type NodeInstallJob struct {
	ID         string    `gorm:"primaryKey;size:64" json:"id"`
	NodeID     uint      `gorm:"index;not null" json:"node_id"`
	RuntimeKey string    `gorm:"size:64;not null" json:"runtime_key"`
	Status     string    `gorm:"size:32;not null" json:"status"`
	Log        string    `gorm:"type:text" json:"log"`
	CreatedAt  time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt  time.Time `gorm:"not null" json:"updated_at"`
}

type Team struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Description string    `gorm:"size:512" json:"description"`
	JoinAble    bool      `gorm:"not null;default:false" json:"join_able"`
	CreatedBy   string    `gorm:"size:36" json:"created_by"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
}

type TeamMember struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TeamID    uint      `gorm:"index;not null" json:"team_id"`
	UserID    string    `gorm:"index;size:36;not null" json:"user_id"`
	Role      string    `gorm:"size:32;not null;default:member" json:"role"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
}

type Workplace struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Description string    `gorm:"size:512" json:"description"`
	Status      string    `gorm:"size:32;not null;default:active" json:"status"`
	CreatedBy   string    `gorm:"size:36" json:"created_by"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
}

type WorkplaceTeam struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WorkplaceID uint      `gorm:"uniqueIndex:idx_workplace_team;not null" json:"workplace_id"`
	TeamID      uint      `gorm:"uniqueIndex:idx_workplace_team;not null" json:"team_id"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
}

func AutoMigrateModels() []interface{} {
	return []interface{}{
		&User{},
		&Project{},
		&ArtifactVersion{},
		&Task{},
		&Schedule{},
		&TaskRun{},
		&Node{},
		&NodeInstallJob{},
		&Team{},
		&TeamMember{},
		&Workplace{},
		&WorkplaceTeam{},
	}
}
