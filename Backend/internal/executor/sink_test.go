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
	var tsCalls, textCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/timeseries/records":
			tsCalls++
		case "/api/v1/text/records":
			textCalls++
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
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
			"type": "timeseries",
			"data": map[string]any{
				"source":      "araneae",
				"metric":      "fed_overnight_rate",
				"timestamp":   time.Now().UTC().Format(time.RFC3339),
				"value":       5.25,
				"hash_key":    "fed_overnight_rate",
				"bucket_date": time.Now().UTC().Format("2006-01-02"),
			},
		},
		{
			"type": "text",
			"data": map[string]any{
				"source":  "araneae",
				"title":   "Fed statement",
				"url":     "https://example.com/fed",
				"content": "sample",
			},
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
			SinkEnabled:            true,
			SinkDirName:            ".araneae/sink",
			HashSlipBaseURL:        server.URL,
			HashSlipTimeseriesPath: "/api/v1/timeseries/records",
			HashSlipTextPath:       "/api/v1/text/records",
		},
		log:        zap.NewNop(),
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	summary, err := app.processSinkArtifacts(context.Background(), contracts.QueueTaskMessage{
		RunID:         "run-1",
		TaskID:        "task-1",
		CorrelationID: "corr-1",
	}, runDir)
	if err != nil {
		t.Fatalf("process sink failed: %v", err)
	}
	if !strings.Contains(summary, "total=2") {
		t.Fatalf("unexpected summary: %s", summary)
	}
	if tsCalls != 1 || textCalls != 1 {
		t.Fatalf("unexpected calls ts=%d text=%d", tsCalls, textCalls)
	}
}
