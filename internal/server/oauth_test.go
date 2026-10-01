package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/turkushan490/turkushan-auth/internal/store"
)

// fakeProvider plays Discord and Google: token endpoint, profile, and Discord server roles.
type fakeProvider struct {
	mu      sync.Mutex
	srv     *httptest.Server
	google  map[string]any
	discord map[string]any
	roles   map[string][]string // Discord server id → role ids; absent = not a member
}

func (f *fakeProvider) set(fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn()
}

// withOAuth points the portal at a fake provider and enables both logins.
func withOAuth(t *testing.T, s *Server, st *store.Store) *fakeProvider {
	t.Helper()
	f := &fakeProvider{roles: map[string][]string{}}
	mux := http.NewServeMux()
	reply := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.Form.Get("code") != "good" || r.Form.Get("client_secret") == "" || !strings.Contains(r.Form.Get("redirect_uri"), "/callback") {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}
		reply(w, map[string]string{"access_token": "tok"})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		reply(w, f.google)
	})
	mux.HandleFunc("/users/@me", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		reply(w, f.discord)
	})
	mux.HandleFunc("/users/@me/guilds/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		guild := strings.Split(strings.TrimPrefix(r.URL.Path, "/users/@me/guilds/"), "/")[0]
		roles, in := f.roles[guild]
		if !in {
			http.Error(w, `{"message":"Unknown Guild"}`, http.StatusNotFound)
			return
		}
		reply(w, map[string]any{"roles": roles})
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)

	for name, label := range map[string]string{"google": "Google", "discord": "Discord"} {
		s.oauth[name] = &oauthProvider{name: name, label: label, authURL: f.srv.URL + "/auth", tokenURL: f.srv.URL + "/token", apiBase: f.srv.URL}
		st.SetSetting(context.Background(), oauthIDKey(name), name+"-client")
		st.SetSetting(context.Background(), oauthSecretKey(name), name+"-secret")
	}
	return f
}

// oauthLogin runs start + callback like a browser would, and returns where the portal sends the visitor.
func (c *client) oauthLogin(provider, startQuery string) string {
	c.t.Helper()
	rec, _ := c.do(http.MethodGet, "/api/oauth/"+provider+"/start?"+startQuery, nil, nil)
	if rec.Code != http.StatusFound {
		c.t.Fatalf("oauth start: %d %s", rec.Code, rec.Body.String())
	}
	authURL, err := url.Parse(rec.Header().Get("Location"))
	if err != nil || authURL.Query().Get("state") == "" {
		return rec.Header().Get("Location") // not sent to the provider (e.g. disabled)
	}
	rec, _ = c.do(http.MethodGet, "/api/oauth/"+provider+"/callback?code=good&state="+url.QueryEscape(authURL.Query().Get("state")), nil, nil)
	if rec.Code != http.StatusFound {
		c.t.Fatalf("oauth callback: %d %s", rec.Code, rec.Body.String())
	}
	return rec.Header().Get("Location")
}

func pendingToken(t *testing.T, location string) string {
	t.Helper()
	if !strings.HasPrefix(location, "/welcome?p=") {
		t.Fatalf("expected the welcome step, got %q", location)
	}
	p, _ := url.QueryUnescape(strings.TrimPrefix(location, "/welcome?p="))
	return p
}

