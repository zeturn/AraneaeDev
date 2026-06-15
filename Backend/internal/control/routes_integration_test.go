package control

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"araneae-go/gen/pb"
	"araneae-go/internal/common"
	"araneae-go/internal/control/security/password"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"
)

func newTestControlApp(t *testing.T) *App {
	t.Helper()

	tmp := t.TempDir()
	cfg := common.ControlConfig{
		JWTSecret:       "test-jwt-secret",
		ExecutionAPIKey: "test-callback-key",
		ArtifactRoot:    filepath.Join(tmp, "artifacts"),
		RSSRoot:         filepath.Join(tmp, "rss"),
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

	hash, err := password.Hash("admin123")
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
		oauthCodes:      map[string]oauthExchangeState{},
	}
	a.setupRoutes()
	return a
}

func newProtectedRun(taskID string) (common.TaskRun, string) {
	runToken := "run-token-" + uuid.NewString()
	return common.TaskRun{
		ID:            uuid.NewString(),
		TaskID:        taskID,
		TriggerSource: "manual",
		NodeQueue:     "default",
		Status:        "queued",
		CreatedAt:     time.Now(),
		CorrelationID: uuid.NewString(),
		RunTokenHash:  hashNodeKey(runToken),
	}, runToken
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

	return loginAndGetTokenFor(t, app, "admin", "admin123")
}

