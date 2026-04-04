package control

import (
	"crypto/subtle"
	"strings"

	"araneae-go/internal/common"
	"araneae-go/internal/control/contracts"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func (a *App) listRecentRuns(c *fiber.Ctx) error {
	role, _ := c.Locals("role").(string)
	uid, _ := c.Locals("uid").(string)
	workplaceID, err := parseOptionalWorkplaceQueryID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	var runs []common.TaskRun
	if isPrivilegedRole(role) {
		query := a.db.Order("created_at desc").Limit(200)
		if workplaceID != nil {
			projectByWorkplace := a.db.Model(&common.Project{}).Select("id").Where("workplace_id = ?", *workplaceID)
			taskByProject := a.db.Model(&common.Task{}).Select("id").Where("project_id IN (?)", projectByWorkplace)
			query = query.Where("task_id IN (?)", taskByProject)
		}
		if err := query.Find(&runs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	} else {
		if workplaceID != nil {
			allowed, accessErr := a.userCanAccessWorkplace(uid, *workplaceID)
			if accessErr != nil {
				return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
			}
			if !allowed {
				return c.JSON(fiber.Map{"records": []common.TaskRun{}, "count": 0})
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

		taskScope := a.db.Model(&common.Task{}).Select("id").Where("project_id IN (?)", projectScope)
		if err := a.db.Where("task_id IN (?)", taskScope).Order("created_at desc").Limit(200).Find(&runs).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}
	return c.JSON(fiber.Map{
		"records": runs,
		"count":   len(runs),
	})
}

func (a *App) runCallback(c *fiber.Ctx) error {
	runID := c.Params("id")
	providedKey := strings.TrimSpace(c.Get("X-Execution-Key"))
	providedRunToken := strings.TrimSpace(c.Get("X-Run-Token"))
	providedCorrelationID := strings.TrimSpace(c.Get("X-Correlation-ID"))
	expectedKey := strings.TrimSpace(a.cfg.ExecutionAPIKey)
	if expectedKey == "" || subtle.ConstantTimeCompare([]byte(providedKey), []byte(expectedKey)) != 1 {
		a.recordSecurityEvent(c, "callback_invalid_key", "critical", "run_id="+runID)
		return fiber.NewError(fiber.StatusUnauthorized, "invalid execution key")
	}
	var existingRun common.TaskRun
	if err := a.db.Where("id = ?", runID).First(&existingRun).Error; err != nil {
		a.recordSecurityEvent(c, "callback_run_not_found", "warning", "run_id="+runID)
		return fiber.NewError(fiber.StatusNotFound, "run not found")
	}
	if isTerminalRunStatus(existingRun.Status) {
		a.recordSecurityEvent(c, "callback_replay_ignored", "warning", "run_id="+runID)
		return c.JSON(fiber.Map{"ok": true, "ignored": true})
	}
	if providedRunToken == "" || hashNodeKey(providedRunToken) != existingRun.RunTokenHash {
		a.recordSecurityEvent(c, "callback_invalid_run_token", "critical", "run_id="+runID)
		return fiber.NewError(fiber.StatusUnauthorized, "invalid run token")
	}
	if providedCorrelationID == "" || subtle.ConstantTimeCompare([]byte(providedCorrelationID), []byte(existingRun.CorrelationID)) != 1 {
		a.recordSecurityEvent(c, "callback_invalid_correlation", "critical", "run_id="+runID)
		return fiber.NewError(fiber.StatusUnauthorized, "invalid correlation id")
	}

	var req contracts.RunCallbackPayload
	if err := c.BodyParser(&req); err != nil {
		a.recordSecurityEvent(c, "callback_invalid_payload", "warning", "run_id="+runID)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status == "" {
		status = "failed"
	}
	if !isAllowedCallbackStatus(status) {
		a.recordSecurityEvent(c, "callback_invalid_status", "warning", "run_id="+runID+" status="+status)
		return fiber.NewError(fiber.StatusBadRequest, "invalid status")
	}
	if len(req.Output) > maxCallbackOutputBytes {
		a.recordSecurityEvent(c, "callback_output_too_large", "warning", "run_id="+runID)
		return fiber.NewError(fiber.StatusBadRequest, "output is too large")
	}
	updates := map[string]interface{}{
		"status":      status,
		"output":      req.Output,
		"exit_code":   req.ExitCode,
		"started_at":  req.StartedAt,
		"finished_at": req.FinishedAt,
	}
	result := a.db.Model(&common.TaskRun{}).Where("id = ? AND status IN ?", runID, []string{"queued", "running"}).Updates(updates)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		a.recordSecurityEvent(c, "callback_update_ignored", "warning", "run_id="+runID)
		return c.JSON(fiber.Map{"ok": true, "ignored": true})
	}

	if status == "success" {
		var run common.TaskRun
		if err := a.db.Where("id = ?", runID).First(&run).Error; err == nil {
			if err := a.triggerNextScheduleChainRun(run); err != nil {
				a.log.Error("chain next step trigger failed", zap.Error(err), zap.String("run_id", runID), zap.String("schedule_id", run.ScheduleID))
			}
		}
	}
	return c.JSON(fiber.Map{"ok": true})
}
