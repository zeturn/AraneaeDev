package control

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"araneae-go/internal/common"

	"github.com/gofiber/fiber/v2"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

func newTestControlApp(t *testing.T) *App {
	t.Helper()

	tmp := t.TempDir()
	cfg := common.ControlConfig{
		JWTSecret:       "test-jwt-secret",
		ExecutionAPIKey: "test-callback-key",
		ArtifactRoot:    filepath.Join(tmp, "artifacts"),
		DBPath:          filepath.Join(tmp, "control.db"),
	}

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sql db handle: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	if err := db.AutoMigrate(common.AutoMigrateModels()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	hash, err := hashPassword("admin123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	admin := common.User{
		ID:           uuid.NewString(),
		Username:     "admin",
		PasswordHash: hash,
		Role:         "admin",
		CreatedAt:    time.Now(),
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	a := &App{
		cfg:             cfg,
		db:              db,
		http:            fiber.New(),
		cron:            cron.New(cron.WithSeconds()),
		cronEntries:     map[string]cron.EntryID{},
		scheduleEntries: map[string]cron.EntryID{},
	}
	a.setupRoutes()
	return a
}

func buildTestZip(t *testing.T, entries map[string]string) []byte {
	t.Helper()

	buf := bytes.NewBuffer(nil)
	zw := zip.NewWriter(buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}

func doJSONRequest(t *testing.T, app *App, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := app.http.Test(req, -1)
	if err != nil {
		t.Fatalf("http test request failed: %v", err)
	}
	rec := httptest.NewRecorder()
	rec.Code = resp.StatusCode
	if _, err := rec.Body.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read response body: %v", err)
	}
	_ = resp.Body.Close()
	return rec
}

func loginAndGetToken(t *testing.T, app *App) string {
	t.Helper()

	rec := doJSONRequest(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username": "admin",
		"password": "admin123",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	tok, _ := out["token"].(string)
	if tok == "" {
		t.Fatalf("missing token in login response: %s", rec.Body.String())
	}
	return tok
}

func createProjectVersionTask(t *testing.T, app *App, token, cronExpr string) (common.Project, common.ArtifactVersion, common.Task) {
	t.Helper()

	projectRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/projects", token, map[string]string{
		"name": "demo-project",
	})
	if projectRec.Code != http.StatusOK {
		t.Fatalf("create project failed: status=%d body=%s", projectRec.Code, projectRec.Body.String())
	}
	var project common.Project
	if err := json.Unmarshal(projectRec.Body.Bytes(), &project); err != nil {
		t.Fatalf("decode project: %v", err)
	}

	zipContent := buildTestZip(t, map[string]string{"run.sh": "echo hello"})
	body := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "demo.zip")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(zipContent); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	uploadPath := fmt.Sprintf("/api/v1/projects/%s/upload", project.ID)
	uploadReq := httptest.NewRequest(http.MethodPost, uploadPath, body)
	uploadReq.Header.Set("Authorization", "Bearer "+token)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadResp, err := app.http.Test(uploadReq, -1)
	if err != nil {
		t.Fatalf("upload request failed: %v", err)
	}
	if uploadResp.StatusCode != http.StatusOK {
		b := bytes.NewBuffer(nil)
		_, _ = b.ReadFrom(uploadResp.Body)
		t.Fatalf("upload failed: status=%d body=%s", uploadResp.StatusCode, b.String())
	}
	uploadBody := bytes.NewBuffer(nil)
	_, _ = uploadBody.ReadFrom(uploadResp.Body)
	_ = uploadResp.Body.Close()

	var version common.ArtifactVersion
	if err := json.Unmarshal(uploadBody.Bytes(), &version); err != nil {
		t.Fatalf("decode version: %v", err)
	}

	taskRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/tasks", token, map[string]string{
		"name":          "demo-task",
		"project_id":    project.ID,
		"version_id":    version.ID,
		"entry_command": "bash run.sh",
		"cron_expr":     cronExpr,
		"node_queue":    "default",
	})
	if taskRec.Code != http.StatusOK {
		t.Fatalf("create task failed: status=%d body=%s", taskRec.Code, taskRec.Body.String())
	}
	var task common.Task
	if err := json.Unmarshal(taskRec.Body.Bytes(), &task); err != nil {
		t.Fatalf("decode task: %v", err)
	}

	return project, version, task
}

