package common

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           string    `gorm:"primaryKey;size:36" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Name         string    `gorm:"size:128" json:"name"`
	Email        string    `gorm:"size:160;index" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         string    `gorm:"size:32;not null" json:"role"`
	CreatedAt    time.Time `gorm:"not null" json:"created_at"`

	// BasaltPass user tokens (persisted so Araneae can perform cross-app token
	// exchange on behalf of the user, e.g. to read their Objectary files).
	BasaltAccessToken  string     `gorm:"size:1024" json:"-"`
	BasaltRefreshToken string     `gorm:"size:1024" json:"-"`
	BasaltTokenExpiry  *time.Time `json:"-"`
}

type Project struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Description string    `gorm:"size:512" json:"description"`
	Language    string    `gorm:"size:64" json:"language"`
	WorkplaceID *uint     `gorm:"index" json:"workplace_id,omitempty"`
	CreatedBy   string    `gorm:"size:36;not null" json:"created_by"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
	ProjectID    string    `gorm:"index;size:36" json:"project_id"`
	VersionID    string    `gorm:"index;size:36" json:"version_id"`
	EntryCommand string    `gorm:"size:512" json:"entry_command"`
	Type         string    `gorm:"size:16;not null;default:code" json:"type"`
	SourceURL    string    `gorm:"size:1024" json:"source_url"`
	MetadataJSON string    `gorm:"type:text" json:"metadata,omitempty"`
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
	NodeQueue     string     `gorm:"size:64;not null;default:default" json:"node_queue"`
	Status        string     `gorm:"size:32;not null" json:"status"`
	Output        string     `gorm:"type:text" json:"output"`
	ExitCode      int        `gorm:"not null;default:0" json:"exit_code"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	CorrelationID string     `gorm:"size:64" json:"correlation_id"`
	RunTokenHash  string     `gorm:"size:64" json:"-"`
	CreatedAt     time.Time  `gorm:"not null" json:"created_at"`
}

type Schedule struct {
	ID              string            `gorm:"primaryKey;size:36" json:"id"`
	Name            string            `gorm:"size:128;not null" json:"name"`
	Description     string            `gorm:"size:512" json:"description"`
	TaskID          string            `gorm:"index;size:36" json:"task_id"`
	ProjectID       string            `gorm:"index;size:36;not null" json:"project_id"`
	VersionID       string            `gorm:"index;size:36;not null" json:"version_id"`
	EntryCommand    string            `gorm:"size:512;not null" json:"entry_command"`
	CronExpr        string            `gorm:"size:128;not null" json:"cron_expr"`
	TriggerType     string            `gorm:"size:32;not null;default:api" json:"trigger_type"`
	RunAt           *time.Time        `json:"run_at"`
	NodeQueue       string            `gorm:"size:64;not null" json:"node_queue"`
	OrderJSON       string            `gorm:"type:text" json:"order"`
	Enabled         bool              `gorm:"not null;default:true" json:"enabled"`
	CreatedBy       string            `gorm:"size:36;not null" json:"created_by"`
	LastTriggeredAt *time.Time        `json:"last_triggered_at"`
	CreatedAt       time.Time         `gorm:"not null" json:"created_at"`
	UpdatedAt       time.Time         `gorm:"not null" json:"updated_at"`
	RunTimes        []ScheduleRunTime `gorm:"foreignKey:ScheduleID;references:ID" json:"run_times"`
}

// ScheduleRunTime is a one-to-many relation: a single schedule (bound to a task) can
// run at multiple specific points in time. Each row represents one firing time.
type ScheduleRunTime struct {
	ID          string     `gorm:"primaryKey;size:36" json:"id"`
	ScheduleID  string     `gorm:"index;size:36;not null" json:"schedule_id"`
	RunAt       time.Time  `gorm:"not null" json:"run_at"`
	TriggeredAt *time.Time `json:"triggered_at"`
	CreatedAt   time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null" json:"updated_at"`
}

type RSSSubscription struct {
	ID            string     `gorm:"primaryKey;size:36" json:"id"`
	WorkplaceID   *uint      `gorm:"index;uniqueIndex:idx_rss_workplace_url" json:"workplace_id,omitempty"`
	URL           string     `gorm:"size:1024;not null;uniqueIndex:idx_rss_workplace_url" json:"url"`
	Title         string     `gorm:"size:255" json:"title"`
	Description   string     `gorm:"size:1024" json:"description"`
	Link          string     `gorm:"size:1024" json:"link"`
	StorageDir    string     `gorm:"size:512;not null" json:"storage_dir"`
	CreatedBy     string     `gorm:"size:36;not null" json:"created_by"`
	LastFetchedAt *time.Time `json:"last_fetched_at"`
	CreatedAt     time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"not null" json:"updated_at"`
}

type RSSItem struct {
	ID             string     `gorm:"primaryKey;size:36" json:"id"`
	SubscriptionID string     `gorm:"index;uniqueIndex:idx_rss_item_identity;size:36;not null" json:"subscription_id"`
	GUID           string     `gorm:"uniqueIndex:idx_rss_item_identity;size:512;not null" json:"guid"`
	Title          string     `gorm:"size:512" json:"title"`
	Link           string     `gorm:"size:1024" json:"link"`
	Author         string     `gorm:"size:255" json:"author"`
	Summary        string     `gorm:"type:text" json:"summary"`
	Content        string     `gorm:"type:text" json:"content"`
	ContentPath    string     `gorm:"size:512;not null" json:"content_path"`
	PublishedAt    *time.Time `json:"published_at"`
	FetchedAt      time.Time  `gorm:"not null" json:"fetched_at"`
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
	AuthTokenHash    string    `gorm:"size:64;index" json:"-"`
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
	IsPersonal  bool      `gorm:"not null;default:false" json:"is_personal"`
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

type SecurityEvent struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	EventType string    `gorm:"index;size:64;not null" json:"event_type"`
	Severity  string    `gorm:"index;size:16;not null" json:"severity"`
	UserID    string    `gorm:"index;size:36" json:"user_id"`
	IPAddress string    `gorm:"size:64" json:"ip_address"`
	Method    string    `gorm:"size:16" json:"method"`
	Path      string    `gorm:"size:255" json:"path"`
	Detail    string    `gorm:"size:1024" json:"detail"`
	CreatedAt time.Time `gorm:"index;not null" json:"created_at"`
}

func personalTeamName(username string) string {
	name := strings.TrimSpace(username)
	if name == "" {
		name = "user"
	}
	return fmt.Sprintf("%s Personal Team", name)
}

func (u *User) AfterCreate(tx *gorm.DB) error {
	now := time.Now()
	team := Team{
		Name:        personalTeamName(u.Username),
		Description: "Auto-created personal team",
		JoinAble:    false,
		IsPersonal:  true,
		CreatedBy:   u.ID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := tx.Create(&team).Error; err != nil {
		return err
	}

	member := TeamMember{
		TeamID:    team.ID,
		UserID:    u.ID,
		Role:      "owner",
		CreatedAt: now,
	}
	if err := tx.Create(&member).Error; err != nil {
		return err
	}

	return nil
}

func AutoMigrateModels() []interface{} {
	return []interface{}{
		&User{},
		&Project{},
		&ArtifactVersion{},
		&Task{},
		&Schedule{},
		&ScheduleRunTime{},
		&TaskRun{},
		&RSSSubscription{},
		&RSSItem{},
		&Node{},
		&NodeInstallJob{},
		&Team{},
		&TeamMember{},
		&Workplace{},
		&WorkplaceTeam{},
		&SecurityEvent{},
	}
}
