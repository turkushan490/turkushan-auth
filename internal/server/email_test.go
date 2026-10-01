package server

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/turkushan490/turkushan-auth/internal/mail/mailtest"
	"github.com/turkushan490/turkushan-auth/internal/store"
)

var tokenRe = regexp.MustCompile(`token=([A-Za-z0-9_-]+)`)

// withMail points the portal at a fake SMTP server.
func withMail(t *testing.T, st *store.Store) *mailtest.Server {
	t.Helper()
	srv := mailtest.Start(t)
	ctx := context.Background()
	st.SetSetting(ctx, store.SettingSMTPHost, srv.Host)
	st.SetSetting(ctx, store.SettingSMTPPort, srv.Port)
	st.SetSetting(ctx, store.SettingSMTPFrom, "Portal <noreply@example.com>")
	return srv
}

func linkToken(t *testing.T, srv *mailtest.Server, wantTo, wantPath string) string {
	t.Helper()
	to, text := srv.Last(t)
	if !strings.Contains(to, wantTo) {
		t.Fatalf("mail went to %q, want %q", to, wantTo)
	}
	if !strings.Contains(text, "https://auth.example.com"+wantPath+"?token=") {
		t.Fatalf("mail has no %s link:\n%s", wantPath, text)
	}
	return tokenRe.FindStringSubmatch(text)[1]
}

func TestEmailVerificationFlow(t *testing.T) {
	h, st := newTestServer(t)
	srv := withMail(t, st)
	addSite(t, st, "mail.example.com", false, true)

	c := newClient(t, h)
	if code, out := c.post("/api/register", map[string]string{"username": "ann", "password": "Secret!1", "email": "Ann@Example.com"}); code != http.StatusOK {
		t.Fatalf("register: %d %v", code, out)
	}
	token := linkToken(t, srv, "ann@example.com", "/verify")

	// Email-only site is closed until the address is verified.
	if res := c.forward(http.MethodGet, "https://mail.example.com/"); res.StatusCode != http.StatusForbidden {
		t.Fatalf("before verify: %d", res.StatusCode)
	}
	if code, out := c.post("/api/verify-email", map[string]string{"token": token}); code != http.StatusOK {
		t.Fatalf("verify: %d %v", code, out)
	}
	if code, _ := c.post("/api/verify-email", map[string]string{"token": token}); code != http.StatusBadRequest {
		t.Errorf("link worked twice: %d", code)
	}
	user := c.session()["user"].(map[string]any)
	if user["email_verified"] != true {
		t.Fatalf("not verified: %v", user)
	}
	if res := c.forward(http.MethodGet, "https://mail.example.com/"); res.StatusCode != http.StatusOK {
		t.Errorf("after verify: %d", res.StatusCode)
	}

	// Changing the address needs the password and makes it unverified again.
	if code, _ := c.post("/api/account/email", map[string]string{"email": "new@example.com", "password": "Wrong!1"}); code != http.StatusUnauthorized {
		t.Errorf("change without password: %d", code)
	}
	if code, out := c.post("/api/account/email", map[string]string{"email": "new@example.com", "password": "Secret!1"}); code != http.StatusOK {
		t.Fatalf("change email: %d %v", code, out)
	}
	linkToken(t, srv, "new@example.com", "/verify")
	if user := c.session()["user"].(map[string]any); user["email_verified"] != false || user["email"] != "new@example.com" {
		t.Errorf("after change: %v", user)
	}
	if code, _ := c.post("/api/account/email/resend", map[string]string{}); code != http.StatusOK {
		t.Errorf("resend: %d", code)
	}
}