func TestUserCreate_AutoCreatesPersonalTeam(t *testing.T) {
	app := newTestControlApp(t)

	hash, err := hashPassword("alice123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := common.User{
		ID:           uuid.NewString(),
		Username:     "alice",
		PasswordHash: hash,
		Role:         "viewer",
		CreatedAt:    time.Now(),
	}
	if err := app.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	var personalTeam common.Team
	if err := app.db.Where("created_by = ? AND is_personal = ?", user.ID, true).First(&personalTeam).Error; err != nil {
		t.Fatalf("personal team not created: %v", err)
	}
	if personalTeam.JoinAble {
		t.Fatalf("personal team should not be joinable")
	}

	var owner common.TeamMember
	if err := app.db.Where("team_id = ? AND user_id = ?", personalTeam.ID, user.ID).First(&owner).Error; err != nil {
		t.Fatalf("personal team owner membership not created: %v", err)
	}
	if owner.Role != "owner" {
		t.Fatalf("expected owner role, got %q", owner.Role)
	}
}

func TestDeleteTeam_RejectsPersonalTeam(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)

	hash, err := hashPassword("bob123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := common.User{
		ID:           uuid.NewString(),
		Username:     "bob",
		PasswordHash: hash,
		Role:         "viewer",
		CreatedAt:    time.Now(),
	}
	if err := app.db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	var personalTeam common.Team
	if err := app.db.Where("created_by = ? AND is_personal = ?", user.ID, true).First(&personalTeam).Error; err != nil {
		t.Fatalf("load personal team: %v", err)
	}

	deletePath := fmt.Sprintf("/api/v1/teams/%d", personalTeam.ID)
	rec := doJSONRequest(t, app, http.MethodDelete, deletePath, token, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when deleting personal team, got %d body=%s", rec.Code, rec.Body.String())
	}

	var count int64
	if err := app.db.Model(&common.Team{}).Where("id = ?", personalTeam.ID).Count(&count).Error; err != nil {
		t.Fatalf("count team: %v", err)
	}
	if count != 1 {
		t.Fatalf("personal team should still exist after delete attempt")
	}
}

