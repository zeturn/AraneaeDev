package control

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"araneae-go/internal/common"
	"araneae-go/internal/control/security/password"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/google/uuid"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type createProjectRequest struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Language    string     `json:"language"`
	Command     string     `json:"command"`
	WorkplaceID *laxString `json:"workplace_id"`
	Workplace   *laxString `json:"workplace"`
}

type updateProjectRequest struct {
	Name        *string    `json:"name"`
	Description *string    `json:"description"`
	Language    *string    `json:"language"`
	Command     *string    `json:"command"`
	WorkplaceID *laxString `json:"workplace_id"`
	Workplace   *laxString `json:"workplace"`
}

type updateArtifactVersionRequest struct {
	FileName *string `json:"file_name"`
}

type createTaskRequest struct {
	Name         string `json:"name"`
	ProjectID    string `json:"project_id"`
	VersionID    string `json:"version_id"`
	EntryCommand string `json:"entry_command"`
	CronExpr     string `json:"cron_expr"`
	NodeQueue    string `json:"node_queue"`
}

type updateTaskRequest struct {
	Name         *string `json:"name"`
	ProjectID    *string `json:"project_id"`
	VersionID    *string `json:"version_id"`
	EntryCommand *string `json:"entry_command"`
	CronExpr     *string `json:"cron_expr"`
	NodeQueue    *string `json:"node_queue"`
	Enabled      *bool   `json:"enabled"`
}

type laxString string

func (s *laxString) UnmarshalJSON(data []byte) error {
	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		*s = laxString(strings.TrimSpace(asString))
		return nil
	}

	var asNumber float64
	if err := json.Unmarshal(data, &asNumber); err == nil {
		*s = laxString(strconv.FormatFloat(asNumber, 'f', -1, 64))
		return nil
	}

	var asNull any
	if err := json.Unmarshal(data, &asNull); err == nil && asNull == nil {
		*s = ""
		return nil
	}

	return errors.New("value must be a string or number")
}

func sanitizeNodeQueue(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return value
		}
	}
	return ""
}

func parseOptionalWorkplaceID(workplaceID *laxString, workplace *laxString) (*uint, error) {
	value := ""
	if workplaceID != nil {
		value = strings.TrimSpace(string(*workplaceID))
	}
	if value == "" && workplace != nil {
		value = strings.TrimSpace(string(*workplace))
	}
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return nil, errors.New("workplace_id must be a positive integer")
	}
	result := uint(parsed)
	return &result, nil
}

func parseOptionalWorkplaceQueryID(c *fiber.Ctx) (*uint, error) {
	raw := strings.TrimSpace(c.Query("workplace_id"))
	if raw == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || parsed == 0 {
		return nil, errors.New("invalid workplace_id")
	}
	result := uint(parsed)
	return &result, nil
}

type createScheduleRequest struct {
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	TaskID       string    `json:"task_id"`
	ProjectID    string    `json:"project_id"`
	VersionID    string    `json:"version_id"`
	EntryCommand string    `json:"entry_command"`
	CronExpr     string    `json:"cron_expr"`
	NodeQueue    laxString `json:"node_queue"`
	Enabled      *bool     `json:"enabled"`
	Order        any       `json:"order"`
}

type updateScheduleRequest struct {
	Name         *string    `json:"name"`
	Description  *string    `json:"description"`
	TaskID       *string    `json:"task_id"`
	ProjectID    *string    `json:"project_id"`
	VersionID    *string    `json:"version_id"`
	EntryCommand *string    `json:"entry_command"`
	CronExpr     *string    `json:"cron_expr"`
	NodeQueue    *laxString `json:"node_queue"`
	Enabled      *bool      `json:"enabled"`
	Order        any        `json:"order"`
}

const (
	maxCallbackOutputBytes    = 1024 * 1024
	maxManualTriggerPerMinute = 20
	maxCallbackPerMinute      = 120
	triggerDuplicateWindow    = 20 * time.Second
)

