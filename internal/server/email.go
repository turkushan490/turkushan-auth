package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/turkushan490/turkushan-auth/internal/auth"
	"github.com/turkushan490/turkushan-auth/internal/mail"
	"github.com/turkushan490/turkushan-auth/internal/store"
)

const (
	verifyTTL = 24 * time.Hour
	resetTTL  = time.Hour
)

func (s *Server) mailConfig(ctx context.Context) (mail.Config, error) {
	var c mail.Config
	var port string
	for key, dst := range map[string]*string{
		store.SettingSMTPHost:     &c.Host,
		store.SettingSMTPPort:     &port,
		store.SettingSMTPUsername: &c.Username,
		store.SettingSMTPPassword: &c.Password,
		store.SettingSMTPFrom:     &c.From,
	} {
		v, err := s.store.Setting(ctx, key)
		if err != nil {
			return c, err
		}
		*dst = v
	}
	c.Port, _ = strconv.Atoi(port)
	return c, nil
}

var mailHTML = template.Must(template.New("mail").Parse(`<!doctype html>
<html><body style="margin:0;padding:24px;background:#f4f4f5;font-family:-apple-system,Segoe UI,Roboto,sans-serif;color:#18181b">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:480px;background:#ffffff;border-radius:12px;padding:32px">
<tr><td>
<p style="margin:0 0 4px;font-size:13px;color:#71717a">{{.Brand}}</p>
<h1 style="margin:0 0 16px;font-size:20px">{{.Title}}</h1>
<p style="margin:0 0 24px;font-size:15px;line-height:1.5">{{.Intro}}</p>
{{if .Link}}<p style="margin:0 0 24px"><a href="{{.Link}}" style="display:inline-block;background:#6366f1;color:#ffffff;text-decoration:none;font-weight:600;padding:12px 20px;border-radius:8px">{{.Button}}</a></p>
<p style="margin:0 0 8px;font-size:13px;color:#71717a">Or copy this link:</p>
<p style="margin:0 0 24px;font-size:13px;word-break:break-all"><a href="{{.Link}}" style="color:#6366f1">{{.Link}}</a></p>{{end}}
<p style="margin:0;font-size:13px;color:#71717a">{{.Footer}}</p>
</td></tr></table></td></tr></table></body></html>`))

type mailContent struct {
	Brand, Title, Intro, Button, Link, Footer string
}

func (s *Server) sendMail(ctx context.Context, to string, c mailContent) error {
	cfg, err := s.mailConfig(ctx)
	if err != nil {
		return err
	}
	c.Brand = s.redirectBase
	var html bytes.Buffer
	if err := mailHTML.Execute(&html, c); err != nil {
		return err
	}
	text := c.Intro + "\n\n"
	if c.Link != "" {
		text += c.Link + "\n\n"
	}
	text += c.Footer + "\n"
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return mail.Send(ctx, cfg, mail.Message{To: to, Subject: c.Title, Text: text, HTML: html.String()})
}

// syncGroups re-applies the auto-add rules to one user; a failure is only logged.
func (s *Server) syncGroups(ctx context.Context, userID int64) {
	if err := s.store.SyncRuleGroups(ctx, userID); err != nil {
		s.log.Error("apply group rules", "user_id", userID, "err", err)
	}
}

func (s *Server) mailReady(ctx context.Context) bool {
	cfg, err := s.mailConfig(ctx)
	return err == nil && cfg.Ready()
}

// sendVerification mails the confirmation link. rd (optional) is the site the
// user was on their way to; after verifying they continue there.
func (s *Server) sendVerification(ctx context.Context, u *store.User, email, rd string) error {
	token, err := s.store.CreateToken(ctx, u.ID, store.TokenVerifyEmail, email, verifyTTL)
	if err != nil {
		return err
	}
	link := s.portalURL("/verify", url.Values{"token": {token}})
	if safe := auth.SafeRedirect(rd, s.redirectBase, s.cfg.AppURL.Scheme == "http"); safe != "" {
		link += "&rd=" + url.QueryEscape(safe)
	}
	return s.sendMail(ctx, email, mailContent{
		Title:  "Verify your email address",
		Intro:  fmt.Sprintf("Hi %s, click the button to confirm this is your email address for %s.", u.Username, s.redirectBase),
		Button: "Verify email",
		Link:   link,
		Footer: "The link works for 24 hours. Didn't ask for this? Then you can ignore this mail.",
	})
}

