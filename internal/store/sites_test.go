package store

import (
	"context"
	"testing"
)

func TestSitesAndAccess(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	siteID, err := s.CreateSite(ctx, &Site{Name: "Manga", Host: "manga.turkushan.com", Upstream: "http://192.168.0.6:3000", RequireApproval: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSite(ctx, &Site{Name: "Dup", Host: "manga.turkushan.com"}); err != ErrHostTaken {
		t.Errorf("duplicate host: %v", err)
	}
	st, err := s.SiteByHost(ctx, "manga.turkushan.com")
	if err != nil || st.ID != siteID || !st.RequireApproval || st.RequireEmail {
		t.Fatalf("by host: %+v %v", st, err)
	}

	userID, _ := s.RegisterUser(ctx, "alice", "hash", "")

	if status, _ := s.AccessStatus(ctx, userID, siteID); status != "" {
		t.Errorf("no request yet: %q", status)
	}
	created, err := s.RequestAccess(ctx, userID, siteID)
	if err != nil || !created {
		t.Fatalf("request: %v %v", created, err)
	}
	if created, _ := s.RequestAccess(ctx, userID, siteID); created {
		t.Error("second request should not create a new row")
	}
	if status, _ := s.AccessStatus(ctx, userID, siteID); status != AccessPending {
		t.Errorf("after request: %q", status)
	}

	if err := s.SetAccess(ctx, userID, siteID, AccessApproved, "turkushan"); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListAccess(ctx)
	if err != nil || len(list) != 1 || list[0].Status != AccessApproved || list[0].DecidedBy != "turkushan" || list[0].SiteHost != "manga.turkushan.com" {
		t.Fatalf("list: %+v %v", list, err)
	}

	// A denied user visiting again must not reset the decision to pending.
	s.SetAccess(ctx, userID, siteID, AccessDenied, "turkushan")
	s.RequestAccess(ctx, userID, siteID)
	if status, _ := s.AccessStatus(ctx, userID, siteID); status != AccessDenied {
		t.Errorf("re-request overwrote denial: %q", status)
	}

	st.Name, st.RequireEmail = "Manga library", true
	if err := s.UpdateSite(ctx, st); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteSite(ctx, siteID); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.ListAccess(ctx); len(list) != 0 {
		t.Error("access rows not removed with the site")
	}
	if err := s.DeleteSite(ctx, siteID); err != ErrNotFound {
		t.Errorf("delete missing: %v", err)
	}
}

func TestSettings(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	if v, err := s.Setting(ctx, SettingDiscordWebhook); v != "" || err != nil {
		t.Fatalf("unset: %q %v", v, err)
	}
	s.SetSetting(ctx, SettingDiscordWebhook, "a")
	s.SetSetting(ctx, SettingDiscordWebhook, "b")
	if v, _ := s.Setting(ctx, SettingDiscordWebhook); v != "b" {
		t.Errorf("got %q", v)
	}
	s.SetSetting(ctx, SettingDiscordWebhook, "")
	if v, _ := s.Setting(ctx, SettingDiscordWebhook); v != "" {
		t.Errorf("not removed: %q", v)
	}
}