func isAllowedCallbackStatus(status string) bool {
	switch status {
	case "running", "success", "failed", "canceled", "cancelled":
		return true
	default:
		return false
	}
}

func isTerminalRunStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success", "failed", "canceled", "cancelled":
		return true
	default:
		return false
	}
}

func (a *App) hasRecentManualRun(taskID, scheduleID string) (bool, error) {
	query := a.db.Model(&common.TaskRun{}).
		Where("trigger_source = ?", "manual").
		Where("status IN ?", []string{"queued", "running"}).
		Where("created_at >= ?", time.Now().Add(-triggerDuplicateWindow))

	if scheduleID != "" {
		query = query.Where("schedule_id = ?", scheduleID)
	} else {
		query = query.Where("task_id = ? AND (schedule_id = '' OR schedule_id IS NULL)", taskID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

type legacyScheduleOrder struct {
	Name     string               `json:"name"`
	Schedule []legacyScheduleStep `json:"schedule"`
}

type legacyScheduleStep struct {
	TaskID     string   `json:"task_id"`
	TaskStatus string   `json:"task_status"`
	Name       string   `json:"name"`
	ProjectID  string   `json:"project_id"`
	Node       []string `json:"node"`
	Trigger    string   `json:"trigger"`
	Crons      string   `json:"crons"`
	Previous   string   `json:"previous"`
}

func (a *App) setupRoutes() {
	a.http.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	loginRateLimit := limiter.New(limiter.Config{
		Max:        10,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			a.recordSecurityEvent(c, "auth_login_rate_limited", "warning", "too many login attempts")
			return fiber.NewError(fiber.StatusTooManyRequests, "too many login attempts")
		},
	})

	manualTriggerRateLimit := limiter.New(limiter.Config{
		Max:        maxManualTriggerPerMinute,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			uid, _ := c.Locals("uid").(string)
			if uid == "" {
				uid = c.IP()
			}
			return uid + ":" + c.Path()
		},
		LimitReached: func(c *fiber.Ctx) error {
			a.recordSecurityEvent(c, "manual_trigger_rate_limited", "warning", "too many trigger attempts")
			return fiber.NewError(fiber.StatusTooManyRequests, "too many trigger attempts")
		},
	})

	callbackRateLimit := limiter.New(limiter.Config{
		Max:        maxCallbackPerMinute,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			a.recordSecurityEvent(c, "callback_rate_limited", "warning", "too many callback attempts")
			return fiber.NewError(fiber.StatusTooManyRequests, "too many callback attempts")
		},
	})

	a.http.Post("/api/v1/auth/login", loginRateLimit, a.login)
	a.http.Get("/api/auth/basaltpass/login", a.basaltPassLogin)
	a.http.Get("/api/auth/basaltpass/login/", a.basaltPassLogin)
	a.http.Get("/api/auth/basaltpass/callback", a.basaltPassCallback)
	a.http.Get("/api/auth/basaltpass/callback/", a.basaltPassCallback)
	a.http.Post("/api/v1/auth/basaltpass/exchange", a.basaltPassExchange)
	a.http.Post("/api/v1/runs/:id/callback", callbackRateLimit, a.runCallback)

	api := a.http.Group("/api/v1", a.authMiddleware)
	api.Post("/projects", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.createProject)
	api.Get("/projects", a.requireAppScope("araneae.read"), a.listProjects)
	api.Get("/projects/:id", a.requireAppScope("araneae.read"), a.getProject)
	api.Put("/projects/:id", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.updateProject)
	api.Delete("/projects/:id", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.deleteProject)
	api.Get("/projects/:id/versions", a.requireAppScope("araneae.read"), a.listProjectVersions)
	api.Get("/projects/:projectID/versions/:versionID", a.requireAppScope("araneae.read"), a.getProjectVersion)
	api.Put("/projects/:projectID/versions/:versionID", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.updateProjectVersion)
	api.Delete("/projects/:projectID/versions/:versionID", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.deleteProjectVersion)
	api.Post("/projects/:id/upload", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.uploadArtifact)
	api.Post("/tasks", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.createTask)
	api.Get("/tasks", a.requireAppScope("araneae.read"), a.listTasks)
	api.Get("/tasks/:id", a.requireAppScope("araneae.read"), a.getTask)
	api.Put("/tasks/:id", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.updateTask)
	api.Delete("/tasks/:id", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.deleteTask)
	api.Post("/tasks/:id/trigger", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), manualTriggerRateLimit, a.triggerTask)
	api.Get("/tasks/:id/runs", a.requireAppScope("araneae.read"), a.listRuns)
	api.Post("/schedules", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.createSchedule)
	api.Get("/schedules", a.requireAppScope("araneae.read"), a.listSchedules)
	api.Get("/schedules/:id", a.requireAppScope("araneae.read"), a.getSchedule)
	api.Put("/schedules/:id", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.updateSchedule)
	api.Delete("/schedules/:id", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.deleteSchedule)
	api.Post("/schedules/:id/enable", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.enableSchedule)
	api.Post("/schedules/:id/disable", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.disableSchedule)
	api.Post("/schedules/:id/trigger", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), manualTriggerRateLimit, a.triggerSchedule)
	api.Get("/schedules/:id/runs", a.requireAppScope("araneae.read"), a.listScheduleRuns)
	api.Get("/runs", a.requireAppScope("araneae.read"), a.listRecentRuns)

	api.Get("/users/", a.requireAppScope("araneae.read"), a.listUsers)
	api.Get("/users", a.requireAppScope("araneae.read"), a.listUsers)
	api.Get("/users/:id/", a.requireAppScope("araneae.read"), a.getUser)
	api.Get("/users/:id", a.requireAppScope("araneae.read"), a.getUser)

	api.Get("/nodes/discover/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.read"), a.discoverNodes)
	api.Post("/nodes/register/", a.requireRoles("admin"), a.requireAppScope("araneae.write"), a.registerNode)
	api.Get("/nodes/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.read"), a.listNodes)
	api.Get("/nodes", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.read"), a.listNodes)
	api.Get("/nodes/:id/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.read"), a.getNode)
	api.Get("/nodes/:id", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.read"), a.getNode)
	api.Put("/nodes/:id/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.updateNode)
	api.Put("/nodes/:id", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.updateNode)
	api.Delete("/nodes/:id/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.deleteNode)
	api.Delete("/nodes/:id", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.deleteNode)
	api.Get("/nodes/:id/status/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.read"), a.getNodeStatus)
	api.Get("/nodes/:id/status", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.read"), a.getNodeStatus)
	api.Get("/nodes/:id/capabilities/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.read"), a.getNodeCapabilities)
	api.Get("/nodes/:id/capabilities", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.read"), a.getNodeCapabilities)
	api.Post("/nodes/:id/refresh_capabilities/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.refreshNodeCapabilities)
	api.Post("/nodes/:id/refresh_capabilities", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.refreshNodeCapabilities)
	api.Get("/nodes/:id/installers/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.read"), a.getNodeInstallers)
	api.Get("/nodes/:id/installers", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.read"), a.getNodeInstallers)
	api.Post("/nodes/:id/install_runtime/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.installRuntime)
	api.Post("/nodes/:id/install_runtime", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.installRuntime)
	api.Get("/nodes/:id/install_status/:jobID/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.read"), a.getInstallStatus)
	api.Get("/nodes/:id/install_status/:jobID", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.read"), a.getInstallStatus)

	api.Get("/teams/my_teams/", a.requireAppScope("araneae.read"), a.listMyTeams)
	api.Get("/teams/my_teams", a.requireAppScope("araneae.read"), a.listMyTeams)
	api.Post("/teams/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.createTeam)
	api.Post("/teams", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.createTeam)
	api.Get("/teams/:id/", a.requireAppScope("araneae.read"), a.getTeam)
	api.Get("/teams/:id", a.requireAppScope("araneae.read"), a.getTeam)
	api.Put("/teams/:id/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.updateTeam)
	api.Put("/teams/:id", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.updateTeam)
	api.Delete("/teams/:id/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.deleteTeam)
	api.Delete("/teams/:id", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.deleteTeam)
	api.Get("/teams/:id/members/", a.requireAppScope("araneae.read"), a.getTeamMembers)
	api.Get("/teams/:id/members", a.requireAppScope("araneae.read"), a.getTeamMembers)
	api.Post("/teams/:id/add_members/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.addTeamMembers)
	api.Post("/teams/:id/add_members", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.addTeamMembers)
	api.Delete("/teams/:id/members/:userID/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.removeTeamMember)
	api.Delete("/teams/:id/members/:userID", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.removeTeamMember)

	api.Get("/workplaces/", a.requireAppScope("araneae.read"), a.listWorkplaces)
	api.Get("/workplaces", a.requireAppScope("araneae.read"), a.listWorkplaces)
	api.Get("/workplaces/my_workplaces/", a.requireAppScope("araneae.read"), a.listMyWorkplaces)
	api.Get("/workplaces/my_workplaces", a.requireAppScope("araneae.read"), a.listMyWorkplaces)
	api.Post("/workplaces/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.createWorkplace)
	api.Post("/workplaces", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.createWorkplace)
	api.Get("/workplaces/:id/", a.requireAppScope("araneae.read"), a.getWorkplace)
	api.Get("/workplaces/:id", a.requireAppScope("araneae.read"), a.getWorkplace)
	api.Put("/workplaces/:id/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.updateWorkplace)
	api.Put("/workplaces/:id", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.updateWorkplace)
	api.Delete("/workplaces/:id/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.deleteWorkplace)
	api.Delete("/workplaces/:id", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.deleteWorkplace)
	api.Post("/workplaces/:id/add_teams/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.addWorkplaceTeams)
	api.Post("/workplaces/:id/add_teams", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.addWorkplaceTeams)
	api.Post("/workplaces/:id/add_people/", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.addWorkplacePeople)
	api.Post("/workplaces/:id/add_people", a.requireRoles("admin", "operator"), a.requireAppScope("araneae.write"), a.addWorkplacePeople)
}

func (a *App) login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	username := strings.TrimSpace(req.Username)
	var user common.User
	if err := a.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		a.recordSecurityEvent(c, "auth_login_failed", "warning", "invalid credentials for username="+username)
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}
	if !password.Verify(req.Password, user.PasswordHash) {
		a.recordSecurityEvent(c, "auth_login_failed", "warning", "invalid credentials for username="+username)
		return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
	}
	token, err := a.issueToken(user.ID, user.Role)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	a.recordSecurityEventWithUser(c, user.ID, "auth_login_success", "info", "username="+username)
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
	workplaceID, err := parseOptionalWorkplaceID(req.WorkplaceID, req.Workplace)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if workplaceID != nil {
		var workplace common.Workplace
		if err := a.db.Select("id").Where("id = ?", *workplaceID).First(&workplace).Error; err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "workplace not found")
		}
		allowed, accessErr := a.canBindWorkplace(c, *workplaceID)
		if accessErr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
		}
		if !allowed {
			return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
		}
	}
	uid, _ := c.Locals("uid").(string)
	now := time.Now()
	p := common.Project{
		ID:          uuid.NewString(),
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Language:    strings.TrimSpace(req.Language),
		Command:     strings.TrimSpace(req.Command),
		WorkplaceID: workplaceID,
		CreatedBy:   uid,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := a.db.Create(&p).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(p)
}

func (a *App) listProjects(c *fiber.Ctx) error {
	var projects []common.Project
	role, _ := c.Locals("role").(string)
	uid, _ := c.Locals("uid").(string)
	workplaceID, err := parseOptionalWorkplaceQueryID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	query := a.db.Order("created_at desc")
	if workplaceID != nil {
		query = query.Where("workplace_id = ?", *workplaceID)
	}
	if !isPrivilegedRole(role) {
		if workplaceID != nil {
			allowed, accessErr := a.userCanAccessWorkplace(uid, *workplaceID)
			if accessErr != nil {
				return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
			}
			if !allowed {
				return c.JSON([]common.Project{})
			}
		}

		accessibleWorkplaceIDs, scopeErr := a.userAccessibleWorkplaceIDs(uid)
		if scopeErr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, scopeErr.Error())
		}
		if len(accessibleWorkplaceIDs) > 0 {
			query = query.Where("created_by = ? OR workplace_id IN ?", uid, accessibleWorkplaceIDs)
		} else {
			query = query.Where("created_by = ?", uid)
		}
	}
	if err := query.Find(&projects).Error; err != nil {
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
	allowed, accessErr := a.canAccessProject(c, project)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !allowed {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}
	return c.JSON(project)
}

func (a *App) updateProject(c *fiber.Ctx) error {
	projectID := strings.TrimSpace(c.Params("id"))
	var project common.Project
	if err := a.db.Where("id = ?", projectID).First(&project).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	var req updateProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if req.Name != nil {
		newName := strings.TrimSpace(*req.Name)
		if newName == "" {
			return fiber.NewError(fiber.StatusBadRequest, "project name is required")
		}
		project.Name = newName
	}

	if req.Description != nil {
		project.Description = strings.TrimSpace(*req.Description)
	}

	if req.Language != nil {
		project.Language = strings.TrimSpace(*req.Language)
	}

	if req.Command != nil {
		project.Command = strings.TrimSpace(*req.Command)
	}

	if req.WorkplaceID != nil || req.Workplace != nil {
		workplaceID, err := parseOptionalWorkplaceID(req.WorkplaceID, req.Workplace)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if workplaceID != nil {
			var workplace common.Workplace
			if err := a.db.Select("id").Where("id = ?", *workplaceID).First(&workplace).Error; err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "workplace not found")
			}
			allowed, accessErr := a.canBindWorkplace(c, *workplaceID)
			if accessErr != nil {
				return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
			}
			if !allowed {
				return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
			}
		}
		project.WorkplaceID = workplaceID
	}

	project.UpdatedAt = time.Now()

	if err := a.db.Save(&project).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(project)
}

func (a *App) deleteProject(c *fiber.Ctx) error {
	projectID := strings.TrimSpace(c.Params("id"))
	var project common.Project
	if err := a.db.Where("id = ?", projectID).First(&project).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}
	canWrite, accessErr := a.canWriteProject(c, project)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canWrite {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}

	var taskIDs []string
	if err := a.db.Model(&common.Task{}).Where("project_id = ?", projectID).Pluck("id", &taskIDs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	var scheduleIDs []string
	if err := a.db.Model(&common.Schedule{}).Where("project_id = ?", projectID).Pluck("id", &scheduleIDs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	for _, taskID := range taskIDs {
		a.unregisterCronTask(taskID)
	}
	for _, scheduleID := range scheduleIDs {
		a.unregisterCronSchedule(scheduleID)
	}

	tx := a.db.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}

	if len(taskIDs) > 0 {
		if err := tx.Where("task_id IN ?", taskIDs).Delete(&common.TaskRun{}).Error; err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}
	if len(scheduleIDs) > 0 {
		if err := tx.Where("schedule_id IN ?", scheduleIDs).Delete(&common.TaskRun{}).Error; err != nil {
			tx.Rollback()
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}
	if err := tx.Where("project_id = ?", projectID).Delete(&common.Schedule{}).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := tx.Where("project_id = ?", projectID).Delete(&common.Task{}).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := tx.Where("project_id = ?", projectID).Delete(&common.ArtifactVersion{}).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := tx.Where("id = ?", projectID).Delete(&common.Project{}).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"ok": true})
}

func (a *App) listProjectVersions(c *fiber.Ctx) error {
	projectID := c.Params("id")
	var project common.Project
	if err := a.db.Where("id = ?", projectID).First(&project).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}
	allowed, accessErr := a.canAccessProject(c, project)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !allowed {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}
	var versions []common.ArtifactVersion
	if err := a.db.Where("project_id = ?", projectID).Order("created_at desc").Find(&versions).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(versions)
}

func (a *App) getProjectVersion(c *fiber.Ctx) error {
	projectID := strings.TrimSpace(c.Params("projectID"))
	versionID := strings.TrimSpace(c.Params("versionID"))

	var project common.Project
	if err := a.db.Where("id = ?", projectID).First(&project).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}
	allowed, accessErr := a.canAccessProject(c, project)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !allowed {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}

	var version common.ArtifactVersion
	if err := a.db.Where("id = ? AND project_id = ?", versionID, projectID).First(&version).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "version not found")
	}
	return c.JSON(version)
}

func (a *App) updateProjectVersion(c *fiber.Ctx) error {
	projectID := strings.TrimSpace(c.Params("projectID"))
	versionID := strings.TrimSpace(c.Params("versionID"))
	var project common.Project
	if err := a.db.Where("id = ?", projectID).First(&project).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}
	canWrite, accessErr := a.canWriteProject(c, project)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canWrite {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}

	var version common.ArtifactVersion
	if err := a.db.Where("id = ? AND project_id = ?", versionID, projectID).First(&version).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "version not found")
	}

	var req updateArtifactVersionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if req.FileName != nil {
		fileName := strings.TrimSpace(*req.FileName)
		if fileName == "" {
			return fiber.NewError(fiber.StatusBadRequest, "file_name is required")
		}
		version.FileName = fileName
	}

	if err := a.db.Save(&version).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(version)
}

