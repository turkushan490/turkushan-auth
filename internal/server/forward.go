package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/turkushan490/turkushan-auth/internal/notify"
	"github.com/turkushan490/turkushan-auth/internal/store"
)

// Outcomes of an access check; the frontend's /pending page shows one per status.
const (
	accessOK            = "ok"
	accessPending       = "pending"
	accessDenied        = "denied"
	accessUnknownSite   = "unknown_site"
	accessEmailRequired = "email_required"
)

type decision struct {
	Status string
	Site   *store.Site
}

// decide works out whether u may open host. With fileRequest set, a first visit
// to an approval-only site files a pending request and notifies the admin.
func (s *Server) decide(ctx context.Context, u *store.User, host string, fileRequest bool) (decision, error) {
	site, err := s.store.SiteByHost(ctx, host)
	if errors.Is(err, store.ErrNotFound) {
		return decision{Status: accessUnknownSite}, nil // deny by default
	}
	if err != nil {
		return decision{}, err
	}
	d := decision{Site: site}

	switch {
	case u.IsAdmin:
		d.Status = accessOK
		return d, nil
	case site.RequireEmail && !u.EmailVerified:
		d.Status = accessEmailRequired
		return d, nil
	case !site.RequireApproval:
		d.Status = accessOK
		return d, nil
	}

	status, err := s.store.AccessStatus(ctx, u.ID, site.ID)
	if err != nil {
		return d, err
	}
	switch status {
	case store.AccessApproved:
		d.Status = accessOK
	case store.AccessDenied:
		d.Status = accessDenied
	default:
		d.Status = accessPending
		if status == "" && fileRequest {
			created, err := s.store.RequestAccess(ctx, u.ID, site.ID)
			if err != nil {
				return d, err
			}
			if created {
				s.log.Info("access requested", "username", u.Username, "site", site.Host)
				s.notifyAdmin(ctx, "New access request",
					fmt.Sprintf("**%s** wants to open **%s** (%s).", u.Username, site.Name, site.Host))
			}
		}
	}
	return d, nil
}

// handleForwardAuth is NPM's auth_request target (GET /api/auth/nginx):
// 200 = let through, 401/403 + X-Auth-Location = send the visitor there.
// nginx uses the original request's method for the subrequest, so any method is accepted.
func (s *Server) handleForwardAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	orig := r.Header.Get("X-Original-URL")
	u, err := url.Parse(orig)
	if orig == "" || err != nil || u.Host == "" {
		writeError(w, http.StatusBadRequest, "Missing X-Original-URL header: check the NPM advanced config.")
		return
	}
	host := strings.ToLower(u.Hostname())

	user := s.currentUser(r)
	if user == nil {
		w.Header().Set("X-Auth-Location", s.portalURL("/login", url.Values{"rd": {orig}}))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	d, err := s.decide(r.Context(), user, host, true)
	if err != nil {
		s.log.Error("forward auth", "host", host, "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if d.Status == accessOK {
		w.Header().Set("X-Auth-User", user.Username)
		if user.EmailVerified {
			w.Header().Set("X-Auth-Email", user.Email)
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("X-Auth-Location", s.portalURL("/pending", url.Values{"site": {host}, "rd": {orig}}))
	w.WriteHeader(http.StatusForbidden)
}

// handleAccess tells the /pending page where the user stands for a site.
func (s *Server) handleAccess(w http.ResponseWriter, r *http.Request) {
	user := s.currentUser(r)
	if user == nil {
		writeError(w, http.StatusUnauthorized, "Please sign in.")
		return
	}
	host := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("site")))
	d, err := s.decide(r.Context(), user, host, false)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	resp := map[string]any{"status": d.Status, "is_admin": user.IsAdmin}
	if d.Site != nil {
		resp["site"] = map[string]string{"name": d.Site.Name, "host": d.Site.Host}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) portalURL(path string, q url.Values) string {
	u := s.cfg.AppURL.String() + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	return u
}

// notifyAdmin posts to the Discord webhook, if one is set, without holding up the request.
func (s *Server) notifyAdmin(ctx context.Context, title, text string) {
	webhook, err := s.store.Setting(ctx, store.SettingDiscordWebhook)
	if err != nil || webhook == "" {
		return
	}
	link := s.portalURL("/admin", nil)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := notify.Discord(ctx, webhook, title, text, link); err != nil {
			s.log.Warn("discord notification failed", "err", err)
		}
	}()
}
