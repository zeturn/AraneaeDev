package control

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"araneae-go/internal/common"
)

func TestBuildCrawlSucceededEnvelopeReportsCustomCrawlerWithHashSlipSlot(t *testing.T) {
	app := newTestControlApp(t)
	slot := map[string]any{
		"dataset_id": "ds_custom",
		"chunk_id":   "chunk_custom",
		"data_type":  "structured",
	}
	metadata, _ := json.Marshal(map[string]any{
		"hashslip_slot": slot,
		"mission_id":    "mission_custom",
		"schema_id":     "blog_post",
	})
	task := common.Task{
		ID:           "task_custom",
		Name:         "custom blog crawler",
		Type:         "code",
		SourceURL:    "https://example.com/blog/rss.xml",
		MetadataJSON: string(metadata),
		NodeQueue:    "default",
		CreatedBy:    "tester",
		CreatedAt:    time.Now(),
	}
	if err := app.db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	finishedAt := time.Now().UTC()
	run := common.TaskRun{
		ID:            "run_custom",
		TaskID:        task.ID,
		Status:        "success",
		Output:        "sink forwarded total=3 timeseries=0 text=0 structured=3 failed=0",
		FinishedAt:    &finishedAt,
		CorrelationID: "trace_custom",
		CreatedAt:     time.Now(),
	}
	envelope, ok, err := app.buildCrawlSucceededEnvelope(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected custom crawler with hashslip_slot to be reportable")
	}
	if envelope.MessageType != "araneae.crawl.succeeded" {
		t.Fatalf("unexpected message type: %s", envelope.MessageType)
	}
	if envelope.Correlation["mission_id"] != "mission_custom" {
		t.Fatalf("unexpected mission correlation: %#v", envelope.Correlation)
	}
	gotSlot, ok := envelope.Payload["hashslip_slot"].(map[string]any)
	if !ok || gotSlot["dataset_id"] != "ds_custom" {
		t.Fatalf("expected hashslip slot in payload, got %#v", envelope.Payload["hashslip_slot"])
	}
	if envelope.Payload["task_type"] != "code" {
		t.Fatalf("expected task_type code, got %#v", envelope.Payload["task_type"])
	}
}

func TestBuildCrawlSucceededEnvelopeSkipsCodeTaskWithoutHashSlipSlot(t *testing.T) {
	app := newTestControlApp(t)
	task := common.Task{
		ID:        "task_no_slot",
		Name:      "maintenance script",
		Type:      "code",
		NodeQueue: "default",
		CreatedBy: "tester",
		CreatedAt: time.Now(),
	}
	if err := app.db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	envelope, ok, err := app.buildCrawlSucceededEnvelope(context.Background(), common.TaskRun{ID: "run_no_slot", TaskID: task.ID})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("expected no report for code task without hashslip slot, got %#v", envelope)
	}
}