// --- signed-in user: email + password ---

type emailRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	RD       string `json:"rd"`
}

// handleSetEmail adds, changes or removes the user's address. Replacing or
// removing a verified address asks for the password, because that address can
// reset the password; adding a first one (nothing to take over yet) does not.
func (s *Server) handleSetEmail(w http.ResponseWriter, r *http.Request) {
	u := s.currentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "Please sign in.")
		return
	}
	var req emailRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	ctx := r.Context()
	if !s.mailPerUser.Allow(strconv.FormatInt(u.ID, 10)) {
		writeError(w, http.StatusTooManyRequests, "Too many email changes. Try again in an hour.")
		return
	}
	if u.EmailVerified {
		if ok, err := auth.VerifyPassword(req.Password, u.PasswordHash); err != nil || !ok {
			writeError(w, http.StatusUnauthorized, "Wrong password.")
			return
		}
	}
	email, err := auth.NormalizeEmail(req.Email)
	if err != nil {
		writeError(w, http.StatusBadRequest, sentence(err))
		return
	}

	if email == "" {
		if err := s.store.SetEmail(ctx, u.ID, ""); err != nil {
			s.serverError(w, r, err)
			return
		}
		s.syncGroups(ctx, u.ID)
		writeJSON(w, http.StatusOK, map[string]string{"message": "Email address removed."})
		return
	}
	if email == u.Email && u.EmailVerified {
		writeJSON(w, http.StatusOK, map[string]string{"message": "That address is already verified."})
		return
	}
	if !s.mailReady(ctx) {
		writeError(w, http.StatusServiceUnavailable, "Email isn't set up on this portal yet. Ask the admin.")
		return
	}
	if taken, err := s.store.EmailInUse(ctx, email, u.ID); err != nil {
		s.serverError(w, r, err)
		return
	} else if taken {
		writeError(w, http.StatusConflict, "That email address is already used by another account.")
		return
	}
	if err := s.store.SetEmail(ctx, u.ID, email); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.syncGroups(ctx, u.ID)
	if err := s.sendVerification(ctx, u, email, req.RD); err != nil {
		s.log.Error("send verification mail", "err", err)
		writeError(w, http.StatusBadGateway, "Your address is saved, but the mail could not be sent. Try \"Send again\" later.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Check your inbox for " + email + " and click the link."})
}

