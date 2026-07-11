package control

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"araneae-go/internal/common"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (a *App) createSchedule(c *fiber.Ctx) error {
	var req createScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var normalizedLegacy *legacyScheduleOrder
	if legacy, ok := parseLegacyOrder(req.Order); ok {
		normalized, err := a.validateAndNormalizeLegacyOrder(legacy)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		normalizedLegacy = &normalized

		if req.Name == "" {
			req.Name = strings.TrimSpace(normalized.Name)
		}
		if len(normalized.Schedule) > 0 {
			step := normalized.Schedule[0]
			if req.Name == "" {
				req.Name = strings.TrimSpace(step.Name)
			}
			if req.TaskID == "" {
				req.TaskID = strings.TrimSpace(step.TaskID)
			}
			if req.ProjectID == "" {
				req.ProjectID = strings.TrimSpace(step.ProjectID)
			}
			if req.TriggerType == "" {
				req.TriggerType = strings.TrimSpace(step.Trigger)
			}
			if req.CronExpr == "" && strings.EqualFold(strings.TrimSpace(step.Trigger), "crons") {
				req.CronExpr = strings.TrimSpace(step.Crons)
			}
			if req.RunAt == "" && strings.EqualFold(strings.TrimSpace(step.Trigger), "datetime") {
				req.RunAt = strings.TrimSpace(step.RunAt)
			}
			if req.NodeQueue == "" && len(step.Node) > 0 {
				req.NodeQueue = laxString(strings.TrimSpace(step.Node[0]))
			}
		}
	}

	if req.TaskID != "" {
		var task common.Task
		if err := a.db.Where("id = ?", req.TaskID).First(&task).Error; err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "task not found")
		}
		if req.ProjectID != "" && strings.TrimSpace(req.ProjectID) != task.ProjectID {
			return fiber.NewError(fiber.StatusBadRequest, "task_id does not belong to project_id")
		}
		if req.VersionID != "" && strings.TrimSpace(req.VersionID) != task.VersionID {
			return fiber.NewError(fiber.StatusBadRequest, "task_id does not belong to version_id")
		}
		if req.ProjectID == "" {
			req.ProjectID = task.ProjectID
		}
		if req.VersionID == "" {
			req.VersionID = task.VersionID
		}
		if req.EntryCommand == "" {
			req.EntryCommand = task.EntryCommand
		}
		if req.NodeQueue == "" {
			req.NodeQueue = laxString(task.NodeQueue)
		}
	}

	req.ProjectID = strings.TrimSpace(req.ProjectID)
	req.VersionID = strings.TrimSpace(req.VersionID)
	req.EntryCommand = strings.TrimSpace(req.EntryCommand)
	req.CronExpr = strings.TrimSpace(req.CronExpr)
	req.TriggerType = strings.ToLower(strings.TrimSpace(req.TriggerType))
	req.RunAt = strings.TrimSpace(req.RunAt)
	req.NodeQueue = laxString(sanitizeNodeQueue(string(req.NodeQueue)))
	req.Name = strings.TrimSpace(req.Name)

	if req.NodeQueue == "" {
		req.NodeQueue = laxString("default")
	}
	if req.Name == "" {
		req.Name = "schedule-" + uuid.NewString()[:8]
	}
	if req.ProjectID == "" || req.VersionID == "" || req.EntryCommand == "" {
		return fiber.NewError(fiber.StatusBadRequest, "project_id, version_id and entry_command are required")
	}
	// Collect multiple firing times for the datetime trigger.
	var collectedRunTimes []time.Time
	if strings.EqualFold(strings.TrimSpace(req.TriggerType), "datetime") || (strings.TrimSpace(req.TriggerType) == "" && strings.TrimSpace(req.RunAt) != "") {
		rawTimes := []string{}
		if strings.TrimSpace(req.RunAt) != "" {
			rawTimes = append(rawTimes, strings.TrimSpace(req.RunAt))
		}
		for _, rt := range req.RunTimes {
			if strings.TrimSpace(rt) != "" {
				rawTimes = append(rawTimes, strings.TrimSpace(rt))
			}
		}
		seen := map[string]bool{}
		for _, r := range rawTimes {
			if seen[r] {
				continue
			}
			seen[r] = true
			parsed, perr := parseScheduleRunAt(r)
			if perr != nil {
				return fiber.NewError(fiber.StatusBadRequest, "invalid run_at ("+r+"): "+perr.Error())
			}
			collectedRunTimes = append(collectedRunTimes, *parsed)
		}
		if len(collectedRunTimes) > 0 {
			sort.Slice(collectedRunTimes, func(i, j int) bool { return collectedRunTimes[i].Before(collectedRunTimes[j]) })
			req.RunAt = formatScheduleRunAt(&collectedRunTimes[0])
		}
	}

	triggerType, cronExpr, runAt, err := normalizeScheduleTrigger(req.TriggerType, req.CronExpr, req.RunAt)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	req.TriggerType = triggerType
	req.CronExpr = cronExpr

	var project common.Project
	if err := a.db.Where("id = ?", req.ProjectID).First(&project).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "project not found")
	}
	canWrite, accessErr := a.canWriteProject(c, project)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canWrite {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	var version common.ArtifactVersion
	if err := a.db.Where("id = ? AND project_id = ?", req.VersionID, req.ProjectID).First(&version).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "version not found")
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	orderJSON := ""
	if req.Order != nil {
		if normalizedLegacy != nil {
			if b, err := json.Marshal(normalizedLegacy); err == nil {
				orderJSON = string(b)
			}
		}
		if orderJSON == "" {
			switch o := req.Order.(type) {
			case string:
				orderJSON = strings.TrimSpace(o)
			default:
				if b, err := json.Marshal(o); err == nil {
					orderJSON = string(b)
				}
			}
		}
	}
	if orderJSON == "" {
		trigger := req.TriggerType
		fallback := legacyScheduleOrder{
			Name: req.Name,
			Schedule: []legacyScheduleStep{{
				TaskID:     req.TaskID,
				TaskStatus: "exist",
				Name:       req.Name,
				ProjectID:  req.ProjectID,
				Node:       []string{string(req.NodeQueue)},
				Trigger:    trigger,
				Crons:      req.CronExpr,
				RunAt:      formatScheduleRunAt(runAt),
			}},
		}
		if b, err := json.Marshal(fallback); err == nil {
			orderJSON = string(b)
		}
	}

	uid, _ := c.Locals("uid").(string)
	now := time.Now()
	schedule := common.Schedule{
		ID:           uuid.NewString(),
		Name:         req.Name,
		Description:  req.Description,
		TaskID:       req.TaskID,
		ProjectID:    req.ProjectID,
		VersionID:    req.VersionID,
		EntryCommand: req.EntryCommand,
		CronExpr:     req.CronExpr,
		TriggerType:  req.TriggerType,
		RunAt:        runAt,
		NodeQueue:    string(req.NodeQueue),
		OrderJSON:    orderJSON,
		Enabled:      enabled,
		CreatedBy:    uid,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := a.db.Create(&schedule).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if req.TriggerType == "datetime" {
		for _, rt := range collectedRunTimes {
			row := common.ScheduleRunTime{
				ID:         uuid.NewString(),
				ScheduleID: schedule.ID,
				RunAt:      rt,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			if err := a.db.Create(&row).Error; err != nil {
				_ = a.db.Delete(&common.Schedule{}, "id = ?", schedule.ID).Error
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
		}
	}
	if schedule.Enabled {
		if err := a.registerCronSchedule(schedule); err != nil {
			_ = a.db.Delete(&common.Schedule{}, "id = ?", schedule.ID).Error
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}
	var created common.Schedule
	if err := a.db.Preload("RunTimes").Where("id = ?", schedule.ID).First(&created).Error; err != nil {
		return c.JSON(schedule)
	}
	return c.JSON(created)
}

func (a *App) listSchedules(c *fiber.Ctx) error {
	role, _ := c.Locals("role").(string)
	uid, _ := c.Locals("uid").(string)
	workplaceID, err := parseOptionalWorkplaceQueryID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var schedules []common.Schedule
	query := a.db.Order("created_at desc")
	if workplaceID != nil {
		projectByWorkplace := a.db.Model(&common.Project{}).Select("id").Where("workplace_id = ?", *workplaceID)
		query = query.Where("project_id IN (?)", projectByWorkplace)
	}

	if !isPrivilegedRole(role) {
		if workplaceID != nil {
			allowed, accessErr := a.userCanAccessWorkplace(uid, *workplaceID)
			if accessErr != nil {
				return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
			}
			if !allowed {
				return c.JSON([]common.Schedule{})
			}
		}

		accessibleWorkplaceIDs, scopeErr := a.userAccessibleWorkplaceIDs(uid)
		if scopeErr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, scopeErr.Error())
		}

		projectScope := a.db.Model(&common.Project{}).Select("id")
		if len(accessibleWorkplaceIDs) > 0 {
			projectScope = projectScope.Where("created_by = ? OR workplace_id IN ?", uid, accessibleWorkplaceIDs)
		} else {
			projectScope = projectScope.Where("created_by = ?", uid)
		}
		if workplaceID != nil {
			projectScope = projectScope.Where("workplace_id = ?", *workplaceID)
		}
		query = query.Where("project_id IN (?)", projectScope)
	}

	if err := query.Preload("RunTimes").Find(&schedules).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(schedules)
}

func (a *App) getSchedule(c *fiber.Ctx) error {
	id := c.Params("id")
	var schedule common.Schedule
	if err := a.db.Preload("RunTimes").Where("id = ?", id).First(&schedule).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "schedule not found")
	}
	allowed, accessErr := a.canAccessSchedule(c, schedule)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !allowed {
		return fiber.NewError(fiber.StatusNotFound, "schedule not found")
	}
	return c.JSON(schedule)
}

