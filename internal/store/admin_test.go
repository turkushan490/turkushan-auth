package store

import (
	"context"
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
	entries, err := s.ListAudit(ctx, 10)
	if err != nil || len(entries) != 1 || entries[0].Action != "user.delete" {
		t.Fatalf("audit: %+v %v", entries, err)
	}
}
