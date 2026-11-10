package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/turkushan490/turkushan-auth/internal/auth"
	"github.com/turkushan490/turkushan-auth/internal/i18n"
	"github.com/turkushan490/turkushan-auth/internal/store"
)

// oauthProvider is one "Continue with …" login. The URLs are fields so tests can point them at a fake provider.
type oauthProvider struct {
	name     string // discord | google, also the URL segment and the identities.provider value
	label    string
	authURL  string
	tokenURL string
	apiBase  string
}

func defaultOAuthProviders() map[string]*oauthProvider {
	return map[string]*oauthProvider{
		"discord": {
			name: "discord", label: "Discord",
			authURL:  "https://discord.com/oauth2/authorize",
			tokenURL: "https://discord.com/api/oauth2/token",
			apiBase:  "https://discord.com/api",
		},
		"google": {
			name: "google", label: "Google",
			authURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			tokenURL: "https://oauth2.googleapis.com/token",
			apiBase:  "https://openidconnect.googleapis.com/v1",
		},
	}
}

var oauthOrder = []string{"discord", "google"}

const (
	oauthCookie = "ta_oauth"
	pendingTTL  = 30 * time.Minute
)

func oauthIDKey(name string) string     { return "oauth_" + name + "_id" }
func oauthSecretKey(name string) string { return "oauth_" + name + "_secret" }

func (s *Server) oauthCreds(ctx context.Context, name string) (id, secret string, err error) {
	if id, err = s.store.Setting(ctx, oauthIDKey(name)); err != nil {
		return "", "", err
	}
	secret, err = s.store.Setting(ctx, oauthSecretKey(name))
	return id, secret, err
}

// enabledProviders lists the logins that are set up, in display order.
func (s *Server) enabledProviders(ctx context.Context) []string {
	out := []string{}
	for _, name := range oauthOrder {
		if id, secret, err := s.oauthCreds(ctx, name); err == nil && id != "" && secret != "" {
			out = append(out, name)
		}
	}
	return out
}

// ensureProviderGroup makes, once per provider, a group named after it ("Discord",
// "Google") that everyone who signs in that way is put in automatically. The
// admin then only has to tick what the group may open. It is not made again
// after the admin deletes it.
func (s *Server) ensureProviderGroup(ctx context.Context, p *oauthProvider) error {
	flag := "oauth_" + p.name + "_group_made"
	if done, err := s.store.Setting(ctx, flag); err != nil || done != "" {
		return err
	}
	rules, err := s.store.ListRules(ctx)
	if err != nil {
		return err
	}
	exists := false
	for _, r := range rules {
		if r.Kind == store.RuleMethod && r.Value == p.name {
			exists = true // the admin already made such a rule
		}
	}
	if !exists {
		id, err := s.store.CreateGroup(ctx, p.label, "Everyone who signs in with "+p.label+".")
		if errors.Is(err, store.ErrGroupNameTaken) {
			groups, lerr := s.store.ListGroups(ctx)
			if lerr != nil {
				return lerr
			}
			for _, g := range groups {
				if strings.EqualFold(g.Name, p.label) {
					id, err = g.ID, nil
				}
			}
		}
		if err != nil {
			return err
		}
		if _, err := s.store.CreateRule(ctx, id, store.RuleMethod, p.name); err != nil {
			return err
		}
		if err := s.store.SyncAllRuleGroups(ctx); err != nil {
			return err
		}
	}
	return s.store.SetSetting(ctx, flag, "1")
}

// ensureProviderGroups runs at start-up for the logins that are already on.
func (s *Server) ensureProviderGroups(ctx context.Context) {
	for _, name := range s.enabledProviders(ctx) {
		if err := s.ensureProviderGroup(ctx, s.oauth[name]); err != nil {
			s.log.Error("create login group", "provider", name, "err", err)
		}
	}
}

func (s *Server) oauthRedirectURI(name string) string {
	return s.cfg.AppURL.String() + "/api/oauth/" + name + "/callback"
}

// oauthState travels in a short-lived cookie between "start" and "callback".
type oauthState struct {
	State    string `json:"s"`
	Provider string `json:"p"`
	RD       string `json:"rd,omitempty"`
	Link     bool   `json:"l,omitempty"` // connect to the signed-in account instead of signing in
	Verifier string `json:"v"`           // PKCE code_verifier
	Expires  int64  `json:"e"`
}

const oauthStateTTL = 10 * time.Minute