func (a *App) updateSchedule(c *fiber.Ctx) error {
	id := c.Params("id")
	var schedule common.Schedule
	if err := a.db.Where("id = ?", id).First(&schedule).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "schedule not found")
	}

	var req updateScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	runAtRaw := formatScheduleRunAt(schedule.RunAt)

	taskRebind := false
	if req.Name != nil {
		schedule.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		schedule.Description = strings.TrimSpace(*req.Description)
	}
	if req.TaskID != nil {
		schedule.TaskID = strings.TrimSpace(*req.TaskID)
		taskRebind = true
	}
	if req.ProjectID != nil {
		schedule.ProjectID = strings.TrimSpace(*req.ProjectID)
	}
	if req.VersionID != nil {
		schedule.VersionID = strings.TrimSpace(*req.VersionID)
	}
	if req.EntryCommand != nil {
		schedule.EntryCommand = strings.TrimSpace(*req.EntryCommand)
	}
	if req.CronExpr != nil {
		schedule.CronExpr = strings.TrimSpace(*req.CronExpr)
	}
	if req.TriggerType != nil {
		schedule.TriggerType = strings.ToLower(strings.TrimSpace(*req.TriggerType))
	}
	if req.RunAt != nil {
		runAtRaw = strings.TrimSpace(*req.RunAt)
	}
	if req.NodeQueue != nil {
		schedule.NodeQueue = sanitizeNodeQueue(string(*req.NodeQueue))
	}
	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}
	if req.Order != nil {
		if legacy, ok := parseLegacyOrder(req.Order); ok {
			normalized, err := a.validateAndNormalizeLegacyOrder(legacy)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			req.Order = normalized
			if len(normalized.Schedule) > 0 {
				step := normalized.Schedule[0]
				if req.TriggerType == nil {
					value := strings.TrimSpace(step.Trigger)
					req.TriggerType = &value
					schedule.TriggerType = strings.ToLower(value)
				}
				if req.CronExpr == nil && strings.EqualFold(strings.TrimSpace(step.Trigger), "crons") {
					schedule.CronExpr = strings.TrimSpace(step.Crons)
				}
				if req.RunAt == nil && strings.EqualFold(strings.TrimSpace(step.Trigger), "datetime") {
					runAtRaw = strings.TrimSpace(step.RunAt)
				}
			}
		}
		orderJSON, err := marshalOrderJSON(req.Order)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		schedule.OrderJSON = orderJSON
	}
	triggerType, cronExpr, runAt, triggerErr := normalizeScheduleTrigger(schedule.TriggerType, schedule.CronExpr, runAtRaw)
	if triggerErr != nil {
		return fiber.NewError(fiber.StatusBadRequest, triggerErr.Error())
	}
	schedule.TriggerType = triggerType
	schedule.CronExpr = cronExpr
	schedule.RunAt = runAt

	if schedule.TriggerType == "datetime" {
		replaceRunTimes := false
		var newRunTimes []time.Time
		if req.RunTimes != nil {
			replaceRunTimes = true
			rawTimes := []string{}
			if req.RunAt != nil && strings.TrimSpace(*req.RunAt) != "" {
				rawTimes = append(rawTimes, strings.TrimSpace(*req.RunAt))
			}
			for _, rt := range *req.RunTimes {
				if strings.TrimSpace(rt) != "" {
					rawTimes = append(rawTimes, strings.TrimSpace(rt))
				}
			}
			seen := map[string]bool{}
			for _, r := range rawTimes {
				if seen[r] {
					continue
				}
				seen[r] = true
				parsed, perr := parseScheduleRunAt(r)
				if perr != nil {
					return fiber.NewError(fiber.StatusBadRequest, "invalid run_at ("+r+"): "+perr.Error())
				}
				newRunTimes = append(newRunTimes, *parsed)
			}
			if len(newRunTimes) == 0 {
				return fiber.NewError(fiber.StatusBadRequest, "at least one run_at is required for datetime trigger")
			}
		} else if req.RunAt != nil {
			replaceRunTimes = true
			parsed, perr := parseScheduleRunAt(strings.TrimSpace(*req.RunAt))
			if perr != nil {
				return fiber.NewError(fiber.StatusBadRequest, perr.Error())
			}
			newRunTimes = []time.Time{*parsed}
		}
		if replaceRunTimes {
			if err := a.db.Where("schedule_id = ?", schedule.ID).Delete(&common.ScheduleRunTime{}).Error; err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, err.Error())
			}
			sort.Slice(newRunTimes, func(i, j int) bool { return newRunTimes[i].Before(newRunTimes[j]) })
			now := time.Now()
			for _, rt := range newRunTimes {
				row := common.ScheduleRunTime{ID: uuid.NewString(), ScheduleID: schedule.ID, RunAt: rt, CreatedAt: now, UpdatedAt: now}
				if err := a.db.Create(&row).Error; err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, err.Error())
				}
			}
			schedule.RunAt = &newRunTimes[0]
		}
	} else {
		if err := a.db.Where("schedule_id = ?", schedule.ID).Delete(&common.ScheduleRunTime{}).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	if schedule.TaskID != "" {
		var task common.Task
		if err := a.db.Where("id = ?", schedule.TaskID).First(&task).Error; err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "task not found")
		}
		if schedule.ProjectID != task.ProjectID {
			return fiber.NewError(fiber.StatusBadRequest, "task_id does not belong to project_id")
		}
		if schedule.VersionID != task.VersionID {
			return fiber.NewError(fiber.StatusBadRequest, "task_id does not belong to version_id")
		}
		if taskRebind {
			if req.ProjectID == nil || schedule.ProjectID == "" {
				schedule.ProjectID = task.ProjectID
			}
			if req.VersionID == nil || schedule.VersionID == "" {
				schedule.VersionID = task.VersionID
			}
			if req.EntryCommand == nil || schedule.EntryCommand == "" {
				schedule.EntryCommand = task.EntryCommand
			}
			if req.NodeQueue == nil || schedule.NodeQueue == "" {
				schedule.NodeQueue = task.NodeQueue
			}
		}
	}

	if schedule.NodeQueue == "" {
		schedule.NodeQueue = "default"
	}
	if schedule.Name == "" {
		schedule.Name = "schedule-" + schedule.ID[:8]
	}
	if schedule.ProjectID == "" || schedule.VersionID == "" || schedule.EntryCommand == "" {
		return fiber.NewError(fiber.StatusBadRequest, "project_id, version_id and entry_command are required")
	}

	var project common.Project
	if err := a.db.Where("id = ?", schedule.ProjectID).First(&project).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "project not found")
	}
	canWriteProject, projectAccessErr := a.canWriteProject(c, project)
	if projectAccessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, projectAccessErr.Error())
	}
	if !canWriteProject {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	var version common.ArtifactVersion
	if err := a.db.Where("id = ? AND project_id = ?", schedule.VersionID, schedule.ProjectID).First(&version).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "version not found")
	}

	schedule.UpdatedAt = time.Now()
	if err := a.db.Save(&schedule).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	a.unregisterCronSchedule(schedule.ID)
	if schedule.Enabled {
		if err := a.registerCronSchedule(schedule); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}

	var updated common.Schedule
	if err := a.db.Preload("RunTimes").Where("id = ?", schedule.ID).First(&updated).Error; err != nil {
		return c.JSON(schedule)
	}
	return c.JSON(updated)
}