func loginAndGetTokenFor(t *testing.T, app *App, username, password string) string {
	t.Helper()

	rec := doJSONRequest(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"username": username,
		"password": password,
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

func countSecurityEvents(t *testing.T, app *App, eventType string) int64 {
	t.Helper()
	var count int64
	if err := app.db.Model(&common.SecurityEvent{}).Where("event_type = ?", eventType).Count(&count).Error; err != nil {
		t.Fatalf("count security events failed: %v", err)
	}
	return count
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

	hash, err := password.Hash("alice123")
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

	hash, err := password.Hash("bob123")
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
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 when deleting team via Araneae, got %d body=%s", rec.Code, rec.Body.String())
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
	if createRec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 when creating team via Araneae, got %d body=%s", createRec.Code, createRec.Body.String())
	}
}

func TestControlRoutes_RequireAuth(t *testing.T) {
	app := newTestControlApp(t)
	rec := doJSONRequest(t, app, http.MethodGet, "/api/v1/projects", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestControlRoutes_ViewerSeesOnlySelfInUsersList(t *testing.T) {
	app := newTestControlApp(t)

	hash, err := password.Hash("alice123")
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

	viewerToken := loginAndGetTokenFor(t, app, "alice", "alice123")
	usersRec := doJSONRequest(t, app, http.MethodGet, "/api/v1/users", viewerToken, nil)
	if usersRec.Code != http.StatusOK {
		t.Fatalf("list users failed: status=%d body=%s", usersRec.Code, usersRec.Body.String())
	}

	var payload struct {
		Results []map[string]any `json:"results"`
		Count   int              `json:"count"`
	}
	if err := json.Unmarshal(usersRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode users response: %v", err)
	}
	if payload.Count != 1 || len(payload.Results) != 1 {
		t.Fatalf("viewer should only see one user record, got count=%d len=%d", payload.Count, len(payload.Results))
	}
	if payload.Results[0]["id"] != user.ID {
		t.Fatalf("viewer should only see self, got %+v", payload.Results[0])
	}
}

func TestControlRoutes_ViewerCannotReadOtherProject(t *testing.T) {
	app := newTestControlApp(t)
	adminToken := loginAndGetToken(t, app)

	projectRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/projects", adminToken, map[string]string{
		"name": "admin-project",
	})
	if projectRec.Code != http.StatusOK {
		t.Fatalf("create project failed: status=%d body=%s", projectRec.Code, projectRec.Body.String())
	}
	var project common.Project
	if err := json.Unmarshal(projectRec.Body.Bytes(), &project); err != nil {
		t.Fatalf("decode project: %v", err)
	}

	hash, err := password.Hash("viewer123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	viewer := common.User{
		ID:           uuid.NewString(),
		Username:     "viewer_a",
		PasswordHash: hash,
		Role:         "viewer",
		CreatedAt:    time.Now(),
	}
	if err := app.db.Create(&viewer).Error; err != nil {
		t.Fatalf("create viewer: %v", err)
	}

	viewerToken := loginAndGetTokenFor(t, app, "viewer_a", "viewer123")

	listRec := doJSONRequest(t, app, http.MethodGet, "/api/v1/projects", viewerToken, nil)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list projects failed: status=%d body=%s", listRec.Code, listRec.Body.String())
	}
	var projects []common.Project
	if err := json.Unmarshal(listRec.Body.Bytes(), &projects); err != nil {
		t.Fatalf("decode project list: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("viewer should not see admin project list, got %d records", len(projects))
	}

	getRec := doJSONRequest(t, app, http.MethodGet, "/api/v1/projects/"+project.ID, viewerToken, nil)
	if getRec.Code != http.StatusNotFound && getRec.Code != http.StatusForbidden {
		t.Fatalf("viewer should not read other project, got %d body=%s", getRec.Code, getRec.Body.String())
	}
}

func TestControlRoutes_ViewerCanReadOwnTeamButNotOthers(t *testing.T) {
	t.Skip("team read path is delegated to BasaltPass S2S and requires external service integration")

	app := newTestControlApp(t)
	adminToken := loginAndGetToken(t, app)

	createTeamRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/teams", adminToken, map[string]any{
		"name":        "admin-private-team",
		"description": "private",
		"join_able":   false,
	})
	if createTeamRec.Code != http.StatusOK {
		t.Fatalf("create admin team failed: status=%d body=%s", createTeamRec.Code, createTeamRec.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createTeamRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created team: %v", err)
	}
	otherTeamID := int(created["id"].(float64))

	hash, err := password.Hash("teamviewer123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	viewer := common.User{
		ID:           uuid.NewString(),
		Username:     "team_viewer",
		PasswordHash: hash,
		Role:         "viewer",
		CreatedAt:    time.Now(),
	}
	if err := app.db.Create(&viewer).Error; err != nil {
		t.Fatalf("create viewer: %v", err)
	}

	var ownPersonalTeam common.Team
	if err := app.db.Where("created_by = ? AND is_personal = ?", viewer.ID, true).First(&ownPersonalTeam).Error; err != nil {
		t.Fatalf("load own personal team: %v", err)
	}

	viewerToken := loginAndGetTokenFor(t, app, "team_viewer", "teamviewer123")

	ownRec := doJSONRequest(t, app, http.MethodGet, fmt.Sprintf("/api/v1/teams/%d", ownPersonalTeam.ID), viewerToken, nil)
	if ownRec.Code != http.StatusOK {
		t.Fatalf("viewer should read own team, got %d body=%s", ownRec.Code, ownRec.Body.String())
	}

	otherRec := doJSONRequest(t, app, http.MethodGet, fmt.Sprintf("/api/v1/teams/%d", otherTeamID), viewerToken, nil)
	if otherRec.Code != http.StatusNotFound {
		t.Fatalf("viewer should not read foreign team, got %d body=%s", otherRec.Code, otherRec.Body.String())
	}

	membersRec := doJSONRequest(t, app, http.MethodGet, fmt.Sprintf("/api/v1/teams/%d/members", otherTeamID), viewerToken, nil)
	if membersRec.Code != http.StatusNotFound {
		t.Fatalf("viewer should not read foreign team members, got %d body=%s", membersRec.Code, membersRec.Body.String())
	}
}