func TestDeleteTeam_AllowsRegularTeam(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)

	createRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/teams", token, map[string]any{
		"name":        "regular-team",
		"description": "test",
		"join_able":   false,
	})
	if createRec.Code != http.StatusOK {
		t.Fatalf("create team failed: status=%d body=%s", createRec.Code, createRec.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode team response: %v", err)
	}
	teamIDFloat, ok := created["id"].(float64)
	if !ok {
		t.Fatalf("missing team id in response: %s", createRec.Body.String())
	}
	deletePath := fmt.Sprintf("/api/v1/teams/%d", int(teamIDFloat))
	deleteRec := doJSONRequest(t, app, http.MethodDelete, deletePath, token, nil)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete regular team failed: status=%d body=%s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestControlRoutes_RequireAuth(t *testing.T) {
	app := newTestControlApp(t)
	rec := doJSONRequest(t, app, http.MethodGet, "/api/v1/projects", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestControlRoutes_CallbackBypassesJWTGroup(t *testing.T) {
	app := newTestControlApp(t)

	rec := doJSONRequest(t, app, http.MethodPost, "/api/v1/runs/fake-run-id/callback", "", map[string]any{
		"status":    "success",
		"output":    "ok",
		"exit_code": 0,
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without callback key, got %d", rec.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/fake-run-id/callback", bytes.NewReader([]byte(`{"status":"success"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Execution-Key", app.cfg.ExecutionAPIKey)
	resp, err := app.http.Test(req, -1)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		b := bytes.NewBuffer(nil)
		_, _ = b.ReadFrom(resp.Body)
		t.Fatalf("expected 404 with callback key on missing run, got %d body=%s", resp.StatusCode, b.String())
	}
	_ = resp.Body.Close()
}

func TestControlRoutes_CallbackUpdatesRun(t *testing.T) {
	app := newTestControlApp(t)

	run := common.TaskRun{
		ID:            uuid.NewString(),
		TaskID:        uuid.NewString(),
		TriggerSource: "manual",
		Status:        "queued",
		CreatedAt:     time.Now(),
		CorrelationID: uuid.NewString(),
	}
	if err := app.db.Create(&run).Error; err != nil {
		t.Fatalf("seed task run failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.ID+"/callback", bytes.NewReader([]byte(`{"status":"success","output":"done","exit_code":0}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Execution-Key", app.cfg.ExecutionAPIKey)
	resp, err := app.http.Test(req, -1)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		b := bytes.NewBuffer(nil)
		_, _ = b.ReadFrom(resp.Body)
		t.Fatalf("expected 200 callback update, got %d body=%s", resp.StatusCode, b.String())
	}
	_ = resp.Body.Close()

	var updated common.TaskRun
	if err := app.db.Where("id = ?", run.ID).First(&updated).Error; err != nil {
		t.Fatalf("load updated run failed: %v", err)
	}
	if updated.Status != "success" || updated.ExitCode != 0 || updated.Output != "done" {
		t.Fatalf("unexpected updated run: status=%s code=%d output=%q", updated.Status, updated.ExitCode, updated.Output)
	}
}

func TestControlRoutes_ProjectUploadAndTaskFlow(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)
	project, _, task := createProjectVersionTask(t, app, token, "")
	if task.ID == "" {
		t.Fatal("task id is empty")
	}

	versionsRec := doJSONRequest(t, app, http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/versions", project.ID), token, nil)
	if versionsRec.Code != http.StatusOK {
		t.Fatalf("list versions failed: status=%d body=%s", versionsRec.Code, versionsRec.Body.String())
	}
	var versions []common.ArtifactVersion
	if err := json.Unmarshal(versionsRec.Body.Bytes(), &versions); err != nil {
		t.Fatalf("decode versions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(versions))
	}

	tasksRec := doJSONRequest(t, app, http.MethodGet, "/api/v1/tasks", token, nil)
	if tasksRec.Code != http.StatusOK {
		t.Fatalf("list tasks failed: status=%d body=%s", tasksRec.Code, tasksRec.Body.String())
	}
	var tasks []common.Task
	if err := json.Unmarshal(tasksRec.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("decode tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
}

func TestControlRoutes_TriggerReturns503WhenQueueUnavailable(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)
	_, _, task := createProjectVersionTask(t, app, token, "")

	rec := doJSONRequest(t, app, http.MethodPost, "/api/v1/tasks/"+task.ID+"/trigger", token, nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when queue unavailable, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestControlRoutes_TaskCronExprIgnored(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)
	project, version, _ := createProjectVersionTask(t, app, token, "")

	rec := doJSONRequest(t, app, http.MethodPost, "/api/v1/tasks", token, map[string]any{
		"name":          "cron-ignored-task",
		"project_id":    project.ID,
		"version_id":    version.ID,
		"entry_command": "bash run.sh",
		"cron_expr":     "*/15 * * * * *",
		"node_queue":    "default",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("create task failed: status=%d body=%s", rec.Code, rec.Body.String())
	}

	var task common.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &task); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	if task.CronExpr != "" {
		t.Fatalf("expected empty task cron_expr, got %q", task.CronExpr)
	}
	if len(app.cronEntries) != 0 {
		t.Fatalf("expected no task cron entries, got %d", len(app.cronEntries))
	}
}

func TestControlRoutes_ScheduleFlowWithoutQueue(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)
	_, _, task := createProjectVersionTask(t, app, token, "")

	scheduleRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules", token, map[string]any{
		"name":      "demo-schedule",
		"task_id":   task.ID,
		"cron_expr": "*/30 * * * * *",
		"enabled":   true,
	})
	if scheduleRec.Code != http.StatusOK {
		t.Fatalf("create schedule failed: status=%d body=%s", scheduleRec.Code, scheduleRec.Body.String())
	}

	var schedule common.Schedule
	if err := json.Unmarshal(scheduleRec.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("decode schedule: %v", err)
	}
	if schedule.ID == "" {
		t.Fatal("expected schedule id")
	}

	listRec := doJSONRequest(t, app, http.MethodGet, "/api/v1/schedules", token, nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list schedules failed: status=%d body=%s", listRec.Code, listRec.Body.String())
	}
	var schedules []common.Schedule
	if err := json.Unmarshal(listRec.Body.Bytes(), &schedules); err != nil {
		t.Fatalf("decode schedules: %v", err)
	}
	if len(schedules) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(schedules))
	}

	triggerRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules/"+schedule.ID+"/trigger", token, nil)
	if triggerRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when schedule queue unavailable, got %d body=%s", triggerRec.Code, triggerRec.Body.String())
	}

	runsRec := doJSONRequest(t, app, http.MethodGet, "/api/v1/schedules/"+schedule.ID+"/runs", token, nil)
	if runsRec.Code != http.StatusOK {
		t.Fatalf("list schedule runs failed: status=%d body=%s", runsRec.Code, runsRec.Body.String())
	}
	var runs []common.TaskRun
	if err := json.Unmarshal(runsRec.Body.Bytes(), &runs); err != nil {
		t.Fatalf("decode schedule runs: %v", err)
	}
	if len(runs) == 0 {
		t.Fatalf("expected at least one failed run after trigger")
	}
	if runs[0].ScheduleID != schedule.ID {
		t.Fatalf("expected schedule run linked to %s, got %s", schedule.ID, runs[0].ScheduleID)
	}
}

func TestControlRoutes_UpdateDeleteSchedule(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)
	_, _, task := createProjectVersionTask(t, app, token, "")

	createRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules", token, map[string]any{
		"name":      "schedule-to-update",
		"task_id":   task.ID,
		"cron_expr": "*/30 * * * * *",
		"enabled":   true,
	})
	if createRec.Code != http.StatusOK {
		t.Fatalf("create schedule failed: status=%d body=%s", createRec.Code, createRec.Body.String())
	}

	var schedule common.Schedule
	if err := json.Unmarshal(createRec.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("decode schedule: %v", err)
	}
	if len(app.scheduleEntries) != 1 {
		t.Fatalf("expected 1 registered schedule entry, got %d", len(app.scheduleEntries))
	}

	updateRec := doJSONRequest(t, app, http.MethodPut, "/api/v1/schedules/"+schedule.ID, token, map[string]any{
		"name":    "schedule-updated",
		"enabled": false,
	})
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update schedule failed: status=%d body=%s", updateRec.Code, updateRec.Body.String())
	}

	var updated common.Schedule
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated schedule: %v", err)
	}
	if updated.Name != "schedule-updated" || updated.Enabled {
		t.Fatalf("unexpected updated schedule state: name=%s enabled=%v", updated.Name, updated.Enabled)
	}
	if len(app.scheduleEntries) != 0 {
		t.Fatalf("expected schedule entries to be unregistered after disable, got %d", len(app.scheduleEntries))
	}

	deleteRec := doJSONRequest(t, app, http.MethodDelete, "/api/v1/schedules/"+schedule.ID, token, nil)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete schedule failed: status=%d body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	getRec := doJSONRequest(t, app, http.MethodGet, "/api/v1/schedules/"+schedule.ID, token, nil)
	if getRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after deleting schedule, got %d body=%s", getRec.Code, getRec.Body.String())
	}
}

