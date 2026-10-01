package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/turkushan490/turkushan-auth/internal/store"
)

// appearance is the look of the portal, set in Admin panel → Appearance.
// Zero values mean "use the default".
type appearance struct {
	SiteName  string `json:"site_name"`  // shown under the logo; default: the base domain
	PageTitle string `json:"page_title"` // browser tab title
	Accent    string `json:"accent"`     // #rrggbb for buttons and links
	Font      string `json:"font"`       // one of allowedFonts
	BgType    string `json:"bg_type"`    // default | color | gradient | image
	BgColor   string `json:"bg_color"`
	BgColor2  string `json:"bg_color2"` // second color of the gradient
	Overlay   int    `json:"overlay"`   // 0-90: how much an image is darkened, in percent

	// Uploaded files live in DATA_DIR/branding; the version busts browser caches.
	LogoType    string `json:"logo_type"` // content type, "" = no logo
	LogoVer     int64  `json:"logo_ver"`
	BgImageType string `json:"bg_image_type"`
	BgImageVer  int64  `json:"bg_image_ver"`
}

var (
	hexColorRe   = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	allowedFonts = map[string]bool{"": true, "inter": true, "montserrat": true, "space-grotesk": true, "nunito": true, "outfit": true, "mono": true}
	bgTypes      = map[string]bool{"": true, "default": true, "color": true, "gradient": true, "image": true}
)

const (
	maxLogoBytes       = 1 << 20 // 1 MB
	maxBackgroundBytes = 8 << 20 // 8 MB
)

func (s *Server) loadAppearance(ctx context.Context) (appearance, error) {
	var a appearance
	v, err := s.store.Setting(ctx, store.SettingAppearance)
	if err != nil || v == "" {
		return a, err
	}
	return a, json.Unmarshal([]byte(v), &a)
}

func (s *Server) saveAppearance(ctx context.Context, a appearance) error {
	b, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return s.store.SetSetting(ctx, store.SettingAppearance, string(b))
}

func (s *Server) siteName(a appearance) string {
	if a.SiteName != "" {
		return a.SiteName
	}
	return s.redirectBase
}

func (s *Server) pageTitle(a appearance) string {
	if a.PageTitle != "" {
		return a.PageTitle
	}
	return s.siteName(a)
}

// appearanceJSON is what every visitor's browser gets to style the pages.
func (s *Server) appearanceJSON(a appearance) map[string]any {
	out := map[string]any{
		"site_name":    s.siteName(a),
		"page_title":   s.pageTitle(a),
		"accent":       a.Accent,
		"font":         a.Font,
		"bg_type":      "default",
		"bg_color":     a.BgColor,
		"bg_color2":    a.BgColor2,
		"overlay":      a.Overlay,
		"logo_url":     "",
		"bg_image_url": "",
	}
	if a.BgType != "" {
		out["bg_type"] = a.BgType
	}
	if a.LogoType != "" {
		out["logo_url"] = fmt.Sprintf("/branding/logo?v=%d", a.LogoVer)
	}
	if a.BgImageType != "" {
		out["bg_image_url"] = fmt.Sprintf("/branding/background?v=%d", a.BgImageVer)
	}
	return out
}

var (
	titleTagRe = regexp.MustCompile(`<title>[^<]*</title>`)
	iconTagRe  = regexp.MustCompile(`<link rel="icon"[^>]*>`)
)

// renderIndex puts the configured tab title and icon into index.html, so they
// are right before any JavaScript runs.
func (s *Server) renderIndex(ctx context.Context) []byte {
	a, err := s.loadAppearance(ctx)
	if err != nil {
		s.log.Error("load appearance", "err", err)
		return s.index
	}
	out := titleTagRe.ReplaceAllLiteral(s.index, []byte("<title>"+html.EscapeString(s.pageTitle(a))+"</title>"))
	if a.LogoType != "" {
		out = iconTagRe.ReplaceAllLiteral(out, []byte(fmt.Sprintf(`<link rel="icon" href="/branding/logo?v=%d">`, a.LogoVer)))
	}
	return out
}

type appearanceInput struct {
	SiteName  string `json:"site_name"`
	PageTitle string `json:"page_title"`
	Accent    string `json:"accent"`
	Font      string `json:"font"`
	BgType    string `json:"bg_type"`
	BgColor   string `json:"bg_color"`
	BgColor2  string `json:"bg_color2"`
	Overlay   int    `json:"overlay"`
}

func (s *Server) handleUpdateAppearance(w http.ResponseWriter, r *http.Request) {
	if !s.need(w, r, store.PermAppearance) {
		return
	}
	var in appearanceInput
	if !decodeJSON(w, r, &in) {
		return
	}
	in.SiteName = strings.TrimSpace(in.SiteName)
	in.PageTitle = strings.TrimSpace(in.PageTitle)
	okColor := func(c string) bool { return c == "" || hexColorRe.MatchString(c) }
	switch {
	case utf8.RuneCountInString(in.SiteName) > 40:
		writeError(w, http.StatusBadRequest, "The site name can be at most 40 characters.")
		return
	case utf8.RuneCountInString(in.PageTitle) > 60:
		writeError(w, http.StatusBadRequest, "The tab title can be at most 60 characters.")
		return
	case !okColor(in.Accent) || !okColor(in.BgColor) || !okColor(in.BgColor2):
		writeError(w, http.StatusBadRequest, "Colors must look like #6366f1.")
		return
	case !allowedFonts[in.Font]:
		writeError(w, http.StatusBadRequest, "Unknown font.")
		return
	case !bgTypes[in.BgType]:
		writeError(w, http.StatusBadRequest, "Unknown background type.")
		return
	case in.Overlay < 0 || in.Overlay > 90:
		writeError(w, http.StatusBadRequest, "Darkness must be between 0 and 90.")
		return
	}

	ctx := r.Context()
	a, err := s.loadAppearance(ctx)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if in.BgType == "image" && a.BgImageType == "" {
		writeError(w, http.StatusBadRequest, "Upload a background image first.")
		return
	}
	a.SiteName, a.PageTitle, a.Accent, a.Font = in.SiteName, in.PageTitle, in.Accent, in.Font
	a.BgType, a.BgColor, a.BgColor2, a.Overlay = in.BgType, in.BgColor, in.BgColor2, in.Overlay
	if err := s.saveAppearance(ctx, a); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "appearance.update", "", "")
	writeJSON(w, http.StatusOK, map[string]any{"appearance": s.appearanceJSON(a)})
}