func TestControlRoutes_WorkplaceMemberCanReadBoundProjectTaskAndSchedule(t *testing.T) {
	app := newTestControlApp(t)
	adminToken := loginAndGetToken(t, app)

	hash, err := password.Hash("workviewer123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	viewer := common.User{
		ID:           uuid.NewString(),
		Username:     "work_viewer",
		PasswordHash: hash,
		Role:         "viewer",
		CreatedAt:    time.Now(),
	}
	if err := app.db.Create(&viewer).Error; err != nil {
		t.Fatalf("create viewer: %v", err)
	}

	var personalTeam common.Team
	if err := app.db.Where("created_by = ? AND is_personal = ?", viewer.ID, true).First(&personalTeam).Error; err != nil {
		t.Fatalf("load viewer personal team: %v", err)
	}

	workplaceRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/workplaces", adminToken, map[string]any{
		"name":        "viewer-workplace",
		"description": "for visibility test",
		"status":      "active",
		"team_id":     personalTeam.ID,
	})
	if workplaceRec.Code != http.StatusOK {
		t.Fatalf("create workplace failed: status=%d body=%s", workplaceRec.Code, workplaceRec.Body.String())
	}
	var workplace map[string]any
	if err := json.Unmarshal(workplaceRec.Body.Bytes(), &workplace); err != nil {
		t.Fatalf("decode workplace response: %v", err)
	}
	workplaceID := uint(workplace["id"].(float64))

	project, version, task := createProjectVersionTask(t, app, adminToken, "")
	if err := app.db.Model(&common.Project{}).Where("id = ?", project.ID).Update("workplace_id", workplaceID).Error; err != nil {
		t.Fatalf("bind project to workplace: %v", err)
	}

	scheduleRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules", adminToken, map[string]any{
		"name":          "viewer-schedule",
		"description":   "for workplace visibility",
		"task_id":       task.ID,
		"project_id":    project.ID,
		"version_id":    version.ID,
		"entry_command": "bash run.sh",
		"node_queue":    "default",
		"enabled":       false,
	})
	if scheduleRec.Code != http.StatusOK {
		t.Fatalf("create schedule failed: status=%d body=%s", scheduleRec.Code, scheduleRec.Body.String())
	}
	var schedule common.Schedule
	if err := json.Unmarshal(scheduleRec.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("decode schedule response: %v", err)
	}

	viewerToken := loginAndGetTokenFor(t, app, "work_viewer", "workviewer123")

	projectsRec := doJSONRequest(t, app, http.MethodGet, fmt.Sprintf("/api/v1/projects?workplace_id=%d", workplaceID), viewerToken, nil)
	if projectsRec.Code != http.StatusOK {
		t.Fatalf("list projects by workplace failed: status=%d body=%s", projectsRec.Code, projectsRec.Body.String())
	}
	var projects []common.Project
	if err := json.Unmarshal(projectsRec.Body.Bytes(), &projects); err != nil {
		t.Fatalf("decode projects response: %v", err)
	}
	if len(projects) != 1 || projects[0].ID != project.ID {
		t.Fatalf("expected one bound project, got %+v", projects)
	}

	tasksRec := doJSONRequest(t, app, http.MethodGet, fmt.Sprintf("/api/v1/tasks?workplace_id=%d", workplaceID), viewerToken, nil)
	if tasksRec.Code != http.StatusOK {
		t.Fatalf("list tasks by workplace failed: status=%d body=%s", tasksRec.Code, tasksRec.Body.String())
	}
	var tasks []common.Task
	if err := json.Unmarshal(tasksRec.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("decode tasks response: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != task.ID {
		t.Fatalf("expected one bound task, got %+v", tasks)
	}

	schedulesRec := doJSONRequest(t, app, http.MethodGet, fmt.Sprintf("/api/v1/schedules?workplace_id=%d", workplaceID), viewerToken, nil)
	if schedulesRec.Code != http.StatusOK {
		t.Fatalf("list schedules by workplace failed: status=%d body=%s", schedulesRec.Code, schedulesRec.Body.String())
	}
	var schedules []common.Schedule
	if err := json.Unmarshal(schedulesRec.Body.Bytes(), &schedules); err != nil {
		t.Fatalf("decode schedules response: %v", err)
	}
	if len(schedules) != 1 || schedules[0].ID != schedule.ID {
		t.Fatalf("expected one bound schedule, got %+v", schedules)
	}
}

func TestControlRoutes_OperatorCannotCreateProjectInForeignWorkplace(t *testing.T) {
	app := newTestControlApp(t)
	adminToken := loginAndGetToken(t, app)

	opHash, err := password.Hash("operator123")
	if err != nil {
		t.Fatalf("hash operator password: %v", err)
	}
	operator := common.User{
		ID:           uuid.NewString(),
		Username:     "operator_a",
		PasswordHash: opHash,
		Role:         "operator",
		CreatedAt:    time.Now(),
	}
	if err := app.db.Create(&operator).Error; err != nil {
		t.Fatalf("create operator: %v", err)
	}
	opToken := loginAndGetTokenFor(t, app, "operator_a", "operator123")

	ownerHash, err := password.Hash("owner123")
	if err != nil {
		t.Fatalf("hash owner password: %v", err)
	}
	owner := common.User{
		ID:           uuid.NewString(),
		Username:     "work_owner",
		PasswordHash: ownerHash,
		Role:         "viewer",
		CreatedAt:    time.Now(),
	}
	if err := app.db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}

	var ownerTeam common.Team
	if err := app.db.Where("created_by = ? AND is_personal = ?", owner.ID, true).First(&ownerTeam).Error; err != nil {
		t.Fatalf("load owner personal team: %v", err)
	}

	workplaceRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/workplaces", adminToken, map[string]any{
		"name":        "foreign-workplace",
		"description": "operator should not bind here",
		"status":      "active",
		"team_id":     ownerTeam.ID,
	})
	if workplaceRec.Code != http.StatusOK {
		t.Fatalf("create workplace failed: status=%d body=%s", workplaceRec.Code, workplaceRec.Body.String())
	}
	var workplace map[string]any
	if err := json.Unmarshal(workplaceRec.Body.Bytes(), &workplace); err != nil {
		t.Fatalf("decode workplace response: %v", err)
	}
	workplaceID := uint(workplace["id"].(float64))

	projectRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/projects", opToken, map[string]any{
		"name":         "cross-workplace-project",
		"workplace_id": workplaceID,
	})
	if projectRec.Code != http.StatusForbidden {
		t.Fatalf("operator should not create project in foreign workplace: status=%d body=%s", projectRec.Code, projectRec.Body.String())
	}
}

