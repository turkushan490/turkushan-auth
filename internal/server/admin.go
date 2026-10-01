package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/turkushan490/turkushan-auth/internal/auth"
	"github.com/turkushan490/turkushan-auth/internal/notify"
	"github.com/turkushan490/turkushan-auth/internal/store"
)

type adminKey struct{}
type rightsKey struct{}

// requireStaff lets through everyone who may use the admin panel: admins and
// members of a group with rights. What they may do there is checked per action.
func (s *Server) requireStaff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := s.currentUser(r)
		if u == nil {
			writeError(w, http.StatusUnauthorized, "Please sign in.")
			return
		}
		rights, err := s.store.UserRights(r.Context(), u)
		if err != nil {
			s.serverError(w, r, err)
			return
		}
		if !rights.Staff() {
			writeError(w, http.StatusForbidden, "You don't have access to the admin panel.")
			return
		}
		ctx := context.WithValue(r.Context(), adminKey{}, u)
		ctx = context.WithValue(ctx, rightsKey{}, rights)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// adminFrom is the signed-in staff member doing the action.
func adminFrom(r *http.Request) *store.User {
	return r.Context().Value(adminKey{}).(*store.User)
}

func rightsFrom(r *http.Request) store.Rights {
	return r.Context().Value(rightsKey{}).(store.Rights)
}

const msgNoRight = "You don't have the right to do that."

// need answers 403 unless the staff member has the global right.
func (s *Server) need(w http.ResponseWriter, r *http.Request, perm string) bool {
	if rightsFrom(r).Has(perm) {
		return true
	}
	writeError(w, http.StatusForbidden, msgNoRight)
	return false
}

// needAdmin answers 403 unless the staff member is a full admin.
func (s *Server) needAdmin(w http.ResponseWriter, r *http.Request) bool {
	if rightsFrom(r).Admin {
		return true
	}
	writeError(w, http.StatusForbidden, "Only admins can do that.")
	return false
}

// audit records an admin action; a failure is logged but doesn't fail the action.
func (s *Server) audit(r *http.Request, action, target, detail string) {
	if err := s.store.Audit(r.Context(), adminFrom(r).Username, action, target, detail,
		clientIP(r, s.cfg.TrustedProxies)); err != nil {
		s.log.Error("audit log", "err", err)
	}
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	return id, err == nil && id > 0
}

// --- overview ---

type siteWithSnippet struct {
	store.Site
	Snippet string `json:"snippet"`
}

// portalAddress is how NPM reaches the portal: the admin setting, else PORTAL_INTERNAL_URL.
func (s *Server) portalAddress(ctx context.Context) (string, error) {
	v, err := s.store.Setting(ctx, store.SettingPortalURL)
	if err != nil || v != "" {
		return v, err
	}
	return s.cfg.PortalInternalURL, nil
}