func TestOAuthNewUser(t *testing.T) {
	s, h, st := newTestServerFull(t)

	// Not set up: no buttons, and starting sends you back.
	c := newClient(t, h)
	if p := c.session()["providers"].([]any); len(p) != 0 {
		t.Fatalf("providers before setup: %v", p)
	}
	if loc := c.oauthLogin("google", ""); loc != "/login?oauth_error=disabled" {
		t.Errorf("disabled provider: %q", loc)
	}

	f := withOAuth(t, s, st)
	f.set(func() {
		f.google = map[string]any{"sub": "g-1", "email": "Nina@Gmail.com", "email_verified": true, "name": "Nina de Vries"}
	})
	if p := c.session()["providers"].([]any); fmt.Sprint(p) != "[discord google]" {
		t.Fatalf("providers: %v", p)
	}

	// First time: welcome step, with a suggested username.
	token := pendingToken(t, c.oauthLogin("google", "rd="+url.QueryEscape("https://manga.example.com/x")))
	if c.session()["authenticated"] != false {
		t.Fatal("signed in before finishing the welcome step")
	}
	_, pending := c.do(http.MethodGet, "/api/oauth/pending?p="+url.QueryEscape(token), nil, nil)
	if pending["suggested_username"] != "ninadevries" || pending["label"] != "Google" {
		t.Fatalf("pending: %v", pending)
	}

	// Skip = take the suggestion, no password.
	code, out := c.post("/api/oauth/complete", map[string]string{"p": token})
	if code != http.StatusOK || out["redirect"] != "https://manga.example.com/x" {
		t.Fatalf("complete: %d %v", code, out)
	}
	user := c.session()["user"].(map[string]any)
	if user["username"] != "ninadevries" || user["email"] != "nina@gmail.com" || user["email_verified"] != true ||
		user["has_password"] != false || fmt.Sprint(user["logins"]) != "[google]" {
		t.Errorf("new user: %v", user)
	}
	if code, _ := c.post("/api/oauth/complete", map[string]string{"p": token}); code != http.StatusNotFound {
		t.Errorf("welcome token worked twice: %d", code)
	}

	// Next time: straight in, no welcome step.
	c2 := newClient(t, h)
	if loc := c2.oauthLogin("google", ""); loc != "https://auth.example.com/" {
		t.Errorf("returning user: %q", loc)
	}
	if u := c2.session()["user"].(map[string]any); u["username"] != "ninadevries" {
		t.Errorf("returning user session: %v", u)
	}

	// Can't disconnect their only way in; after setting a password they can.
	if rec, _ := c2.do(http.MethodDelete, "/api/account/logins/google", nil, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("unlink only login: %d", rec.Code)
	}
	if code, _ := c2.post("/api/account/password", map[string]string{"new": "Secret!1"}); code != http.StatusOK {
		t.Fatalf("set first password: %d", code)
	}
	if rec, _ := c2.do(http.MethodDelete, "/api/account/logins/google", nil, nil); rec.Code != http.StatusOK {
		t.Errorf("unlink with a password: %d", rec.Code)
	}

	// Choosing a username and password in the welcome step.
	f.set(func() { f.google = map[string]any{"sub": "g-2", "email": "bo@gmail.com", "email_verified": true, "name": "!!!"} })
	c3 := newClient(t, h)
	token = pendingToken(t, c3.oauthLogin("google", "rd="+url.QueryEscape("https://evil.com/")))
	for body, want := range map[*map[string]string]int{
		{"p": token, "username": "ninadevries"}:               http.StatusConflict,
		{"p": token, "username": "x"}:                         http.StatusBadRequest,
		{"p": token, "username": "bo.b", "password": "weak"}:  http.StatusBadRequest,
	} {
		if code, _ := c3.post("/api/oauth/complete", *body); code != want {
			t.Errorf("complete %v: %d, want %d", *body, code, want)
		}
	}
	code, out = c3.post("/api/oauth/complete", map[string]string{"p": token, "username": "Bo.B", "password": "Secret!1"})
	if code != http.StatusOK || out["redirect"] != "https://auth.example.com/" {
		t.Fatalf("complete with username: %d %v", code, out)
	}
	if code, _ := newClient(t, h).post("/api/login", map[string]string{"username": "bo.b", "password": "Secret!1"}); code != http.StatusOK {
		t.Errorf("password login after welcome step: %d", code)
	}
}