func TestControlRoutes_OperatorCannotUpdateForeignTeam(t *testing.T) {
	app := newTestControlApp(t)
	adminToken := loginAndGetToken(t, app)

	operatorHash, err := password.Hash("operator-team-123")
	if err != nil {
		t.Fatalf("hash operator password: %v", err)
	}
	operator := common.User{
		ID:           uuid.NewString(),
		Username:     "operator_team",
		PasswordHash: operatorHash,
		Role:         "operator",
		CreatedAt:    time.Now(),
	}
	if err := app.db.Create(&operator).Error; err != nil {
		t.Fatalf("create operator: %v", err)
	}
	operatorToken := loginAndGetTokenFor(t, app, "operator_team", "operator-team-123")

	teamRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/teams", adminToken, map[string]any{
		"name":        "admin-owned-team",
		"description": "private",
		"join_able":   false,
	})
	if teamRec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 when creating team via Araneae, got %d body=%s", teamRec.Code, teamRec.Body.String())
	}

	rec := doJSONRequest(t, app, http.MethodPut, "/api/v1/teams/1", operatorToken, map[string]any{
		"name": "tampered-team",
	})
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 when updating team via Araneae, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestControlRoutes_OperatorCannotUpdateForeignWorkplace(t *testing.T) {
	app := newTestControlApp(t)
	adminToken := loginAndGetToken(t, app)

	operatorHash, err := password.Hash("operator-workplace-123")
	if err != nil {
		t.Fatalf("hash operator password: %v", err)
	}
	operator := common.User{
		ID:           uuid.NewString(),
		Username:     "operator_workplace",
		PasswordHash: operatorHash,
		Role:         "operator",
		CreatedAt:    time.Now(),
	}
	if err := app.db.Create(&operator).Error; err != nil {
		t.Fatalf("create operator: %v", err)
	}
	operatorToken := loginAndGetTokenFor(t, app, "operator_workplace", "operator-workplace-123")

	workplaceRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/workplaces", adminToken, map[string]any{
		"name":        "admin-owned-workplace",
		"description": "private",
		"status":      "active",
	})
	if workplaceRec.Code != http.StatusOK {
		t.Fatalf("create workplace failed: status=%d body=%s", workplaceRec.Code, workplaceRec.Body.String())
	}
	var workplace map[string]any
	if err := json.Unmarshal(workplaceRec.Body.Bytes(), &workplace); err != nil {
		t.Fatalf("decode workplace response: %v", err)
	}

	rec := doJSONRequest(t, app, http.MethodPut, fmt.Sprintf("/api/v1/workplaces/%d", int(workplace["id"].(float64))), operatorToken, map[string]any{
		"name": "tampered-workplace",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("operator should not update foreign workplace: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestControlRoutes_CreateScheduleRejectsTaskProjectVersionMismatch(t *testing.T) {
	app := newTestControlApp(t)
	adminToken := loginAndGetToken(t, app)

	_, _, taskA := createProjectVersionTask(t, app, adminToken, "")
	projectB, versionB, _ := createProjectVersionTask(t, app, adminToken, "")

	scheduleRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules", adminToken, map[string]any{
		"name":          "mismatch-schedule",
		"task_id":       taskA.ID,
		"project_id":    projectB.ID,
		"version_id":    versionB.ID,
		"entry_command": "bash run.sh",
		"node_queue":    "default",
		"enabled":       false,
	})
	if scheduleRec.Code != http.StatusBadRequest {
		t.Fatalf("schedule with mismatched task/project/version should fail: status=%d body=%s", scheduleRec.Code, scheduleRec.Body.String())
	}
}

