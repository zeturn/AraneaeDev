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

// TaskEnvelope is the canonical transport envelope for Araneae task
// requests. The task itself is deliberately kept under payload so every
// RabbitMQ event in the platform has the same transport shape.
type TaskEnvelope struct {
	MessageType    string           `json:"message_type"`
	SchemaVersion  string           `json:"schema_version"`
	EventID        string           `json:"event_id"`
	TraceID        string           `json:"trace_id"`
	CausationID    string           `json:"causation_id,omitempty"`
	IdempotencyKey string           `json:"idempotency_key"`
	Producer       string           `json:"producer"`
	RetryCount     int              `json:"retry_count"`
	OccurredAt     string           `json:"occurred_at"`
	Correlation    map[string]any   `json:"correlation"`
	Payload        QueueTaskMessage `json:"payload"`
}

type RunCallbackPayload struct {
	Status     string     `json:"status"`
	Output     string     `json:"output"`
	ExitCode   int        `json:"exit_code"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}