func TestOAuthLinking(t *testing.T) {
	s, h, st := newTestServerFull(t)
	srv := withMail(t, st)
	f := withOAuth(t, s, st)

	// An existing account with a verified address.
	owner := newClient(t, h)
	owner.post("/api/register", map[string]string{"username": "eva", "password": "Secret!1", "email": "eva@example.com"})
	owner.post("/api/verify-email", map[string]string{"token": linkToken(t, srv, "eva@example.com", "/verify")})

	// Google says the address is NOT verified: never linked on that alone.
	f.set(func() { f.google = map[string]any{"sub": "g-eva", "email": "eva@example.com", "email_verified": false, "name": "Eva"} })
	c := newClient(t, h)
	pendingToken(t, c.oauthLogin("google", ""))

	// Verified address: that's the same person, signed in to the existing account.
	f.set(func() { f.google["email_verified"] = true })
	c = newClient(t, h)
	if loc := c.oauthLogin("google", ""); loc != "https://auth.example.com/" {
		t.Fatalf("auto-link: %q", loc)
	}
	if u := c.session()["user"].(map[string]any); u["username"] != "eva" || fmt.Sprint(u["logins"]) != "[google]" {
		t.Errorf("auto-linked session: %v", u)
	}

	// Connecting Discord from the account page.
	f.set(func() { f.discord = map[string]any{"id": "d-1", "username": "evagamer", "email": "other@example.com", "verified": true} })
	if loc := owner.oauthLogin("discord", "link=1"); loc != "/?connected=discord" {
		t.Fatalf("link: %q", loc)
	}
	if u := owner.session()["user"].(map[string]any); fmt.Sprint(u["logins"]) != "[discord google]" {
		t.Errorf("logins after linking: %v", u["logins"])
	}
	// Linking needs a signed-in user.
	if rec, _ := newClient(t, h).do(http.MethodGet, "/api/oauth/discord/start?link=1", nil, nil); rec.Header().Get("Location") != "/login" {
		t.Errorf("link while signed out: %q", rec.Header().Get("Location"))
	}
	// Someone else can't connect a Discord account that already belongs to eva.
	other := signedIn(t, h, st, "mallory", false)
	if loc := other.oauthLogin("discord", "link=1"); loc != "/?oauth_error=taken" {
		t.Errorf("link a taken account: %q", loc)
	}

	// A blocked account can't get in through Discord either.
	st.DB.Exec(`UPDATE users SET status = 'blocked' WHERE username = 'eva'`)
	c = newClient(t, h)
	if loc := c.oauthLogin("discord", ""); loc != "/login?oauth_error=blocked" {
		t.Errorf("blocked user: %q", loc)
	}
	if c.session()["authenticated"] != false {
		t.Error("blocked user got a session")
	}
}

func TestOAuthStateAndErrors(t *testing.T) {
	s, h, st := newTestServerFull(t)
	f := withOAuth(t, s, st)
	f.set(func() { f.google = map[string]any{"sub": "g-9", "email": "x@example.com", "email_verified": true, "name": "X"} })

	callback := func(c *client, query string) string {
		rec, _ := c.do(http.MethodGet, "/api/oauth/google/callback?"+query, nil, nil)
		return rec.Header().Get("Location")
	}
	start := func(c *client) string {
		rec, _ := c.do(http.MethodGet, "/api/oauth/google/start", nil, nil)
		u, _ := url.Parse(rec.Header().Get("Location"))
		if !strings.HasPrefix(rec.Header().Get("Location"), f.srv.URL+"/auth?") || u.Query().Get("client_id") != "google-client" ||
			u.Query().Get("redirect_uri") != "https://auth.example.com/api/oauth/google/callback" {
			t.Fatalf("authorize URL: %s", rec.Header().Get("Location"))
		}
		return u.Query().Get("state")
	}

	// A callback nobody started (login CSRF) is refused.
	if loc := callback(newClient(t, h), "code=good&state=whatever"); loc != "/login?oauth_error=state" {
		t.Errorf("callback without start: %q", loc)
	}
	c := newClient(t, h)
	start(c)
	if loc := callback(c, "code=good&state=forged"); loc != "/login?oauth_error=state" {
		t.Errorf("wrong state: %q", loc)
	}
	// The state can't be replayed after a failed attempt.
	c = newClient(t, h)
	state := start(c)
	callback(c, "code=good&state=forged")
	if loc := callback(c, "code=good&state="+state); loc != "/login?oauth_error=state" {
		t.Errorf("state reused: %q", loc)
	}
	// The state belongs to one provider.
	c = newClient(t, h)
	state = start(c)
	if rec, _ := c.do(http.MethodGet, "/api/oauth/discord/callback?code=good&state="+state, nil, nil); rec.Header().Get("Location") != "/login?oauth_error=state" {
		t.Errorf("state used at another provider: %q", rec.Header().Get("Location"))
	}

	c = newClient(t, h)
	if loc := callback(c, "error=access_denied&state="+start(c)); loc != "/login?oauth_error=cancelled" {
		t.Errorf("cancelled: %q", loc)
	}
	c = newClient(t, h)
	if loc := callback(c, "code=bad&state="+start(c)); loc != "/login?oauth_error=provider" {
		t.Errorf("bad code: %q", loc)
	}
	if rec, _ := c.do(http.MethodGet, "/api/oauth/facebook/start", nil, nil); rec.Code != http.StatusNotFound {
		t.Errorf("unknown provider: %d", rec.Code)
	}
}

