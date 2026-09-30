package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/turkushan490/turkushan-auth/internal/config"
	"github.com/turkushan490/turkushan-auth/internal/db"
	"github.com/turkushan490/turkushan-auth/internal/store"
)

func newTestServer(t *testing.T) (http.Handler, *store.Store) {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if err := db.Migrate(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	dist := fstest.MapFS{
		"index.html":      {Data: []byte("<!doctype html><title>spa</title>")},
		"assets/app-1.js": {Data: []byte("console.log(1)")},
		"favicon.svg":     {Data: []byte("<svg/>")},
	}
	appURL, _ := url.Parse("https://auth.example.com")
	cfg := &config.Config{AppURL: appURL, CookieDomain: ".example.com"}
	st := store.New(d)
	h, err := New(cfg, st, slog.New(slog.NewTextHandler(io.Discard, nil)), dist)
	if err != nil {
		t.Fatal(err)
	}
	return h, st
}

// client keeps cookies between requests and sends the CSRF header like the frontend.
type client struct {
	t       *testing.T
	h       http.Handler
	cookies map[string]string
	last    []*http.Cookie
}

func newClient(t *testing.T, h http.Handler) *client {
	c := &client{t: t, h: h, cookies: map[string]string{}}
	c.do(http.MethodGet, "/api/session", nil, nil) // picks up the CSRF cookie
	return c
}

func (c *client) do(method, path string, body any, headers map[string]string) (*httptest.ResponseRecorder, map[string]any) {
	c.t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet {
		req.Header.Set(csrfHeader, c.cookies[csrfCookie])
	}
	for k, v := range c.cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	for k, v := range headers {
		if v == "" {
			req.Header.Del(k)
		} else {
			req.Header.Set(k, v)
		}
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)

	c.last = rec.Result().Cookies()
	for _, ck := range c.last {
		if ck.MaxAge < 0 || ck.Value == "" {
			delete(c.cookies, ck.Name)
		} else {
			c.cookies[ck.Name] = ck.Value
		}
	}
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	return rec, out
}

func (c *client) post(path string, body any) (int, map[string]any) {
	c.t.Helper()
	rec, out := c.do(http.MethodPost, path, body, nil)
	return rec.Code, out
}

func (c *client) session() map[string]any {
	c.t.Helper()
	_, out := c.do(http.MethodGet, "/api/session", nil, nil)
	return out
}

func TestHealthz(t *testing.T) {
	h, _ := newTestServer(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestSPA(t *testing.T) {
	h, _ := newTestServer(t)
	get := func(method, p string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(method, p, nil))
		return rec
	}

	for _, p := range []string{"/", "/login", "/admin/users", "/index.html"} {
		rec := get(http.MethodGet, p)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<title>spa</title>") {
			t.Errorf("%s: got %d %q", p, rec.Code, rec.Body.String())
		}
	}

	rec := get(http.MethodGet, "/assets/app-1.js")
	if rec.Code != http.StatusOK || rec.Body.String() != "console.log(1)" {
		t.Errorf("asset: got %d %q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("asset not cached: %q", rec.Header().Get("Cache-Control"))
	}

	rec = get(http.MethodGet, "/api/nope")
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Header().Get("Content-Type"), "json") {
		t.Errorf("unknown api: got %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	if rec := get(http.MethodPost, "/login"); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST on page: got %d", rec.Code)
	}
	for _, hdr := range []string{"Content-Security-Policy", "X-Frame-Options", "X-Content-Type-Options", "Strict-Transport-Security", "Referrer-Policy"} {
		if get(http.MethodGet, "/").Header().Get(hdr) == "" {
			t.Errorf("missing %s", hdr)
		}
	}
}

func TestRegisterLoginLogout(t *testing.T) {
	h, _ := newTestServer(t)
	c := newClient(t, h)

	if c.cookies[csrfCookie] == "" {
		t.Fatal("no CSRF cookie from /api/session")
	}
	if s := c.session(); s["authenticated"] != false || s["brand"] != "example.com" {
		t.Fatalf("fresh session: %v", s)
	}

	code, out := c.post("/api/register", map[string]string{
		"username": " Alice ", "password": "Secret!1", "rd": "https://lrr.example.com/archive?id=7",
	})
	if code != http.StatusOK || out["redirect"] != "https://lrr.example.com/archive?id=7" {
		t.Fatalf("register: %d %v", code, out)
	}

	var sc *http.Cookie
	for _, ck := range c.last {
		if ck.Name == sessionCookie {
			sc = ck
		}
	}
	if sc == nil {
		t.Fatal("no session cookie after register")
	}
	if !sc.HttpOnly || !sc.Secure || sc.SameSite != http.SameSiteLaxMode || sc.Domain != "example.com" || sc.Path != "/" {
		t.Errorf("session cookie attributes: %+v", sc)
	}

	s := c.session()
	user, _ := s["user"].(map[string]any)
	if s["authenticated"] != true || user["username"] != "alice" || user["is_admin"] != false {
		t.Fatalf("after register: %v", s)
	}

	if code, _ := c.post("/api/logout", map[string]string{}); code != http.StatusOK {
		t.Fatalf("logout: %d", code)
	}
	if s := c.session(); s["authenticated"] != false {
		t.Fatalf("after logout: %v", s)
	}

	// Wrong password, then right password with a foreign rd: falls back to the portal.
	if code, out := c.post("/api/login", map[string]string{"username": "alice", "password": "Wrong!123"}); code != http.StatusUnauthorized || out["error"] != msgBadLogin {
		t.Fatalf("wrong password: %d %v", code, out)
	}
	code, out = c.post("/api/login", map[string]string{"username": "ALICE", "password": "Secret!1", "rd": "https://evil.com/"})
	if code != http.StatusOK || out["redirect"] != "https://auth.example.com/" {
		t.Fatalf("login: %d %v", code, out)
	}
	if s := c.session(); s["authenticated"] != true {
		t.Fatalf("after login: %v", s)
	}
}

func TestLogoutKillsSessionEverywhere(t *testing.T) {
	h, _ := newTestServer(t)
	a := newClient(t, h)
	a.post("/api/register", map[string]string{"username": "dave", "password": "Secret!1"})
	stolen := a.cookies[sessionCookie]

	a.post("/api/logout", map[string]string{})

	// Reusing the old cookie value after logout must not work.
	b := newClient(t, h)
	b.cookies[sessionCookie] = stolen
	if s := b.session(); s["authenticated"] != false {
		t.Fatal("session still valid after logout")
	}
}

func TestRegisterValidation(t *testing.T) {
	h, _ := newTestServer(t)
	c := newClient(t, h)

	cases := []struct {
		body map[string]string
		code int
	}{
		{map[string]string{"username": "ab", "password": "Secret!1"}, http.StatusBadRequest},
		{map[string]string{"username": "erin", "password": "secret!1"}, http.StatusBadRequest},  // no capital
		{map[string]string{"username": "erin", "password": "Secret11"}, http.StatusBadRequest},  // no symbol
		{map[string]string{"username": "erin", "password": "Se!1"}, http.StatusBadRequest},      // too short
		{map[string]string{"username": "erin", "password": "Secret!1", "email": "nope"}, http.StatusBadRequest},
		{map[string]string{"username": "erin", "password": "Secret!1", "email": "Erin@Example.com"}, http.StatusOK},
		{map[string]string{"username": "Erin", "password": "Secret!1"}, http.StatusConflict},
		// erin hasn't verified the address yet, so it doesn't block someone else.
		{map[string]string{"username": "frank", "password": "Secret!1", "email": "erin@example.com"}, http.StatusOK},
	}
	for i, tc := range cases {
		code, out := c.post("/api/register", tc.body)
		if code != tc.code {
			t.Errorf("case %d %v: got %d %v, want %d", i, tc.body, code, out, tc.code)
		}
		if code != http.StatusOK && out["error"] == nil {
			t.Errorf("case %d: no error message", i)
		}
	}
}

func TestLockout(t *testing.T) {
	h, _ := newTestServer(t)
	c := newClient(t, h)
	c.post("/api/register", map[string]string{"username": "bob", "password": "Secret!1"})
	c.post("/api/logout", map[string]string{})

	for i := 0; i < store.MaxFailedLogins; i++ {
		if code, _ := c.post("/api/login", map[string]string{"username": "bob", "password": "Nope!123"}); code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: %d", i+1, code)
		}
	}
	if code, _ := c.post("/api/login", map[string]string{"username": "bob", "password": "Secret!1"}); code != http.StatusLocked {
		t.Fatalf("locked account accepted the right password: %d", code)
	}

	// Unknown users get the same answer as a wrong password.
	if code, out := c.post("/api/login", map[string]string{"username": "nobody", "password": "Secret!1"}); code != http.StatusUnauthorized || out["error"] != msgBadLogin {
		t.Fatalf("unknown user: %d %v", code, out)
	}
}

