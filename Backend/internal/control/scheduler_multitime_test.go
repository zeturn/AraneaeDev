package control

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"araneae-go/internal/common"
)

type testRunTime struct {
	ID          string  `json:"id"`
	RunAt       string  `json:"run_at"`
	TriggeredAt *string `json:"triggered_at"`
}

type testSchedule struct {
	ID          string       `json:"id"`
	TriggerType string       `json:"trigger_type"`
	RunAt       string       `json:"run_at"`
	Enabled     bool         `json:"enabled"`
	RunTimes    []testRunTime `json:"run_times"`
}

func futureRunAt(offset time.Duration) string {
	return time.Now().Add(offset).Format(time.RFC3339)
}

func countRunTimeRows(t *testing.T, app *App, scheduleID string) int {
	t.Helper()
	var count int64
	if err := app.db.Model(&common.ScheduleRunTime{}).Where("schedule_id = ?", scheduleID).Count(&count).Error; err != nil {
		t.Fatalf("count run_time rows: %v", err)
	}
	return int(count)
}

func countPendingTimers(app *App, scheduleID string) int {
	count := 0
	for key := range app.scheduleTimers {
		if key == scheduleID || len(key) > len(scheduleID) && key[:len(scheduleID)+1] == scheduleID+"/" {
			count++
		}
	}
	return count
}

func createMultiTimeSchedule(t *testing.T, app *App, token string, runTimes []string) testSchedule {
	t.Helper()
	_, _, task := createProjectVersionTask(t, app, token, "")

	payload := map[string]any{
		"name":         "multi-time-schedule",
		"trigger_type": "datetime",
		"run_times":    runTimes,
		"task_id":      task.ID,
		"project_id":   task.ProjectID,
		"version_id":   task.VersionID,
		"entry_command": task.EntryCommand,
		"enabled":      true,
	}
	rec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules", token, payload)
	if rec.Code != http.StatusOK {
		t.Fatalf("create schedule status=%d body=%s", rec.Code, rec.Body.String())
	}
	var sched testSchedule
	if err := json.Unmarshal(rec.Body.Bytes(), &sched); err != nil {
		t.Fatalf("decode schedule: %v", err)
	}
	return sched
}

func TestCreateScheduleMultipleRunTimes(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)

	times := []string{futureRunAt(3 * time.Hour), futureRunAt(1 * time.Hour), futureRunAt(2 * time.Hour)}
	sched := createMultiTimeSchedule(t, app, token, times)

	if sched.TriggerType != "datetime" {
		t.Fatalf("expected trigger_type datetime, got %s", sched.TriggerType)
	}
	if len(sched.RunTimes) != 3 {
		t.Fatalf("expected 3 run_times, got %d (%+v)", len(sched.RunTimes), sched.RunTimes)
	}
	// Earliest time becomes schedule.RunAt.
	if sched.RunAt != times[1] {
		t.Fatalf("expected schedule.run_at=%s (earliest), got %s", times[1], sched.RunAt)
	}
	// DB must persist one row per time.
	if n := countRunTimeRows(t, app, sched.ID); n != 3 {
		t.Fatalf("expected 3 run_time rows in DB, got %d", n)
	}
	// Scheduler must register one timer per pending time.
	if n := countPendingTimers(app, sched.ID); n != 3 {
		t.Fatalf("expected 3 pending timers, got %d", n)
	}
}