func (a *App) deleteSchedule(c *fiber.Ctx) error {
	id := c.Params("id")
	var schedule common.Schedule
	if err := a.db.Preload("RunTimes").Where("id = ?", id).First(&schedule).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "schedule not found")
	}
	canWrite, accessErr := a.canWriteSchedule(c, schedule)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canWrite {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}

	a.unregisterCronSchedule(schedule.ID)
	if err := a.db.Where("schedule_id = ?", schedule.ID).Delete(&common.ScheduleRunTime{}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := a.db.Delete(&common.Schedule{}, "id = ?", schedule.ID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (a *App) enableSchedule(c *fiber.Ctx) error {
	return a.setScheduleEnabled(c, true)
}

func (a *App) disableSchedule(c *fiber.Ctx) error {
	return a.setScheduleEnabled(c, false)
}

func (a *App) setScheduleEnabled(c *fiber.Ctx, enabled bool) error {
	id := c.Params("id")
	var schedule common.Schedule
	if err := a.db.Preload("RunTimes").Where("id = ?", id).First(&schedule).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "schedule not found")
	}
	canWrite, accessErr := a.canWriteSchedule(c, schedule)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canWrite {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}

	schedule.Enabled = enabled
	schedule.UpdatedAt = time.Now()
	if err := a.db.Save(&schedule).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	a.unregisterCronSchedule(schedule.ID)
	if schedule.Enabled {
		if err := a.registerCronSchedule(schedule); err != nil {
			schedule.Enabled = false
			schedule.UpdatedAt = time.Now()
			_ = a.db.Save(&schedule).Error
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}

	return c.JSON(schedule)
}

func (a *App) triggerSchedule(c *fiber.Ctx) error {
	id := c.Params("id")
	var schedule common.Schedule
	if err := a.db.Preload("RunTimes").Where("id = ?", id).First(&schedule).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "schedule not found")
	}
	canWrite, accessErr := a.canWriteSchedule(c, schedule)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canWrite {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	hasRecent, recentErr := a.hasRecentManualRun("", schedule.ID)
	if recentErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, recentErr.Error())
	}
	if hasRecent {
		a.recordSecurityEvent(c, "schedule_trigger_duplicate_blocked", "warning", "schedule_id="+schedule.ID)
		return fiber.NewError(fiber.StatusTooManyRequests, "schedule trigger is already in progress")
	}
	run, err := a.publishScheduleRun(schedule, "manual")
	if err != nil {
		if errors.Is(err, errQueueUnavailable) {
			return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(run)
}

func (a *App) listScheduleRuns(c *fiber.Ctx) error {
	id := c.Params("id")
	var schedule common.Schedule
	if err := a.db.Preload("RunTimes").Where("id = ?", id).First(&schedule).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "schedule not found")
	}
	allowed, accessErr := a.canAccessSchedule(c, schedule)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !allowed {
		return fiber.NewError(fiber.StatusNotFound, "schedule not found")
	}

	var runs []common.TaskRun
	if err := a.db.Where("schedule_id = ?", id).Order("created_at desc").Find(&runs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(runs)
}

