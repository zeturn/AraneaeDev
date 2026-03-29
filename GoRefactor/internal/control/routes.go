package control

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"araneae-go/internal/common"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type createProjectRequest struct {
	Name string `json:"name"`
}

type createTaskRequest struct {
	Name         string `json:"name"`
	ProjectID    string `json:"project_id"`
	VersionID    string `json:"version_id"`
	EntryCommand string `json:"entry_command"`
	CronExpr     string `json:"cron_expr"`
	NodeQueue    string `json:"node_queue"`
}

type createScheduleRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	TaskID       string `json:"task_id"`
	ProjectID    string `json:"project_id"`
	VersionID    string `json:"version_id"`
	EntryCommand string `json:"entry_command"`
	CronExpr     string `json:"cron_expr"`
	NodeQueue    string `json:"node_queue"`
	Mode         string `json:"mode"`
	Enabled      *bool  `json:"enabled"`
	Order        any    `json:"order"`
}

type updateScheduleRequest struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	TaskID       *string `json:"task_id"`
	ProjectID    *string `json:"project_id"`
	VersionID    *string `json:"version_id"`
	EntryCommand *string `json:"entry_command"`
	CronExpr     *string `json:"cron_expr"`
	NodeQueue    *string `json:"node_queue"`
	Mode         *string `json:"mode"`
	Enabled      *bool   `json:"enabled"`
	Order        any     `json:"order"`
}

type legacyScheduleOrder struct {
	Name     string               `json:"name"`
	Schedule []legacyScheduleStep `json:"schedule"`
}

type legacyScheduleStep struct {
	TaskID    string   `json:"task_id"`
	Name      string   `json:"name"`
	ProjectID string   `json:"project_id"`
	Node      []string `json:"node"`
	Crons     string   `json:"crons"`
}

func (a *App) setupRoutes() {
	a.http.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	a.http.Post("/api/v1/auth/login", a.login)
	a.http.Post("/api/v1/runs/:id/callback", a.runCallback)

	api := a.http.Group("/api/v1", a.authMiddleware)
	api.Post("/projects", a.requireRoles("admin", "operator"), a.createProject)
	api.Get("/projects", a.listProjects)
	api.Get("/projects/:id", a.getProject)
	api.Get("/projects/:id/versions", a.listProjectVersions)
	api.Post("/projects/:id/upload", a.requireRoles("admin", "operator"), a.uploadArtifact)
	api.Post("/tasks", a.requireRoles("admin", "operator"), a.createTask)
	api.Get("/tasks", a.listTasks)
	api.Post("/tasks/:id/trigger", a.requireRoles("admin", "operator"), a.triggerTask)
	api.Get("/tasks/:id/runs", a.listRuns)
	api.Post("/schedules", a.requireRoles("admin", "operator"), a.createSchedule)
	api.Get("/schedules", a.listSchedules)
	api.Get("/schedules/:id", a.getSchedule)
	api.Put("/schedules/:id", a.requireRoles("admin", "operator"), a.updateSchedule)
	api.Delete("/schedules/:id", a.requireRoles("admin", "operator"), a.deleteSchedule)
	api.Post("/schedules/:id/enable", a.requireRoles("admin", "operator"), a.enableSchedule)
	api.Post("/schedules/:id/disable", a.requireRoles("admin", "operator"), a.disableSchedule)
	api.Post("/schedules/:id/trigger", a.requireRoles("admin", "operator"), a.triggerSchedule)
	api.Get("/schedules/:id/runs", a.listScheduleRuns)
	api.Get("/runs", a.listRecentRuns)
}

func (a *App) login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	var user common.User
	if err := a.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}
	if !verifyPassword(req.Password, user.PasswordHash) {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}
	token, err := a.issueToken(user.ID, user.Role)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{
		"token": token,
		"user":  fiber.Map{"id": user.ID, "username": user.Username, "role": user.Role},
	})
}

func (a *App) createProject(c *fiber.Ctx) error {
	var req createProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if strings.TrimSpace(req.Name) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "project name is required")
	}
	uid, _ := c.Locals("uid").(string)
	p := common.Project{
		ID:        uuid.NewString(),
		Name:      req.Name,
		CreatedBy: uid,
		CreatedAt: time.Now(),
	}
	if err := a.db.Create(&p).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(p)
}

func (a *App) listProjects(c *fiber.Ctx) error {
	var projects []common.Project
	if err := a.db.Order("created_at desc").Find(&projects).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(projects)
}

func (a *App) getProject(c *fiber.Ctx) error {
	projectID := c.Params("id")
	var project common.Project
	if err := a.db.Where("id = ?", projectID).First(&project).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}
	return c.JSON(project)
}

