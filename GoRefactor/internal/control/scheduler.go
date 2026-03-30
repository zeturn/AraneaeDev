package control

import (
	"fmt"

	"araneae-go/internal/common"
	"go.uber.org/zap"
)

func (a *App) loadCronTasks() error {
	var tasks []common.Task
	if err := a.db.Where("enabled = ? AND cron_expr <> ''", true).Find(&tasks).Error; err != nil {
		return err
	}
	for _, t := range tasks {
		if err := a.registerCronTask(t); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) registerCronTask(task common.Task) error {
	if task.CronExpr == "" {
		return nil
	}
	a.cronMu.Lock()
	defer a.cronMu.Unlock()

	if old, ok := a.cronEntries[task.ID]; ok {
		a.cron.Remove(old)
	}
	entryID, err := a.cron.AddFunc(task.CronExpr, func() {
		if _, e := a.publishTaskRun(task, "schedule", ""); e != nil {
			a.log.Error("scheduled trigger failed", zap.Error(e), zap.String("task_id", task.ID))
		}
	})
	if err != nil {
		return fmt.Errorf("register cron task %s: %w", task.ID, err)
	}
	a.cronEntries[task.ID] = entryID
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
	if err := a.db.Where("enabled = ? AND cron_expr <> ''", true).Find(&schedules).Error; err != nil {
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
	if schedule.CronExpr == "" {
		return nil
	}
	a.cronMu.Lock()
	defer a.cronMu.Unlock()

	if old, ok := a.scheduleEntries[schedule.ID]; ok {
		a.cron.Remove(old)
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
	return nil
}

func (a *App) unregisterCronSchedule(scheduleID string) {
	a.cronMu.Lock()
	defer a.cronMu.Unlock()

	if old, ok := a.scheduleEntries[scheduleID]; ok {
		a.cron.Remove(old)
		delete(a.scheduleEntries, scheduleID)
	}
}
