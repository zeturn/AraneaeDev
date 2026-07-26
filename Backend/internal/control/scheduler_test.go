package control

import (
	"testing"
	"time"

	"araneae-go/internal/common"
)

func TestRunCronScheduleReloadsEnabledState(t *testing.T) {
	app := newTestControlApp(t)
	schedule := common.Schedule{
		ID:          "disabled-cron-schedule",
		Name:        "disabled cron schedule",
		TriggerType: "crons",
		CronExpr:    "* * * * *",
		Enabled:     false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := app.db.Create(&schedule).Error; err != nil {
		t.Fatal(err)
	}

	published := false
	app.runPublisher = func(received common.Schedule, source string) (*common.TaskRun, error) {
		published = true
		if received.ID != schedule.ID || source != "schedule" {
			t.Fatalf("unexpected publish request: %#v source=%q", received, source)
		}
		return &common.TaskRun{}, nil
	}

	app.runCronSchedule(schedule.ID)
	if published {
		t.Fatal("disabled schedule was published")
	}

	if err := app.db.Model(&common.Schedule{}).Where("id = ?", schedule.ID).Update("enabled", true).Error; err != nil {
		t.Fatal(err)
	}
	app.runCronSchedule(schedule.ID)
	if !published {
		t.Fatal("enabled schedule was not published")
	}
}