// oauthCookieName uses the __Host- prefix on https: the browser then refuses
// the cookie from any other subdomain, so a sibling app can't plant one.
func (s *Server) oauthCookieName() string {
	if s.secureCookies() {
		return "__Host-" + oauthCookie
	}
	return oauthCookie
}

func (s *Server) setOAuthCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name: s.oauthCookieName(), Value: value, Path: "/", MaxAge: maxAge,
		HttpOnly: true, Secure: s.secureCookies(), SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) oauthMAC(payload string) string {
	m := hmac.New(sha256.New, append([]byte("oauth-state:"), s.cfg.SessionSecret...))
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// encodeOAuthState signs the state, so only this portal can have made the cookie.
func (s *Server) encodeOAuthState(st oauthState) string {
	raw, _ := json.Marshal(st)
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return payload + "." + s.oauthMAC(payload)
}

func (s *Server) decodeOAuthState(r *http.Request) (oauthState, bool) {
	var st oauthState
	c, err := r.Cookie(s.oauthCookieName())
	if err != nil {
		return st, false
	}
	payload, mac, ok := strings.Cut(c.Value, ".")
	if !ok || !hmac.Equal([]byte(mac), []byte(s.oauthMAC(payload))) {
		return st, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil || json.Unmarshal(raw, &st) != nil || st.Expires < time.Now().Unix() {
		return oauthState{}, false
	}
	return st, true
}

// pkceChallenge is the S256 code_challenge for a verifier (RFC 7636).
func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// handleOAuthStart sends the visitor to Discord/Google.
func (s *Server) handleOAuthStart(w http.ResponseWriter, r *http.Request) {
	p := s.oauth[chi.URLParam(r, "provider")]
	if p == nil {
		http.NotFound(w, r)
		return
	}
	ctx := r.Context()
	clientID, secret, err := s.oauthCreds(ctx, p.name)
	if err != nil || clientID == "" || secret == "" {
		s.oauthFail(w, r, false, "disabled")
		return
	}
	link := r.URL.Query().Get("link") == "1"
	if link && s.currentUser(r) == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	state, err := store.NewToken()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	verifier, err := store.NewToken() // 43 url-safe characters, the PKCE minimum
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.setOAuthCookie(w, s.encodeOAuthState(oauthState{
		State: state, Provider: p.name, RD: r.URL.Query().Get("rd"), Link: link,
		Verifier: verifier, Expires: time.Now().Add(oauthStateTTL).Unix(),
	}), int(oauthStateTTL/time.Second))

	q := url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {s.oauthRedirectURI(p.name)},
		"response_type":         {"code"},
		"state":                 {state},
		"code_challenge":        {pkceChallenge(verifier)},
		"code_challenge_method": {"S256"},
	}
	switch p.name {
	case "google":
		q.Set("scope", "openid email profile")
		q.Set("prompt", "select_account")
	case "discord":
		scope := "identify email"
		// Only ask to see server roles when a rule actually needs them.
		if guilds, err := s.discordRuleGuilds(ctx); err == nil && len(guilds) > 0 {
			scope += " guilds.members.read"
		}
		q.Set("scope", scope)
	}
	http.Redirect(w, r, p.authURL+"?"+q.Encode(), http.StatusFound)
}

// oauthFail sends the visitor back with a short error code the page turns into a message.
func (s *Server) oauthFail(w http.ResponseWriter, r *http.Request, link bool, code string) {
	target := "/login"
	if link {
		target = "/"
	}
	http.Redirect(w, r, target+"?oauth_error="+url.QueryEscape(code), http.StatusFound)
}

type oauthProfile struct {
	Subject  string
	Email    string
	Verified bool
	Display  string
	Data     string // identities.data
}

// handleOAuthCallback is where Discord/Google send the visitor back to.
func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	p := s.oauth[chi.URLParam(r, "provider")]
	if p == nil {
		http.NotFound(w, r)
		return
	}
	ctx := r.Context()
	w.Header().Set("Cache-Control", "no-store")

	st, valid := s.decodeOAuthState(r)
	s.setOAuthCookie(w, "", -1)
	// The state must match the one we sent from this browser, or someone else started this login.
	if !valid || st.State == "" || st.Provider != p.name ||
		subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("state")), []byte(st.State)) != 1 {
		s.oauthFail(w, r, false, "state")
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		s.oauthFail(w, r, st.Link, "cancelled")
		return
	}
	clientID, secret, err := s.oauthCreds(ctx, p.name)
	if err != nil || clientID == "" {
		s.oauthFail(w, r, st.Link, "disabled")
		return
	}

	token, err := s.oauthExchange(ctx, p, clientID, secret, r.URL.Query().Get("code"), st.Verifier)
	if err != nil {
		s.log.Error("oauth token exchange", "provider", p.name, "err", err)
		s.oauthFail(w, r, st.Link, "provider")
		return
	}
	profile, err := s.oauthProfile(ctx, p, token)
	if err != nil || profile.Subject == "" {
		s.log.Error("oauth profile", "provider", p.name, "err", err)
		s.oauthFail(w, r, st.Link, "provider")
		return
	}

	current := s.currentUser(r)
	ident, err := s.store.IdentityBySubject(ctx, p.name, profile.Subject)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		s.serverError(w, r, err)
		return
	}

	var userID int64
	switch {
	case ident != nil:
		// Known Discord/Google account.
		if st.Link && current != nil && current.ID != ident.UserID {
			s.oauthFail(w, r, true, "taken")
			return
		}
		userID = ident.UserID
	case st.Link:
		if current == nil {
			s.oauthFail(w, r, false, "state")
			return
		}
		userID = current.ID
	default:
		// New here. If their verified address belongs to an account, that's them,
		// except for admins and staff: whoever controls a Discord/Google account
		// with that address shouldn't get the admin panel without the password.
		// They connect it themselves from the account page instead.
		if profile.Verified && profile.Email != "" {
			if u, err := s.store.UserByVerifiedEmail(ctx, profile.Email); err == nil {
				rights, err := s.store.UserRights(ctx, u)
				if err != nil {
					s.serverError(w, r, err)
					return
				}
				if rights.Staff() {
					s.log.Warn("oauth sign-in matched a staff account by email; not linked", "username", u.Username, "provider", p.name)
					s.oauthFail(w, r, false, "link_first")
					return
				}
				userID = u.ID
			} else if !errors.Is(err, store.ErrNotFound) {
				s.serverError(w, r, err)
				return
			}
		}
		if userID == 0 {
			// Let them pick a username (and maybe a password) first.
			pendingToken, err := s.store.CreatePending(ctx, &store.Pending{
				Provider: p.name, Subject: profile.Subject, Email: profile.Email, Verified: profile.Verified,
				Display: profile.Display, Data: profile.Data, RD: st.RD,
			}, pendingTTL)
			if err != nil {
				s.serverError(w, r, err)
				return
			}
			http.Redirect(w, r, "/welcome?p="+url.QueryEscape(pendingToken), http.StatusFound)
			return
		}
	}

	// Blocked accounts stop here, before anything about them is saved or synced.
	u, err := s.store.UserByID(ctx, userID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if u.Status != "active" {
		s.oauthFail(w, r, st.Link, "blocked")
		return
	}
	if err := s.store.SaveIdentity(ctx, &store.Identity{
		Provider: p.name, Subject: profile.Subject, UserID: userID,
		Email: profile.Email, Display: profile.Display, Data: profile.Data,
	}); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.syncGroups(ctx, userID)

	if st.Link {
		http.Redirect(w, r, "/?connected="+p.name, http.StatusFound)
		return
	}
	ip := clientIP(r, s.cfg.TrustedProxies)
	if err := s.store.RecordLoginSuccess(ctx, u.ID); err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := s.startSession(w, r, u.ID, ip); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.log.Info("login", "username", u.Username, "ip", ip, "via", p.name)
	http.Redirect(w, r, s.safeRedirect(st.RD), http.StatusFound)
}

