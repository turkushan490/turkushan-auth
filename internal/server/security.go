package server

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"net/netip"
	"strings"

	"github.com/turkushan490/turkushan-auth/internal/auth"
	"github.com/turkushan490/turkushan-auth/internal/store"
)

const (
	sessionCookie = "ta_session"
	csrfCookie    = "ta_csrf"
	csrfHeader    = "X-CSRF-Token"
	// The server-side session (30 days, sliding) decides validity; the cookie
	// just has to outlive it.
	cookieMaxAge = 365 * 24 * 60 * 60
)

func (s *Server) secureCookies() bool { return s.cfg.AppURL.Scheme == "https" }

// setSessionCookie shares the session with every subdomain under COOKIE_DOMAIN (SSO).
func (s *Server) setSessionCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Domain:   s.cfg.CookieDomain,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.secureCookies(),
		SameSite: http.SameSiteLaxMode,
	})
}

// currentUser returns the logged-in user, or nil.
func (s *Server) currentUser(r *http.Request) *store.User {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return nil
	}
	u, err := s.store.SessionUser(r.Context(), c.Value)
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			s.log.Error("session lookup", "err", err)
		}
		return nil
	}
	return u
}

// startSession replaces any existing session (no session fixation) and sets the cookie.
func (s *Server) startSession(w http.ResponseWriter, r *http.Request, userID int64, ip string) error {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		if err := s.store.DeleteSession(r.Context(), c.Value); err != nil {
			return err
		}
	}
	token, err := s.store.CreateSession(r.Context(), userID, ip, r.UserAgent())
	if err != nil {
		return err
	}
	s.setSessionCookie(w, token, cookieMaxAge)
	return nil
}

// ensureCSRF gives the browser a CSRF token cookie. The frontend echoes it in
// the X-CSRF-Token header (double-submit), which another site cannot do.
func (s *Server) ensureCSRF(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(csrfCookie); err == nil && len(c.Value) >= 32 {
		return
	}
	token, err := store.NewToken()
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		Secure:   s.secureCookies(),
		SameSite: http.SameSiteStrictMode,
	})
}

// csrfProtect guards every state-changing API call.
func (s *Server) csrfProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != s.appOrigin {
			writeError(w, http.StatusForbidden, "Request blocked. Open the portal via "+s.cfg.AppURL.String()+" and try again.")
			return
		}
		c, err := r.Cookie(csrfCookie)
		if err != nil || c.Value == "" ||
			subtle.ConstantTimeCompare([]byte(c.Value), []byte(r.Header.Get(csrfHeader))) != 1 {
			writeError(w, http.StatusForbidden, "Security token missing or expired. Reload the page and try again.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP returns the visitor's IP. Proxy headers are only believed when the
// direct peer is a trusted proxy (NPM); otherwise anyone could spoof them.
func clientIP(r *http.Request, trusted []netip.Prefix) string {
	peer, ok := parseIP(r.RemoteAddr)
	if !ok {
		return r.RemoteAddr
	}
	isTrusted := func(a netip.Addr) bool {
		for _, p := range trusted {
			if p.Contains(a) {
				return true
			}
		}
		return false
	}
	if !isTrusted(peer) {
		return peer.String()
	}
	if a, ok := parseIP(r.Header.Get("X-Real-IP")); ok {
		return a.String()
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Walk from the right: the last address not added by a trusted proxy is the client.
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			a, ok := parseIP(parts[i])
			if !ok {
				break
			}
			if !isTrusted(a) || i == 0 {
				return a.String()
			}
		}
	}
	return peer.String()
}

// parseIP accepts "1.2.3.4", "1.2.3.4:5678" and "[::1]:5678".
func parseIP(s string) (netip.Addr, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return netip.Addr{}, false
	}
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr().Unmap(), true
	}
	if a, err := netip.ParseAddr(s); err == nil {
		return a.Unmap(), true
	}
	return netip.Addr{}, false
}

// safeRedirect returns rd when it is allowed, else the portal's own page.
func (s *Server) safeRedirect(rd string) string {
	if u := auth.SafeRedirect(rd, s.redirectBase, s.cfg.AppURL.Scheme == "http"); u != "" {
		return u
	}
	return s.cfg.AppURL.String() + "/"
}