func TestControlRoutes_EnableDisableSchedule(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)
	_, _, task := createProjectVersionTask(t, app, token, "")

	createRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules", token, map[string]any{
		"name":      "toggle-schedule",
		"task_id":   task.ID,
		"cron_expr": "*/30 * * * * *",
		"enabled":   true,
	})
	if createRec.Code != http.StatusOK {
		t.Fatalf("create schedule failed: status=%d body=%s", createRec.Code, createRec.Body.String())
	}

	var schedule common.Schedule
	if err := json.Unmarshal(createRec.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("decode schedule: %v", err)
	}
	if len(app.scheduleEntries) != 1 {
		t.Fatalf("expected one schedule cron entry, got %d", len(app.scheduleEntries))
	}

	disableRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules/"+schedule.ID+"/disable", token, nil)
	if disableRec.Code != http.StatusOK {
		t.Fatalf("disable schedule failed: status=%d body=%s", disableRec.Code, disableRec.Body.String())
	}
	if len(app.scheduleEntries) != 0 {
		t.Fatalf("expected no schedule cron entries after disable, got %d", len(app.scheduleEntries))
	}

	enableRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules/"+schedule.ID+"/enable", token, nil)
	if enableRec.Code != http.StatusOK {
		t.Fatalf("enable schedule failed: status=%d body=%s", enableRec.Code, enableRec.Body.String())
	}
	if len(app.scheduleEntries) != 1 {
		t.Fatalf("expected one schedule cron entry after enable, got %d", len(app.scheduleEntries))
	}
}

