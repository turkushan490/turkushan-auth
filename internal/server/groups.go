package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/turkushan490/turkushan-auth/internal/auth"
	"github.com/turkushan490/turkushan-auth/internal/store"
)

const inviteTTL = 72 * time.Hour

// mayManageGroup reports whether a staff member may rename, delete or change
// the members and rules of a group. Non-admins only may when they hold every
// right the group gives, so nobody can hand out more than they have themselves.
func (s *Server) mayManageGroup(ctx context.Context, rights store.Rights, g *store.Group) (bool, error) {
	if rights.Admin {
		return true, nil
	}
	for _, p := range g.Perms {
		if !rights.Has(p) {
			return false, nil
		}
	}
	groupSites, err := s.store.ListGroupSites(ctx)
	if err != nil {
		return false, err
	}
	for _, gs := range groupSites {
		if gs.GroupID == g.ID && !rights.CanApprove(gs.SiteID) {
			return false, nil
		}
	}
	return true, nil
}

// targetGroup loads the group in the URL for someone with the "groups" right who may manage it.
func (s *Server) targetGroup(w http.ResponseWriter, r *http.Request) *store.Group {
	if !s.need(w, r, store.PermGroups) {
		return nil
	}
	id, ok := pathID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid group.")
		return nil
	}
	g, err := s.store.GroupByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "That group no longer exists.")
		return nil
	}
	if err != nil {
		s.serverError(w, r, err)
		return nil
	}
	ok, err = s.mayManageGroup(r.Context(), rightsFrom(r), g)
	if err != nil {
		s.serverError(w, r, err)
		return nil
	}
	if !ok {
		writeError(w, http.StatusForbidden, "This group has rights you don't have yourself, so you can't change it.")
		return nil
	}
	return g
}

type groupInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func cleanGroup(in groupInput) (name, description, msg string) {
	name = strings.TrimSpace(in.Name)
	description = strings.TrimSpace(in.Description)
	switch {
	case name == "" || utf8.RuneCountInString(name) > 40:
		return "", "", "Give the group a name (max 40 characters)."
	case strings.EqualFold(name, "admins") || strings.EqualFold(name, "admin"):
		return "", "", "\"Admins\" is the built-in group. Pick another name."
	case utf8.RuneCountInString(description) > 200:
		return "", "", "The description can be at most 200 characters."
	}
	return name, description, ""
}