func (s *Server) handleResendVerification(w http.ResponseWriter, r *http.Request) {
	u := s.currentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "Please sign in.")
		return
	}
	var req struct {
		RD string `json:"rd"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if u.Email == "" || u.EmailVerified {
		writeError(w, http.StatusBadRequest, "There is no unverified email address to send to.")
		return
	}
	if !s.mailPerUser.Allow(strconv.FormatInt(u.ID, 10)) {
		writeError(w, http.StatusTooManyRequests, "Too many mails sent. Try again in an hour.")
		return
	}
	if !s.mailReady(r.Context()) {
		writeError(w, http.StatusServiceUnavailable, "Email isn't set up on this portal yet. Ask the admin.")
		return
	}
	if err := s.sendVerification(r.Context(), u, u.Email, req.RD); err != nil {
		s.log.Error("send verification mail", "err", err)
		writeError(w, http.StatusBadGateway, "The mail could not be sent. Try again later.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Sent. Check your inbox for " + u.Email + "."})
}

type changePasswordRequest struct {
	Current string `json:"current"`
	New     string `json:"new"`
}

// handleChangePassword signs the user out everywhere else and keeps this browser signed in.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	u := s.currentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "Please sign in.")
		return
	}
	var req changePasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !s.loginPerUser.Allow(u.Username) {
		writeError(w, http.StatusTooManyRequests, "Too many attempts. Wait a minute and try again.")
		return
	}
	if u.PasswordHash != "" {
		if ok, err := auth.VerifyPassword(req.Current, u.PasswordHash); err != nil || !ok {
			writeError(w, http.StatusUnauthorized, "Your current password is wrong.")
			return
		}
	}
	if err := auth.ValidatePassword(req.New); err != nil {
		writeError(w, http.StatusBadRequest, sentence(err))
		return
	}
	hash, err := auth.HashPassword(req.New)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := s.store.SetPassword(r.Context(), u.ID, hash); err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := s.startSession(w, r, u.ID, clientIP(r, s.cfg.TrustedProxies)); err != nil {
		s.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Password changed. Other devices were signed out."})
}

// --- links from mails ---

type tokenRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (s *Server) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !s.tokenPerIP.Allow(clientIP(r, s.cfg.TrustedProxies)) {
		writeError(w, http.StatusTooManyRequests, "Too many attempts. Try again later.")
		return
	}
	ctx := r.Context()
	userID, email, err := s.store.ConsumeToken(ctx, req.Token, store.TokenVerifyEmail)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "This link is invalid, already used or expired. Request a new one on your account page.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	ok, err := s.store.VerifyEmail(ctx, userID, email)
	if errors.Is(err, store.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "Another account already verified this email address.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if !ok {
		writeError(w, http.StatusBadRequest, "This link is for an email address you no longer use.")
		return
	}
	s.syncGroups(ctx, userID)
	s.log.Info("email verified", "user_id", userID)
	writeJSON(w, http.StatusOK, map[string]string{"email": email})
}

type forgotRequest struct {
	Login string `json:"login"` // username or email
}

// handleForgotPassword always answers the same, so it can't be used to find out
// which usernames or addresses exist.
func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	login := strings.ToLower(strings.TrimSpace(req.Login))
	if !s.forgotPerIP.Allow(clientIP(r, s.cfg.TrustedProxies)) || !s.forgotPerLogin.Allow(login) {
		writeError(w, http.StatusTooManyRequests, "Too many requests. Try again in an hour.")
		return
	}
	done := map[string]string{"message": "If that account has a verified email address, a reset link is on its way. No mail? Ask the admin to reset your password."}

	ctx := r.Context()
	var u *store.User
	var err error
	if strings.Contains(login, "@") {
		u, err = s.store.UserByVerifiedEmail(ctx, login)
	} else {
		u, err = s.store.UserByUsername(ctx, auth.NormalizeUsername(login))
	}
	if err != nil || !u.EmailVerified || u.Status != "active" {
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			s.log.Error("forgot password lookup", "err", err)
		}
		writeJSON(w, http.StatusOK, done)
		return
	}

	token, err := s.store.CreateToken(ctx, u.ID, store.TokenResetPassword, u.Email, resetTTL)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	err = s.sendMail(ctx, u.Email, mailContent{
		Title:  "Reset your password",
		Intro:  fmt.Sprintf("Hi %s, someone (hopefully you) asked to reset your password for %s. Click the button to choose a new one.", u.Username, s.redirectBase),
		Button: "Choose a new password",
		Link:   s.portalURL("/reset", url.Values{"token": {token}}),
		Footer: "The link works for 1 hour and only once. Didn't ask for this? Ignore this mail; your password stays the same.",
	})
	if err != nil {
		s.log.Error("send reset mail", "err", err)
	} else {
		s.log.Info("password reset mail sent", "username", u.Username)
	}
	writeJSON(w, http.StatusOK, done)
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !s.tokenPerIP.Allow(clientIP(r, s.cfg.TrustedProxies)) {
		writeError(w, http.StatusTooManyRequests, "Too many attempts. Try again later.")
		return
	}
	// Check the new password first, so a typo doesn't use up the link.
	if err := auth.ValidatePassword(req.Password); err != nil {
		writeError(w, http.StatusBadRequest, sentence(err))
		return
	}
	ctx := r.Context()
	userID, tokenEmail, err := s.store.ConsumeToken(ctx, req.Token, store.TokenResetPassword)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusBadRequest, "This link is invalid, already used or expired. Request a new one.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := s.store.SetPassword(ctx, userID, hash); err != nil {
		s.serverError(w, r, err)
		return
	}
	// The link arrived in their mailbox, so that address is theirs (this is how invites get verified).
	if tokenEmail != "" {
		if _, err := s.store.VerifyEmail(ctx, userID, tokenEmail); err != nil && !errors.Is(err, store.ErrEmailTaken) {
			s.log.Error("verify email after reset", "err", err)
		}
		s.syncGroups(ctx, userID)
	}
	s.log.Info("password reset via email", "user_id", userID)
	writeJSON(w, http.StatusOK, map[string]string{"redirect": "/login"})
}

// --- admin: mail settings ---

type mailSettingsInput struct {
	Host     *string `json:"smtp_host"`
	Port     *int    `json:"smtp_port"`
	Username *string `json:"smtp_username"`
	Password *string `json:"smtp_password"` // nil or "" keeps the saved one
	From     *string `json:"smtp_from"`
}

// applyMailSettings validates and stores the mail fields that were sent.
// It returns a message for the admin when something is invalid.
func (s *Server) applyMailSettings(ctx context.Context, in mailSettingsInput) (string, error) {
	set := func(key, v string) error { return s.store.SetSetting(ctx, key, v) }
	if in.Host != nil {
		h := strings.TrimSpace(*in.Host)
		if h != "" && !hostRe.MatchString(strings.ToLower(h)) && net.ParseIP(h) == nil {
			return "Enter the mail server as a hostname, like smtp.resend.com.", nil
		}
		if err := set(store.SettingSMTPHost, h); err != nil {
			return "", err
		}
	}
	if in.Port != nil {
		if *in.Port < 1 || *in.Port > 65535 {
			return "The port must be between 1 and 65535 (usually 465 or 587).", nil
		}
		if err := set(store.SettingSMTPPort, strconv.Itoa(*in.Port)); err != nil {
			return "", err
		}
	}
	if in.Username != nil {
		if err := set(store.SettingSMTPUsername, strings.TrimSpace(*in.Username)); err != nil {
			return "", err
		}
	}
	if in.Password != nil && *in.Password != "" {
		if err := set(store.SettingSMTPPassword, *in.Password); err != nil {
			return "", err
		}
	}
	if in.From != nil {
		f := strings.TrimSpace(*in.From)
		if f != "" && !mail.ValidFrom(f) {
			return "Enter the sender like noreply@" + s.redirectBase + " or " + s.redirectBase + " <noreply@" + s.redirectBase + ">.", nil
		}
		if err := set(store.SettingSMTPFrom, f); err != nil {
			return "", err
		}
	}
	return "", nil
}

func (s *Server) handleTestMail(w http.ResponseWriter, r *http.Request) {
	if !s.need(w, r, store.PermSettings) {
		return
	}
	var req struct {
		To string `json:"to"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	to, err := auth.NormalizeEmail(req.To)
	if err != nil || to == "" {
		writeError(w, http.StatusBadRequest, "Enter the address to send the test mail to.")
		return
	}
	if !s.mailReady(r.Context()) {
		writeError(w, http.StatusBadRequest, "Fill in and save the mail server, port and sender first.")
		return
	}
	err = s.sendMail(r.Context(), to, mailContent{
		Title:  "Test mail",
		Intro:  "Email from your login portal works. Verification and password-reset mails will look like this.",
		Footer: "Sent from the admin panel.",
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "Sending failed: "+err.Error())
		return
	}
	s.audit(r, "settings.test_mail", to, "")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