func TestFirstEmailNeedsNoPasswordAndLinkContinuesToSite(t *testing.T) {
	h, st := newTestServer(t)
	srv := withMail(t, st)
	addSite(t, st, "mail.example.com", false, true)
	c := newClient(t, h)
	c.post("/api/register", map[string]string{"username": "eva", "password": "Secret!1"})

	// The sign-up page can ask what the site needs before the visitor has an account.
	_, info := newClient(t, h).do(http.MethodGet, "/api/site?rd=https%3A%2F%2Fmail.example.com%2Fx", nil, nil)
	if info["known"] != true || info["require_email"] != true {
		t.Errorf("site info: %v", info)
	}
	if _, info := c.do(http.MethodGet, "/api/site?rd=https%3A%2F%2Fnope.example.com%2F", nil, nil); info["known"] != false {
		t.Errorf("unknown site info: %v", info)
	}

	_, acc := c.do(http.MethodGet, "/api/access?site=mail.example.com", nil, nil)
	if acc["status"] != accessEmailRequired || acc["mail_ready"] != true || acc["email"] != "" {
		t.Fatalf("access: %v", acc)
	}

	// Adding a first address: no password, and the link leads on to the site.
	const rd = "https://mail.example.com/page?x=1"
	if code, out := c.post("/api/account/email", map[string]string{"email": "eva@example.com", "rd": rd}); code != http.StatusOK {
		t.Fatalf("first email: %d %v", code, out)
	}
	_, text := srv.Last(t)
	if !strings.Contains(text, "&rd=https%3A%2F%2Fmail.example.com%2Fpage%3Fx%3D1") {
		t.Errorf("verification link has no rd:\n%s", text)
	}
	// A typo can be corrected without the password too, as long as nothing is verified.
	if code, _ := c.post("/api/account/email", map[string]string{"email": "eva2@example.com", "rd": "https://evil.com/"}); code != http.StatusOK {
		t.Fatalf("correct address: %d", code)
	}
	if _, text := srv.Last(t); strings.Contains(text, "evil.com") {
		t.Error("foreign rd ended up in the mail")
	}
	token := linkToken(t, srv, "eva2@example.com", "/verify")
	if code, _ := c.post("/api/verify-email", map[string]string{"token": token}); code != http.StatusOK {
		t.Fatalf("verify: %d", code)
	}
	// Now it's verified: changing it needs the password again.
	if code, _ := c.post("/api/account/email", map[string]string{"email": "eva3@example.com"}); code != http.StatusUnauthorized {
		t.Errorf("change verified address without password: %d", code)
	}
}

func TestForgotAndResetPassword(t *testing.T) {
	h, st := newTestServer(t)
	srv := withMail(t, st)

	c := newClient(t, h)
	c.post("/api/register", map[string]string{"username": "bea", "password": "Secret!1", "email": "bea@example.com"})
	c.post("/api/verify-email", map[string]string{"token": linkToken(t, srv, "bea@example.com", "/verify")})
	other := newClient(t, h)
	other.post("/api/login", map[string]string{"username": "bea", "password": "Secret!1"})

	// Unknown accounts get the same answer and no mail.
	before := srv.Count()
	code, unknown := c.post("/api/password/forgot", map[string]string{"login": "nobody"})
	if code != http.StatusOK || srv.Count() != before {
		t.Fatalf("unknown user: %d, mails %d -> %d", code, before, srv.Count())
	}

	code, known := c.post("/api/password/forgot", map[string]string{"login": "BEA@example.com"})
	if code != http.StatusOK || known["message"] != unknown["message"] {
		t.Fatalf("known user answer differs: %v vs %v", known, unknown)
	}
	token := linkToken(t, srv, "bea@example.com", "/reset")

	if code, _ := c.post("/api/password/reset", map[string]string{"token": token, "password": "weak"}); code != http.StatusBadRequest {
		t.Errorf("weak password: %d", code)
	}
	// The weak attempt must not have used up the link.
	if code, out := c.post("/api/password/reset", map[string]string{"token": token, "password": "Nieuw!pass2"}); code != http.StatusOK {
		t.Fatalf("reset: %d %v", code, out)
	}
	if code, _ := c.post("/api/password/reset", map[string]string{"token": token, "password": "Again!pass3"}); code != http.StatusBadRequest {
		t.Errorf("reset link worked twice: %d", code)
	}
	if s := other.session(); s["authenticated"] != false {
		t.Error("other sessions survived a password reset")
	}
	if code, _ := newClient(t, h).post("/api/login", map[string]string{"username": "bea", "password": "Secret!1"}); code != http.StatusUnauthorized {
		t.Errorf("old password still works: %d", code)
	}
	if code, _ := newClient(t, h).post("/api/login", map[string]string{"username": "bea", "password": "Nieuw!pass2"}); code != http.StatusOK {
		t.Errorf("new password: %d", code)
	}

	// Without a verified email: no mail.
	c2 := newClient(t, h)
	c2.post("/api/register", map[string]string{"username": "cas", "password": "Secret!1"})
	before = srv.Count()
	c2.post("/api/password/forgot", map[string]string{"login": "cas"})
	if srv.Count() != before {
		t.Error("reset mail sent to an account without a verified email")
	}
}

