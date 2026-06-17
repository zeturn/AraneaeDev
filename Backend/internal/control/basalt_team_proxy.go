package control

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func (a *App) basaltS2SCredentials() (string, string, error) {
	clientID := strings.TrimSpace(a.cfg.BasaltClientID)
	clientSecret := strings.TrimSpace(a.cfg.BasaltClientSecret)
	if clientID == "" || clientSecret == "" {
		return "", "", errors.New("missing BASALTPASS_OAUTH_CLIENT_ID or BASALTPASS_OAUTH_CLIENT_SECRET")
	}
	// BasaltPass /api/v1/s2s/* uses client headers directly instead of OAuth client_credentials.
	return clientID, clientSecret, nil
}

func (a *App) basaltS2SGet(path string, query url.Values) (map[string]any, error) {
	clientID, clientSecret, err := a.basaltS2SCredentials()
	if err != nil {
		return nil, err
	}

	u := trimBasaltURL(a.cfg.BasaltInternalBaseURL) + path
	if encoded := query.Encode(); encoded != "" {
		u += "?" + encoded
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("client_id", clientID)
	req.Header.Set("client_secret", clientSecret)
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("s2s request %s failed with status %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (a *App) basaltS2SPost(path string, body any) (map[string]any, error) {
	clientID, clientSecret, err := a.basaltS2SCredentials()
	if err != nil {
		return nil, err
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	u := trimBasaltURL(a.cfg.BasaltInternalBaseURL) + path
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("client_id", clientID)
	req.Header.Set("client_secret", clientSecret)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("s2s request %s failed with status %d: %s", path, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var payload map[string]any
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func toAnySlice(v any) []any {
	if v == nil {
		return nil
	}
	if out, ok := v.([]any); ok {
		return out
	}
	return nil
}

func toMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	out, _ := v.(map[string]any)
	return out
}

func (a *App) basaltLookupUserIDByEmail(email string) (uint, error) {
	q := url.Values{}
	q.Set("email", strings.TrimSpace(email))
	payload, err := a.basaltS2SGet("/api/v1/s2s/users/lookup", q)
	if err != nil {
		return 0, err
	}
	data := toMap(payload["data"])
	users := toAnySlice(data["users"])
	if len(users) == 0 {
		return 0, errors.New("user not found in BasaltPass tenant")
	}
	first := toMap(users[0])
	rawID := first["id"]
	switch v := rawID.(type) {
	case float64:
		if v <= 0 {
			return 0, errors.New("invalid user id from BasaltPass")
		}
		return uint(v), nil
	case string:
		parsed, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		if err != nil || parsed == 0 {
			return 0, errors.New("invalid user id from BasaltPass")
		}
		return uint(parsed), nil
	default:
		return 0, errors.New("invalid user id from BasaltPass")
	}
}

func (a *App) basaltGetUserTeams(email string) ([]fiber.Map, error) {
	userID, err := a.basaltLookupUserIDByEmail(email)
	if err != nil {
		if strings.Contains(err.Error(), "user not found in BasaltPass tenant") {
			return a.basaltListTeams()
		}
		return nil, err
	}
	payload, err := a.basaltS2SGet("/api/v1/s2s/users/"+strconv.FormatUint(uint64(userID), 10)+"/teams", nil)
	if err != nil {
		return nil, err
	}
	data := toMap(payload["data"])
	items := toAnySlice(data["teams"])
	out := make([]fiber.Map, 0, len(items))
	for _, item := range items {
		row := toMap(item)
		out = append(out, fiber.Map{
			"id":          row["id"],
			"name":        row["name"],
			"description": row["description"],
			"join_able":   false,
			"is_personal": false,
			"role":        row["role"],
			"created_at":  row["created_at"],
			"updated_at":  row["updated_at"],
		})
	}
	return out, nil
}

func (a *App) basaltListTeams() ([]fiber.Map, error) {
	payload, err := a.basaltS2SGet("/api/v1/s2s/teams", nil)
	if err != nil {
		return nil, err
	}
	data := toMap(payload["data"])
	items := toAnySlice(data["teams"])
	out := make([]fiber.Map, 0, len(items))
	for _, item := range items {
		row := toMap(item)
		out = append(out, fiber.Map{
			"id":            row["id"],
			"name":          row["name"],
			"description":   row["description"],
			"join_able":     false,
			"is_personal":   false,
			"members_count": row["member_count"],
			"created_at":    row["created_at"],
			"updated_at":    row["updated_at"],
		})
	}
	return out, nil
}

func (a *App) basaltCreateTeam(name, description, ownerEmail string) (fiber.Map, error) {
	ownerEmail = strings.TrimSpace(ownerEmail)
	if ownerEmail == "" {
		return nil, errors.New("email is required to create a BasaltPass team owner")
	}
	ownerID, err := a.basaltLookupUserIDByEmail(ownerEmail)
	if err != nil {
		return nil, err
	}
	if ownerID == 0 {
		return nil, errors.New("invalid BasaltPass owner user id")
	}

	reqBody := map[string]any{
		"name":          strings.TrimSpace(name),
		"description":   strings.TrimSpace(description),
		"owner_user_id": ownerID,
	}

	payload, err := a.basaltS2SPost("/api/v1/s2s/teams", reqBody)
	if err != nil {
		return nil, err
	}
	data := toMap(payload["data"])
	return fiber.Map{
		"id":            data["id"],
		"name":          data["name"],
		"description":   data["description"],
		"join_able":     false,
		"is_personal":   false,
		"members_count": data["member_count"],
		"created_at":    data["created_at"],
		"updated_at":    data["updated_at"],
	}, nil
}

func (a *App) basaltGetTeamDetail(teamID string) (fiber.Map, error) {
	payload, err := a.basaltS2SGet("/api/v1/s2s/teams/"+url.PathEscape(strings.TrimSpace(teamID)), nil)
	if err != nil {
		return nil, err
	}
	data := toMap(payload["data"])
	return fiber.Map{
		"id":            data["id"],
		"name":          data["name"],
		"description":   data["description"],
		"join_able":     false,
		"is_personal":   false,
		"created_by":    "",
		"created_at":    data["created_at"],
		"updated_at":    data["updated_at"],
		"members_count": data["member_count"],
	}, nil
}

func (a *App) basaltGetTeamMembers(teamID string) ([]fiber.Map, error) {
	payload, err := a.basaltS2SGet("/api/v1/s2s/teams/"+url.PathEscape(strings.TrimSpace(teamID)), nil)
	if err != nil {
		return nil, err
	}
	data := toMap(payload["data"])
	members := toAnySlice(data["members"])
	out := make([]fiber.Map, 0, len(members))
	for _, member := range members {
		m := toMap(member)
		user := toMap(m["user"])
		out = append(out, fiber.Map{
			"user": fiber.Map{
				"id":       user["id"],
				"username": user["nickname"],
				"email":    user["email"],
			},
			"role": m["role"],
		})
	}
	return out, nil
}
