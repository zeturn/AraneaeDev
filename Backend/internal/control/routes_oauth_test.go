package control

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCreatePKCEPairUsesRFC7636S256Encoding(t *testing.T) {
	verifier, challenge, err := createPKCEPair()
	if err != nil {
		t.Fatalf("createPKCEPair failed: %v", err)
	}
	if verifier == "" || challenge == "" {
		t.Fatal("expected verifier and challenge")
	}

	sum := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge != expected {
		t.Fatalf("unexpected challenge encoding: got %s want %s", challenge, expected)
	}
}

func TestBasaltPassLoginRedirectsToAuthorizeURL(t *testing.T) {
	app := newTestControlApp(t)
	app.cfg.BasaltOAuthEnabled = true
	app.cfg.BasaltBaseURL = "http://basalt.example"
	app.cfg.BasaltInternalBaseURL = "http://basalt-internal.example"
	app.cfg.BasaltClientID = "client-123"
	app.cfg.BasaltRedirectURI = "http://localhost:8180/api/auth/basaltpass/callback/"
	app.cfg.BasaltScope = "openid profile email"

	req := httptest.NewRequest(http.MethodGet, "/api/auth/basaltpass/login/?next=%2Faprons%2Fworkplaces", nil)
	resp, err := app.http.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}

	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	if got := location.Scheme + "://" + location.Host + location.Path; got != "http://basalt.example/api/v1/oauth/authorize" {
		t.Fatalf("unexpected redirect target: %s", got)
	}

	query := location.Query()
	if query.Get("client_id") != "client-123" {
		t.Fatalf("unexpected client_id: %s", query.Get("client_id"))
	}
	if query.Get("redirect_uri") != "http://localhost:8180/api/auth/basaltpass/callback/" {
		t.Fatalf("unexpected redirect_uri: %s", query.Get("redirect_uri"))
	}
	if query.Get("code_challenge") == "" {
		t.Fatal("missing code_challenge")
	}
	if query.Get("state") == "" {
		t.Fatal("missing state")
	}
	if query.Get("nonce") == "" {
		t.Fatal("missing nonce")
	}

	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected oauth cookies to be set")
	}
}

