package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/turkushan490/turkushan-auth/internal/store"
)

// A helper with the "users" right can't take over someone who has rights they lack.
func TestUsersRightCantTakeOverMorePowerfulStaff(t *testing.T) {
	h, st := newTestServer(t)
	admin := signedIn(t, h, st, "boss", true)
	helper := signedIn(t, h, st, "helper", false)
	signedIn(t, h, st, "sam", false)
	signedIn(t, h, st, "joe", false)
	helperID, samID, joeID := userID(t, st, "helper"), userID(t, st, "sam"), userID(t, st, "joe")

	helpers := mkGroup(t, admin, "Helpers")
	admin.put(fmt.Sprintf("/api/admin/groups/%d/perms", helpers), map[string]any{"perms": []string{"users"}})
	addMember(t, admin, helpers, helperID, true)
	settings := mkGroup(t, admin, "Settings people")
	admin.put(fmt.Sprintf("/api/admin/groups/%d/perms", settings), map[string]any{"perms": []string{"settings"}})
	addMember(t, admin, settings, samID, true)

	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPut, fmt.Sprintf("/api/admin/users/%d/password", samID), map[string]string{"password": "Hacked!1"}},
		{http.MethodPut, fmt.Sprintf("/api/admin/users/%d/status", samID), map[string]string{"status": "blocked"}},
		{http.MethodDelete, fmt.Sprintf("/api/admin/users/%d", samID), nil},
	} {
		if rec, out := helper.do(tc.method, tc.path, tc.body, nil); rec.Code != http.StatusForbidden {
			t.Errorf("%s %s on a settings holder: %d %v", tc.method, tc.path, rec.Code, out)
		}
	}
	// A plain user is still fine.
	if code, out := helper.put(fmt.Sprintf("/api/admin/users/%d/password", joeID), map[string]string{"password": "Reset!pw1"}); code != http.StatusOK {
		t.Errorf("reset a plain user: %d %v", code, out)
	}
	// Admins can do it.
	if code, _ := admin.put(fmt.Sprintf("/api/admin/users/%d/password", samID), map[string]string{"password": "Reset!pw1"}); code != http.StatusOK {
		t.Errorf("admin resets a settings holder: %d", code)
	}
}

// The "settings" right can't touch mail or login apps, and a new mail host drops the saved password.
func TestMailSettingsAdminOnlyAndPasswordCleared(t *testing.T) {
	h, st := newTestServer(t)
	ctx := context.Background()
	admin := signedIn(t, h, st, "boss", true)
	helper := signedIn(t, h, st, "helper", false)
	g := mkGroup(t, admin, "Settings")
	admin.put(fmt.Sprintf("/api/admin/groups/%d/perms", g), map[string]any{"perms": []string{"settings"}})
	addMember(t, admin, g, userID(t, st, "helper"), true)

	if code, out := admin.put("/api/admin/settings", map[string]any{
		"smtp_host": "smtp.resend.com", "smtp_port": 465, "smtp_username": "resend", "smtp_password": "s3cret", "smtp_from": "noreply@example.com",
	}); code != http.StatusOK {
		t.Fatalf("admin saves mail: %d %v", code, out)
	}

	for _, body := range []map[string]any{
		{"smtp_host": "evil.example.net"},
		{"smtp_port": 2525},
		{"oauth": map[string]any{"google": map[string]any{"client_id": "x", "client_secret": "y"}}},
	} {
		if code, _ := helper.put("/api/admin/settings", body); code != http.StatusForbidden {
			t.Errorf("settings helper changed %v: %d", body, code)
		}
	}
	if pw, _ := st.Setting(ctx, store.SettingSMTPPassword); pw != "s3cret" {
		t.Fatalf("password changed by a refused request: %q", pw)
	}
	// Portal address is still theirs to change.
	if code, out := helper.put("/api/admin/settings", map[string]any{"portal_url": "http://192.168.0.6:3010"}); code != http.StatusOK {
		t.Errorf("settings helper sets portal address: %d %v", code, out)
	}

	// Same host, other fields: password kept.
	admin.put("/api/admin/settings", map[string]any{"smtp_host": "SMTP.resend.com", "smtp_from": "hi@example.com"})
	if pw, _ := st.Setting(ctx, store.SettingSMTPPassword); pw != "s3cret" {
		t.Errorf("password lost without a host change: %q", pw)
	}
	// New host without a new password: the old one is not sent there.
	admin.put("/api/admin/settings", map[string]any{"smtp_host": "smtp.other.com"})
	if pw, _ := st.Setting(ctx, store.SettingSMTPPassword); pw != "" {
		t.Errorf("password kept for a new host: %q", pw)
	}
	// New host with a new password: saved.
	admin.put("/api/admin/settings", map[string]any{"smtp_host": "smtp.third.com", "smtp_password": "n3w"})
	if pw, _ := st.Setting(ctx, store.SettingSMTPPassword); pw != "n3w" {
		t.Errorf("new password not saved: %q", pw)
	}
}