func TestControlRoutes_RegisterNodeRequiresPairKey(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	fakeExecutor := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/verify" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get(nodeAuthHeader) != "pair-ok" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid node key"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","queue":"node-a"}`))
	}))
	fakeExecutor.Listener = listener
	fakeExecutor.Start()
	defer fakeExecutor.Close()

	host, portRaw, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split host port failed: %v", err)
	}
	port, err := net.LookupPort("tcp", portRaw)
	if err != nil {
		t.Fatalf("lookup port failed: %v", err)
	}

	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)

	missingKeyRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/nodes/register/", token, map[string]any{
		"ip":   host,
		"name": "executor-a",
		"port": port,
	})
	if missingKeyRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when pair_key missing, got %d body=%s", missingKeyRec.Code, missingKeyRec.Body.String())
	}

	rec := doJSONRequest(t, app, http.MethodPost, "/api/v1/nodes/register/", token, map[string]any{
		"ip":        host,
		"name":      "executor-a",
		"port":      port,
		"grpc_port": 9190,
		"pair_key":  "pair-ok",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 register node, got %d body=%s", rec.Code, rec.Body.String())
	}

	var node common.Node
	if err := json.Unmarshal(rec.Body.Bytes(), &node); err != nil {
		t.Fatalf("decode node response failed: %v", err)
	}
	if node.CeleryQueue != "node-a" {
		t.Fatalf("expected queue node-a from executor verify, got %q", node.CeleryQueue)
	}

	var persisted common.Node
	if err := app.db.Where("id = ?", node.ID).First(&persisted).Error; err != nil {
		t.Fatalf("load persisted node failed: %v", err)
	}
	if persisted.AuthTokenHash != hashNodeKey("pair-ok") {
		t.Fatalf("expected persisted auth hash %q, got %q", hashNodeKey("pair-ok"), persisted.AuthTokenHash)
	}
}