func TestChangePassword(t *testing.T) {
	h, st := newTestServer(t)
	c := signedIn(t, h, st, "dirk", false)
	other := newClient(t, h)
	other.post("/api/login", map[string]string{"username": "dirk", "password": testPassword})

	if code, _ := c.post("/api/account/password", map[string]string{"current": "Wrong!1", "new": "Better!pass1"}); code != http.StatusUnauthorized {
		t.Errorf("wrong current: %d", code)
	}
	if code, _ := c.post("/api/account/password", map[string]string{"current": testPassword, "new": "weak"}); code != http.StatusBadRequest {
		t.Errorf("weak new: %d", code)
	}
	if code, out := c.post("/api/account/password", map[string]string{"current": testPassword, "new": "Better!pass1"}); code != http.StatusOK {
		t.Fatalf("change: %d %v", code, out)
	}
	if s := c.session(); s["authenticated"] != true {
		t.Error("this browser was signed out")
	}
	if s := other.session(); s["authenticated"] != false {
		t.Error("other device still signed in")
	}
}

func TestMailSettings(t *testing.T) {
	h, st := newTestServer(t)
	srv := mailtest.Start(t)
	admin := signedIn(t, h, st, "boss", true)

	if code, _ := admin.post("/api/admin/settings/test-mail", map[string]string{"to": "boss@example.com"}); code != http.StatusBadRequest {
		t.Errorf("test mail without settings: %d", code)
	}
	for _, bad := range []map[string]any{
		{"smtp_from": "not an address"},
		{"smtp_port": 0},
		{"smtp_host": "bad host;"},
	} {
		if code, _ := admin.put("/api/admin/settings", bad); code != http.StatusBadRequest {
			t.Errorf("settings %v: %d", bad, code)
		}
	}
	port, _ := strconv.Atoi(srv.Port)
	good := map[string]any{"smtp_host": srv.Host, "smtp_port": port, "smtp_username": "resend", "smtp_password": "re_secret", "smtp_from": "noreply@example.com"}
	if code, out := admin.put("/api/admin/settings", good); code != http.StatusOK {
		t.Fatalf("save: %d %v", code, out)
	}
	_, data := admin.do(http.MethodGet, "/api/admin/data", nil, nil)
	smtp := data["settings"].(map[string]any)["smtp"].(map[string]any)
	if smtp["password_set"] != true || smtp["ready"] != true {
		t.Errorf("smtp settings: %v", smtp)
	}
	if _, leaked := smtp["password"]; leaked {
		t.Error("SMTP password sent to the browser")
	}
	// Saving without a password keeps the stored one.
	admin.put("/api/admin/settings", map[string]any{"smtp_password": ""})
	if v, _ := st.Setting(context.Background(), store.SettingSMTPPassword); v != "re_secret" {
		t.Errorf("password lost: %q", v)
	}
	if code, out := admin.post("/api/admin/settings/test-mail", map[string]string{"to": "boss@example.com"}); code != http.StatusOK {
		t.Fatalf("test mail: %d %v", code, out)
	}
	if to, _ := srv.Last(t); !strings.Contains(to, "boss@example.com") {
		t.Errorf("test mail to %q", to)
	}
}