func parseLegacyOrder(raw any) (legacyScheduleOrder, bool) {
	if raw == nil {
		return legacyScheduleOrder{}, false
	}
	var data []byte
	switch v := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return legacyScheduleOrder{}, false
		}
		data = []byte(trimmed)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return legacyScheduleOrder{}, false
		}
		data = b
	}
	var parsed legacyScheduleOrder
	if err := json.Unmarshal(data, &parsed); err != nil {
		return legacyScheduleOrder{}, false
	}
	return parsed, true
}

func (a *App) validateAndNormalizeLegacyOrder(order legacyScheduleOrder) (legacyScheduleOrder, error) {
	order.Name = strings.TrimSpace(order.Name)
	if len(order.Schedule) == 0 {
		return legacyScheduleOrder{}, errors.New("order.schedule must contain at least one task")
	}

	normalized := make([]legacyScheduleStep, 0, len(order.Schedule))
	for idx, rawStep := range order.Schedule {
		step := rawStep
		step.TaskID = strings.TrimSpace(step.TaskID)
		if step.TaskID == "" {
			return legacyScheduleOrder{}, errors.New("each schedule step must reference an existing task_id")
		}

		var task common.Task
		if err := a.db.Where("id = ?", step.TaskID).First(&task).Error; err != nil {
			return legacyScheduleOrder{}, errors.New("schedule step references a task that does not exist")
		}

		step.Name = strings.TrimSpace(step.Name)
		if step.Name == "" {
			step.Name = task.Name
		}

		step.ProjectID = strings.TrimSpace(step.ProjectID)
		if step.ProjectID == "" {
			step.ProjectID = task.ProjectID
		}

		cleanNodes := make([]string, 0, len(step.Node))
		for _, node := range step.Node {
			trimmed := strings.TrimSpace(node)
			if trimmed != "" {
				cleanNodes = append(cleanNodes, trimmed)
			}
		}
		if len(cleanNodes) == 0 {
			if queue := strings.TrimSpace(task.NodeQueue); queue != "" {
				cleanNodes = []string{queue}
			}
		}
		step.Node = cleanNodes

		if idx == 0 {
			trigger := strings.ToLower(strings.TrimSpace(step.Trigger))
			if trigger == "" {
				if strings.TrimSpace(step.Crons) != "" {
					trigger = "crons"
				} else if strings.TrimSpace(step.RunAt) != "" {
					trigger = "datetime"
				} else {
					trigger = "api"
				}
			}
			if trigger != "crons" && trigger != "api" && trigger != "datetime" {
				return legacyScheduleOrder{}, errors.New("first schedule step trigger must be crons, api or datetime")
			}

			step.Trigger = trigger
			step.Crons = strings.TrimSpace(step.Crons)
			step.RunAt = strings.TrimSpace(step.RunAt)
			if trigger == "crons" && step.Crons == "" {
				return legacyScheduleOrder{}, errors.New("cron expression is required when first step trigger is crons")
			}
			if trigger == "datetime" {
				runAt, err := parseScheduleRunAt(step.RunAt)
				if err != nil {
					return legacyScheduleOrder{}, err
				}
				step.RunAt = runAt.Format(time.RFC3339)
				step.Crons = ""
			}
			if trigger == "api" || trigger == "crons" {
				step.RunAt = ""
			}
			if trigger == "api" {
				step.Crons = ""
			}
			step.Previous = ""
		} else {
			trigger := strings.ToLower(strings.TrimSpace(step.Trigger))
			if trigger != "" && trigger != "previous" {
				return legacyScheduleOrder{}, errors.New("only the first schedule step can use crons or api trigger")
			}
			prevName := normalized[idx-1].Name
			if prevName == "" {
				prevName = normalized[idx-1].TaskID
			}
			step.Trigger = "previous"
			step.Crons = ""
			step.RunAt = ""
			step.Previous = prevName
		}

		step.TaskStatus = "exist"
		normalized = append(normalized, step)
	}

	if order.Name == "" {
		order.Name = normalized[0].Name
	}
	order.Schedule = normalized
	return order, nil
}