func (s *Server) oauthExchange(ctx context.Context, p *oauthProvider, clientID, secret, code, verifier string) (string, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {s.oauthRedirectURI(p.name)},
		"client_id":     {clientID},
		"client_secret": {secret},
		"code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := s.oauthHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK || out.AccessToken == "" {
		return "", fmt.Errorf("token endpoint answered %s", resp.Status)
	}
	return out.AccessToken, nil
}

// oauthGet fetches JSON with the visitor's access token and returns the HTTP status.
func (s *Server) oauthGet(ctx context.Context, rawURL, token string, v any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := s.oauthHTTP.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, nil
	}
	return resp.StatusCode, json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(v)
}

func (s *Server) oauthProfile(ctx context.Context, p *oauthProvider, token string) (*oauthProfile, error) {
	switch p.name {
	case "google":
		var g struct {
			Sub           string `json:"sub"`
			Email         string `json:"email"`
			EmailVerified any    `json:"email_verified"` // bool, or "true" in older responses
			Name          string `json:"name"`
		}
		if code, err := s.oauthGet(ctx, p.apiBase+"/userinfo", token, &g); err != nil || code != http.StatusOK {
			return nil, fmt.Errorf("userinfo: status %d: %v", code, err)
		}
		email, _ := auth.NormalizeEmail(g.Email)
		verified := g.EmailVerified == true || g.EmailVerified == "true"
		return &oauthProfile{Subject: g.Sub, Email: email, Verified: verified && email != "", Display: g.Name}, nil

	case "discord":
		var d struct {
			ID         string `json:"id"`
			Username   string `json:"username"`
			GlobalName string `json:"global_name"`
			Email      string `json:"email"`
			Verified   bool   `json:"verified"`
		}
		if code, err := s.oauthGet(ctx, p.apiBase+"/users/@me", token, &d); err != nil || code != http.StatusOK {
			return nil, fmt.Errorf("users/@me: status %d: %v", code, err)
		}
		email, _ := auth.NormalizeEmail(d.Email)
		display := d.Username
		if display == "" {
			display = d.GlobalName
		}
		prof := &oauthProfile{Subject: d.ID, Email: email, Verified: d.Verified && email != "", Display: display}

		// Roles in the servers that auto-add rules look at; a server they're not in is simply left out.
		guilds, err := s.discordRuleGuilds(ctx)
		if err != nil {
			return nil, err
		}
		roles := map[string][]string{}
		for _, guild := range guilds {
			var m struct {
				Roles []string `json:"roles"`
			}
			code, err := s.oauthGet(ctx, p.apiBase+"/users/@me/guilds/"+guild+"/member", token, &m)
			if err != nil {
				return nil, err
			}
			if code == http.StatusOK {
				if m.Roles == nil {
					m.Roles = []string{}
				}
				roles[guild] = m.Roles
			}
		}
		data, _ := json.Marshal(map[string]any{"guilds": roles})
		prof.Data = string(data)
		return prof, nil
	}
	return nil, errors.New("unknown provider")
}

