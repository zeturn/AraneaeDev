package control

import (
	"fmt"
	"strings"
	"time"

	"araneae-go/internal/common"
	"github.com/google/uuid"
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
	a.clearScheduleTimers(schedule.ID)

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
		runTimes, err := a.loadPendingRunTimes(schedule)
		if err != nil {
			return err
		}
		if len(runTimes) == 0 {
			// No pending firing times left: disable the schedule so it does not linger.
			if schedule.Enabled {
				_ = a.db.Model(&common.Schedule{}).Where("id = ?", schedule.ID).Update("enabled", false).Error
			}
			return nil
		}
		for _, rt := range runTimes {
			delay := time.Until(rt.RunAt)
			if delay <= 0 {
				go a.runOneTimeSchedule(schedule, rt.ID)
				continue
			}
			timer := time.AfterFunc(delay, func() {
				a.runOneTimeSchedule(schedule, rt.ID)
			})
			a.scheduleTimers[schedule.ID+"/"+rt.ID] = timer
		}
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
	a.clearScheduleTimers(scheduleID)
}

// clearScheduleTimers stops and removes every pending one-time timer belonging to a
// schedule. Timers are keyed as "<scheduleID>/<runTimeID>" so we match by prefix.
func (a *App) clearScheduleTimers(scheduleID string) {
	for key, timer := range a.scheduleTimers {
		if key == scheduleID || strings.HasPrefix(key, scheduleID+"/") {
			timer.Stop()
			delete(a.scheduleTimers, key)
		}
	}
}

// loadPendingRunTimes returns the not-yet-fired firing times for a schedule, ordered
// ascending. For backwards compatibility, a legacy datetime schedule that only has a
// single Schedule.RunAt (and no ScheduleRunTime rows yet) is migrated into a row.
func (a *App) loadPendingRunTimes(schedule common.Schedule) ([]common.ScheduleRunTime, error) {
	var runTimes []common.ScheduleRunTime
	if err := a.db.Where("schedule_id = ? AND triggered_at IS NULL", schedule.ID).Order("run_at asc").Find(&runTimes).Error; err != nil {
		return nil, err
	}
	if len(runTimes) > 0 {
		return runTimes, nil
	}
	if schedule.TriggerType == "datetime" && schedule.RunAt != nil {
		rt := common.ScheduleRunTime{
			ID:         uuid.NewString(),
			ScheduleID: schedule.ID,
			RunAt:      *schedule.RunAt,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := a.db.Create(&rt).Error; err != nil {
			return nil, err
		}
		return []common.ScheduleRunTime{rt}, nil
	}
	return nil, nil
}

func (a *App) runOneTimeSchedule(schedule common.Schedule, runTimeID string) {
	var fresh common.Schedule
	if err := a.db.Where("id = ?", schedule.ID).First(&fresh).Error; err != nil {
		return
	}
	if !fresh.Enabled {
		return
	}
	var rt common.ScheduleRunTime
	if err := a.db.Where("id = ?", runTimeID).First(&rt).Error; err != nil {
		return
	}
	if rt.TriggeredAt != nil {
		return
	}
	if _, err := a.runPublisher(fresh, "schedule_datetime"); err != nil {
		a.log.Error("datetime trigger failed", zap.Error(err), zap.String("schedule_id", fresh.ID))
		return
	}

	now := time.Now()
	if err := a.db.Model(&common.ScheduleRunTime{}).Where("id = ?", runTimeID).Updates(map[string]any{
		"triggered_at": &now,
		"updated_at":   now,
	}).Error; err != nil {
		a.log.Error("failed to mark run_time triggered", zap.Error(err), zap.String("schedule_id", fresh.ID))
	}

	var pending []common.ScheduleRunTime
	a.db.Where("schedule_id = ? AND triggered_at IS NULL", fresh.ID).Order("run_at asc").Find(&pending)
	if len(pending) == 0 {
		a.db.Model(&common.Schedule{}).Where("id = ?", fresh.ID).Updates(map[string]any{
			"enabled":           false,
			"run_at":            nil,
			"last_triggered_at": &now,
			"updated_at":        now,
		})
		a.unregisterCronSchedule(fresh.ID)
		return
	}
	a.db.Model(&common.Schedule{}).Where("id = ?", fresh.ID).Updates(map[string]any{
		"run_at":            pending[0].RunAt,
		"last_triggered_at": &now,
		"updated_at":        now,
	})
}
