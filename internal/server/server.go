// Package server wires up the HTTP routes.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/turkushan490/turkushan-auth/internal/config"
	"github.com/turkushan490/turkushan-auth/internal/ratelimit"
	"github.com/turkushan490/turkushan-auth/internal/store"
)

type Server struct {
	cfg   *config.Config
	store *store.Store
	log   *slog.Logger
	dist  fs.FS
	index []byte

	appOrigin    string // scheme://host of APP_URL, for the CSRF origin check
	redirectBase string // rd must point at this domain or a subdomain of it

	loginPerIP   *ratelimit.Limiter
	loginPerUser *ratelimit.Limiter
	registerPerIP *ratelimit.Limiter
}

// New returns the portal's HTTP handler. dist is the built web app.
func New(cfg *config.Config, st *store.Store, log *slog.Logger, dist fs.FS) (http.Handler, error) {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return nil, fmt.Errorf("web build is missing index.html: %w", err)
	}
	base := strings.TrimPrefix(cfg.CookieDomain, ".")
	if base == "" {
		base = cfg.AppURL.Hostname()
	}
	s := &Server{
		cfg: cfg, store: st, log: log, dist: dist, index: index,
		appOrigin:     cfg.AppURL.Scheme + "://" + cfg.AppURL.Host,
		redirectBase:  base,
		loginPerIP:    ratelimit.New(20, time.Minute),
		loginPerUser:  ratelimit.New(10, time.Minute),
		registerPerIP: ratelimit.New(5, time.Hour),
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders(cfg.AppURL.Scheme == "https"))

	r.Get("/healthz", s.healthz)
	r.Route("/api", func(r chi.Router) {
		// NPM's auth_request subrequest: no CSRF token, and any HTTP method.
		r.HandleFunc("/auth/nginx", s.handleForwardAuth)

		r.Group(func(r chi.Router) {
			r.Use(s.csrfProtect)
			r.Get("/session", s.handleSession)
			r.Post("/login", s.handleLogin)
			r.Post("/register", s.handleRegister)
			r.Post("/logout", s.handleLogout)
			r.Get("/access", s.handleAccess)

			r.Route("/admin", func(r chi.Router) {
				r.Use(s.requireAdmin)
				r.Get("/data", s.handleAdminData)
				r.Get("/audit", s.handleAdminAudit)
				r.Post("/sites", s.handleCreateSite)
				r.Put("/sites/{id}", s.handleUpdateSite)
				r.Delete("/sites/{id}", s.handleDeleteSite)
				r.Put("/access", s.handleSetAccess)
				r.Put("/users/{id}/status", s.handleSetUserStatus)
				r.Put("/users/{id}/password", s.handleSetUserPassword)
				r.Delete("/users/{id}", s.handleDeleteUser)
				r.Put("/settings", s.handleUpdateSettings)
				r.Post("/settings/test-discord", s.handleTestDiscord)
			})
		})
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusNotFound, "Not found.")
		})
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		})
	})
	r.NotFound(s.spa)
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
	})
	return r, nil
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.store.DB.PingContext(ctx); err != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte("ok"))
}

// spa serves built assets, and index.html for every other page so the
// frontend router can handle /login, /admin, etc.
func (s *Server) spa(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeError(w, http.StatusNotFound, "Not found.")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name != "" && name != "index.html" {
		if st, err := fs.Stat(s.dist, name); err == nil && !st.IsDir() {
			if strings.HasPrefix(name, "assets/") {
				// Vite puts a content hash in asset file names.
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.ServeFileFS(w, r, s.dist, name)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(s.index)
}

func securityHeaders(https bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "same-origin")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
			h.Set("Content-Security-Policy",
				"default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; "+
					"frame-ancestors 'none'; base-uri 'none'; form-action 'self'; object-src 'none'")
			if https {
				// Only the portal's own host; subdomains may still be plain http on the LAN.
				h.Set("Strict-Transport-Security", "max-age=31536000")
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError sends {"error": msg}; msg is shown to the user as is.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) serverError(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("request failed", "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "Something went wrong. Try again.")
}

// decodeJSON reads a small JSON body into v, answering the error itself.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Expected JSON.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request.")
		return false
	}
	return true
}
