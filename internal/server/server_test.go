package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/turkushan490/turkushan-auth/internal/config"
	"github.com/turkushan490/turkushan-auth/internal/db"
	"github.com/turkushan490/turkushan-auth/internal/store"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if err := db.Migrate(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	dist := fstest.MapFS{
		"index.html":      {Data: []byte("<!doctype html><title>spa</title>")},
		"assets/app-1.js": {Data: []byte("console.log(1)")},
		"favicon.svg":     {Data: []byte("<svg/>")},
	}
	h, err := New(&config.Config{}, store.New(d), slog.New(slog.NewTextHandler(io.Discard, nil)), dist)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func get(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestHealthz(t *testing.T) {
	rec := get(t, newTestServer(t), http.MethodGet, "/healthz")
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestSPA(t *testing.T) {
	h := newTestServer(t)

	for _, p := range []string{"/", "/login", "/admin/users", "/index.html"} {
		rec := get(t, h, http.MethodGet, p)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<title>spa</title>") {
			t.Errorf("%s: got %d %q", p, rec.Code, rec.Body.String())
		}
	}

	rec := get(t, h, http.MethodGet, "/assets/app-1.js")
	if rec.Code != http.StatusOK || rec.Body.String() != "console.log(1)" {
		t.Errorf("asset: got %d %q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("asset not cached: %q", rec.Header().Get("Cache-Control"))
	}

	rec = get(t, h, http.MethodGet, "/api/nope")
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Header().Get("Content-Type"), "json") {
		t.Errorf("unknown api: got %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}

	if rec := get(t, h, http.MethodPost, "/login"); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST on page: got %d", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := get(t, newTestServer(t), http.MethodGet, "/")
	for _, h := range []string{"Content-Security-Policy", "X-Frame-Options", "X-Content-Type-Options"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("missing %s", h)
		}
	}
}