func TestControlRoutes_CallbackBypassesJWTGroup(t *testing.T) {
	app := newTestControlApp(t)
	run, runToken := newProtectedRun(uuid.NewString())
	if err := app.db.Create(&run).Error; err != nil {
		t.Fatalf("seed protected run failed: %v", err)
	}

	rec := doJSONRequest(t, app, http.MethodPost, "/api/v1/runs/"+run.ID+"/callback", "", map[string]any{
		"status":    "success",
		"output":    "ok",
		"exit_code": 0,
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without callback key, got %d", rec.Code)
	}
	if got := countSecurityEvents(t, app, "callback_invalid_key"); got == 0 {
		t.Fatalf("expected callback_invalid_key security event")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.ID+"/callback", bytes.NewReader([]byte(`{"status":"success"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Execution-Key", app.cfg.ExecutionAPIKey)
	req.Header.Set("X-Run-Token", runToken)
	req.Header.Set("X-Correlation-ID", run.CorrelationID)
	resp, err := app.http.Test(req, -1)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		b := bytes.NewBuffer(nil)
		_, _ = b.ReadFrom(resp.Body)
		t.Fatalf("expected 200 with valid callback auth, got %d body=%s", resp.StatusCode, b.String())
	}
	_ = resp.Body.Close()
}

func TestControlRoutes_CallbackUpdatesRun(t *testing.T) {
	app := newTestControlApp(t)

	run, runToken := newProtectedRun(uuid.NewString())
	if err := app.db.Create(&run).Error; err != nil {
		t.Fatalf("seed task run failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.ID+"/callback", bytes.NewReader([]byte(`{"status":"success","output":"done","exit_code":0}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Execution-Key", app.cfg.ExecutionAPIKey)
	req.Header.Set("X-Run-Token", runToken)
	req.Header.Set("X-Correlation-ID", run.CorrelationID)
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

func TestControlRoutes_CallbackIsIdempotentAfterTerminalState(t *testing.T) {
	app := newTestControlApp(t)

	run, runToken := newProtectedRun(uuid.NewString())
	if err := app.db.Create(&run).Error; err != nil {
		t.Fatalf("seed task run failed: %v", err)
	}

	first := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.ID+"/callback", bytes.NewReader([]byte(`{"status":"success","output":"done","exit_code":0}`)))
	first.Header.Set("Content-Type", "application/json")
	first.Header.Set("X-Execution-Key", app.cfg.ExecutionAPIKey)
	first.Header.Set("X-Run-Token", runToken)
	first.Header.Set("X-Correlation-ID", run.CorrelationID)
	firstResp, err := app.http.Test(first, -1)
	if err != nil {
		t.Fatalf("first callback request failed: %v", err)
	}
	if firstResp.StatusCode != http.StatusOK {
		b := bytes.NewBuffer(nil)
		_, _ = b.ReadFrom(firstResp.Body)
		t.Fatalf("expected 200 for first callback, got %d body=%s", firstResp.StatusCode, b.String())
	}
	_ = firstResp.Body.Close()

	second := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.ID+"/callback", bytes.NewReader([]byte(`{"status":"failed","output":"tampered","exit_code":1}`)))
	second.Header.Set("Content-Type", "application/json")
	second.Header.Set("X-Execution-Key", app.cfg.ExecutionAPIKey)
	second.Header.Set("X-Run-Token", runToken)
	second.Header.Set("X-Correlation-ID", run.CorrelationID)
	secondResp, err := app.http.Test(second, -1)
	if err != nil {
		t.Fatalf("second callback request failed: %v", err)
	}
	if secondResp.StatusCode != http.StatusOK {
		b := bytes.NewBuffer(nil)
		_, _ = b.ReadFrom(secondResp.Body)
		t.Fatalf("expected 200 for replay callback, got %d body=%s", secondResp.StatusCode, b.String())
	}
	_ = secondResp.Body.Close()

	var updated common.TaskRun
	if err := app.db.Where("id = ?", run.ID).First(&updated).Error; err != nil {
		t.Fatalf("load updated run failed: %v", err)
	}
	if updated.Status != "success" || updated.ExitCode != 0 || updated.Output != "done" {
		t.Fatalf("replay callback should not overwrite terminal result: status=%s code=%d output=%q", updated.Status, updated.ExitCode, updated.Output)
	}
}

func TestControlRoutes_CallbackRejectsWrongRunToken(t *testing.T) {
	app := newTestControlApp(t)
	run, _ := newProtectedRun(uuid.NewString())
	if err := app.db.Create(&run).Error; err != nil {
		t.Fatalf("seed task run failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.ID+"/callback", bytes.NewReader([]byte(`{"status":"success"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Execution-Key", app.cfg.ExecutionAPIKey)
	req.Header.Set("X-Run-Token", "wrong-token")
	req.Header.Set("X-Correlation-ID", run.CorrelationID)
	resp, err := app.http.Test(req, -1)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		b := bytes.NewBuffer(nil)
		_, _ = b.ReadFrom(resp.Body)
		t.Fatalf("expected 401 for wrong run token, got %d body=%s", resp.StatusCode, b.String())
	}
}

func TestControlRoutes_CallbackRequiresSignatureInProduction(t *testing.T) {
	app := newTestControlApp(t)
	app.cfg.Environment = "production"
	run, runToken := newProtectedRun(uuid.NewString())
	if err := app.db.Create(&run).Error; err != nil {
		t.Fatalf("seed task run failed: %v", err)
	}

	body := []byte(`{"status":"success","output":"done","exit_code":0}`)
	unsigned := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.ID+"/callback", bytes.NewReader(body))
	unsigned.Header.Set("Content-Type", "application/json")
	unsigned.Header.Set("X-Execution-Key", app.cfg.ExecutionAPIKey)
	unsigned.Header.Set("X-Run-Token", runToken)
	unsigned.Header.Set("X-Correlation-ID", run.CorrelationID)
	unsignedResp, err := app.http.Test(unsigned, -1)
	if err != nil {
		t.Fatalf("unsigned callback request failed: %v", err)
	}
	if unsignedResp.StatusCode != http.StatusUnauthorized {
		b := bytes.NewBuffer(nil)
		_, _ = b.ReadFrom(unsignedResp.Body)
		t.Fatalf("expected 401 for unsigned production callback, got %d body=%s", unsignedResp.StatusCode, b.String())
	}
	_ = unsignedResp.Body.Close()

	ts := fmt.Sprintf("%d", time.Now().Unix())
	sig := common.BuildCallbackSignature(app.cfg.ExecutionAPIKey, ts, run.ID, runToken, run.CorrelationID, body)
	signed := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+run.ID+"/callback", bytes.NewReader(body))
	signed.Header.Set("Content-Type", "application/json")
	signed.Header.Set("X-Execution-Key", app.cfg.ExecutionAPIKey)
	signed.Header.Set("X-Run-Token", runToken)
	signed.Header.Set("X-Correlation-ID", run.CorrelationID)
	signed.Header.Set(common.CallbackTimestampHeader, ts)
	signed.Header.Set(common.CallbackSignatureHeader, sig)
	signedResp, err := app.http.Test(signed, -1)
	if err != nil {
		t.Fatalf("signed callback request failed: %v", err)
	}
	if signedResp.StatusCode != http.StatusOK {
		b := bytes.NewBuffer(nil)
		_, _ = b.ReadFrom(signedResp.Body)
		t.Fatalf("expected 200 for signed callback, got %d body=%s", signedResp.StatusCode, b.String())
	}
	_ = signedResp.Body.Close()
}

func TestControl_GetArtifactRequiresRunScopedAuth(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)
	project, version, _ := createProjectVersionTask(t, app, token, "")

	node := common.Node{
		Name:           "executor-a",
		Status:         "active",
		IPAddress:      "127.0.0.1",
		Port:           4280,
		GRPCPort:       9190,
		CeleryQueue:    "default",
		AuthTokenHash:  hashNodeKey("pair-key"),
		IsEnabled:      true,
		LastActiveTime: time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := app.db.Create(&node).Error; err != nil {
		t.Fatalf("seed node failed: %v", err)
	}

	run, runToken := newProtectedRun(uuid.NewString())
	if err := app.db.Create(&run).Error; err != nil {
		t.Fatalf("seed run failed: %v", err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		controlNodeAuthMetadata, "pair-key",
		controlRunIDMetadata, run.ID,
		controlRunTokenMetadata, runToken,
		controlCorrelationIDMD, run.CorrelationID,
	))
	ctx = context.WithValue(ctx, authenticatedNodeContextKey, node)

	if _, err := app.GetArtifact(ctx, &pb.GetArtifactRequest{ProjectId: project.ID, VersionId: version.ID}); err != nil {
		t.Fatalf("expected artifact fetch to succeed, got %v", err)
	}

	badCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		controlNodeAuthMetadata, "pair-key",
		controlRunIDMetadata, run.ID,
		controlRunTokenMetadata, "wrong-token",
		controlCorrelationIDMD, run.CorrelationID,
	))
	badCtx = context.WithValue(badCtx, authenticatedNodeContextKey, node)
	if _, err := app.GetArtifact(badCtx, &pb.GetArtifactRequest{ProjectId: project.ID, VersionId: version.ID}); err == nil {
		t.Fatalf("expected artifact fetch to fail with wrong run token")
	}
}

