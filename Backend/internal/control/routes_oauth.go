package control

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"araneae-go/internal/common"
	"araneae-go/internal/control/security/password"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	basaltStateCookie    = "araneae_basalt_state"
	basaltVerifierCookie = "araneae_basalt_verifier"
	basaltNextCookie     = "araneae_basalt_next"
	basaltCookiePath     = "/api/auth/basaltpass"
	basaltCookieMaxAge   = 600
	basaltExchangeTTL    = 5 * time.Minute
)

type oauthExchangeState struct {
	Token     string
	Next      string
	ExpiresAt time.Time
}

type basaltExchangeRequest struct {
	Code string `json:"code"`
}

func trimBasaltURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

func safeFrontendNext(raw string) string {
	next := strings.TrimSpace(raw)
	if next == "" {
		return "/aprons/workplaces"
	}
	if strings.HasPrefix(next, "/") {
		return next
	}
	return "/aprons/workplaces"
}

func randomURLSafeToken(byteLen int) (string, error) {
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func createPKCEPair() (string, string, error) {
	verifier, err := randomURLSafeToken(32)
	if err != nil {
		return "", "", err
	}
	challengeBytes := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(challengeBytes[:]), nil
}

func (a *App) setBasaltOAuthCookies(c *fiber.Ctx, state, verifier, next string) {
	secure := strings.HasPrefix(strings.ToLower(strings.TrimSpace(a.cfg.FrontendBaseURL)), "https://")
	for _, entry := range []struct {
		name  string
		value string
	}{
		{name: basaltStateCookie, value: state},
		{name: basaltVerifierCookie, value: verifier},
		{name: basaltNextCookie, value: next},
	} {
		c.Cookie(&fiber.Cookie{
			Name:     entry.name,
			Value:    entry.value,
			Path:     basaltCookiePath,
			HTTPOnly: true,
			SameSite: "lax",
			Secure:   secure,
			MaxAge:   basaltCookieMaxAge,
		})
	}
}

func (a *App) clearBasaltOAuthCookies(c *fiber.Ctx) {
	secure := strings.HasPrefix(strings.ToLower(strings.TrimSpace(a.cfg.FrontendBaseURL)), "https://")
	for _, name := range []string{basaltStateCookie, basaltVerifierCookie, basaltNextCookie} {
		c.Cookie(&fiber.Cookie{
			Name:     name,
			Value:    "",
			Path:     basaltCookiePath,
			HTTPOnly: true,
			SameSite: "lax",
			Secure:   secure,
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
		})
	}
}

func (a *App) buildBasaltAuthorizeURL(state, challenge string) (string, error) {
	if !a.cfg.BasaltOAuthEnabled {
		return "", errors.New("BasaltPass OAuth is disabled")
	}
	if strings.TrimSpace(a.cfg.BasaltClientID) == "" {
		return "", errors.New("missing BASALTPASS_OAUTH_CLIENT_ID")
	}
	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", strings.TrimSpace(a.cfg.BasaltClientID))
	values.Set("redirect_uri", strings.TrimSpace(a.cfg.BasaltRedirectURI))
	values.Set("scope", strings.TrimSpace(a.cfg.BasaltScope))
	values.Set("state", state)
	values.Set("code_challenge", challenge)
	values.Set("code_challenge_method", "S256")
	return trimBasaltURL(a.cfg.BasaltBaseURL) + "/api/v1/oauth/authorize?" + values.Encode(), nil
}

func (a *App) exchangeBasaltCode(code, verifier string) (map[string]any, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", strings.TrimSpace(a.cfg.BasaltClientID))
	form.Set("code", code)
	form.Set("redirect_uri", strings.TrimSpace(a.cfg.BasaltRedirectURI))
	form.Set("code_verifier", verifier)
	if secret := strings.TrimSpace(a.cfg.BasaltClientSecret); secret != "" {
		form.Set("client_secret", secret)
	}

	req, err := http.NewRequest(http.MethodPost, trimBasaltURL(a.cfg.BasaltInternalBaseURL)+"/api/v1/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (a *App) fetchBasaltUserInfo(accessToken string) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, trimBasaltURL(a.cfg.BasaltInternalBaseURL)+"/api/v1/oauth/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("userinfo failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (a *App) introspectBasaltToken(token string) (map[string]any, error) {
	form := url.Values{}
	form.Set("token", token)

	req, err := http.NewRequest(http.MethodPost, trimBasaltURL(a.cfg.BasaltInternalBaseURL)+"/api/v1/oauth/introspect", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if clientID := strings.TrimSpace(a.cfg.BasaltClientID); clientID != "" {
		req.SetBasicAuth(clientID, strings.TrimSpace(a.cfg.BasaltClientSecret))
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("introspect failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func extractStringValue(payload map[string]any, keys ...string) string {
	tryExtract := func(source map[string]any) string {
		for _, key := range keys {
			if value, ok := source[key].(string); ok {
				value = strings.TrimSpace(value)
				if value != "" {
					return value
				}
			}
		}
		return ""
	}

	if value := tryExtract(payload); value != "" {
		return value
	}

	for _, nestedKey := range []string{"data", "result", "payload"} {
		nested, ok := payload[nestedKey].(map[string]any)
		if !ok {
			continue
		}
		if value := tryExtract(nested); value != "" {
			return value
		}
	}
	return ""
}

func normalizeBasaltUsername(subject string) string {
	hash := sha256.Sum256([]byte(subject))
	return "bp_" + hex.EncodeToString(hash[:])[:24]
}

func basaltDisplayNameFromClaims(claims map[string]any) string {
	name := extractStringValue(claims, "name", "full_name", "display_name", "preferred_username", "nickname", "username")
	if name != "" {
		return name
	}
	given := extractStringValue(claims, "given_name", "first_name")
	family := extractStringValue(claims, "family_name", "last_name")
	if given == "" && family == "" {
		email := extractStringValue(claims, "email")
		if at := strings.Index(email, "@"); at > 0 {
			return strings.TrimSpace(email[:at])
		}
		return extractStringValue(claims, "sub", "user_id", "id")
	}
	return strings.TrimSpace(strings.TrimSpace(given) + " " + strings.TrimSpace(family))
}

func (a *App) findOrCreateBasaltUser(subject, scopeRaw string, claims map[string]any) (*common.User, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return nil, errors.New("missing subject")
	}
	username := normalizeBasaltUsername(subject)
	role := a.basaltRoleFromIdentity(scopeRaw, claims)
	profileName := basaltDisplayNameFromClaims(claims)
	email := extractStringValue(claims, "email", "preferred_username", "upn")
	if a.isBasaltAdminEmail(email) {
		role = "admin"
	}
	var user common.User
	err := a.db.Where("username = ?", username).First(&user).Error
	if err == nil {
		if role != "admin" && a.isBasaltAdminEmail(user.Email) {
			role = "admin"
		}
		updates := map[string]any{}
		if user.Role != role {
			updates["role"] = role
			user.Role = role
		}
		if profileName != "" && strings.TrimSpace(user.Name) != profileName {
			updates["name"] = profileName
			user.Name = profileName
		}
		if email != "" && strings.TrimSpace(user.Email) != email {
			updates["email"] = email
			user.Email = email
		}
		if len(updates) > 0 {
			if updateErr := a.db.Model(&common.User{}).Where("id = ?", user.ID).Updates(updates).Error; updateErr != nil {
				return nil, updateErr
			}
		}
		if syncErr := a.syncBasaltGroups(user, claims); syncErr != nil {
			return nil, syncErr
		}
		return &user, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	randomPassword, err := randomURLSafeToken(24)
	if err != nil {
		return nil, err
	}
	passwordHash, err := password.Hash(randomPassword)
	if err != nil {
		return nil, err
	}

	user = common.User{
		ID:           uuid.NewString(),
		Username:     username,
		Name:         profileName,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    time.Now(),
	}
	if err := a.db.Create(&user).Error; err != nil {
		return nil, err
	}
	if syncErr := a.syncBasaltGroups(user, claims); syncErr != nil {
		return nil, syncErr
	}
	return &user, nil
}

func (a *App) basaltPassLogin(c *fiber.Ctx) error {
	safeNext := safeFrontendNext(c.Query("next"))
	state, err := randomURLSafeToken(24)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	verifier, challenge, err := createPKCEPair()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	loginURL, err := a.buildBasaltAuthorizeURL(state, challenge)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	a.setBasaltOAuthCookies(c, state, verifier, safeNext)
	return c.Redirect(loginURL, fiber.StatusFound)
}

func (a *App) basaltPassCallback(c *fiber.Ctx) error {
	defer a.clearBasaltOAuthCookies(c)

	if oauthError := strings.TrimSpace(c.Query("error")); oauthError != "" {
		description := strings.TrimSpace(c.Query("error_description"))
		if description != "" {
			oauthError = oauthError + ": " + description
		}
		return fiber.NewError(fiber.StatusBadRequest, oauthError)
	}

	state := strings.TrimSpace(c.Query("state"))
	code := strings.TrimSpace(c.Query("code"))
	cookieState := strings.TrimSpace(c.Cookies(basaltStateCookie))
	verifier := strings.TrimSpace(c.Cookies(basaltVerifierCookie))
	next := safeFrontendNext(c.Cookies(basaltNextCookie))
	if state == "" || cookieState == "" || state != cookieState {
		return fiber.NewError(fiber.StatusBadRequest, "invalid oauth state")
	}
	if code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing authorization code")
	}
	if verifier == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing oauth verifier")
	}

	tokenPayload, err := a.exchangeBasaltCode(code, verifier)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	accessToken := extractStringValue(tokenPayload, "access_token", "accessToken", "access", "token")
	if accessToken == "" {
		return fiber.NewError(fiber.StatusBadGateway, "missing access token")
	}

	userInfo, err := a.fetchBasaltUserInfo(accessToken)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	subject := extractStringValue(userInfo, "sub", "user_id", "id")
	if subject == "" {
		return fiber.NewError(fiber.StatusBadGateway, "missing subject from userinfo")
	}

	scopeRaw := normalizeScopes(extractStringValue(tokenPayload, "scope"))
	identityClaims := mergeClaims(tokenPayload, userInfo)
	user, err := a.findOrCreateBasaltUser(subject, scopeRaw, identityClaims)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	displayName := basaltDisplayNameFromClaims(identityClaims)
	email := extractStringValue(identityClaims, "email")
	if displayName == "" {
		displayName = strings.TrimSpace(user.Name)
	}
	if email == "" {
		email = strings.TrimSpace(user.Email)
	}
	token, err := a.issueTokenWithIdentity(user.ID, user.Role, displayName, email)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	exchangeCode, err := randomURLSafeToken(24)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	a.storeOAuthExchangeCode(exchangeCode, token, next)

	callbackPath := strings.TrimSpace(a.cfg.BasaltCallbackPath)
	if callbackPath == "" {
		callbackPath = "/oauth/callback"
	}
	frontendBase := strings.TrimRight(strings.TrimSpace(a.cfg.FrontendBaseURL), "/")
	if frontendBase == "" {
		frontendBase = "http://localhost:5109"
	}
	redirectURL := fmt.Sprintf("%s%s?code=%s&next=%s", frontendBase, callbackPath, url.QueryEscape(exchangeCode), url.QueryEscape(next))
	return c.Redirect(redirectURL, fiber.StatusFound)
}

func (a *App) basaltPassExchange(c *fiber.Ctx) error {
	var req basaltExchangeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	token, next, ok := a.consumeOAuthExchangeCode(req.Code)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid oauth exchange code")
	}
	return c.JSON(fiber.Map{
		"access": token,
		"next":   next,
	})
}

func (a *App) storeOAuthExchangeCode(code, token, next string) {
	a.oauthMu.Lock()
	defer a.oauthMu.Unlock()

	now := time.Now()
	for key, state := range a.oauthCodes {
		if !state.ExpiresAt.After(now) {
			delete(a.oauthCodes, key)
		}
	}
	a.oauthCodes[strings.TrimSpace(code)] = oauthExchangeState{
		Token:     strings.TrimSpace(token),
		Next:      safeFrontendNext(next),
		ExpiresAt: now.Add(basaltExchangeTTL),
	}
}

func (a *App) consumeOAuthExchangeCode(code string) (string, string, bool) {
	a.oauthMu.Lock()
	defer a.oauthMu.Unlock()

	state, ok := a.oauthCodes[strings.TrimSpace(code)]
	if !ok {
		return "", "", false
	}
	delete(a.oauthCodes, strings.TrimSpace(code))
	if !state.ExpiresAt.After(time.Now()) {
		return "", "", false
	}
	return state.Token, state.Next, true
}