// Only admins can open a site to everyone; the "sites" right keeps approval on.
func TestSitesRightCantOpenSites(t *testing.T) {
	h, st := newTestServer(t)
	admin := signedIn(t, h, st, "boss", true)
	helper := signedIn(t, h, st, "helper", false)
	g := mkGroup(t, admin, "Sites")
	admin.put(fmt.Sprintf("/api/admin/groups/%d/perms", g), map[string]any{"perms": []string{"sites"}})
	addMember(t, admin, g, userID(t, st, "helper"), true)

	site := func(host string, approval bool) map[string]any {
		return map[string]any{"name": host, "host": host, "upstream": "http://192.168.0.6:3000", "require_approval": approval}
	}
	if code, _ := helper.post("/api/admin/sites", site("open.example.com", false)); code != http.StatusForbidden {
		t.Errorf("helper creates an open site: %d", code)
	}
	if code, out := helper.post("/api/admin/sites", site("closed.example.com", true)); code != http.StatusOK {
		t.Fatalf("helper creates an approval site: %d %v", code, out)
	}
	closed, _ := st.SiteByHost(context.Background(), "closed.example.com")
	if code, _ := helper.put(fmt.Sprintf("/api/admin/sites/%d", closed.ID), site("closed.example.com", false)); code != http.StatusForbidden {
		t.Errorf("helper opens a closed site: %d", code)
	}
	// A site the admin already opened can still be edited by the helper.
	openID := addSite(t, st, "already.example.com", false, false)
	if code, out := helper.put(fmt.Sprintf("/api/admin/sites/%d", openID), site("already.example.com", false)); code != http.StatusOK {
		t.Errorf("helper edits an already-open site: %d %v", code, out)
	}
	if code, _ := admin.post("/api/admin/sites", site("admin-open.example.com", false)); code != http.StatusOK {
		t.Errorf("admin creates an open site: %d", code)
	}
}

// The NPM snippet must use $host, never the raw Host header, and pass the identity on.
func TestSnippetUsesMatchedHost(t *testing.T) {
	sn := nginxSnippet("http://192.168.0.6:3000", "http://192.168.0.6:3010")
	if strings.Contains(sn, "$http_host") {
		t.Error("snippet still uses $http_host")
	}
	for _, want := range []string{
		"x-original-url $scheme://$host$request_uri",
		"proxy_set_header Remote-User $auth_user",
		"proxy_set_header Remote-Email $auth_email",
		"proxy_set_header Remote-Groups $auth_groups",
		"proxy_set_header Host $host",
	} {
		if !strings.Contains(sn, want) {
			t.Errorf("snippet lacks %q", want)
		}
	}
}

func TestForwardAuthSendsGroups(t *testing.T) {
	h, st := newTestServer(t)
	siteID := addSite(t, st, "manga.example.com", true, false)
	admin := signedIn(t, h, st, "boss", true)
	ann := signedIn(t, h, st, "ann", false)
	for _, name := range []string{"Readers", "Friends"} {
		g := mkGroup(t, admin, name)
		admin.put(fmt.Sprintf("/api/admin/groups/%d/sites", g), map[string]any{"site_id": siteID, "can_open": true})
		addMember(t, admin, g, userID(t, st, "ann"), true)
	}
	res := ann.forward(http.MethodGet, "https://manga.example.com/")
	if res.StatusCode != http.StatusOK || res.Header.Get("X-Auth-User") != "ann" || res.Header.Get("X-Auth-Groups") != "Friends,Readers" {
		t.Errorf("forward auth: %d user=%q groups=%q", res.StatusCode, res.Header.Get("X-Auth-User"), res.Header.Get("X-Auth-Groups"))
	}
}