func TestRunOneTimeScheduleMarksTriggered(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)

	times := []string{futureRunAt(1 * time.Hour), futureRunAt(2 * time.Hour)}
	sched := createMultiTimeSchedule(t, app, token, times)

	var rts []common.ScheduleRunTime
	if err := app.db.Where("schedule_id = ?", sched.ID).Order("run_at asc").Find(&rts).Error; err != nil {
		t.Fatalf("load run_times: %v", err)
	}
	if len(rts) != 2 {
		t.Fatalf("expected 2 run_times, got %d", len(rts))
	}
	rt1, rt2 := rts[0], rts[1]

	// Fire the first time.
	var schedule common.Schedule
	if err := app.db.Where("id = ?", sched.ID).First(&schedule).Error; err != nil {
		t.Fatalf("load schedule: %v", err)
	}
	app.runOneTimeSchedule(schedule, rt1.ID)

	var fired1 common.ScheduleRunTime
	if err := app.db.Where("id = ?", rt1.ID).First(&fired1).Error; err != nil {
		t.Fatalf("load rt1: %v", err)
	}
	if fired1.TriggeredAt == nil {
		t.Fatalf("expected rt1 to be marked triggered")
	}
	var runCount int64
	app.db.Model(&common.TaskRun{}).Where("schedule_id = ? AND trigger_source = ?", sched.ID, "schedule_datetime").Count(&runCount)
	if runCount != 1 {
		t.Fatalf("expected 1 published run after first firing, got %d", runCount)
	}
	var afterFirst common.Schedule
	app.db.Where("id = ?", sched.ID).First(&afterFirst)
	if !afterFirst.Enabled {
		t.Fatalf("schedule should stay enabled after first of multiple times")
	}
	if afterFirst.RunAt == nil || afterFirst.RunAt.Format(time.RFC3339) != rt2.RunAt.Format(time.RFC3339) {
		t.Fatalf("expected schedule.run_at advanced to second time")
	}

	// Fire the second (last) time: schedule should become disabled.
	app.runOneTimeSchedule(afterFirst, rt2.ID)
	var fired2 common.ScheduleRunTime
	app.db.Where("id = ?", rt2.ID).First(&fired2)
	if fired2.TriggeredAt == nil {
		t.Fatalf("expected rt2 to be marked triggered")
	}
	var afterSecond common.Schedule
	app.db.Where("id = ?", sched.ID).First(&afterSecond)
	if afterSecond.Enabled {
		t.Fatalf("schedule should be disabled after all times consumed")
	}
	if countPendingTimers(app, sched.ID) != 0 {
		t.Fatalf("expected no pending timers after schedule disabled")
	}
}

func TestUpdateScheduleReplacesRunTimes(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)

	sched := createMultiTimeSchedule(t, app, token, []string{futureRunAt(1 * time.Hour)})
	if n := countRunTimeRows(t, app, sched.ID); n != 1 {
		t.Fatalf("expected 1 run_time after create, got %d", n)
	}

	// Replace with two new times.
	newTimes := []string{futureRunAt(5 * time.Hour), futureRunAt(6 * time.Hour)}
	rec := doJSONRequest(t, app, http.MethodPut, "/api/v1/schedules/"+sched.ID, token, map[string]any{
		"trigger_type": "datetime",
		"run_times":    newTimes,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update schedule status=%d body=%s", rec.Code, rec.Body.String())
	}
	var updated testSchedule
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated schedule: %v", err)
	}
	if len(updated.RunTimes) != 2 {
		t.Fatalf("expected 2 run_times after update, got %d", len(updated.RunTimes))
	}
	if n := countRunTimeRows(t, app, sched.ID); n != 2 {
		t.Fatalf("expected 2 run_time rows after replacement, got %d", n)
	}
	if updated.RunAt != newTimes[0] {
		t.Fatalf("expected schedule.run_at=%s after update, got %s", newTimes[0], updated.RunAt)
	}
}

func TestUpdateScheduleSwitchToCronsClearsRunTimes(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)

	sched := createMultiTimeSchedule(t, app, token, []string{futureRunAt(1 * time.Hour), futureRunAt(2 * time.Hour)})
	if n := countRunTimeRows(t, app, sched.ID); n != 2 {
		t.Fatalf("expected 2 run_times after create, got %d", n)
	}

	rec := doJSONRequest(t, app, http.MethodPut, "/api/v1/schedules/"+sched.ID, token, map[string]any{
		"trigger_type": "crons",
		"cron_expr":    "0 * * * * *",
		"run_times":    []string{},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update schedule status=%d body=%s", rec.Code, rec.Body.String())
	}
	if n := countRunTimeRows(t, app, sched.ID); n != 0 {
		t.Fatalf("expected run_time rows cleared when switching to crons, got %d", n)
	}
}