func (a *App) deleteProjectVersion(c *fiber.Ctx) error {
	projectID := strings.TrimSpace(c.Params("projectID"))
	versionID := strings.TrimSpace(c.Params("versionID"))
	var project common.Project
	if err := a.db.Where("id = ?", projectID).First(&project).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}
	canWrite, accessErr := a.canWriteProject(c, project)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canWrite {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}

	var version common.ArtifactVersion
	if err := a.db.Where("id = ? AND project_id = ?", versionID, projectID).First(&version).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "version not found")
	}

	var refs int64
	if err := a.db.Model(&common.Task{}).Where("version_id = ?", versionID).Count(&refs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if refs > 0 {
		return fiber.NewError(fiber.StatusBadRequest, "version is used by tasks and cannot be deleted")
	}

	if err := a.db.Where("id = ? AND project_id = ?", versionID, projectID).Delete(&common.ArtifactVersion{}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"ok": true})
}

func (a *App) uploadArtifact(c *fiber.Ctx) error {
	projectID := c.Params("id")
	var p common.Project
	if err := a.db.Where("id = ?", projectID).First(&p).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}
	canWrite, accessErr := a.canWriteProject(c, p)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canWrite {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
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
	uid, _ := c.Locals("uid").(string)
	task := common.Task{
		ID:           uuid.NewString(),
		Name:         req.Name,
		ProjectID:    req.ProjectID,
		VersionID:    req.VersionID,
		EntryCommand: req.EntryCommand,
		CronExpr:     "",
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
	return c.JSON(task)
}

func (a *App) triggerTask(c *fiber.Ctx) error {
	taskID := c.Params("id")
	var task common.Task
	if err := a.db.Where("id = ?", taskID).First(&task).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "task not found")
	}
	canWrite, accessErr := a.canWriteTask(c, task)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canWrite {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	hasRecent, recentErr := a.hasRecentManualRun(task.ID, "")
	if recentErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, recentErr.Error())
	}
	if hasRecent {
		a.recordSecurityEvent(c, "task_trigger_duplicate_blocked", "warning", "task_id="+task.ID)
		return fiber.NewError(fiber.StatusTooManyRequests, "task trigger is already in progress")
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

func (a *App) getTask(c *fiber.Ctx) error {
	taskID := strings.TrimSpace(c.Params("id"))
	var task common.Task
	if err := a.db.Where("id = ?", taskID).First(&task).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "task not found")
	}
	allowed, accessErr := a.canAccessTask(c, task)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !allowed {
		return fiber.NewError(fiber.StatusNotFound, "task not found")
	}
	return c.JSON(task)
}