func TestLoginRateLimit(t *testing.T) {
	h, _ := newTestServer(t)
	c := newClient(t, h)
	var last int
	for i := 0; i < 25; i++ {
		// A different username each time, so only the per-IP limit can trigger.
		last, _ = c.post("/api/login", map[string]string{"username": fmt.Sprintf("guess%d", i), "password": "Nope!123"})
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("25 logins from one IP: last status %d, want 429", last)
	}
}

func TestRegisterRateLimit(t *testing.T) {
	h, _ := newTestServer(t)
	c := newClient(t, h)
	var last int
	for i := 0; i < 7; i++ {
		last, _ = c.post("/api/register", map[string]string{"username": fmt.Sprintf("bot%d", i), "password": "Secret!1"})
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("7 registrations from one IP: last status %d, want 429", last)
	}
}

func TestBlockedUserCannotLogin(t *testing.T) {
	h, st := newTestServer(t)
	c := newClient(t, h)
	c.post("/api/register", map[string]string{"username": "gina", "password": "Secret!1"})
	st.DB.Exec(`UPDATE users SET status = 'blocked' WHERE username = 'gina'`)

	if s := c.session(); s["authenticated"] != false {
		t.Fatal("blocked user still has a valid session")
	}
	if code, _ := c.post("/api/login", map[string]string{"username": "gina", "password": "Secret!1"}); code != http.StatusForbidden {
		t.Fatalf("blocked login: %d", code)
	}
}

func TestCSRF(t *testing.T) {
	h, _ := newTestServer(t)
	c := newClient(t, h)
	body := map[string]string{"username": "henk", "password": "Secret!1"}

	rec, _ := c.do(http.MethodPost, "/api/register", body, map[string]string{csrfHeader: ""})
	if rec.Code != http.StatusForbidden {
		t.Errorf("missing CSRF header: %d", rec.Code)
	}
	rec, _ = c.do(http.MethodPost, "/api/register", body, map[string]string{csrfHeader: "forged"})
	if rec.Code != http.StatusForbidden {
		t.Errorf("wrong CSRF header: %d", rec.Code)
	}
	rec, _ = c.do(http.MethodPost, "/api/register", body, map[string]string{"Origin": "https://evil.com"})
	if rec.Code != http.StatusForbidden {
		t.Errorf("foreign origin: %d", rec.Code)
	}
	rec, _ = c.do(http.MethodPost, "/api/register", body, map[string]string{"Origin": "https://auth.example.com"})
	if rec.Code != http.StatusOK {
		t.Errorf("own origin: %d %s", rec.Code, rec.Body.String())
	}
}
