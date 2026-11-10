package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/turkushan490/turkushan-auth/internal/auth"
	"github.com/turkushan490/turkushan-auth/internal/i18n"
	"github.com/turkushan490/turkushan-auth/internal/store"
)

const msgBadLogin = "Wrong username or password."

func userJSON(u *store.User) map[string]any {
	return map[string]any{
		"username":       u.Username,
		"is_admin":       u.IsAdmin,
		"email":          u.Email,
		"email_verified": u.EmailVerified,
		"has_password":   u.PasswordHash != "",
		"lang":           u.Lang,
	}
}

// handleSetLanguage stores the language picked in the web app, so the mails
// the portal sends this person are in that language too. "" = follow the browser.
func (s *Server) handleSetLanguage(w http.ResponseWriter, r *http.Request) {
	u := s.currentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "Please sign in.")
		return
	}
	var req struct {
		Lang string `json:"lang"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	lang := i18n.Canonical(req.Lang)
	if req.Lang != "" && lang == "" {
		writeError(w, http.StatusBadRequest, "Unknown language.")
		return
	}
	if err := s.store.SetLang(r.Context(), u.ID, lang); err != nil {
		s.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"lang": lang})
}

// handleSession tells the frontend who is logged in and hands out the CSRF cookie.
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	s.ensureCSRF(w, r)
	resp := map[string]any{
		"app_url":       s.cfg.AppURL.String(),
		"brand":         s.redirectBase,
		"authenticated": false,
	}
	a, err := s.loadAppearance(r.Context())
	if err != nil {
		s.log.Error("load appearance", "err", err)
	}
	resp["appearance"] = s.appearanceJSON(a)
	resp["providers"] = s.enabledProviders(r.Context())
	if u := s.currentUser(r); u != nil {
		resp["authenticated"] = true
		user := userJSON(u)
		// staff: may open the admin panel (admins and members of a group with rights).
		if rights, err := s.store.UserRights(r.Context(), u); err != nil {
			s.log.Error("load rights", "err", err)
		} else {
			user["staff"] = rights.Staff()
		}
		logins := []string{}
		if idents, err := s.store.ListIdentities(r.Context(), u.ID); err != nil {
			s.log.Error("load logins", "err", err)
		} else {
			for _, i := range idents {
				logins = append(logins, i.Provider)
			}
		}
		user["logins"] = logins
		resp["user"] = user
	}
	writeJSON(w, http.StatusOK, resp)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	RD       string `json:"rd"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	ctx := r.Context()
	ip := clientIP(r, s.cfg.TrustedProxies)
	username := auth.NormalizeUsername(req.Username)

	if !s.loginPerIP.Allow(ip) || !s.loginPerUser.Allow(username) {
		writeError(w, http.StatusTooManyRequests, "Too many attempts. Wait a minute and try again.")
		return
	}

	u, err := s.store.UserByUsername(ctx, username)
	if errors.Is(err, store.ErrNotFound) {
		auth.DummyVerify(req.Password)
		writeError(w, http.StatusUnauthorized, msgBadLogin)
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	locked := u.LockedUntil > time.Now().Unix()

	ok := false
	if u.PasswordHash == "" {
		// No password set (invited, or signs in with Discord/Google): never matches.
		auth.DummyVerify(req.Password)
	} else if ok, err = auth.VerifyPassword(req.Password, u.PasswordHash); err != nil {
		s.serverError(w, r, err)
		return
	}
	// A locked account answers like a wrong password, unless the password is
	// right: only the owner learns about the lock, and guessers can't use it to
	// find out which usernames exist.
	if locked {
		if ok {
			writeError(w, http.StatusLocked, "Too many wrong passwords. This account is locked for 15 minutes.")
		} else {
			writeError(w, http.StatusUnauthorized, msgBadLogin)
		}
		return
	}
	if !ok {
		locked, err := s.store.RecordLoginFailure(ctx, u.ID)
		if err != nil {
			s.serverError(w, r, err)
			return
		}
		if locked {
			s.log.Warn("account locked after failed logins", "username", u.Username, "ip", ip)
		}
		writeError(w, http.StatusUnauthorized, msgBadLogin)
		return
	}

	// Checked after the password, so a blocked status is not revealed to guessers.
	if u.Status != "active" {
		writeError(w, http.StatusForbidden, "This account is blocked.")
		return
	}

	if err := s.store.RecordLoginSuccess(ctx, u.ID); err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := s.startSession(w, r, u.ID, ip); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.log.Info("login", "username", u.Username, "ip", ip)
	writeJSON(w, http.StatusOK, map[string]string{"redirect": s.safeRedirect(req.RD)})
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	RD       string `json:"rd"`
	Lang     string `json:"lang"` // the language picked in the web app, if any
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	ctx := r.Context()
	ip := clientIP(r, s.cfg.TrustedProxies)

	username := auth.NormalizeUsername(req.Username)
	if err := auth.ValidateUsername(username); err != nil {
		writeError(w, http.StatusBadRequest, sentence(err))
		return
	}
	if err := auth.ValidatePassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, sentence(err))
		return
	}
	email, err := auth.NormalizeEmail(req.Email)
	if err != nil {
		writeError(w, http.StatusBadRequest, sentence(err))
		return
	}
	if !s.registerPerIP.Allow(ip) {
		writeError(w, http.StatusTooManyRequests, "Too many new accounts from your network. Try again later.")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.serverError(w, r, err)
		return
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

	lang := i18n.Canonical(req.Lang)
	if lang != "" {
		if err := s.store.SetLang(ctx, id, lang); err != nil {
			s.log.Error("save language", "err", err)
		}
	}
	s.log.Info("user registered", "username", username, "ip", ip)
	if err := s.store.SyncRuleGroups(ctx, id); err != nil {
		s.log.Error("apply group rules", "err", err)
	}
	s.notifyAdmin(ctx, "New account", fmt.Sprintf("**%s** created an account.", username))
	if err := s.startSession(w, r, id, ip); err != nil {
		s.serverError(w, r, err)
		return
	}
	// Best effort: the account works without it, and the user can send it again later.
	if email != "" && s.mailReady(ctx) {
		newUser := &store.User{ID: id, Username: username, Lang: lang}
		if err := s.sendVerification(ctx, newUser, email, req.RD, s.mailLang(ctx, newUser, r)); err != nil {
			s.log.Warn("verification mail after sign-up failed", "err", err)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"redirect": s.safeRedirect(req.RD)})
}

// handleLogout deletes the session server-side, so the cookie is dead on every subdomain.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		if err := s.store.DeleteSession(r.Context(), c.Value); err != nil {
			s.serverError(w, r, err)
			return
		}
	}
	s.setSessionCookie(w, "", -1)
	writeJSON(w, http.StatusOK, map[string]string{"redirect": "/login"})
}

// sentence turns a Go error ("password needs ...") into a user message ("Password needs ....").
func sentence(err error) string {
	msg := err.Error()
	r, size := utf8.DecodeRuneInString(msg)
	msg = string(unicode.ToUpper(r)) + msg[size:]
	if last, _ := utf8.DecodeLastRuneInString(msg); last != '.' && last != '?' && last != '!' {
		msg += "."
	}
	return msg
}
