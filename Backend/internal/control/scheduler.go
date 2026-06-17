package control

import (
	"fmt"
	"time"

	"araneae-go/internal/common"
	"go.uber.org/zap"
)

func (a *App) loadCronTasks() error {
	// Task-level cron is disabled: only schedules are allowed to run by cron.
	if err := a.db.Model(&common.Task{}).Where("cron_expr <> ''").Update("cron_expr", "").Error; err != nil {
		return err
	}
	return nil
}

func (a *App) registerCronTask(task common.Task) error {
	_ = task
	return nil
}

func (a *App) unregisterCronTask(taskID string) {
	a.cronMu.Lock()
	defer a.cronMu.Unlock()

	if old, ok := a.cronEntries[taskID]; ok {
		a.cron.Remove(old)
		delete(a.cronEntries, taskID)
	}
}

func (a *App) loadCronSchedules() error {
	var schedules []common.Schedule
	if err := a.db.Where("enabled = ?", true).Find(&schedules).Error; err != nil {
		return err
	}
	for _, s := range schedules {
		if err := a.registerCronSchedule(s); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) registerCronSchedule(schedule common.Schedule) error {
	a.cronMu.Lock()
	defer a.cronMu.Unlock()

	if old, ok := a.scheduleEntries[schedule.ID]; ok {
		a.cron.Remove(old)
		delete(a.scheduleEntries, schedule.ID)
	}
	if timer, ok := a.scheduleTimers[schedule.ID]; ok {
		timer.Stop()
		delete(a.scheduleTimers, schedule.ID)
	}

	triggerType := schedule.TriggerType
	if triggerType == "" {
		if schedule.CronExpr != "" {
			triggerType = "crons"
		} else if schedule.RunAt != nil {
			triggerType = "datetime"
		} else {
			triggerType = "api"
		}
	}
	switch triggerType {
	case "crons":
		if schedule.CronExpr == "" {
			return nil
		}
		entryID, err := a.cron.AddFunc(schedule.CronExpr, func() {
			if _, e := a.publishScheduleRun(schedule, "schedule"); e != nil {
				a.log.Error("scheduled trigger failed", zap.Error(e), zap.String("schedule_id", schedule.ID))
			}
		})
		if err != nil {
			return fmt.Errorf("register cron schedule %s: %w", schedule.ID, err)
		}
		a.scheduleEntries[schedule.ID] = entryID
	case "datetime":
		if schedule.RunAt == nil {
			return nil
		}
		delay := time.Until(*schedule.RunAt)
		if delay <= 0 {
			go a.runOneTimeSchedule(schedule)
			return nil
		}
		timer := time.AfterFunc(delay, func() {
			a.runOneTimeSchedule(schedule)
		})
		a.scheduleTimers[schedule.ID] = timer
	default:
		return nil
	}
	return nil
}

func (a *App) unregisterCronSchedule(scheduleID string) {
	a.cronMu.Lock()
	defer a.cronMu.Unlock()

	if old, ok := a.scheduleEntries[scheduleID]; ok {
		a.cron.Remove(old)
		delete(a.scheduleEntries, scheduleID)
	}
	if timer, ok := a.scheduleTimers[scheduleID]; ok {
		timer.Stop()
		delete(a.scheduleTimers, scheduleID)
	}
}

func (a *App) runOneTimeSchedule(schedule common.Schedule) {
	var fresh common.Schedule
	if err := a.db.Where("id = ?", schedule.ID).First(&fresh).Error; err != nil {
		return
	}
	if !fresh.Enabled {
		return
	}
	if fresh.LastTriggeredAt != nil {
		return
	}
	if _, err := a.publishScheduleRun(fresh, "schedule_datetime"); err != nil {
		a.log.Error("datetime trigger failed", zap.Error(err), zap.String("schedule_id", fresh.ID))
		return
	}
	now := time.Now()
	_ = a.db.Model(&common.Schedule{}).Where("id = ?", fresh.ID).Updates(map[string]any{
		"enabled":    false,
		"updated_at": now,
	}).Error
	a.unregisterCronSchedule(fresh.ID)
}
