package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var (
	tinyPNG  = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01")
	tinyJPEG = []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")
	tinySVG  = []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"><script>alert(1)</script></svg>`)
)

// upload sends a raw file body like the admin panel does.
func (c *client) upload(method, path string, data []byte) *httptest.ResponseRecorder {
	c.t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set(csrfHeader, c.cookies[csrfCookie])
	for k, v := range c.cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	return rec
}

func fetch(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestAppearance(t *testing.T) {
	h, st := newTestServer(t)
	admin := signedIn(t, h, st, "boss", true)
	user := signedIn(t, h, st, "bob", false)

	// Defaults: base domain as name, default background, no files.
	a := newClient(t, h).session()["appearance"].(map[string]any)
	if a["site_name"] != "example.com" || a["bg_type"] != "default" || a["logo_url"] != "" {
		t.Fatalf("defaults: %v", a)
	}
	if rec := fetch(h, "/branding/logo"); rec.Code != http.StatusNotFound {
		t.Errorf("logo before upload: %d", rec.Code)
	}

	// Only admins can change it.
	if code, _ := user.put("/api/admin/appearance", map[string]any{"site_name": "Hacked"}); code != http.StatusForbidden {
		t.Errorf("non-admin update: %d", code)
	}
	if rec := user.upload(http.MethodPost, "/api/admin/appearance/logo", tinyPNG); rec.Code != http.StatusForbidden {
		t.Errorf("non-admin upload: %d", rec.Code)
	}

	for _, bad := range []map[string]any{
		{"accent": "red"},
		{"accent": "#12345"},
		{"bg_color": "#fff; background:url(//evil)"},
		{"font": "comic-sans"},
		{"bg_type": "video"},
		{"overlay": 100},
		{"site_name": strings.Repeat("x", 41)},
		{"bg_type": "image"}, // nothing uploaded yet
	} {
		if code, _ := admin.put("/api/admin/appearance", bad); code != http.StatusBadRequest {
			t.Errorf("appearance %v: %d", bad, code)
		}
	}

	good := map[string]any{
		"site_name": "Turk's <Portal>", "page_title": "Sign in & relax", "accent": "#ff5500", "font": "inter",
		"bg_type": "gradient", "bg_color": "#101020", "bg_color2": "#302010", "overlay": 40,
	}
	if code, out := admin.put("/api/admin/appearance", good); code != http.StatusOK {
		t.Fatalf("update: %d %v", code, out)
	}
	a = newClient(t, h).session()["appearance"].(map[string]any)
	if a["site_name"] != "Turk's <Portal>" || a["accent"] != "#ff5500" || a["font"] != "inter" || a["bg_type"] != "gradient" {
		t.Errorf("after update: %v", a)
	}
	// The tab title lands in the HTML, escaped.
	if body := fetch(h, "/login").Body.String(); !strings.Contains(body, "<title>Sign in &amp; relax</title>") {
		t.Errorf("index title: %s", body)
	}

	// Uploads: type comes from the content, size is limited.
	if rec := admin.upload(http.MethodPost, "/api/admin/appearance/logo", []byte("<html><script>alert(1)</script>")); rec.Code != http.StatusBadRequest {
		t.Errorf("html as logo: %d", rec.Code)
	}
	if rec := admin.upload(http.MethodPost, "/api/admin/appearance/background", tinySVG); rec.Code != http.StatusBadRequest {
		t.Errorf("svg as background: %d", rec.Code)
	}
	if rec := admin.upload(http.MethodPost, "/api/admin/appearance/logo", append(append([]byte{}, tinyPNG...), make([]byte, maxLogoBytes)...)); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized logo: %d", rec.Code)
	}
	if rec := admin.upload(http.MethodPost, "/api/admin/appearance/nope", tinyPNG); rec.Code != http.StatusNotFound {
		t.Errorf("unknown kind: %d", rec.Code)
	}

	if rec := admin.upload(http.MethodPost, "/api/admin/appearance/logo", tinyPNG); rec.Code != http.StatusOK {
		t.Fatalf("logo upload: %d %s", rec.Code, rec.Body.String())
	}
	if rec := admin.upload(http.MethodPost, "/api/admin/appearance/background", tinyJPEG); rec.Code != http.StatusOK {
		t.Fatalf("background upload: %d %s", rec.Code, rec.Body.String())
	}
	a = newClient(t, h).session()["appearance"].(map[string]any)
	logoURL, _ := a["logo_url"].(string)
	if !strings.HasPrefix(logoURL, "/branding/logo?v=") || a["bg_type"] != "image" || a["bg_image_url"] == "" {
		t.Fatalf("after uploads: %v", a)
	}
	rec := fetch(h, logoURL)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" || !bytes.Equal(rec.Body.Bytes(), tinyPNG) {
		t.Errorf("serve logo: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("logo not cacheable: %q", rec.Header().Get("Cache-Control"))
	}
	if body := fetch(h, "/").Body.String(); !strings.Contains(body, `<link rel="icon" href="`+logoURL+`">`) {
		t.Errorf("index icon: %s", body)
	}

	// An SVG logo is allowed, but served so that scripts inside it can't run.
	if rec := admin.upload(http.MethodPost, "/api/admin/appearance/logo", tinySVG); rec.Code != http.StatusOK {
		t.Fatalf("svg logo: %d", rec.Code)
	}
	rec = fetch(h, "/branding/logo")
	if rec.Header().Get("Content-Type") != "image/svg+xml" || !strings.Contains(rec.Header().Get("Content-Security-Policy"), "sandbox") {
		t.Errorf("svg headers: %s / %s", rec.Header().Get("Content-Type"), rec.Header().Get("Content-Security-Policy"))
	}

	// Removing the background falls back to the default look.
	if rec := admin.upload(http.MethodDelete, "/api/admin/appearance/background", nil); rec.Code != http.StatusOK {
		t.Fatalf("delete background: %d", rec.Code)
	}
	a = newClient(t, h).session()["appearance"].(map[string]any)
	if a["bg_type"] != "default" || a["bg_image_url"] != "" {
		t.Errorf("after delete: %v", a)
	}
	if rec := fetch(h, "/branding/background"); rec.Code != http.StatusNotFound {
		t.Errorf("background after delete: %d", rec.Code)
	}
}