// handleAdminData returns what the panel shows, limited to what this staff member may see.
func (s *Server) handleAdminData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	me, rights := adminFrom(r), rightsFrom(r)
	fail := func(err error) bool {
		if err != nil {
			s.serverError(w, r, err)
		}
		return err != nil
	}

	sites, err := s.store.ListSites(ctx)
	if fail(err) {
		return
	}
	portal, err := s.portalAddress(ctx)
	if fail(err) {
		return
	}
	siteList := make([]siteWithSnippet, len(sites))
	for i, st := range sites {
		if rights.Has(store.PermSites) {
			siteList[i] = siteWithSnippet{Site: st, Snippet: nginxSnippet(st.Upstream, portal)}
		} else {
			st.Upstream = "" // internal addresses are only for whoever manages sites
			siteList[i] = siteWithSnippet{Site: st}
		}
	}

	// Requests: only for the sites this person may approve.
	access, err := s.store.ListAccess(ctx)
	if fail(err) {
		return
	}
	visible := []store.AccessEntry{}
	for _, a := range access {
		if rights.CanApprove(a.SiteID) || rights.Has(store.PermUsers) {
			visible = append(visible, a)
		}
	}

	approveSites := []int64{}
	for id := range rights.ApproveSites {
		approveSites = append(approveSites, id)
	}
	perms := []string{}
	for _, p := range store.AllPerms {
		if rights.Has(p) {
			perms = append(perms, p)
		}
	}
	settings := map[string]any{
		"base_domain": s.redirectBase,
		"app_url":     s.cfg.AppURL.String(),
		"portal_url":  portal,
	}
	resp := map[string]any{
		"me": map[string]any{
			"id": me.ID, "username": me.Username, "admin": rights.Admin,
			"perms": perms, "approve_sites": approveSites,
		},
		"sites":       siteList,
		"access":      visible,
		"users":       []store.UserSummary{},
		"groups":      []store.Group{},
		"memberships": []store.Membership{},
		"group_sites": []store.GroupSite{},
		"rules":       []store.Rule{},
		"identities":  []store.Identity{},
		"all_perms":   store.AllPerms,
		"mail_ready":  s.mailReady(ctx),
		"settings":    settings,
	}

	if rights.Has(store.PermUsers) || rights.Has(store.PermGroups) {
		users, err := s.store.ListUsers(ctx)
		if fail(err) {
			return
		}
		groups, err := s.store.ListGroups(ctx)
		if fail(err) {
			return
		}
		memberships, err := s.store.ListMemberships(ctx)
		if fail(err) {
			return
		}
		groupSites, err := s.store.ListGroupSites(ctx)
		if fail(err) {
			return
		}
		rules, err := s.store.ListRules(ctx)
		if fail(err) {
			return
		}
		resp["users"], resp["groups"], resp["memberships"] = users, groups, memberships
		identities, err := s.store.ListAllIdentities(ctx)
		if fail(err) {
			return
		}
		resp["group_sites"], resp["rules"], resp["identities"] = groupSites, rules, identities
	}

	if rights.Has(store.PermSettings) {
		webhook, err := s.store.Setting(ctx, store.SettingDiscordWebhook)
		if fail(err) {
			return
		}
		mc, err := s.mailConfig(ctx)
		if fail(err) {
			return
		}
		logins := map[string]any{}
		for name, p := range s.oauth {
			id, secret, err := s.oauthCreds(ctx, name)
			if fail(err) {
				return
			}
			logins[name] = map[string]any{ // the secret itself is never sent back
				"label": p.label, "client_id": id, "secret_set": secret != "",
				"redirect_uri": s.oauthRedirectURI(name),
			}
		}
		settings["oauth"] = logins
		settings["discord_webhook_set"] = webhook != ""
		settings["smtp"] = map[string]any{ // the password itself is never sent back
			"host":         mc.Host,
			"port":         mc.Port,
			"username":     mc.Username,
			"from":         mc.From,
			"password_set": mc.Password != "",
			"ready":        mc.Ready(),
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminAudit(w http.ResponseWriter, r *http.Request) {
	if !s.need(w, r, store.PermAudit) {
		return
	}
	entries, err := s.store.ListAudit(r.Context(), 300)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

// nginxSnippet is the NPM "Advanced" config for one protected site.
func nginxSnippet(upstream, portal string) string {
	if portal == "" {
		portal = "http://YOUR-SERVER-IP:3010"
	}
	if upstream == "" {
		upstream = "http://YOUR-APP-IP:PORT"
	}
	return fmt.Sprintf(`location / {
  proxy_pass %s;

  auth_request /turkushan-auth;
  auth_request_set $redirection_url $upstream_http_x_auth_location;
  error_page 401 403 =302 $redirection_url;
}

location /turkushan-auth {
  internal;
  proxy_pass %s/api/auth/nginx;
  proxy_pass_request_body off;
  proxy_set_header content-length "";
  proxy_set_header x-original-url $scheme://$http_host$request_uri;
  proxy_set_header x-original-method $request_method;
  proxy_set_header x-forwarded-for $proxy_add_x_forwarded_for;
  proxy_set_header x-real-ip $remote_addr;
}
`, upstream, portal)
}

// --- sites ---

type siteInput struct {
	Name            string `json:"name"`
	Host            string `json:"host"`
	Upstream        string `json:"upstream"`
	RequireApproval bool   `json:"require_approval"`
	RequireEmail    bool   `json:"require_email"`
}

var hostRe = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z][a-z0-9-]{0,62}$`)

// validURL accepts http(s)://host[:port] with nothing that could break out of the nginx config.
func validBaseURL(raw string) bool {
	if strings.ContainsAny(raw, " \t\r\n;{}'\"$\\") {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" &&
		u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == ""
}

// cleanSite validates the form; the returned string is a message for the admin.
func (s *Server) cleanSite(in siteInput) (*store.Site, string) {
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 64 {
		return nil, "Give the site a name (max 64 characters)."
	}

	host := strings.ToLower(strings.TrimSpace(in.Host))
	host = strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://")
	host = strings.TrimSuffix(host, "/")
	if !hostRe.MatchString(host) {
		return nil, "Enter a hostname like manga." + s.redirectBase + "."
	}
	if host != s.redirectBase && !strings.HasSuffix(host, "."+s.redirectBase) {
		return nil, "The site must be under " + s.redirectBase + ", otherwise the login cookie doesn't reach it."
	}
	if host == s.cfg.AppURL.Hostname() {
		return nil, "That's the login portal itself."
	}

	upstream := strings.TrimSuffix(strings.TrimSpace(in.Upstream), "/")
	if !validBaseURL(upstream) {
		return nil, "Enter where the app runs, like http://192.168.0.6:3000."
	}
	// Pointing a site at itself (or at the portal) makes NPM loop and answer 502.
	up, _ := url.Parse(upstream)
	upHost := strings.ToLower(up.Hostname())
	if upHost == host || upHost == s.cfg.AppURL.Hostname() {
		return nil, "\"Where the app runs\" must be the app's own address (IP and port, like http://192.168.0.6:3000), not " + up.Host + "."
	}
	if portal, err := s.portalAddress(context.Background()); err == nil && portal != "" && strings.EqualFold(upstream, portal) {
		return nil, "That's the login portal's address. Enter where the app itself runs, like http://192.168.0.6:3000."
	}

	return &store.Site{
		Name: name, Host: host, Upstream: upstream,
		RequireApproval: in.RequireApproval, RequireEmail: in.RequireEmail,
	}, ""
}

func siteFlags(st *store.Site) string {
	return fmt.Sprintf("approval=%v email=%v upstream=%s", st.RequireApproval, st.RequireEmail, st.Upstream)
}

func (s *Server) handleCreateSite(w http.ResponseWriter, r *http.Request) {
	if !s.need(w, r, store.PermSites) {
		return
	}
	var in siteInput
	if !decodeJSON(w, r, &in) {
		return
	}
	st, msg := s.cleanSite(in)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if _, err := s.store.CreateSite(r.Context(), st); err != nil {
		if errors.Is(err, store.ErrHostTaken) {
			writeError(w, http.StatusConflict, st.Host+" is already added.")
			return
		}
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "site.create", st.Host, siteFlags(st))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleUpdateSite(w http.ResponseWriter, r *http.Request) {
	if !s.need(w, r, store.PermSites) {
		return
	}
	id, ok := pathID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid site.")
		return
	}
	var in siteInput
	if !decodeJSON(w, r, &in) {
		return
	}
	st, msg := s.cleanSite(in)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	st.ID = id
	switch err := s.store.UpdateSite(r.Context(), st); {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "That site no longer exists.")
		return
	case errors.Is(err, store.ErrHostTaken):
		writeError(w, http.StatusConflict, st.Host+" is already added.")
		return
	case err != nil:
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "site.update", st.Host, siteFlags(st))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteSite(w http.ResponseWriter, r *http.Request) {
	if !s.need(w, r, store.PermSites) {
		return
	}
	id, ok := pathID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid site.")
		return
	}
	st, err := s.store.SiteByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "That site no longer exists.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := s.store.DeleteSite(r.Context(), id); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "site.delete", st.Host, "")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- access ---

type accessInput struct {
	UserID int64  `json:"user_id"`
	SiteID int64  `json:"site_id"`
	Status string `json:"status"`
}

func (s *Server) handleSetAccess(w http.ResponseWriter, r *http.Request) {
	var in accessInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Status != store.AccessApproved && in.Status != store.AccessDenied {
		writeError(w, http.StatusBadRequest, "Status must be approved or denied.")
		return
	}
	if !rightsFrom(r).CanApprove(in.SiteID) {
		writeError(w, http.StatusForbidden, msgNoRight)
		return
	}
	ctx := r.Context()
	username, err := s.store.UsernameByID(ctx, in.UserID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "That user no longer exists.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	st, err := s.store.SiteByID(ctx, in.SiteID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "That site no longer exists.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := s.store.SetAccess(ctx, in.UserID, in.SiteID, in.Status, adminFrom(r).Username); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "access."+in.Status, username+" → "+st.Host, "")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- users ---

// targetUser loads the user in the URL for a user-management action. It needs
// the "users" right, never works on yourself, and only admins may touch admins.
func (s *Server) targetUser(w http.ResponseWriter, r *http.Request, verb string) *store.User {
	if !s.need(w, r, store.PermUsers) {
		return nil
	}
	id, ok := pathID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid user.")
		return nil
	}
	u, err := s.store.UserByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "That user no longer exists.")
		return nil
	}
	if err != nil {
		s.serverError(w, r, err)
		return nil
	}
	if u.ID == adminFrom(r).ID {
		writeError(w, http.StatusForbidden, "You can't "+verb+" your own account here.")
		return nil
	}
	if u.IsAdmin && !rightsFrom(r).Admin {
		writeError(w, http.StatusForbidden, "Only admins can "+verb+" an admin account.")
		return nil
	}
	return u
}

func (s *Server) handleSetUserStatus(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Status != "active" && in.Status != "blocked" {
		writeError(w, http.StatusBadRequest, "Status must be active or blocked.")
		return
	}
	u := s.targetUser(w, r, "block")
	if u == nil {
		return
	}
	if err := s.store.SetUserStatus(r.Context(), u.ID, in.Status); err != nil {
		s.serverError(w, r, err)
		return
	}
	action := "user.block"
	if in.Status == "active" {
		action = "user.unblock"
	}
	s.audit(r, action, u.Username, "")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleSetUserPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := auth.ValidatePassword(in.Password); err != nil {
		writeError(w, http.StatusBadRequest, sentence(err))
		return
	}
	u := s.targetUser(w, r, "reset the password of")
	if u == nil {
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := s.store.SetPassword(r.Context(), u.ID, hash); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "user.password_reset", u.Username, "")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	u := s.targetUser(w, r, "delete")
	if u == nil {
		return
	}
	if err := s.store.DeleteUser(r.Context(), u.ID); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "user.delete", u.Username, "")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- settings ---

type settingsInput struct {
	PortalURL      *string `json:"portal_url"`
	DiscordWebhook *string `json:"discord_webhook"` // "" removes it
	mailSettingsInput
	// OAuth is keyed by provider (discord, google). An empty secret keeps the saved one.
	OAuth map[string]struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Remove       bool   `json:"remove"`
	} `json:"oauth"`
}

func (in settingsInput) hasMail() bool {
	m := in.mailSettingsInput
	return m.Host != nil || m.Port != nil || m.Username != nil || m.Password != nil || m.From != nil
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	if !s.need(w, r, store.PermSettings) {
		return
	}
	var in settingsInput
	if !decodeJSON(w, r, &in) {
		return
	}
	ctx := r.Context()
	if in.PortalURL != nil {
		v := strings.TrimSuffix(strings.TrimSpace(*in.PortalURL), "/")
		if v != "" && !validBaseURL(v) {
			writeError(w, http.StatusBadRequest, "Enter the portal address like http://192.168.0.6:3010.")
			return
		}
		if err := s.store.SetSetting(ctx, store.SettingPortalURL, v); err != nil {
			s.serverError(w, r, err)
			return
		}
		s.audit(r, "settings.portal_url", v, "")
	}
	if in.DiscordWebhook != nil {
		v := strings.TrimSpace(*in.DiscordWebhook)
		if v != "" && !notify.ValidDiscordWebhook(v) {
			writeError(w, http.StatusBadRequest, "That is not a Discord webhook URL. It looks like https://discord.com/api/webhooks/…")
			return
		}
		if err := s.store.SetSetting(ctx, store.SettingDiscordWebhook, v); err != nil {
			s.serverError(w, r, err)
			return
		}
		detail := "set"
		if v == "" {
			detail = "removed"
		}
		s.audit(r, "settings.discord_webhook", "", detail)
	}
	if in.hasMail() {
		msg, err := s.applyMailSettings(ctx, in.mailSettingsInput)
		if err != nil {
			s.serverError(w, r, err)
			return
		}
		if msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		s.audit(r, "settings.mail", "", "")
	}
	for name, o := range in.OAuth {
		p := s.oauth[name]
		if p == nil {
			writeError(w, http.StatusBadRequest, "Unknown login method.")
			return
		}
		id, secret := strings.TrimSpace(o.ClientID), strings.TrimSpace(o.ClientSecret)
		if o.Remove {
			id, secret = "", ""
		} else if id == "" || len(id) > 200 || len(secret) > 200 || strings.ContainsAny(id+secret, " \t\r\n") {
			writeError(w, http.StatusBadRequest, "Fill in the client ID from "+p.label+".")
			return
		}
		if err := s.store.SetSetting(ctx, oauthIDKey(name), id); err != nil {
			s.serverError(w, r, err)
			return
		}
		if secret != "" || o.Remove {
			if err := s.store.SetSetting(ctx, oauthSecretKey(name), secret); err != nil {
				s.serverError(w, r, err)
				return
			}
		}
		detail := "set up"
		if o.Remove {
			detail = "removed"
		} else if err := s.ensureProviderGroup(ctx, p); err != nil {
			s.log.Error("create login group", "provider", name, "err", err)
		}
		s.audit(r, "settings.login", p.label, detail)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleTestDiscord(w http.ResponseWriter, r *http.Request) {
	if !s.need(w, r, store.PermSettings) {
		return
	}
	webhook, err := s.store.Setting(r.Context(), store.SettingDiscordWebhook)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if webhook == "" {
		writeError(w, http.StatusBadRequest, "Save a Discord webhook first.")
		return
	}
	err = notify.Discord(r.Context(), webhook, "Test message",
		"Notifications from your login portal arrive here.", s.portalURL("/admin", nil))
	if err != nil {
		writeError(w, http.StatusBadGateway, "Discord did not accept the message: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
