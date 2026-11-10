package store

import (
	"context"
	"fmt"
	"testing"
)

func TestAdminUserActions(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	id, _ := s.RegisterUser(ctx, "alice", "hash1", "Alice@example.com")
	token, _ := s.CreateSession(ctx, id, "", "")
	s.RecordLoginFailure(ctx, id)

	users, err := s.ListUsers(ctx)
	if err != nil || len(users) != 1 || users[0].Username != "alice" || users[0].Status != "active" {
		t.Fatalf("list: %+v %v", users, err)
	}

	if err := s.SetUserStatus(ctx, id, "blocked"); err != nil {
		t.Fatal(err)
	}
	var n int
	s.DB.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id = ?`, id).Scan(&n)
	if n != 0 {
		t.Error("blocking did not end sessions")
	}
	s.SetUserStatus(ctx, id, "active")

	token, _ = s.CreateSession(ctx, id, "", "")
	if err := s.SetPassword(ctx, id, "hash2"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SessionUser(ctx, token); err != ErrNotFound {
		t.Error("password reset did not end sessions")
	}
	u, _ := s.UserByID(ctx, id)
	if u.PasswordHash != "hash2" || u.FailedAttempts != 0 {
		t.Errorf("after reset: %+v", u)
	}

	siteID, _ := s.CreateSite(ctx, &Site{Name: "Manga", Host: "manga.example.com"})
	s.RequestAccess(ctx, id, siteID)
	if p, _ := s.CountPending(ctx); p != 1 {
		t.Errorf("pending: %d", p)
	}

	if err := s.DeleteUser(ctx, id); err != nil {
		t.Fatal(err)
	}
	if p, _ := s.CountPending(ctx); p != 0 {
		t.Error("access not removed with the user")
	}

	s.Audit(ctx, "turkushan", "user.delete", "alice", "", "192.0.2.1")
	entries, err := s.ListAudit(ctx, "", 0, 10)
	if err != nil || len(entries) != 1 || entries[0].Action != "user.delete" {
		t.Fatalf("audit: %+v %v", entries, err)
	}
}

func TestAuditPagingAndSearch(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		s.Audit(ctx, "boss", "site.create", fmt.Sprintf("site%d.example.com", i), "", "")
	}
	s.Audit(ctx, "helper", "user.block", "mallory", "100%_sure", "")

	first, _ := s.ListAudit(ctx, "", 0, 3)
	if len(first) != 3 || first[0].Target != "mallory" {
		t.Fatalf("first page: %+v", first)
	}
	next, _ := s.ListAudit(ctx, "", first[2].ID, 3)
	if len(next) != 3 || next[0].ID >= first[2].ID {
		t.Fatalf("next page: %+v", next)
	}
	if hits, _ := s.ListAudit(ctx, "MALLORY", 0, 10); len(hits) != 1 {
		t.Errorf("search by target: %d", len(hits))
	}
	if hits, _ := s.ListAudit(ctx, "site.create", 0, 10); len(hits) != 5 {
		t.Errorf("search by action: %d", len(hits))
	}
	// % and _ are matched literally, not as wildcards.
	if hits, _ := s.ListAudit(ctx, "%_", 0, 10); len(hits) != 1 {
		t.Errorf("wildcards not escaped: %d", len(hits))
	}
}