// discordRuleGuilds lists the Discord server ids that auto-add rules refer to.
func (s *Server) discordRuleGuilds(ctx context.Context) ([]string, error) {
	rules, err := s.store.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := []string{}
	for _, r := range rules {
		guild := ""
		switch r.Kind {
		case store.RuleDiscordServer:
			guild = r.Value
		case store.RuleDiscordRole:
			guild, _, _ = strings.Cut(r.Value, ":")
		}
		if guild != "" && !seen[guild] {
			seen[guild] = true
			out = append(out, guild)
		}
	}
	return out, nil
}

// suggestUsername turns a Discord/Google name into something that fits the username rules.
func suggestUsername(display string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(display) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		case (r == '.' || r == '_' || r == '-') && b.Len() > 0:
			b.WriteRune(r)
		}
		if b.Len() >= 24 {
			break
		}
	}
	name := strings.TrimRight(b.String(), "._-")
	if len(name) < 3 {
		return "user"
	}
	return name
}

// freeUsername returns base, or base with a number behind it when it's taken.
func (s *Server) freeUsername(ctx context.Context, base string) (string, error) {
	for i := 1; i < 200; i++ {
		name := base
		if i > 1 {
			name = fmt.Sprintf("%s%d", base, i)
		}
		taken, err := s.store.UsernameTaken(ctx, name)
		if err != nil {
			return "", err
		}
		if !taken && auth.ValidateUsername(name) == nil {
			return name, nil
		}
	}
	return "", errors.New("no free username found")
}

// handleOAuthPending tells the welcome page who is finishing their sign-up.
func (s *Server) handleOAuthPending(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p, err := s.store.GetPending(ctx, r.URL.Query().Get("p"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "This sign-up has expired. Start again from the sign-in page.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	suggested, err := s.freeUsername(ctx, suggestUsername(p.Display))
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"provider": p.Provider, "label": s.oauth[p.Provider].label,
		"display": p.Display, "email": p.Email, "suggested_username": suggested,
	})
}

type oauthCompleteRequest struct {
	P        string `json:"p"`
	Username string `json:"username"` // "" = take the suggestion
	Password string `json:"password"` // "" = no password, sign in with Discord/Google only
	Lang     string `json:"lang"`     // the language picked in the web app, if any
}

