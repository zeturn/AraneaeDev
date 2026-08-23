package contracts

import (
	"encoding/json"
	"testing"
	"time"
)

func sampleTask() QueueTaskMessage {
	return QueueTaskMessage{
		RunID:         "run_1",
		TaskID:        "task_1",
		NodeQueue:     "default",
		CorrelationID: "corr_1",
		RunToken:      "token_1",
	}
}

func TestDecodeTaskMessageCanonicalEnvelope(t *testing.T) {
	payload := sampleTask()
	raw, err := json.Marshal(TaskEnvelope{
		MessageType:    "araneae.task.requested",
		SchemaVersion:  "1.0",
		EventID:        "evt_1",
		TraceID:        "corr_1",
		IdempotencyKey: "araneae.task.requested:run_1",
		Producer:       "araneae",
		OccurredAt:     time.Now().UTC().Format(time.RFC3339Nano),
		Correlation:    map[string]any{"run_id": "run_1"},
		Payload:        payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeTaskMessage(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.RunID != payload.RunID || got.TaskID != payload.TaskID {
		t.Fatalf("unexpected task: %+v", got)
	}
}

func TestDecodeTaskMessageRejectsFlatPayload(t *testing.T) {
	payload := sampleTask()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeTaskMessage(raw); err == nil {
		t.Fatal("expected flat task payload to be rejected")
	}
}