func (a *App) listProjectVersions(c *fiber.Ctx) error {
	projectID := c.Params("id")
	var versions []common.ArtifactVersion
	if err := a.db.Where("project_id = ?", projectID).Order("created_at desc").Find(&versions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(versions)
}

func (a *App) uploadArtifact(c *fiber.Ctx) error {
	projectID := c.Params("id")
	var p common.Project
	if err := a.db.Where("id = ?", projectID).First(&p).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}
	content, fileName, err := loadUploadedFile(c, "file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	versionID := uuid.NewString()
	storagePath, sha, err := writeArtifactFile(a.cfg.ArtifactRoot, p.ID, versionID, fileName, content)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	version := common.ArtifactVersion{
		ID:          versionID,
		ProjectID:   p.ID,
		FileName:    fileName,
		StoragePath: storagePath,
		SHA256:      sha,
		CreatedAt:   time.Now(),
	}
	if err := a.db.Create(&version).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(version)
}

func (a *App) createTask(c *fiber.Ctx) error {
	var req createTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if req.NodeQueue == "" {
		req.NodeQueue = "default"
	}
	if strings.TrimSpace(req.EntryCommand) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "entry_command is required")
	}
	var project common.Project
	if err := a.db.Where("id = ?", req.ProjectID).First(&project).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "project not found")
	}
	var version common.ArtifactVersion
	if err := a.db.Where("id = ? AND project_id = ?", req.VersionID, req.ProjectID).First(&version).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "version not found")
	}
	uid, _ := c.Locals("uid").(string)
	task := common.Task{
		ID:           uuid.NewString(),
		Name:         req.Name,
		ProjectID:    req.ProjectID,
		VersionID:    req.VersionID,
		EntryCommand: req.EntryCommand,
		CronExpr:     strings.TrimSpace(req.CronExpr),
		NodeQueue:    req.NodeQueue,
		Enabled:      true,
		CreatedBy:    uid,
		CreatedAt:    time.Now(),
	}
	if task.Name == "" {
		task.Name = "task-" + task.ID[:8]
	}
	if err := a.db.Create(&task).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if task.CronExpr != "" {
		if err := a.registerCronTask(task); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}
	return c.JSON(task)
}

