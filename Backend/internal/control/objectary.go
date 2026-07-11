package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"araneae-go/internal/common"
)

// objectaryTokenCache caches the exchanged Objectary token per user for a short
// window to avoid hitting BasaltPass token-exchange on every request.
type objectaryTokenCacheEntry struct {
	token     string
	expiresAt time.Time
}

type objectaryTokenCache struct {
	mu      sync.Mutex
	entries map[string]objectaryTokenCacheEntry
}

var objTokenCache = &objectaryTokenCache{entries: map[string]objectaryTokenCacheEntry{}}

func (c *objectaryTokenCache) get(uid string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[uid]
	if !ok {
		return "", false
	}
	if !e.expiresAt.After(time.Now()) {
		delete(c.entries, uid)
		return "", false
	}
	return e.token, true
}

func (c *objectaryTokenCache) set(uid, token string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[uid] = objectaryTokenCacheEntry{token: token, expiresAt: time.Now().Add(ttl)}
}

// objectaryTokenForCurrentUser resolves an Objectary-scoped access token for the
// current Araneae user by: (1) reading the cached exchange, (2) reusing the
// persisted BasaltPass token (refreshing it if expired), then (3) performing a
// BasaltPass cross-app token exchange for the Objectary resource.
func (a *App) objectaryTokenForCurrentUser(c *fiber.Ctx) (string, error) {
	uid, _ := c.Locals("uid").(string)
	if uid == "" {
		return "", errors.New("missing user identity")
	}
	if tok, ok := objTokenCache.get(uid); ok {
		return tok, nil
	}

	var user common.User
	if err := a.db.Where("id = ?", uid).First(&user).Error; err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}
	if strings.TrimSpace(user.BasaltAccessToken) == "" {
		return "", errors.New("no BasaltPass credentials stored for this user; please log in via BasaltPass again")
	}

	// Refresh the BasaltPass token if it has expired and we have a refresh token.
	if user.BasaltTokenExpiry != nil && user.BasaltTokenExpiry.Before(time.Now()) {
		if strings.TrimSpace(user.BasaltRefreshToken) == "" {
			return "", errors.New("BasaltPass token expired; please log in via BasaltPass again")
		}
		newAccess, newRefresh, newExpiry, err := a.refreshBasaltToken(user.BasaltRefreshToken)
		if err != nil {
			return "", fmt.Errorf("failed to refresh BasaltPass token: %w", err)
		}
		user.BasaltAccessToken = newAccess
		if newRefresh != "" {
			user.BasaltRefreshToken = newRefresh
		}
		user.BasaltTokenExpiry = newExpiry
		if err := a.db.Model(&user).Updates(map[string]any{
			"basalt_access_token":  user.BasaltAccessToken,
			"basalt_refresh_token": user.BasaltRefreshToken,
			"basalt_token_expiry":  user.BasaltTokenExpiry,
		}).Error; err != nil {
			return "", fmt.Errorf("failed to persist refreshed token: %w", err)
		}
	}

	objToken, err := a.exchangeObjectaryToken(user.BasaltAccessToken)
	if err != nil {
		return "", err
	}
	objTokenCache.set(uid, objToken, 4*time.Minute)
	return objToken, nil
}

// refreshBasaltToken uses the BasaltPass refresh_token grant to obtain a new
// access token for the Araneae user.
func (a *App) refreshBasaltToken(refreshToken string) (string, string, *time.Time, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", strings.TrimSpace(refreshToken))
	form.Set("client_id", strings.TrimSpace(a.cfg.BasaltClientID))
	if secret := strings.TrimSpace(a.cfg.BasaltClientSecret); secret != "" {
		form.Set("client_secret", secret)
	}

	req, err := http.NewRequest(http.MethodPost, trimBasaltURL(a.cfg.BasaltInternalBaseURL)+"/api/v1/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", "", nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return "", "", nil, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", "", nil, err
	}
	access := extractStringValue(payload, "access_token", "accessToken", "access")
	if access == "" {
		return "", "", nil, errors.New("token refresh returned no access_token")
	}
	refresh := extractStringValue(payload, "refresh_token", "refreshToken")
	var expiry *time.Time
	if raw, ok := payload["expires_in"]; ok {
		var seconds int
		switch v := raw.(type) {
		case float64:
			seconds = int(v)
		case int:
			seconds = v
		case string:
			if parsed, perr := parseStrictInt(v); perr == nil {
				seconds = parsed
			}
		}
		if seconds > 0 {
			e := time.Now().Add(time.Duration(seconds) * time.Second)
			expiry = &e
		}
	}
	return access, refresh, expiry, nil
}