func (a *App) updateTask(c *fiber.Ctx) error {
	taskID := strings.TrimSpace(c.Params("id"))
	var task common.Task
	if err := a.db.Where("id = ?", taskID).First(&task).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "task not found")
	}

	var req updateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return fiber.NewError(fiber.StatusBadRequest, "task name is required")
		}
		task.Name = name
	}
	if req.ProjectID != nil {
		task.ProjectID = strings.TrimSpace(*req.ProjectID)
	}
	if req.VersionID != nil {
		task.VersionID = strings.TrimSpace(*req.VersionID)
	}
	if req.EntryCommand != nil {
		task.EntryCommand = strings.TrimSpace(*req.EntryCommand)
	}
	if req.CronExpr != nil {
		task.CronExpr = ""
	}
	if req.NodeQueue != nil {
		task.NodeQueue = strings.TrimSpace(*req.NodeQueue)
	}
	if req.Enabled != nil {
		task.Enabled = *req.Enabled
	}

	if task.ProjectID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "project_id is required")
	}
	if task.VersionID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "version_id is required")
	}
	if task.EntryCommand == "" {
		return fiber.NewError(fiber.StatusBadRequest, "entry_command is required")
	}
	if task.NodeQueue == "" {
		task.NodeQueue = "default"
	}
	task.CronExpr = ""

	var project common.Project
	if err := a.db.Where("id = ?", task.ProjectID).First(&project).Error; err != nil {
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
	if err := a.db.Where("id = ? AND project_id = ?", task.VersionID, task.ProjectID).First(&version).Error; err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "version not found")
	}

	if err := a.db.Save(&task).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(task)
}

