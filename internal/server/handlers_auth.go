package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/turkushan490/turkushan-auth/internal/auth"
	"github.com/turkushan490/turkushan-auth/internal/store"
)

const msgBadLogin = "Wrong username or password."

func userJSON(u *store.User) map[string]any {
	return map[string]any{
		"username":       u.Username,
		"is_admin":       u.IsAdmin,
		"email":          u.Email,
		"email_verified": u.EmailVerified,
	}
}

// handleSession tells the frontend who is logged in and hands out the CSRF cookie.
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	s.ensureCSRF(w, r)
	resp := map[string]any{
		"app_url":       s.cfg.AppURL.String(),
		"brand":         s.redirectBase,
		"authenticated": false,
	}
	if u := s.currentUser(r); u != nil {
		resp["authenticated"] = true
		resp["user"] = userJSON(u)
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
	if u.LockedUntil > time.Now().Unix() {
		writeError(w, http.StatusLocked, "Too many wrong passwords. This account is locked for 15 minutes.")
		return
	}

	ok, err := auth.VerifyPassword(req.Password, u.PasswordHash)
	if err != nil {
		s.serverError(w, r, err)
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

	s.log.Info("user registered", "username", username, "ip", ip)
	s.notifyAdmin(ctx, "New account", fmt.Sprintf("**%s** created an account.", username))
	if err := s.startSession(w, r, id, ip); err != nil {
		s.serverError(w, r, err)
		return
	}
	// Best effort: the account works without it, and the user can send it again later.
	if email != "" && s.mailReady(ctx) {
		if err := s.sendVerification(ctx, &store.User{ID: id, Username: username}, email, req.RD); err != nil {
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