// exchangeObjectaryToken performs a BasaltPass OAuth 2.0 token exchange (RFC 8693)
// to obtain an Objectary-scoped access token for the current user.
func (a *App) exchangeObjectaryToken(subjectToken string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	form.Set("subject_token", strings.TrimSpace(subjectToken))
	form.Set("subject_token_type", "urn:ietf:params:oauth:token-type:access_token")
	form.Set("resource", strings.TrimSpace(a.cfg.ObjectaryResource))
	form.Set("scope", strings.TrimSpace(a.cfg.ObjectaryScope))

	req, err := http.NewRequest(http.MethodPost, trimBasaltURL(a.cfg.BasaltInternalBaseURL)+"/api/v1/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(strings.TrimSpace(a.cfg.BasaltClientID), strings.TrimSpace(a.cfg.BasaltClientSecret))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	token := extractStringValue(payload, "access_token", "accessToken", "access")
	if token == "" {
		return "", errors.New("token exchange returned no access_token")
	}
	return token, nil
}

func (a *App) objectaryListNodesHandler(c *fiber.Ctx) error {
	if !a.cfg.ObjectaryEnabled {
		return fiber.NewError(fiber.StatusServiceUnavailable, "objectary integration is disabled")
	}
	objToken, err := a.objectaryTokenForCurrentUser(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	provider := strings.TrimSpace(c.Query("provider"))
	if provider == "" {
		provider = a.cfg.ObjectaryProvider
	}
	parentID := strings.TrimSpace(c.Query("parentId"))
	if parentID == "" {
		parentID = "root"
	}
	data, err := a.objectaryListNodes(provider, parentID, objToken)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	return c.JSON(data)
}

func (a *App) objectaryListNodes(provider, parentID, objToken string) (map[string]any, error) {
	u := strings.TrimRight(strings.TrimSpace(a.cfg.ObjectaryBaseURL), "/") +
		"/api/v1/nodes/" + url.PathEscape(parentID) + "/children?provider=" + url.QueryEscape(provider)
	return a.objectaryJSONGet(u, objToken)
}

func (a *App) objectaryJSONGet(rawURL, objToken string) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+objToken)
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("objectary request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return data, nil
}

type objectaryImportRequest struct {
	ProjectID string `json:"project_id"`
	NodeID    string `json:"node_id"`
	Key       string `json:"key"`
	Provider  string `json:"provider"`
}

func (a *App) objectaryImportHandler(c *fiber.Ctx) error {
	if !a.cfg.ObjectaryEnabled {
		return fiber.NewError(fiber.StatusServiceUnavailable, "objectary integration is disabled")
	}
	var req objectaryImportRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	projectID := strings.TrimSpace(req.ProjectID)
	nodeID := strings.TrimSpace(req.NodeID)
	if projectID == "" || nodeID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "project_id and node_id are required")
	}

	var p common.Project
	if err := a.db.Where("id = ?", projectID).First(&p).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "project not found")
	}
	canWrite, accessErr := a.canWriteProject(c, p)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canWrite {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions to write this project")
	}

	objToken, err := a.objectaryTokenForCurrentUser(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		provider = a.cfg.ObjectaryProvider
	}

	content, fileName, err := a.objectaryDownload(provider, nodeID, objToken)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
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

func (a *App) objectaryDownload(provider, nodeID, objToken string) ([]byte, string, error) {
	u := strings.TrimRight(strings.TrimSpace(a.cfg.ObjectaryBaseURL), "/") +
		"/api/v1/nodes/" + url.PathEscape(nodeID) + "/download?provider=" + url.QueryEscape(provider)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+objToken)

	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return nil, "", fmt.Errorf("objectary download failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxArtifactUploadBytes)+1))
	if err != nil {
		return nil, "", err
	}
	if len(content) > maxArtifactUploadBytes {
		return nil, "", fmt.Errorf("file from Objectary exceeds max upload size of %d bytes", maxArtifactUploadBytes)
	}
	fileName := fileNameFromContentDisposition(resp.Header.Get("Content-Disposition"))
	if fileName == "" {
		fileName = strings.TrimSpace(nodeID)
	}
	return content, fileName, nil
}

func fileNameFromContentDisposition(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	for _, part := range strings.Split(header, ";") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(strings.ToLower(part), "filename=") {
			continue
		}
		value := strings.TrimPrefix(part, "filename=")
		value = strings.Trim(value, `"`)
		return value
	}
	return ""
}

func parseStrictInt(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, errors.New("empty")
	}
	var n int
	_, err := fmt.Sscanf(raw, "%d", &n)
	return n, err
}
