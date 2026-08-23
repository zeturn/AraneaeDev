package contracts

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type QueueTaskMessage struct {
	RunID         string         `json:"run_id"`
	TaskID        string         `json:"task_id"`
	ProjectID     string         `json:"project_id"`
	VersionID     string         `json:"version_id"`
	EntryCommand  string         `json:"entry_command"`
	Type          string         `json:"type"`
	SourceURL     string         `json:"source_url"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	NodeQueue     string         `json:"node_queue"`
	CorrelationID string         `json:"correlation_id"`
	RunToken      string         `json:"run_token"`
}

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

// DecodeTaskMessage accepts only the canonical task envelope.
func DecodeTaskMessage(raw []byte) (QueueTaskMessage, error) {
	var envelope TaskEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return QueueTaskMessage{}, err
	}
	if envelope.MessageType != "araneae.task.requested" {
		return QueueTaskMessage{}, fmt.Errorf("unsupported task message_type %q", envelope.MessageType)
	}
	if envelope.SchemaVersion != "1.0" {
		return QueueTaskMessage{}, fmt.Errorf("unsupported task schema_version %q", envelope.SchemaVersion)
	}
	for name, value := range map[string]string{
		"event_id": envelope.EventID, "trace_id": envelope.TraceID,
		"idempotency_key": envelope.IdempotencyKey, "producer": envelope.Producer,
		"occurred_at": envelope.OccurredAt,
	} {
		if strings.TrimSpace(value) == "" {
			return QueueTaskMessage{}, fmt.Errorf("task envelope %s is required", name)
		}
	}
	if envelope.Producer != "araneae" {
		return QueueTaskMessage{}, fmt.Errorf("unexpected task envelope producer %q", envelope.Producer)
	}
	if envelope.RetryCount < 0 {
		return QueueTaskMessage{}, fmt.Errorf("task envelope retry_count must not be negative")
	}
	if _, err := time.Parse(time.RFC3339Nano, envelope.OccurredAt); err != nil {
		return QueueTaskMessage{}, fmt.Errorf("task envelope occurred_at is invalid: %w", err)
	}
	if envelope.Correlation == nil {
		return QueueTaskMessage{}, fmt.Errorf("task envelope correlation is required")
	}
	if err := validateTask(envelope.Payload); err != nil {
		return QueueTaskMessage{}, err
	}
	return envelope.Payload, nil
}

func validateTask(task QueueTaskMessage) error {
	if strings.TrimSpace(task.RunID) == "" || strings.TrimSpace(task.TaskID) == "" {
		return fmt.Errorf("task message run_id and task_id are required")
	}
	if strings.TrimSpace(task.NodeQueue) == "" {
		return fmt.Errorf("task message node_queue is required")
	}
	if strings.TrimSpace(task.RunToken) == "" || strings.TrimSpace(task.CorrelationID) == "" {
		return fmt.Errorf("task message run_token and correlation_id are required")
	}
	return nil
}

type CallbackPayload struct {
	Status     string     `json:"status"`
	Output     string     `json:"output"`
	ExitCode   int        `json:"exit_code"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}