func TestOAuthDiscordRoleRules(t *testing.T) {
	s, h, st := newTestServerFull(t)
	f := withOAuth(t, s, st)
	admin := signedIn(t, h, st, "boss", true)
	vip := mkGroup(t, admin, "VIP")
	members := mkGroup(t, admin, "Server members")
	discordUsers := mkGroup(t, admin, "Discord users")
	for group, rule := range map[int64][2]string{
		vip:          {"discord_role", "111111:222222"},
		members:      {"discord_server", "111111"},
		discordUsers: {"method", "discord"},
	} {
		if code, out := admin.post(fmt.Sprintf("/api/admin/groups/%d/rules", group), map[string]any{"kind": rule[0], "value": rule[1]}); code != http.StatusOK {
			t.Fatalf("rule %v: %d %v", rule, code, out)
		}
	}

	in := func(group int64) bool {
		var n int
		st.DB.QueryRow(`SELECT COUNT(*) FROM user_groups ug JOIN users u ON u.id = ug.user_id WHERE u.username = 'gamer' AND ug.group_id = ?`, group).Scan(&n)
		return n == 1
	}

	f.set(func() {
		f.discord = map[string]any{"id": "d-7", "username": "Gamer", "email": "gamer@example.com", "verified": true}
		f.roles["111111"] = []string{"222222", "333333"}
	})
	c := newClient(t, h)
	// With role rules, the portal asks Discord for permission to read server roles.
	rec, _ := c.do(http.MethodGet, "/api/oauth/discord/start", nil, nil)
	if !strings.Contains(rec.Header().Get("Location"), "guilds.members.read") {
		t.Errorf("scope without guilds.members.read: %s", rec.Header().Get("Location"))
	}
	c.post("/api/oauth/complete", map[string]string{"p": pendingToken(t, c.oauthLogin("discord", ""))})
	if !in(vip) || !in(members) || !in(discordUsers) {
		t.Fatalf("after first login: vip=%v members=%v discord=%v", in(vip), in(members), in(discordUsers))
	}

	// Role taken away in Discord: gone at the next Discord login.
	f.set(func() { f.roles["111111"] = []string{"333333"} })
	newClient(t, h).oauthLogin("discord", "")
	if in(vip) || !in(members) {
		t.Errorf("after role removed: vip=%v members=%v", in(vip), in(members))
	}
	// Left the server.
	f.set(func() { delete(f.roles, "111111") })
	newClient(t, h).oauthLogin("discord", "")
	if in(vip) || in(members) || !in(discordUsers) {
		t.Errorf("after leaving the server: vip=%v members=%v discord=%v", in(vip), in(members), in(discordUsers))
	}
}

func TestOAuthSettings(t *testing.T) {
	_, h, st := newTestServerFull(t)
	admin := signedIn(t, h, st, "boss", true)

	if code, _ := admin.put("/api/admin/settings", map[string]any{"oauth": map[string]any{"facebook": map[string]string{"client_id": "x"}}}); code != http.StatusBadRequest {
		t.Errorf("unknown provider: %d", code)
	}
	if code, _ := admin.put("/api/admin/settings", map[string]any{"oauth": map[string]any{"discord": map[string]string{"client_id": ""}}}); code != http.StatusBadRequest {
		t.Errorf("empty client id: %d", code)
	}
	set := map[string]any{"oauth": map[string]any{"discord": map[string]string{"client_id": "123", "client_secret": "s3cret"}}}
	if code, out := admin.put("/api/admin/settings", set); code != http.StatusOK {
		t.Fatalf("save: %d %v", code, out)
	}
	_, data := admin.do(http.MethodGet, "/api/admin/data", nil, nil)
	discord := data["settings"].(map[string]any)["oauth"].(map[string]any)["discord"].(map[string]any)
	if discord["client_id"] != "123" || discord["secret_set"] != true ||
		discord["redirect_uri"] != "https://auth.example.com/api/oauth/discord/callback" {
		t.Errorf("settings: %v", discord)
	}
	if strings.Contains(fmt.Sprint(data), "s3cret") {
		t.Error("client secret sent to the browser")
	}
	if p := admin.session()["providers"].([]any); fmt.Sprint(p) != "[discord]" {
		t.Errorf("providers: %v", p)
	}
	// Saving without a secret keeps it; remove turns the login off.
	admin.put("/api/admin/settings", map[string]any{"oauth": map[string]any{"discord": map[string]string{"client_id": "456"}}})
	if v, _ := st.Setting(context.Background(), oauthSecretKey("discord")); v != "s3cret" {
		t.Errorf("secret lost: %q", v)
	}
	admin.put("/api/admin/settings", map[string]any{"oauth": map[string]any{"discord": map[string]any{"remove": true}}})
	if p := admin.session()["providers"].([]any); len(p) != 0 {
		t.Errorf("providers after remove: %v", p)
	}
}