func TestOAuthPKCEAndSignedState(t *testing.T) {
	s, h, st := newTestServerFull(t)
	f := withOAuth(t, s, st)
	f.set(func() {
		f.google = map[string]any{"sub": "g-1", "email": "p@example.com", "email_verified": true, "name": "Pia"}
	})

	c := newClient(t, h)
	rec, _ := c.do(http.MethodGet, "/api/oauth/google/start", nil, nil)
	authURL, _ := url.Parse(rec.Header().Get("Location"))
	q := authURL.Query()
	if q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" {
		t.Fatalf("no PKCE in authorize URL: %s", authURL)
	}
	if _, ok := c.cookies["__Host-"+oauthCookie]; !ok {
		t.Fatalf("state cookie not __Host- prefixed: %v", c.cookies)
	}
	c.do(http.MethodGet, "/api/oauth/google/callback?code=good&state="+url.QueryEscape(q.Get("state")), nil, nil)
	f.mu.Lock()
	verifier := f.verifier
	f.mu.Unlock()
	if verifier == "" || pkceChallenge(verifier) != q.Get("code_challenge") {
		t.Errorf("token request verifier %q doesn't match challenge %q", verifier, q.Get("code_challenge"))
	}

	// A state cookie this portal didn't sign is refused, even with a matching state.
	c = newClient(t, h)
	c.cookies["__Host-"+oauthCookie] = "eyJzIjoieCIsInAiOiJnb29nbGUiLCJ2IjoieCIsImUiOjk5OTk5OTk5OTl9.forged"
	if rec, _ := c.do(http.MethodGet, "/api/oauth/google/callback?code=good&state=x", nil, nil); rec.Header().Get("Location") != "/login?oauth_error=state" {
		t.Errorf("forged state cookie: %q", rec.Header().Get("Location"))
	}
}

// Discord/Google never sign someone straight into a staff account by email.
func TestOAuthDoesNotAutoLinkStaff(t *testing.T) {
	s, h, st := newTestServerFull(t)
	srv := withMail(t, st)
	f := withOAuth(t, s, st)
	owner := newClient(t, h)
	owner.post("/api/register", map[string]string{"username": "chief", "password": "Secret!1", "email": "chief@example.com"})
	owner.post("/api/verify-email", map[string]string{"token": linkToken(t, srv, "chief@example.com", "/verify")})
	st.DB.Exec(`UPDATE users SET is_admin = 1 WHERE username = 'chief'`)

	f.set(func() {
		f.google = map[string]any{"sub": "g-x", "email": "chief@example.com", "email_verified": true, "name": "Chief"}
	})
	c := newClient(t, h)
	if loc := c.oauthLogin("google", ""); loc != "/login?oauth_error=link_first" {
		t.Errorf("staff auto-link: %q", loc)
	}
	if c.session()["authenticated"] != false {
		t.Error("signed in to a staff account through Google")
	}
	// Connecting it from the account page still works.
	if loc := owner.oauthLogin("google", "link=1"); loc != "/?connected=google" {
		t.Errorf("staff links from account page: %q", loc)
	}
}

// Double-submitting the welcome step makes one account.
func TestOAuthCompleteOnlyOnce(t *testing.T) {
	s, h, st := newTestServerFull(t)
	f := withOAuth(t, s, st)
	f.set(func() {
		f.google = map[string]any{"sub": "g-d", "email": "d@example.com", "email_verified": true, "name": "Dana"}
	})
	c := newClient(t, h)
	token := pendingToken(t, c.oauthLogin("google", ""))

	// A username clash after the claim puts the sign-up back.
	signedIn(t, h, st, "taken", false)
	if code, _ := c.post("/api/oauth/complete", map[string]string{"p": token, "username": "taken"}); code != http.StatusConflict {
		t.Fatalf("taken username: %d", code)
	}
	if code, out := c.post("/api/oauth/complete", map[string]string{"p": token, "username": "dana"}); code != http.StatusOK {
		t.Fatalf("complete: %d %v", code, out)
	}
	if code, _ := c.post("/api/oauth/complete", map[string]string{"p": token, "username": "dana2"}); code == http.StatusOK {
		t.Error("second submit made another account")
	}
	var n int
	st.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE username LIKE 'dana%'`).Scan(&n)
	if n != 1 {
		t.Errorf("accounts made: %d", n)
	}
}

// Users who only sign in with Discord/Google can change a verified address.
func TestOAuthOnlyUserChangesEmail(t *testing.T) {
	s, h, st := newTestServerFull(t)
	withMail(t, st)
	f := withOAuth(t, s, st)
	f.set(func() {
		f.google = map[string]any{"sub": "g-e", "email": "e@example.com", "email_verified": true, "name": "Eli"}
	})
	c := newClient(t, h)
	c.post("/api/oauth/complete", map[string]string{"p": pendingToken(t, c.oauthLogin("google", ""))})
	if code, out := c.post("/api/account/email", map[string]string{"email": "eli@example.com"}); code != http.StatusOK {
		t.Errorf("change email without a password set: %d %v", code, out)
	}
}
