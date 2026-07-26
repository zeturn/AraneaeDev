package control

import (
	"encoding/json"
	"net/http"
	"testing"

	"araneae-go/internal/common"
)

func TestRSSSourceTaskCanUseScheduleWithoutCodeArtifact(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)

	taskRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/tasks", token, map[string]any{
		"name":       "example source",
		"type":       "rss",
		"source_url": "https://example.com/feed.xml",
		"metadata": map[string]any{
			"hashslip_slot": map[string]any{"dataset_id": "news-raw", "data_type": "structured"},
		},
	})
	if taskRec.Code != http.StatusOK {
		t.Fatalf("create RSS task status=%d body=%s", taskRec.Code, taskRec.Body.String())
	}
	var task common.Task
	if err := json.Unmarshal(taskRec.Body.Bytes(), &task); err != nil {
		t.Fatalf("decode task: %v", err)
	}

	scheduleRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules", token, map[string]any{
		"name":         "poll example source",
		"task_id":      task.ID,
		"trigger_type": "crons",
		"cron_expr":    "*/15 * * * *",
		"enabled":      false,
	})
	if scheduleRec.Code != http.StatusOK {
		t.Fatalf("create source schedule status=%d body=%s", scheduleRec.Code, scheduleRec.Body.String())
	}
	var schedule common.Schedule
	if err := json.Unmarshal(scheduleRec.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("decode schedule: %v", err)
	}
	if schedule.ProjectID != "" || schedule.VersionID != "" || schedule.EntryCommand != "" {
		t.Fatalf("source schedule unexpectedly requires code artifact: %#v", schedule)
	}
	steps, err := app.resolveScheduleExecutionSteps(schedule)
	if err != nil || len(steps) != 1 {
		t.Fatalf("resolve source schedule steps=%#v err=%v", steps, err)
	}
	if steps[0].Type != "rss" || steps[0].SourceURL != task.SourceURL {
		t.Fatalf("source schedule did not preserve RSS task: %#v", steps[0])
	}
}