func TestBasaltPassCallbackRedirectsToFrontendCallback(t *testing.T) {
	fakeBasalt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/oauth/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if r.Form.Get("code") != "code-123" {
				t.Fatalf("unexpected code: %s", r.Form.Get("code"))
			}
			if r.Form.Get("code_verifier") != "verifier-123" {
				t.Fatalf("unexpected verifier: %s", r.Form.Get("code_verifier"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "bp-access-token"})
		case "/api/v1/oauth/userinfo":
			if got := r.Header.Get("Authorization"); got != "Bearer bp-access-token" {
				t.Fatalf("unexpected authorization header: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sub":   "user-subject-1",
				"name":  "Basalt Demo User",
				"email": "demo.user@example.com",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer fakeBasalt.Close()

	app := newTestControlApp(t)
	app.cfg.BasaltOAuthEnabled = true
	app.cfg.BasaltBaseURL = fakeBasalt.URL
	app.cfg.BasaltInternalBaseURL = fakeBasalt.URL
	app.cfg.BasaltClientID = "client-123"
	app.cfg.BasaltClientSecret = "secret-123"
	app.cfg.BasaltRedirectURI = "http://localhost:8180/api/auth/basaltpass/callback/"
	app.cfg.BasaltScope = "openid profile email"
	app.cfg.FrontendBaseURL = "http://localhost:5109"
	app.cfg.BasaltCallbackPath = "/oauth/callback"

	req := httptest.NewRequest(http.MethodGet, "/api/auth/basaltpass/callback/?code=code-123&state=state-123", nil)
	req.AddCookie(&http.Cookie{Name: basaltStateCookie, Value: "state-123"})
	req.AddCookie(&http.Cookie{Name: basaltVerifierCookie, Value: "verifier-123"})
	req.AddCookie(&http.Cookie{Name: basaltNextCookie, Value: "/aprons/workplaces"})

	resp, err := app.http.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}

	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	if got := location.Scheme + "://" + location.Host + location.Path; got != "http://localhost:5109/oauth/callback" {
		t.Fatalf("unexpected frontend callback: %s", got)
	}
	if location.Query().Get("next") != "/aprons/workplaces" {
		t.Fatalf("unexpected next: %s", location.Query().Get("next"))
	}

	exchangeCode := location.Query().Get("code")
	if exchangeCode == "" {
		t.Fatal("missing exchange code in redirect")
	}
	if location.Query().Get("access") != "" {
		t.Fatal("redirect should not expose access token in query")
	}

	exchangeReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/basaltpass/exchange", strings.NewReader(`{"code":"`+exchangeCode+`"}`))
	exchangeReq.Header.Set("Content-Type", "application/json")
	exchangeResp, err := app.http.Test(exchangeReq, -1)
	if err != nil {
		t.Fatalf("exchange request failed: %v", err)
	}
	defer exchangeResp.Body.Close()
	if exchangeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from exchange, got %d", exchangeResp.StatusCode)
	}
	var exchangePayload map[string]any
	if err := json.NewDecoder(exchangeResp.Body).Decode(&exchangePayload); err != nil {
		t.Fatalf("decode exchange response: %v", err)
	}
	issuedToken, _ := exchangePayload["access"].(string)
	if issuedToken == "" {
		t.Fatal("missing access token in exchange response")
	}
	claims, err := app.parseToken(issuedToken)
	if err != nil {
		t.Fatalf("parse issued token: %v", err)
	}
	if claims.Role != "viewer" {
		t.Fatalf("unexpected role: %s", claims.Role)
	}
	if claims.Name != "Basalt Demo User" {
		t.Fatalf("unexpected name in issued token: %q", claims.Name)
	}
	if claims.Email != "demo.user@example.com" {
		t.Fatalf("unexpected email in issued token: %q", claims.Email)
	}
	if exchangePayload["next"] != "/aprons/workplaces" {
		t.Fatalf("unexpected exchange next: %v", exchangePayload["next"])
	}

	profileReq := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+claims.UserID, nil)
	profileReq.Header.Set("Authorization", "Bearer "+issuedToken)
	profileResp, err := app.http.Test(profileReq, -1)
	if err != nil {
		t.Fatalf("profile request failed: %v", err)
	}
	defer profileResp.Body.Close()
	if profileResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from profile api, got %d", profileResp.StatusCode)
	}
	var profilePayload map[string]any
	if err := json.NewDecoder(profileResp.Body).Decode(&profilePayload); err != nil {
		t.Fatalf("decode profile response: %v", err)
	}
	if got, _ := profilePayload["name"].(string); got != "Basalt Demo User" {
		t.Fatalf("unexpected profile name: %q", got)
	}
	if got, _ := profilePayload["email"].(string); got != "demo.user@example.com" {
		t.Fatalf("unexpected profile email: %q", got)
	}

	var userCount int64
	if err := app.db.Model(&struct{}{}).Table("users").Where("id = ?", claims.UserID).Count(&userCount).Error; err != nil {
		t.Fatalf("count created user: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("expected created oauth user, got count=%d", userCount)
	}

	for _, cookie := range resp.Cookies() {
		if strings.HasPrefix(cookie.Name, "araneae_basalt_") && cookie.Value == "" && cookie.Expires.IsZero() {
			t.Fatalf("expected oauth cookie %s to be cleared explicitly", cookie.Name)
		}
	}

	replayReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/basaltpass/exchange", strings.NewReader(`{"code":"`+exchangeCode+`"}`))
	replayReq.Header.Set("Content-Type", "application/json")
	replayResp, err := app.http.Test(replayReq, -1)
	if err != nil {
		t.Fatalf("replay exchange request failed: %v", err)
	}
	defer replayResp.Body.Close()
	if replayResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for replayed exchange code, got %d", replayResp.StatusCode)
	}
}

func TestBasaltPassCallbackAcceptsWrappedAccessTokenPayload(t *testing.T) {
	fakeBasalt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/oauth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"access_token": "bp-access-token"},
			})
		case "/api/v1/oauth/userinfo":
			if got := r.Header.Get("Authorization"); got != "Bearer bp-access-token" {
				t.Fatalf("unexpected authorization header: %s", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"sub": "user-subject-2"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer fakeBasalt.Close()

	app := newTestControlApp(t)
	app.cfg.BasaltOAuthEnabled = true
	app.cfg.BasaltBaseURL = fakeBasalt.URL
	app.cfg.BasaltInternalBaseURL = fakeBasalt.URL
	app.cfg.BasaltClientID = "client-123"
	app.cfg.BasaltClientSecret = "secret-123"
	app.cfg.BasaltRedirectURI = "http://localhost:8180/api/auth/basaltpass/callback/"
	app.cfg.BasaltScope = "openid profile email"
	app.cfg.FrontendBaseURL = "http://localhost:5109"
	app.cfg.BasaltCallbackPath = "/oauth/callback"

	req := httptest.NewRequest(http.MethodGet, "/api/auth/basaltpass/callback/?code=code-456&state=state-456", nil)
	req.AddCookie(&http.Cookie{Name: basaltStateCookie, Value: "state-456"})
	req.AddCookie(&http.Cookie{Name: basaltVerifierCookie, Value: "verifier-456"})
	req.AddCookie(&http.Cookie{Name: basaltNextCookie, Value: "/aprons/workplaces"})

	resp, err := app.http.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}

	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	if got := location.Scheme + "://" + location.Host + location.Path; got != "http://localhost:5109/oauth/callback" {
		t.Fatalf("unexpected frontend callback: %s", got)
	}
	if location.Query().Get("code") == "" {
		t.Fatal("missing exchange code in redirect")
	}
}