func (a *App) deleteTask(c *fiber.Ctx) error {
	taskID := strings.TrimSpace(c.Params("id"))
	var task common.Task
	if err := a.db.Where("id = ?", taskID).First(&task).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "task not found")
	}
	canWrite, accessErr := a.canWriteTask(c, task)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canWrite {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}

	a.unregisterCronTask(taskID)

	tx := a.db.Begin()
	if tx.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, tx.Error.Error())
	}
	if err := tx.Where("task_id = ?", taskID).Delete(&common.TaskRun{}).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := tx.Where("id = ?", taskID).Delete(&common.Task{}).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"ok": true})
}

func (a *App) listTasks(c *fiber.Ctx) error {
	var tasks []common.Task
	role, _ := c.Locals("role").(string)
	uid, _ := c.Locals("uid").(string)
	workplaceID, err := parseOptionalWorkplaceQueryID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
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
				return c.JSON([]common.Task{})
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
	if err := query.Find(&tasks).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(tasks)
}

func (a *App) listRuns(c *fiber.Ctx) error {
	taskID := c.Params("id")
	var task common.Task
	if err := a.db.Where("id = ?", taskID).First(&task).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "task not found")
	}
	allowed, accessErr := a.canAccessTask(c, task)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !allowed {
		return fiber.NewError(fiber.StatusNotFound, "task not found")
	}
	var runs []common.TaskRun
	if err := a.db.Where("task_id = ?", taskID).Order("created_at desc").Find(&runs).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(runs)
}