// sniffImage returns the image type from the file's content (never from what
// the browser claims), or "" when it isn't an image we accept.
func sniffImage(data []byte) string {
	switch ct := http.DetectContentType(data); ct {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return ct
	}
	head := data
	if len(head) > 1024 {
		head = head[:1024]
	}
	trimmed := bytes.TrimLeft(bytes.TrimPrefix(head, []byte("\xef\xbb\xbf")), " \t\r\n")
	if (bytes.HasPrefix(trimmed, []byte("<svg")) || bytes.HasPrefix(trimmed, []byte("<?xml"))) && bytes.Contains(head, []byte("<svg")) {
		return "image/svg+xml"
	}
	return ""
}

type brandingKind struct {
	name    string // logo | background
	max     int64
	allowed map[string]bool
	hint    string
}

var brandingKinds = map[string]brandingKind{
	"logo": {"logo", maxLogoBytes,
		map[string]bool{"image/png": true, "image/jpeg": true, "image/webp": true, "image/gif": true, "image/svg+xml": true},
		"Use a PNG, JPG, WebP, GIF or SVG image."},
	"background": {"background", maxBackgroundBytes,
		map[string]bool{"image/png": true, "image/jpeg": true, "image/webp": true},
		"Use a PNG, JPG or WebP image."},
}

func (s *Server) brandingPath(kind string) string {
	return filepath.Join(s.cfg.DataDir, "branding", kind)
}

// handleUploadBranding stores an uploaded logo or background. The request body is the file itself.
func (s *Server) handleUploadBranding(w http.ResponseWriter, r *http.Request) {
	if !s.need(w, r, store.PermAppearance) {
		return
	}
	kind, ok := brandingKinds[chi.URLParam(r, "kind")]
	if !ok {
		writeError(w, http.StatusNotFound, "Not found.")
		return
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, kind.max+1))
	if err != nil || int64(len(data)) > kind.max {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("That file is too big (max %d MB).", kind.max>>20))
		return
	}
	ct := sniffImage(data)
	if !kind.allowed[ct] {
		writeError(w, http.StatusBadRequest, "That file type isn't supported. "+kind.hint)
		return
	}

	path := s.brandingPath(kind.name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		s.serverError(w, r, err)
		return
	}
	// Write next to the target and rename, so a half-written file is never served.
	if err := os.WriteFile(path+".tmp", data, 0o640); err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := os.Rename(path+".tmp", path); err != nil {
		s.serverError(w, r, err)
		return
	}

	ctx := r.Context()
	a, err := s.loadAppearance(ctx)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	ver := time.Now().UnixMilli()
	if kind.name == "logo" {
		a.LogoType, a.LogoVer = ct, ver
	} else {
		a.BgImageType, a.BgImageVer = ct, ver
		a.BgType = "image"
	}
	if err := s.saveAppearance(ctx, a); err != nil {
		s.serverError(w, r, err)
		return
	}
	s.audit(r, "appearance."+kind.name, "", fmt.Sprintf("uploaded %s, %d KB", ct, len(data)/1024))
	writeJSON(w, http.StatusOK, map[string]any{"appearance": s.appearanceJSON(a)})
}

func (s *Server) handleDeleteBranding(w http.ResponseWriter, r *http.Request) {
	if !s.need(w, r, store.PermAppearance) {
		return
	}
	kind, ok := brandingKinds[chi.URLParam(r, "kind")]
	if !ok {
		writeError(w, http.StatusNotFound, "Not found.")
		return
	}
	ctx := r.Context()
	a, err := s.loadAppearance(ctx)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if kind.name == "logo" {
		a.LogoType, a.LogoVer = "", 0
	} else {
		a.BgImageType, a.BgImageVer = "", 0
		if a.BgType == "image" {
			a.BgType = "default"
		}
	}
	if err := s.saveAppearance(ctx, a); err != nil {
		s.serverError(w, r, err)
		return
	}
	if err := os.Remove(s.brandingPath(kind.name)); err != nil && !os.IsNotExist(err) {
		s.log.Warn("remove branding file", "err", err)
	}
	s.audit(r, "appearance."+kind.name, "", "removed")
	writeJSON(w, http.StatusOK, map[string]any{"appearance": s.appearanceJSON(a)})
}

// handleBranding serves the uploaded logo or background to everyone.
func (s *Server) handleBranding(w http.ResponseWriter, r *http.Request) {
	kind, ok := brandingKinds[chi.URLParam(r, "kind")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	a, err := s.loadAppearance(r.Context())
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	ct := a.LogoType
	if kind.name == "background" {
		ct = a.BgImageType
	}
	if ct == "" {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(s.brandingPath(kind.name))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", ct)
	// The URL carries a version, so browsers may keep it forever.
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	// An SVG opened directly must not be able to run scripts on the portal's origin.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	http.ServeContent(w, r, "", info.ModTime(), f)
}
