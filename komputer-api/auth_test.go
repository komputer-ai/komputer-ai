package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newAuthTestRouter(token string, origins []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(corsMiddleware(origins))
	r.Use(authMiddleware(token))
	ok := func(c *gin.Context) { c.String(http.StatusOK, "ok") }
	r.GET("/healthz", ok)
	r.GET("/readyz", ok)
	r.GET("/api/metrics", ok)
	r.GET("/api/v1/oauth/callback", ok)
	r.GET("/api/v1/agents", ok)
	r.GET("/api/v1/agents/:name/ws", ok)
	return r
}

func doReq(r http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthDisabledWhenTokenEmpty(t *testing.T) {
	r := newAuthTestRouter("", nil)
	w := doReq(r, httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 with auth off", w.Code)
	}
}

func TestAuthRejectsMissingAndWrongToken(t *testing.T) {
	r := newAuthTestRouter("s3cret", nil)
	for name, header := range map[string]string{
		"missing":   "",
		"wrong":     "Bearer nope",
		"no scheme": "s3cret",
		"basic":     "Basic s3cret",
	} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		w := doReq(r, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401", name, w.Code)
		}
		if w.Header().Get("WWW-Authenticate") == "" {
			t.Errorf("%s: missing WWW-Authenticate header", name)
		}
	}
}

func TestAuthAcceptsBearerToken(t *testing.T) {
	r := newAuthTestRouter("s3cret", nil)
	for _, h := range []string{"Bearer s3cret", "bearer s3cret"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
		req.Header.Set("Authorization", h)
		if w := doReq(r, req); w.Code != http.StatusOK {
			t.Errorf("%q: status = %d, want 200", h, w.Code)
		}
	}
}

func TestAuthExemptPaths(t *testing.T) {
	r := newAuthTestRouter("s3cret", nil)
	for _, p := range []string{"/healthz", "/readyz", "/api/metrics", "/api/v1/oauth/callback"} {
		if w := doReq(r, httptest.NewRequest(http.MethodGet, p, nil)); w.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200 without a token", p, w.Code)
		}
	}
}

func TestAuthQueryTokenOnlyForWebSocket(t *testing.T) {
	r := newAuthTestRouter("s3cret", nil)

	ws := httptest.NewRequest(http.MethodGet, "/api/v1/agents/a/ws?access_token=s3cret", nil)
	ws.Header.Set("Upgrade", "websocket")
	if w := doReq(r, ws); w.Code != http.StatusOK {
		t.Errorf("websocket with query token: status = %d, want 200", w.Code)
	}

	plain := httptest.NewRequest(http.MethodGet, "/api/v1/agents?access_token=s3cret", nil)
	if w := doReq(r, plain); w.Code != http.StatusUnauthorized {
		t.Errorf("plain request with query token: status = %d, want 401", w.Code)
	}
}

func TestAuthAllowsPreflightAndInProcess(t *testing.T) {
	r := newAuthTestRouter("s3cret", nil)
	if w := doReq(r, httptest.NewRequest(http.MethodOptions, "/api/v1/agents", nil)); w.Code != http.StatusNoContent {
		t.Errorf("preflight: status = %d, want 204", w.Code)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req = req.WithContext(withInProcess(req.Context()))
	if w := doReq(r, req); w.Code != http.StatusOK {
		t.Errorf("in-process: status = %d, want 200", w.Code)
	}
}

func TestCORSAllowList(t *testing.T) {
	r := newAuthTestRouter("", []string{"https://ui.example.com"})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "https://ui.example.com")
	if got := doReq(r, req).Header().Get("Access-Control-Allow-Origin"); got != "https://ui.example.com" {
		t.Errorf("allowed origin: ACAO = %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	if got := doReq(r, req).Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("other origin: ACAO = %q, want none", got)
	}
}

func TestCORSDefaultAllowsAll(t *testing.T) {
	r := newAuthTestRouter("", nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "https://anything.example.com")
	if got := doReq(r, req).Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("default ACAO = %q, want *", got)
	}
}

func TestCORSOriginsFromEnv(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://a.example.com , *, https://b.example.com ")
	got := corsOriginsFromEnv()
	if len(got) != 2 || got[0] != "https://a.example.com" || got[1] != "https://b.example.com" {
		t.Errorf("corsOriginsFromEnv = %v", got)
	}
}