func TestControlRoutes_ViewerCannotListNodes(t *testing.T) {
	app := newTestControlApp(t)

	hash, err := password.Hash("viewer-node-123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	viewer := common.User{
		ID:           uuid.NewString(),
		Username:     "viewer_nodes",
		PasswordHash: hash,
		Role:         "viewer",
		CreatedAt:    time.Now(),
	}
	if err := app.db.Create(&viewer).Error; err != nil {
		t.Fatalf("create viewer: %v", err)
	}

	viewerToken := loginAndGetTokenFor(t, app, "viewer_nodes", "viewer-node-123")
	rec := doJSONRequest(t, app, http.MethodGet, "/api/v1/nodes", viewerToken, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("viewer should not list nodes: status=%d body=%s", rec.Code, rec.Body.String())
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

func TestControlRoutes_RSSSubscriptionFetchesAndStoresItems(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)

	feed := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Demo Feed</title>
    <description>Demo feed description</description>
    <link>https://example.test/</link>
    <item>
      <title>First Item</title>
      <link>https://example.test/first</link>
      <guid>first-guid</guid>
      <description>First summary</description>
      <pubDate>Mon, 15 Jun 2026 10:00:00 +0000</pubDate>
    </item>
    <item>
      <title>Second Item</title>
      <link>https://example.test/second</link>
      <guid>second-guid</guid>
      <description>Second summary</description>
    </item>
  </channel>
</rss>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(feed))
	}))
	defer server.Close()

	createRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/rss/subscriptions", token, map[string]string{
		"url": server.URL + "/feed.xml",
	})
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create rss subscription failed: status=%d body=%s", createRec.Code, createRec.Body.String())
	}
	var result rssFetchResult
	if err := json.Unmarshal(createRec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode rss create response: %v", err)
	}
	if result.Subscription.ID == "" || result.Subscription.Title != "Demo Feed" {
		t.Fatalf("unexpected subscription: %+v", result.Subscription)
	}
	if result.Created != 2 || len(result.Items) != 2 {
		t.Fatalf("expected two created rss items, got created=%d len=%d", result.Created, len(result.Items))
	}
	if _, err := os.Stat(filepath.Join(result.Subscription.StorageDir, "feed.xml")); err != nil {
		t.Fatalf("feed xml was not stored locally: %v", err)
	}
	for _, item := range result.Items {
		if item.ContentPath == "" {
			t.Fatalf("missing rss item content path: %+v", item)
		}
		if _, err := os.Stat(item.ContentPath); err != nil {
			t.Fatalf("rss item was not stored locally: %v", err)
		}
	}

	refreshRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/rss/subscriptions/"+result.Subscription.ID+"/refresh", token, nil)
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("refresh rss subscription failed: status=%d body=%s", refreshRec.Code, refreshRec.Body.String())
	}
	var refreshed rssFetchResult
	if err := json.Unmarshal(refreshRec.Body.Bytes(), &refreshed); err != nil {
		t.Fatalf("decode rss refresh response: %v", err)
	}
	if refreshed.Created != 0 || refreshed.Updated != 2 {
		t.Fatalf("expected refresh to update two existing items, got created=%d updated=%d", refreshed.Created, refreshed.Updated)
	}

	itemsRec := doJSONRequest(t, app, http.MethodGet, "/api/v1/rss/subscriptions/"+result.Subscription.ID+"/items", token, nil)
	if itemsRec.Code != http.StatusOK {
		t.Fatalf("list rss items failed: status=%d body=%s", itemsRec.Code, itemsRec.Body.String())
	}
	var items []common.RSSItem
	if err := json.Unmarshal(itemsRec.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode rss items: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected two rss items, got %d", len(items))
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

func TestControlRoutes_TriggerTaskRejectsRecentManualDuplicate(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)
	_, _, task := createProjectVersionTask(t, app, token, "")

	existing := common.TaskRun{
		ID:            uuid.NewString(),
		TaskID:        task.ID,
		TriggerSource: "manual",
		Status:        "running",
		CreatedAt:     time.Now(),
		CorrelationID: uuid.NewString(),
	}
	if err := app.db.Create(&existing).Error; err != nil {
		t.Fatalf("seed existing run failed: %v", err)
	}

	rec := doJSONRequest(t, app, http.MethodPost, "/api/v1/tasks/"+task.ID+"/trigger", token, nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for duplicate task trigger, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := countSecurityEvents(t, app, "task_trigger_duplicate_blocked"); got == 0 {
		t.Fatalf("expected task_trigger_duplicate_blocked security event")
	}
}

func TestControlRoutes_TriggerScheduleRejectsRecentManualDuplicate(t *testing.T) {
	app := newTestControlApp(t)
	token := loginAndGetToken(t, app)
	_, _, task := createProjectVersionTask(t, app, token, "")

	scheduleRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules", token, map[string]any{
		"name":      "dedup-schedule",
		"task_id":   task.ID,
		"cron_expr": "",
		"enabled":   false,
	})
	if scheduleRec.Code != http.StatusOK {
		t.Fatalf("create schedule failed: status=%d body=%s", scheduleRec.Code, scheduleRec.Body.String())
	}
	var schedule common.Schedule
	if err := json.Unmarshal(scheduleRec.Body.Bytes(), &schedule); err != nil {
		t.Fatalf("decode schedule: %v", err)
	}

	existing := common.TaskRun{
		ID:            uuid.NewString(),
		TaskID:        task.ID,
		ScheduleID:    schedule.ID,
		TriggerSource: "manual",
		Status:        "queued",
		CreatedAt:     time.Now(),
		CorrelationID: uuid.NewString(),
	}
	if err := app.db.Create(&existing).Error; err != nil {
		t.Fatalf("seed existing schedule run failed: %v", err)
	}

	rec := doJSONRequest(t, app, http.MethodPost, "/api/v1/schedules/"+schedule.ID+"/trigger", token, nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for duplicate schedule trigger, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := countSecurityEvents(t, app, "schedule_trigger_duplicate_blocked"); got == 0 {
		t.Fatalf("expected schedule_trigger_duplicate_blocked security event")
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