// handleOAuthComplete creates the account after a first Discord/Google sign-in.
func (s *Server) handleOAuthComplete(w http.ResponseWriter, r *http.Request) {
	var req oauthCompleteRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	ctx := r.Context()
	ip := clientIP(r, s.cfg.TrustedProxies)
	p, err := s.store.GetPending(ctx, req.P)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "This sign-up has expired. Start again from the sign-in page.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}

	username := auth.NormalizeUsername(req.Username)
	if username == "" {
		if username, err = s.freeUsername(ctx, suggestUsername(p.Display)); err != nil {
			s.serverError(w, r, err)
			return
		}
	} else if err := auth.ValidateUsername(username); err != nil {
		writeError(w, http.StatusBadRequest, sentence(err))
		return
	}
	hash := ""
	if req.Password != "" {
		if err := auth.ValidatePassword(req.Password); err != nil {
			writeError(w, http.StatusBadRequest, sentence(err))
			return
		}
		if hash, err = auth.HashPassword(req.Password); err != nil {
			s.serverError(w, r, err)
			return
		}
	}
	if !s.registerPerIP.Allow(ip) {
		writeError(w, http.StatusTooManyRequests, "Too many new accounts from your network. Try again later.")
		return
	}
	if taken, err := s.store.UsernameTaken(ctx, username); err != nil {
		s.serverError(w, r, err)
		return
	} else if taken {
		writeError(w, http.StatusConflict, "That username is already taken.")
		return
	}
	// Only one request can claim the sign-up: a double click makes one account.
	if claimed, err := s.store.ClaimPending(ctx, req.P); err != nil {
		s.serverError(w, r, err)
		return
	} else if !claimed {
		writeError(w, http.StatusConflict, "This sign-up is already finished. Sign in from the sign-in page.")
		return
	}
	restore := func() {
		if err := s.store.RestorePending(ctx, req.P, p, pendingTTL); err != nil {
			s.log.Error("restore pending sign-up", "err", err)
		}
	}

	// Someone else may have verified this address in the meantime; then leave it off.
	email := p.Email
	if email != "" {
		if taken, err := s.store.EmailInUse(ctx, email, 0); err != nil {
			restore()
			s.serverError(w, r, err)
			return
		} else if taken {
			email = ""
		}
	}
	id, err := s.store.RegisterUser(ctx, username, hash, email)
	if errors.Is(err, store.ErrUsernameTaken) {
		restore()
		writeError(w, http.StatusConflict, "That username is already taken.")
		return
	}
	if err != nil {
		restore()
		s.serverError(w, r, err)
		return
	}
	if email != "" && p.Verified {
		// Discord/Google already checked this address.
		if _, err := s.store.VerifyEmail(ctx, id, email); err != nil && !errors.Is(err, store.ErrEmailTaken) {
			s.log.Error("verify provider email", "err", err)
		}
	}
	if err := s.store.SaveIdentity(ctx, &store.Identity{
		Provider: p.Provider, Subject: p.Subject, UserID: id, Email: p.Email, Display: p.Display, Data: p.Data,
	}); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.syncGroups(ctx, id)
	if lang := i18n.Canonical(req.Lang); lang != "" {
		if err := s.store.SetLang(ctx, id, lang); err != nil {
			s.log.Error("save language", "err", err)
		}
	}

	label := s.oauth[p.Provider].label
	s.log.Info("user registered", "username", username, "ip", ip, "via", p.Provider)
	s.notifyAdmin(ctx, "New account", fmt.Sprintf("**%s** created an account with %s.", username, label))
	if err := s.startSession(w, r, id, ip); err != nil {
		s.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"redirect": s.safeRedirect(p.RD)})
}

// handleUnlinkLogin disconnects Discord/Google from the signed-in account.
func (s *Server) handleUnlinkLogin(w http.ResponseWriter, r *http.Request) {
	u := s.currentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "Please sign in.")
		return
	}
	p := s.oauth[chi.URLParam(r, "provider")]
	if p == nil {
		writeError(w, http.StatusNotFound, "Not found.")
		return
	}
	ctx := r.Context()
	idents, err := s.store.ListIdentities(ctx, u.ID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	// Never leave an account without any way to sign in.
	if u.PasswordHash == "" && len(idents) <= 1 {
		writeError(w, http.StatusBadRequest, "Set a password first. Otherwise you can't sign in anymore.")
		return
	}
	if err := s.store.DeleteIdentity(ctx, u.ID, p.name); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.syncGroups(ctx, u.ID)
	writeJSON(w, http.StatusOK, map[string]string{"message": p.label + " is disconnected.", "code": "disconnected", "provider": p.label})
}