func (a *App) triggerTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	var task common.Task
	if err := a.db.Where("id = ?", taskID).First(&task).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "task not found")
	}
	run, err := a.publishTaskRun(task, "manual", "")
	if err != nil {
		if errors.Is(err, errQueueUnavailable) {
			return fiber.NewError(fiber.StatusServiceUnavailable, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(run)
}

func (a *App) listTasks(c *fiber.Ctx) error {
	var tasks []common.Task
	if err := a.db.Order("created_at desc").Find(&tasks).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(tasks)
}

func (a *App) listRuns(c *fiber.Ctx) error {
	taskID := c.Params("id")
	var runs []common.TaskRun
	if err := a.db.Where("task_id = ?", taskID).Order("created_at desc").Find(&runs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(runs)
}

func (a *App) createSchedule(c *fiber.Ctx) error {
	var req createScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if legacy, ok := parseLegacyOrder(req.Order); ok {
		if req.Name == "" {
			req.Name = strings.TrimSpace(legacy.Name)
		}
		if len(legacy.Schedule) > 0 {
			step := legacy.Schedule[0]
			if req.Name == "" {
				req.Name = strings.TrimSpace(step.Name)
			}
			if req.TaskID == "" {
				req.TaskID = strings.TrimSpace(step.TaskID)
			}
			if req.ProjectID == "" {
				req.ProjectID = strings.TrimSpace(step.ProjectID)
			}
			if req.CronExpr == "" {
				req.CronExpr = strings.TrimSpace(step.Crons)
			}
			if req.NodeQueue == "" && len(step.Node) > 0 {
				req.NodeQueue = strings.TrimSpace(step.Node[0])
			}
		}
	}

	if req.TaskID != "" {
		var task common.Task
		if err := a.db.Where("id = ?", req.TaskID).First(&task).Error; err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "task not found")
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
			req.NodeQueue = task.NodeQueue
		}
	}

	req.ProjectID = strings.TrimSpace(req.ProjectID)
	req.VersionID = strings.TrimSpace(req.VersionID)
	req.EntryCommand = strings.TrimSpace(req.EntryCommand)
	req.CronExpr = strings.TrimSpace(req.CronExpr)
	req.NodeQueue = strings.TrimSpace(req.NodeQueue)
	req.Name = strings.TrimSpace(req.Name)
	req.Mode = strings.TrimSpace(req.Mode)

	if req.NodeQueue == "" {
		req.NodeQueue = "default"
	}
	if req.Mode == "" {
		req.Mode = "recurring"
	}
	if req.Name == "" {
		req.Name = "schedule-" + uuid.NewString()[:8]
	}
	if req.ProjectID == "" || req.VersionID == "" || req.EntryCommand == "" || req.CronExpr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "project_id, version_id, entry_command and cron_expr are required")
	}

	var project common.Project
	if err := a.db.Where("id = ?", req.ProjectID).First(&project).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "project not found")
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
		switch o := req.Order.(type) {
		case string:
			orderJSON = strings.TrimSpace(o)
		default:
			if b, err := json.Marshal(o); err == nil {
				orderJSON = string(b)
			}
		}
	}
	if orderJSON == "" {
		fallback := legacyScheduleOrder{
			Name: req.Name,
			Schedule: []legacyScheduleStep{{
				TaskID:    req.TaskID,
				Name:      req.Name,
				ProjectID: req.ProjectID,
				Node:      []string{req.NodeQueue},
				Crons:     req.CronExpr,
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
		NodeQueue:    req.NodeQueue,
		Mode:         req.Mode,
		OrderJSON:    orderJSON,
		Enabled:      enabled,
		CreatedBy:    uid,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := a.db.Create(&schedule).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if schedule.Enabled {
		if err := a.registerCronSchedule(schedule); err != nil {
			_ = a.db.Delete(&common.Schedule{}, "id = ?", schedule.ID).Error
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}
	return c.JSON(schedule)
}

func (a *App) listSchedules(c *fiber.Ctx) error {
	var schedules []common.Schedule
	if err := a.db.Order("created_at desc").Find(&schedules).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(schedules)
}

func (a *App) getSchedule(c *fiber.Ctx) error {
	id := c.Params("id")
	var schedule common.Schedule
	if err := a.db.Where("id = ?", id).First(&schedule).Error; err != nil {
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
	if req.NodeQueue != nil {
		schedule.NodeQueue = strings.TrimSpace(*req.NodeQueue)
	}
	if req.Mode != nil {
		schedule.Mode = strings.TrimSpace(*req.Mode)
	}
	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}
	if req.Order != nil {
		orderJSON, err := marshalOrderJSON(req.Order)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		schedule.OrderJSON = orderJSON
	}

	if schedule.TaskID != "" {
		var task common.Task
		if err := a.db.Where("id = ?", schedule.TaskID).First(&task).Error; err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "task not found")
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
	if schedule.Mode == "" {
		schedule.Mode = "recurring"
	}
	if schedule.Name == "" {
		schedule.Name = "schedule-" + schedule.ID[:8]
	}
	if schedule.ProjectID == "" || schedule.VersionID == "" || schedule.EntryCommand == "" || schedule.CronExpr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "project_id, version_id, entry_command and cron_expr are required")
	}

	var project common.Project
	if err := a.db.Where("id = ?", schedule.ProjectID).First(&project).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "project not found")
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

	return c.JSON(schedule)
}

func (a *App) deleteSchedule(c *fiber.Ctx) error {
	id := c.Params("id")
	var schedule common.Schedule
	if err := a.db.Where("id = ?", id).First(&schedule).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "schedule not found")
	}

	a.unregisterCronSchedule(schedule.ID)
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
	if err := a.db.Where("id = ?", id).First(&schedule).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "schedule not found")
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
	if err := a.db.Where("id = ?", id).First(&schedule).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "schedule not found")
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
	var runs []common.TaskRun
	if err := a.db.Where("schedule_id = ?", id).Order("created_at desc").Find(&runs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(runs)
}

func (a *App) listRecentRuns(c *fiber.Ctx) error {
	var runs []common.TaskRun
	if err := a.db.Order("created_at desc").Limit(200).Find(&runs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{
		"records": runs,
		"count":   len(runs),
	})
}

func (a *App) runCallback(c *fiber.Ctx) error {
	if c.Get("X-Execution-Key") != a.cfg.ExecutionAPIKey {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid execution key")
	}
	runID := c.Params("id")
	var req runCallbackPayload
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "failed"
	}
	updates := map[string]interface{}{
		"status":      status,
		"output":      req.Output,
		"exit_code":   req.ExitCode,
		"started_at":  req.StartedAt,
		"finished_at": req.FinishedAt,
	}
	result := a.db.Model(&common.TaskRun{}).Where("id = ?", runID).Updates(updates)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "run not found")
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
