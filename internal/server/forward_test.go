package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/turkushan490/turkushan-auth/internal/auth"
	"github.com/turkushan490/turkushan-auth/internal/store"
)

const testPassword = "Secret!1"

// signedIn creates a user and returns a client logged in as them.
func signedIn(t *testing.T, h http.Handler, st *store.Store, username string, admin bool) *client {
	t.Helper()
	hash, _ := auth.HashPassword(testPassword)
	if _, err := st.CreateUser(context.Background(), username, hash, admin); err != nil {
		t.Fatal(err)
	}
	c := newClient(t, h)
	if code, out := c.post("/api/login", map[string]string{"username": username, "password": testPassword}); code != http.StatusOK {
		t.Fatalf("login %s: %d %v", username, code, out)
	}
	return c
}

// forward simulates NPM's auth_request subrequest for url.
func (c *client) forward(method, orig string) *http.Response {
	c.t.Helper()
	rec, _ := c.do(method, "/api/auth/nginx", nil, map[string]string{"X-Original-URL": orig, csrfHeader: ""})
	return rec.Result()
}

func addSite(t *testing.T, st *store.Store, host string, approval, email bool) int64 {
	t.Helper()
	id, err := st.CreateSite(context.Background(), &store.Site{
		Name: host, Host: host, Upstream: "http://192.168.0.6:3000", RequireApproval: approval, RequireEmail: email,
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestForwardAuth(t *testing.T) {
	h, st := newTestServer(t)
	mangaID := addSite(t, st, "manga.example.com", true, false)
	addSite(t, st, "open.example.com", false, false)
	addSite(t, st, "mail.example.com", false, true)

	const orig = "https://manga.example.com/reader?id=5"

	// Not signed in: 401 and a login link that brings the visitor back.
	anon := newClient(t, h)
	res := anon.forward(http.MethodGet, orig)
	wantLogin := "https://auth.example.com/login?rd=" + url.QueryEscape(orig)
	if res.StatusCode != http.StatusUnauthorized || res.Header.Get("X-Auth-Location") != wantLogin {
		t.Fatalf("anonymous: %d %q", res.StatusCode, res.Header.Get("X-Auth-Location"))
	}

	// Broken NPM config.
	if res := anon.forward(http.MethodGet, ""); res.StatusCode != http.StatusBadRequest {
		t.Errorf("missing header: %d", res.StatusCode)
	}

	alice := signedIn(t, h, st, "alice", false)

	// First visit to an approval site: 403 to /pending and a pending request is filed.
	res = alice.forward(http.MethodGet, orig)
	loc := res.Header.Get("X-Auth-Location")
	if res.StatusCode != http.StatusForbidden || !strings.HasPrefix(loc, "https://auth.example.com/pending?") ||
		!strings.Contains(loc, "site=manga.example.com") {
		t.Fatalf("first visit: %d %q", res.StatusCode, loc)
	}
	if n, _ := st.CountPending(context.Background()); n != 1 {
		t.Fatalf("pending requests: %d", n)
	}

	// /api/access tells the pending page the same thing.
	_, out := alice.do(http.MethodGet, "/api/access?site=manga.example.com", nil, nil)
	if out["status"] != accessPending {
		t.Errorf("access status: %v", out)
	}

	// Admin approves → alice gets in, with her username passed along.
	admin := signedIn(t, h, st, "boss", true)
	u, _ := st.UserByUsername(context.Background(), "alice")
	if code, out := admin.put("/api/admin/access", map[string]any{"user_id": u.ID, "site_id": mangaID, "status": "approved"}); code != http.StatusOK {
		t.Fatalf("approve: %d %v", code, out)
	}
	res = alice.forward(http.MethodGet, orig)
	if res.StatusCode != http.StatusOK || res.Header.Get("X-Auth-User") != "alice" {
		t.Fatalf("approved: %d user=%q", res.StatusCode, res.Header.Get("X-Auth-User"))
	}

	// nginx keeps the original method for the subrequest; POSTs must not hit CSRF.
	if res := alice.forward(http.MethodPost, orig); res.StatusCode != http.StatusOK {
		t.Errorf("POST subrequest: %d", res.StatusCode)
	}

	// Revoked → 403 again.
	admin.put("/api/admin/access", map[string]any{"user_id": u.ID, "site_id": mangaID, "status": "denied"})
	if res := alice.forward(http.MethodGet, orig); res.StatusCode != http.StatusForbidden {
		t.Errorf("denied: %d", res.StatusCode)
	}

	// Open site: any signed-in user.
	if res := alice.forward(http.MethodGet, "https://open.example.com/"); res.StatusCode != http.StatusOK {
		t.Errorf("open site: %d", res.StatusCode)
	}
	// Email-only site without a verified email.
	if res := alice.forward(http.MethodGet, "https://mail.example.com/"); res.StatusCode != http.StatusForbidden {
		t.Errorf("email site: %d", res.StatusCode)
	}
	_, out = alice.do(http.MethodGet, "/api/access?site=mail.example.com", nil, nil)
	if out["status"] != accessEmailRequired {
		t.Errorf("email site status: %v", out)
	}
	// Unknown site: denied by default, even for a signed-in user.
	if res := alice.forward(http.MethodGet, "https://secret.example.com/"); res.StatusCode != http.StatusForbidden {
		t.Errorf("unknown site: %d", res.StatusCode)
	}
	// Admins get into every registered site.
	if res := admin.forward(http.MethodGet, orig); res.StatusCode != http.StatusOK {
		t.Errorf("admin: %d", res.StatusCode)
	}

	// Signing out on the portal ends access on the site.
	alice.post("/api/logout", map[string]string{})
	if res := alice.forward(http.MethodGet, "https://open.example.com/"); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("after logout: %d", res.StatusCode)
	}
}

func (c *client) put(path string, body any) (int, map[string]any) {
	c.t.Helper()
	rec, out := c.do(http.MethodPut, path, body, nil)
	return rec.Code, out
}

func (c *client) del(path string) (int, map[string]any) {
	c.t.Helper()
	rec, out := c.do(http.MethodDelete, path, nil, nil)
	return rec.Code, out
}

func TestAdminAPI(t *testing.T) {
	h, st := newTestServer(t)
	ctx := context.Background()

	// Only admins get in.
	if rec, _ := newClient(t, h).do(http.MethodGet, "/api/admin/data", nil, nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous: %d", rec.Code)
	}
	bob := signedIn(t, h, st, "bob", false)
	if rec, _ := bob.do(http.MethodGet, "/api/admin/data", nil, nil); rec.Code != http.StatusForbidden {
		t.Errorf("non-admin: %d", rec.Code)
	}

	admin := signedIn(t, h, st, "boss", true)
	site := map[string]any{"name": "Manga", "host": "https://Manga.example.com/", "upstream": "http://192.168.0.6:3000", "require_approval": true}
	if code, out := admin.post("/api/admin/sites", site); code != http.StatusOK {
		t.Fatalf("create site: %d %v", code, out)
	}
	if code, _ := admin.post("/api/admin/sites", site); code != http.StatusConflict {
		t.Errorf("duplicate site: %d", code)
	}
	for _, bad := range []map[string]any{
		{"name": "Evil", "host": "evil.com", "upstream": "http://1.2.3.4"},
		{"name": "Portal", "host": "auth.example.com", "upstream": "http://1.2.3.4"},
		{"name": "Inject", "host": "x.example.com", "upstream": "http://1.2.3.4; return 200"},
		{"name": "Path", "host": "y.example.com", "upstream": "http://1.2.3.4/path"},
		{"name": "", "host": "z.example.com", "upstream": "http://1.2.3.4"},
	} {
		if code, _ := admin.post("/api/admin/sites", bad); code != http.StatusBadRequest {
			t.Errorf("site %v: %d", bad, code)
		}
	}

	// The snippet uses the site's upstream; the portal address comes from settings.
	_, data := admin.do(http.MethodGet, "/api/admin/data", nil, nil)
	sites := data["sites"].([]any)
	snip := sites[0].(map[string]any)["snippet"].(string)
	if sites[0].(map[string]any)["host"] != "manga.example.com" ||
		!strings.Contains(snip, "proxy_pass http://192.168.0.6:3000;") || !strings.Contains(snip, "YOUR-SERVER-IP") {
		t.Fatalf("site/snippet: %v", sites[0])
	}
	if code, _ := admin.put("/api/admin/settings", map[string]any{"portal_url": "http://192.168.0.6:3010/"}); code != http.StatusOK {
		t.Fatalf("portal url: %d", code)
	}
	_, data = admin.do(http.MethodGet, "/api/admin/data", nil, nil)
	snip = data["sites"].([]any)[0].(map[string]any)["snippet"].(string)
	if !strings.Contains(snip, "proxy_pass http://192.168.0.6:3010/api/auth/nginx;") {
		t.Errorf("snippet without portal url:\n%s", snip)
	}

	// Settings validation.
	if code, _ := admin.put("/api/admin/settings", map[string]any{"discord_webhook": "https://evil.com/hook"}); code != http.StatusBadRequest {
		t.Errorf("bad webhook: %d", code)
	}
	if code, _ := admin.put("/api/admin/settings", map[string]any{"portal_url": "http://x;y"}); code != http.StatusBadRequest {
		t.Errorf("bad portal url: %d", code)
	}

	// User actions.
	bobUser, _ := st.UserByUsername(ctx, "bob")
	bossUser, _ := st.UserByUsername(ctx, "boss")
	bobPath := fmt.Sprintf("/api/admin/users/%d", bobUser.ID)

	if code, _ := admin.put(bobPath+"/status", map[string]string{"status": "blocked"}); code != http.StatusOK {
		t.Fatalf("block: %d", code)
	}
	if s := bob.session(); s["authenticated"] != false {
		t.Error("blocked user still signed in")
	}
	admin.put(bobPath+"/status", map[string]string{"status": "active"})

	if code, _ := admin.put(bobPath+"/password", map[string]string{"password": "weak"}); code != http.StatusBadRequest {
		t.Errorf("weak reset password: %d", code)
	}
	if code, _ := admin.put(bobPath+"/password", map[string]string{"password": "Nieuw!pass1"}); code != http.StatusOK {
		t.Fatalf("reset password: %d", code)
	}
	if code, _ := newClient(t, h).post("/api/login", map[string]string{"username": "bob", "password": "Nieuw!pass1"}); code != http.StatusOK {
		t.Errorf("login with reset password: %d", code)
	}

	bossPath := fmt.Sprintf("/api/admin/users/%d", bossUser.ID)
	if code, _ := admin.put(bossPath+"/status", map[string]string{"status": "blocked"}); code != http.StatusForbidden {
		t.Errorf("block self: %d", code)
	}
	if code, _ := admin.del(bossPath); code != http.StatusForbidden {
		t.Errorf("delete self: %d", code)
	}
	if code, _ := admin.del(bobPath); code != http.StatusOK {
		t.Errorf("delete: %d", code)
	}
	if code, _ := admin.del(bobPath); code != http.StatusNotFound {
		t.Errorf("delete again: %d", code)
	}

	// Every action is in the audit log.
	_, audit := admin.do(http.MethodGet, "/api/admin/audit", nil, nil)
	actions := map[string]bool{}
	for _, e := range audit["entries"].([]any) {
		actions[e.(map[string]any)["action"].(string)] = true
	}
	for _, want := range []string{"site.create", "settings.portal_url", "user.block", "user.unblock", "user.password_reset", "user.delete"} {
		if !actions[want] {
			t.Errorf("audit log misses %s (have %v)", want, actions)
		}
	}
}