func marshalOrderJSON(raw any) (string, error) {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

func normalizeScheduleTrigger(triggerType, cronExpr, runAtRaw string) (string, string, *time.Time, error) {
	trigger := strings.ToLower(strings.TrimSpace(triggerType))
	cron := strings.TrimSpace(cronExpr)
	runAtRaw = strings.TrimSpace(runAtRaw)
	if trigger == "" {
		if cron != "" {
			trigger = "crons"
		} else if runAtRaw != "" {
			trigger = "datetime"
		} else {
			trigger = "api"
		}
	}
	switch trigger {
	case "crons":
		if cron == "" {
			return "", "", nil, errors.New("cron expression is required when trigger_type is crons")
		}
		return trigger, cron, nil, nil
	case "datetime":
		runAt, err := parseScheduleRunAt(runAtRaw)
		if err != nil {
			return "", "", nil, err
		}
		return trigger, "", runAt, nil
	case "api":
		return trigger, "", nil, nil
	default:
		return "", "", nil, errors.New("trigger_type must be one of: api, crons, datetime")
	}
}

func parseScheduleRunAt(raw string) (*time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errors.New("run_at is required when trigger_type is datetime")
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return nil, errors.New("run_at must be RFC3339 with timezone and seconds (example: 2026-06-18T09:30:00+08:00)")
		}
	}
	if !strings.Contains(value, "T") {
		return nil, errors.New("run_at must be RFC3339 with timezone and seconds (example: 2026-06-18T09:30:00+08:00)")
	}
	return &parsed, nil
}

func formatScheduleRunAt(runAt *time.Time) string {
	if runAt == nil {
		return ""
	}
	return runAt.Format(time.RFC3339)
}
