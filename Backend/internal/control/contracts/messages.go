package contracts

import "time"

type QueueTaskMessage struct {
	RunID        string `json:"run_id"`
	TaskID       string `json:"task_id"`
	ProjectID    string `json:"project_id"`
	VersionID    string `json:"version_id"`
	EntryCommand string `json:"entry_command"`
	Type         string `json:"type"`
	SourceURL    string `json:"source_url"`
	// Metadata is copied from the task so an external source can declare its
	// destination contract without coupling the executor to a business domain.
	Metadata      map[string]any `json:"metadata,omitempty"`
	NodeQueue     string         `json:"node_queue"`
	CorrelationID string         `json:"correlation_id"`
	RunToken      string         `json:"run_token"`
}

type RunCallbackPayload struct {
	Status     string     `json:"status"`
	Output     string     `json:"output"`
	ExitCode   int        `json:"exit_code"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}