func TestScheduleChainResolveSteps(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)
	_, _, task := createProjectVersionTask(t, app, token, "")

	scheduleRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules", token, map[string]any{
		"name":      "chain-schedule",
		"task_id":   task.ID,
		"cron_expr": "*/30 * * * * *",
		"enabled":   false,
		"order": map[string]any{
			"name": "chain",
			"schedule": []map[string]any{
				{"task_id": task.ID, "project_id": task.ProjectID, "node": []string{"default"}},
				{"task_id": task.ID, "project_id": task.ProjectID, "node": []string{"default"}},
			},
		},
	})
	if scheduleRec.Code != http.StatusOK {
		t.Fatalf("create chain schedule failed: status=%d body=%s", scheduleRec.Code, scheduleRec.Body.String())
	}

	var schedule common.Schedule
	if err := json.Unmarshal(scheduleRec.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("decode chain schedule: %v", err)
	}

	steps, err := app.resolveScheduleExecutionSteps(schedule)
	if err != nil {
		t.Fatalf("resolve chain steps failed: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 chain steps, got %d", len(steps))
	}
	if steps[0].TaskID == "" || steps[0].EntryCommand == "" {
		t.Fatalf("first step not resolved correctly: %+v", steps[0])
	}
}

func TestScheduleChainTriggerNextStepWithoutQueue(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)
	_, _, task := createProjectVersionTask(t, app, token, "")

	scheduleRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules", token, map[string]any{
		"name":      "chain-no-queue",
		"task_id":   task.ID,
		"cron_expr": "*/30 * * * * *",
		"enabled":   false,
		"order": map[string]any{
			"name": "chain",
			"schedule": []map[string]any{
				{"task_id": task.ID, "project_id": task.ProjectID, "node": []string{"default"}},
				{"task_id": task.ID, "project_id": task.ProjectID, "node": []string{"default"}},
			},
		},
	})
	if scheduleRec.Code != http.StatusOK {
		t.Fatalf("create chain schedule failed: status=%d body=%s", scheduleRec.Code, scheduleRec.Body.String())
	}

	var schedule common.Schedule
	if err := json.Unmarshal(scheduleRec.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("decode chain schedule: %v", err)
	}

	err := app.triggerNextScheduleChainRun(common.TaskRun{
		ID:         uuid.NewString(),
		ScheduleID: schedule.ID,
		ChainID:    uuid.NewString(),
		ChainIndex: 0,
		ChainTotal: 2,
	})
	if err == nil {
		t.Fatalf("expected queue unavailable error when triggering next chain step")
	}
}

func TestControlRoutes_CreateScheduleWithAPIFirstTrigger(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)
	_, _, task := createProjectVersionTask(t, app, token, "")

	scheduleRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules", token, map[string]any{
		"name":    "api-first-schedule",
		"enabled": true,
		"order": map[string]any{
			"name": "api-chain",
			"schedule": []map[string]any{
				{"task_id": task.ID, "trigger": "api", "node": []string{"default"}},
				{"task_id": task.ID, "trigger": "previous", "node": []string{"default"}},
			},
		},
	})
	if scheduleRec.Code != http.StatusOK {
		t.Fatalf("create api-trigger schedule failed: status=%d body=%s", scheduleRec.Code, scheduleRec.Body.String())
	}

	var schedule common.Schedule
	if err := json.Unmarshal(scheduleRec.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("decode schedule: %v", err)
	}
	if schedule.CronExpr != "" {
		t.Fatalf("expected empty cron_expr for api-triggered schedule, got %q", schedule.CronExpr)
	}
	if len(app.scheduleEntries) != 0 {
		t.Fatalf("expected no cron registration for api-triggered schedule, got %d", len(app.scheduleEntries))
	}

	steps, err := app.resolveScheduleExecutionSteps(schedule)
	if err != nil {
		t.Fatalf("resolve schedule steps failed: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 chain steps, got %d", len(steps))
	}
}

func TestControlRoutes_RejectScheduleWhenNonFirstStepUsesCronOrAPI(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)
	_, _, task := createProjectVersionTask(t, app, token, "")

	scheduleRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules", token, map[string]any{
		"name":    "invalid-chain-trigger",
		"enabled": false,
		"order": map[string]any{
			"name": "invalid-chain",
			"schedule": []map[string]any{
				{"task_id": task.ID, "trigger": "crons", "crons": "*/30 * * * * *", "node": []string{"default"}},
				{"task_id": task.ID, "trigger": "crons", "crons": "*/45 * * * * *", "node": []string{"default"}},
			},
		},
	})
	if scheduleRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when non-first step uses cron/api trigger, got %d body=%s", scheduleRec.Code, scheduleRec.Body.String())
	}
}
