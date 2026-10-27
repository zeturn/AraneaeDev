package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"araneae-go/internal/common"
	"araneae-go/internal/executor/contracts"

	"go.uber.org/zap"
)

func TestProcessSinkArtifacts_ForwardsToHashSlip(t *testing.T) {
	var datasetCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"test-token","expires_in":300}`))
			return
		}
		if r.URL.Path != "/api/v1/datasets/ds_sink/records" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Pipeline-Run-Id") != "run-1" || r.Header.Get("X-Pipeline-Task-Id") != "task-1" {
			t.Fatalf("missing canonical provenance headers: %v", r.Header)
		}
		if r.Header.Get("X-Araneae-Run-Id") != "" || r.Header.Get("X-Araneae-Task-Id") != "" {
			t.Fatalf("deprecated provenance headers must not be sent: %v", r.Header)
		}
		datasetCalls++
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	runDir := t.TempDir()
	sinkDir := filepath.Join(runDir, ".araneae", "sink")
	if err := os.MkdirAll(sinkDir, 0o755); err != nil {
		t.Fatalf("mkdir sink dir failed: %v", err)
	}
	records := []map[string]any{
		{
			"message_type":    "araneae.record.ready",
			"schema_version":  "1.0",
			"event_id":        "evt_timeseries",
			"trace_id":        "corr-1",
			"idempotency_key": "araneae.record:evt_timeseries",
			"producer":        "araneae",
			"retry_count":     0,
			"occurred_at":     time.Now().UTC().Format(time.RFC3339Nano),
			"correlation":     map[string]any{"trace_id": "corr-1"},
			"data": map[string]any{
				"source":      "araneae",
				"metric":      "fed_overnight_rate",
				"timestamp":   time.Now().UTC().Format(time.RFC3339),
				"value":       5.25,
				"hash_key":    "fed_overnight_rate",
				"bucket_date": time.Now().UTC().Format("2006-01-02"),
			},
			"metadata": map[string]any{"record_type": "timeseries"},
		},
		{
			"message_type":    "araneae.record.ready",
			"schema_version":  "1.0",
			"event_id":        "evt_text",
			"trace_id":        "corr-1",
			"idempotency_key": "araneae.record:evt_text",
			"producer":        "araneae",
			"retry_count":     0,
			"occurred_at":     time.Now().UTC().Format(time.RFC3339Nano),
			"correlation":     map[string]any{"trace_id": "corr-1"},
			"data": map[string]any{
				"source":  "araneae",
				"title":   "Fed statement",
				"url":     "https://example.com/fed",
				"content": "sample",
			},
			"metadata": map[string]any{"record_type": "text"},
		},
	}
	f, err := os.Create(filepath.Join(sinkDir, "events.jsonl"))
	if err != nil {
		t.Fatalf("create events file failed: %v", err)
	}
	for _, rec := range records {
		b, _ := json.Marshal(rec)
		if _, err := f.WriteString(string(b) + "\n"); err != nil {
			t.Fatalf("write record failed: %v", err)
		}
	}
	_ = f.Close()

	app := &App{
		cfg: common.ExecutorConfig{
			SinkEnabled:        true,
			SinkDirName:        ".araneae/sink",
			HashSlipBaseURL:    server.URL,
			BasaltTokenURL:     server.URL + "/oauth/token",
			BasaltClientID:     "test-client",
			BasaltClientSecret: "test-secret",
		},
		log:        zap.NewNop(),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	summary, err := app.processSinkArtifacts(context.Background(), contracts.QueueTaskMessage{
		RunID:         "run-1",
		TaskID:        "task-1",
		CorrelationID: "corr-1",
		Metadata:      map[string]any{"hashslip_slot": map[string]any{"dataset_id": "ds_sink"}},
	}, runDir)
	if err != nil {
		t.Fatalf("process sink failed: %v", err)
	}
	if !strings.Contains(summary, "total=2") {
		t.Fatalf("unexpected summary: %s", summary)
	}
	if datasetCalls != 2 {
		t.Fatalf("unexpected dataset calls=%d", datasetCalls)
	}
}

func TestProcessSinkArtifacts_ForwardsStructuredEventToTaskHashSlipSlot(t *testing.T) {
	var datasetCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"test-token","expires_in":300}`))
			return
		}
		if r.URL.Path != "/api/v1/datasets/ds_target/records" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Pipeline-Schema-Version") != "1.0" || r.Header.Get("X-Pipeline-Message-Type") != "hashslip.record.ingest" {
			t.Fatalf("missing canonical pipeline headers: %v", r.Header)
		}
		if r.Header.Get("X-Pipeline-Event-Id") == "" || r.Header.Get("X-Pipeline-Idempotency-Key") == "" || r.Header.Get("X-Pipeline-Producer") != "araneae" {
			t.Fatalf("incomplete canonical pipeline headers: %v", r.Header)
		}
		datasetCalls++
		var payload struct {
			Data     map[string]any `json:"data"`
			Source   map[string]any `json:"source"`
			Metadata map[string]any `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.Data["id"] != "article-1" {
			t.Fatalf("expected stable id to be preserved, got %#v", payload.Data["id"])
		}
		stored, ok := payload.Data["payload"].(map[string]any)
		if !ok || stored["title"] != "Example article" {
			t.Fatalf("expected source record to be stored as payload, got %#v", payload.Data)
		}
		if payload.Source["name"] != "https://example.com/blog/rss.xml" {
			t.Fatalf("expected canonical source envelope, got %#v", payload.Source)
		}
		if payload.Metadata["record_type"] != "structured" {
			t.Fatalf("expected canonical record metadata, got %#v", payload.Metadata)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	runDir := t.TempDir()
	sinkDir := filepath.Join(runDir, ".araneae", "sink")
	if err := os.MkdirAll(sinkDir, 0o755); err != nil {
		t.Fatalf("mkdir sink dir failed: %v", err)
	}
	record := map[string]any{
		"message_type":    "araneae.record.ready",
		"schema_version":  "1.0",
		"event_id":        "evt_structured",
		"trace_id":        "corr-1",
		"idempotency_key": "araneae.record:evt_structured",
		"producer":        "araneae",
		"retry_count":     0,
		"occurred_at":     time.Now().UTC().Format(time.RFC3339Nano),
		"correlation":     map[string]any{"trace_id": "corr-1"},
		"data": map[string]any{
			"instance_id": "source-a",
			"schema_id":   "article",
			"data": map[string]any{
				"id":    "article-1",
				"title": "Example article",
			},
		},
		"metadata": map[string]any{"record_type": "structured"},
	}
	raw, _ := json.Marshal(record)
	if err := os.WriteFile(filepath.Join(sinkDir, "events.jsonl"), append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write sink event failed: %v", err)
	}

	app := &App{
		cfg: common.ExecutorConfig{
			SinkEnabled:        true,
			SinkDirName:        ".araneae/sink",
			HashSlipBaseURL:    server.URL,
			BasaltTokenURL:     server.URL + "/oauth/token",
			BasaltClientID:     "test-client",
			BasaltClientSecret: "test-secret",
		},
		log:        zap.NewNop(),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	summary, err := app.processSinkArtifacts(context.Background(), contracts.QueueTaskMessage{
		RunID:         "run-1",
		TaskID:        "task-1",
		CorrelationID: "corr-1",
		SourceURL:     "https://example.com/blog/rss.xml",
		Metadata: map[string]any{
			"hashslip_slot": map[string]any{
				"dataset_id": "ds_target",
			},
		},
	}, runDir)
	if err != nil {
		t.Fatalf("process sink failed: %v", err)
	}
	if !strings.Contains(summary, "structured=1") {
		t.Fatalf("unexpected summary: %s", summary)
	}
	if datasetCalls != 1 {
		t.Fatalf("expected dataset call once, got %d", datasetCalls)
	}
}
