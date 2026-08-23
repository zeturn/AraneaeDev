package executor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAppendSinkEnvelopeWritesCanonicalRecordContract(t *testing.T) {
	dir := t.TempDir()
	if err := appendSinkEnvelope(dir, "structured", json.RawMessage(`{"title":"event"}`), map[string]any{"schema_id": "event.v1"}, "source.example"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var envelope sinkEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if err := validateSinkEnvelope(envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Metadata["record_type"] != "structured" || envelope.Source["name"] != "source.example" {
		t.Fatalf("unexpected canonical metadata/source: %+v %+v", envelope.Metadata, envelope.Source)
	}
}