func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	if !s.need(w, r, store.PermGroups) {
		return
	}
	var in groupInput
	if !decodeJSON(w, r, &in) {
		return
	}
	name, description, msg := cleanGroup(in)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	id, err := s.store.CreateGroup(r.Context(), name, description)
	if errors.Is(err, store.ErrGroupNameTaken) {
		writeError(w, http.StatusConflict, "There is already a group with that name.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "group.create", name, "")
	writeJSON(w, http.StatusOK, map[string]any{"id": id})
}

func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	g := s.targetGroup(w, r)
	if g == nil {
		return
	}
	var in groupInput
	if !decodeJSON(w, r, &in) {
		return
	}
	name, description, msg := cleanGroup(in)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	err := s.store.UpdateGroup(r.Context(), g.ID, name, description)
	if errors.Is(err, store.ErrGroupNameTaken) {
		writeError(w, http.StatusConflict, "There is already a group with that name.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "group.update", name, "")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	g := s.targetGroup(w, r)
	if g == nil {
		return
	}
	if err := s.store.DeleteGroup(r.Context(), g.ID); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "group.delete", g.Name, "")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleSetGroupPerms changes which global rights a group gives. Admins only:
// rights are what the whole permission system hangs on.
func (s *Server) handleSetGroupPerms(w http.ResponseWriter, r *http.Request) {
	if !s.needAdmin(w, r) {
		return
	}
	g := s.targetGroup(w, r)
	if g == nil {
		return
	}
	var in struct {
		Perms []string `json:"perms"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := s.store.SetGroupPerms(r.Context(), g.ID, in.Perms); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "group.rights", g.Name, strings.Join(in.Perms, ","))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleSetGroupSite changes what a group may do with one site. Admins only.
func (s *Server) handleSetGroupSite(w http.ResponseWriter, r *http.Request) {
	if !s.needAdmin(w, r) {
		return
	}
	g := s.targetGroup(w, r)
	if g == nil {
		return
	}
	var in struct {
		SiteID     int64 `json:"site_id"`
		CanOpen    bool  `json:"can_open"`
		CanApprove bool  `json:"can_approve"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	site, err := s.store.SiteByID(r.Context(), in.SiteID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "That site no longer exists.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := s.store.SetGroupSite(r.Context(), g.ID, site.ID, in.CanOpen, in.CanApprove); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "group.site", g.Name+" → "+site.Host, fmt.Sprintf("open=%v approve=%v", in.CanOpen, in.CanApprove))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleSetGroupMember(w http.ResponseWriter, r *http.Request) {
	g := s.targetGroup(w, r)
	if g == nil {
		return
	}
	var in struct {
		UserID int64 `json:"user_id"`
		Member bool  `json:"member"`
	}
	if !decodeJSON(w, r, &in) {
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
	action := "group.member_add"
	if in.Member {
		err = s.store.AddMember(ctx, in.UserID, g.ID)
	} else {
		action = "group.member_remove"
		err = s.store.RemoveMember(ctx, in.UserID, g.ID)
		if err == nil {
			// A rule may put them right back; that's then shown as "by rule".
			err = s.store.SyncRuleGroups(ctx, in.UserID)
		}
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.audit(r, action, username+" → "+g.Name, "")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- auto-add rules ---

var (
	discordIDRe   = regexp.MustCompile(`^[0-9]{5,25}$`)
	discordRoleRe = regexp.MustCompile(`^[0-9]{5,25}:[0-9]{5,25}$`)
)

// cleanRule validates a rule and normalizes its value; msg is for the admin.
func cleanRule(kind, value string) (string, string) {
	value = strings.TrimSpace(value)
	switch kind {
	case store.RuleEveryone:
		return "", ""
	case store.RuleMethod:
		if value == "password" || value == "discord" || value == "google" {
			return value, ""
		}
		return "", "Choose a login method."
	case store.RuleEmail:
		email, err := auth.NormalizeEmail(value)
		if err != nil || email == "" {
			return "", "Enter a full email address."
		}
		return email, ""
	case store.RuleDomain:
		domain := strings.ToLower(strings.TrimPrefix(value, "@"))
		if !hostRe.MatchString(domain) {
			return "", "Enter a domain like example.com."
		}
		return domain, ""
	case store.RuleDiscordServer:
		if !discordIDRe.MatchString(value) {
			return "", "Enter the Discord server ID (a long number)."
		}
		return value, ""
	case store.RuleDiscordRole:
		if !discordRoleRe.MatchString(value) {
			return "", "Enter the Discord server ID and role ID (two long numbers)."
		}
		return value, ""
	}
	return "", "Unknown kind of rule."
}

func (s *Server) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	g := s.targetGroup(w, r)
	if g == nil {
		return
	}
	var in struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	value, msg := cleanRule(in.Kind, in.Value)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	ctx := r.Context()
	if _, err := s.store.CreateRule(ctx, g.ID, in.Kind, value); err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := s.store.SyncAllRuleGroups(ctx); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "group.rule_add", g.Name, strings.TrimSpace(in.Kind+" "+value))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	if !s.need(w, r, store.PermGroups) {
		return
	}
	id, ok := pathID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid rule.")
		return
	}
	ctx := r.Context()
	rule, err := s.store.RuleByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "That rule no longer exists.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	g, err := s.store.GroupByID(ctx, rule.GroupID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if ok, err := s.mayManageGroup(ctx, rightsFrom(r), g); err != nil {
		s.serverError(w, r, err)
		return
	} else if !ok {
		writeError(w, http.StatusForbidden, "This group has rights you don't have yourself, so you can't change it.")
		return
	}
	if err := s.store.DeleteRule(ctx, id); err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := s.store.SyncAllRuleGroups(ctx); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "group.rule_remove", g.Name, strings.TrimSpace(rule.Kind+" "+rule.Value))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- users: create, admin ---

type createUserInput struct {
	Username string  `json:"username"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Invite   bool    `json:"invite"` // mail a link to choose their own password instead
	GroupIDs []int64 `json:"group_ids"`
	Admin    bool    `json:"admin"`
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if !s.need(w, r, store.PermUsers) {
		return
	}
	var in createUserInput
	if !decodeJSON(w, r, &in) {
		return
	}
	ctx := r.Context()
	rights := rightsFrom(r)

	username := auth.NormalizeUsername(in.Username)
	if err := auth.ValidateUsername(username); err != nil {
		writeError(w, http.StatusBadRequest, sentence(err))
		return
	}
	email, err := auth.NormalizeEmail(in.Email)
	if err != nil {
		writeError(w, http.StatusBadRequest, sentence(err))
		return
	}
	if in.Admin && !rights.Admin {
		writeError(w, http.StatusForbidden, "Only admins can create admin accounts.")
		return
	}

	hash := "" // no password yet: they choose one via the invite link
	if in.Invite {
		if email == "" {
			writeError(w, http.StatusBadRequest, "An invite needs their email address.")
			return
		}
		if !s.mailReady(ctx) {
			writeError(w, http.StatusBadRequest, "Email isn't set up yet (Settings → Email). Set a password yourself instead.")
			return
		}
	} else {
		if err := auth.ValidatePassword(in.Password); err != nil {
			writeError(w, http.StatusBadRequest, sentence(err))
			return
		}
		if hash, err = auth.HashPassword(in.Password); err != nil {
			s.serverError(w, r, err)
			return
		}
	}

	// Check the groups before creating anything.
	var groups []*store.Group
	if len(in.GroupIDs) > 0 && !rights.Has(store.PermGroups) {
		writeError(w, http.StatusForbidden, "You don't have the right to put users in groups.")
		return
	}
	for _, id := range in.GroupIDs {
		g, err := s.store.GroupByID(ctx, id)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "One of the groups no longer exists.")
			return
		}
		if err != nil {
			s.serverError(w, r, err)
			return
		}
		if ok, err := s.mayManageGroup(ctx, rights, g); err != nil {
			s.serverError(w, r, err)
			return
		} else if !ok {
			writeError(w, http.StatusForbidden, "You can't add users to "+g.Name+": it has rights you don't have yourself.")
			return
		}
		groups = append(groups, g)
	}

	id, err := s.store.RegisterUser(ctx, username, hash, email)
	switch {
	case errors.Is(err, store.ErrUsernameTaken):
		writeError(w, http.StatusConflict, "That username is already taken.")
		return
	case errors.Is(err, store.ErrEmailTaken):
		writeError(w, http.StatusConflict, "That email address is already used by another account.")
		return
	case err != nil:
		s.serverError(w, r, err)
		return
	}
	if in.Admin {
		if err := s.store.SetAdmin(ctx, id, true); err != nil {
			s.serverError(w, r, err)
			return
		}
	}
	names := []string{}
	for _, g := range groups {
		if err := s.store.AddMember(ctx, id, g.ID); err != nil {
			s.serverError(w, r, err)
			return
		}
		names = append(names, g.Name)
	}
	if err := s.store.SyncRuleGroups(ctx, id); err != nil {
		s.serverError(w, r, err)
		return
	}
	detail := "password set by " + adminFrom(r).Username
	if in.Invite {
		detail = "invite mailed to " + email
	}
	if in.Admin {
		detail += ", admin"
	}
	if len(names) > 0 {
		detail += ", groups: " + strings.Join(names, ", ")
	}
	s.audit(r, "user.create", username, detail)

	if in.Invite {
		token, err := s.store.CreateToken(ctx, id, store.TokenResetPassword, email, inviteTTL)
		if err == nil {
			err = s.sendMail(ctx, email, mailContent{
				Title:  "You're invited",
				Intro:  fmt.Sprintf("%s made an account for you on %s. Your username is %s. Click the button to choose your password.", adminFrom(r).Username, s.redirectBase, username),
				Button: "Choose my password",
				Link:   s.portalURL("/reset", url.Values{"token": {token}}) + "&invite=1",
				Footer: "The link works for 3 days and only once.",
			})
		}
		if err != nil {
			s.log.Error("send invite", "err", err)
			writeJSON(w, http.StatusOK, map[string]any{"id": id, "warning": "The account is created, but the invite mail could not be sent. Use Reset password to set one yourself."})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id})
}

// handleSetAdmin puts a user in, or takes them out of, the built-in Admins group.
func (s *Server) handleSetAdmin(w http.ResponseWriter, r *http.Request) {
	if !s.needAdmin(w, r) {
		return
	}
	var in struct {
		Admin bool `json:"admin"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	id, ok := pathID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid user.")
		return
	}
	ctx := r.Context()
	u, err := s.store.UserByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "That user no longer exists.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if u.ID == adminFrom(r).ID {
		writeError(w, http.StatusForbidden, "You can't remove your own admin rights. Ask another admin.")
		return
	}
	if u.Status != "active" && in.Admin {
		writeError(w, http.StatusBadRequest, "Unblock this user before making them admin.")
		return
	}
	if err := s.store.SetAdmin(ctx, u.ID, in.Admin); err != nil {
		s.serverError(w, r, err)
		return
	}
	action := "admin.promote"
	if !in.Admin {
		action = "admin.demote"
	}
	s.audit(r, action, u.Username, "")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
