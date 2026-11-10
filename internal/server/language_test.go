package server

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestLanguageIsSavedAndUsedForMail(t *testing.T) {
	h, st := newTestServer(t)
	srv := withMail(t, st)

	// Picked at sign-up: stored, and the verification mail is in that language.
	c := newClient(t, h)
	if code, out := c.post("/api/register", map[string]string{"username": "nora", "password": "Secret!1", "email": "nora@example.com", "lang": "NL"}); code != http.StatusOK {
		t.Fatalf("register: %d %v", code, out)
	}
	if u := c.session()["user"].(map[string]any); u["lang"] != "nl" {
		t.Errorf("lang after register: %v", u["lang"])
	}
	_, text := srv.Last(t)
	if !strings.Contains(text, "Hoi nora") {
		t.Errorf("verification mail: %s", text)
	}

	// Changed later from the language picker.
	if rec, out := c.do(http.MethodPut, "/api/account/language", map[string]string{"lang": "zh-tw"}, nil); rec.Code != http.StatusOK || out["lang"] != "zh-TW" {
		t.Fatalf("set language: %d %v", rec.Code, out)
	}
	if rec, _ := c.do(http.MethodPut, "/api/account/language", map[string]string{"lang": "klingon"}, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown language: %d", rec.Code)
	}
	if rec, _ := newClient(t, h).do(http.MethodPut, "/api/account/language", map[string]string{"lang": "nl"}, nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("signed out: %d", rec.Code)
	}
	u, _ := st.UserByUsername(context.Background(), "nora")
	if u.Lang != "zh-TW" {
		t.Errorf("stored lang: %q", u.Lang)
	}
}

func TestErrorsCarryACode(t *testing.T) {
	h, _ := newTestServer(t)
	c := newClient(t, h)
	_, out := c.post("/api/login", map[string]string{"username": "nobody", "password": "Secret!1"})
	if out["code"] != "bad_login" || out["error"] != msgBadLogin {
		t.Errorf("bad login: %v", out)
	}
	_, out = c.post("/api/register", map[string]string{"username": "ok-name", "password": "short"})
	if out["code"] != "password_short" {
		t.Errorf("short password: %v", out)
	}
}

func TestLanguageSettings(t *testing.T) {
	h, st := newTestServer(t)
	admin := signedIn(t, h, st, "boss", true)
	put := func(body map[string]any) (int, map[string]any) {
		rec, out := admin.do(http.MethodPut, "/api/admin/appearance", body, nil)
		return rec.Code, out
	}
	if code, out := put(map[string]any{"default_lang": "tr", "languages": []string{"en", "tr", "nl", "tr"}}); code != http.StatusOK {
		t.Fatalf("save languages: %d %v", code, out)
	}
	look := newClient(t, h).session()["appearance"].(map[string]any)
	if look["default_lang"] != "tr" || len(look["languages"].([]any)) != 3 {
		t.Errorf("session appearance: %v", look)
	}
	if code, _ := put(map[string]any{"default_lang": "de"}); code != http.StatusBadRequest {
		t.Errorf("default outside the offered list: %d", code)
	}
	if code, _ := put(map[string]any{"languages": []string{"xx"}}); code != http.StatusBadRequest {
		t.Errorf("unknown language: %d", code)
	}
	// Other appearance saves keep the language settings.
	if code, _ := put(map[string]any{"site_name": "Home"}); code != http.StatusOK {
		t.Fatal("save site name")
	}
	if look := newClient(t, h).session()["appearance"].(map[string]any); look["default_lang"] != "tr" {
		t.Errorf("language settings lost: %v", look)
	}
}
