package store

import (
	"context"
	"testing"
)

func TestSessions(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	id, err := s.RegisterUser(ctx, "alice", "hash", "")
	if err != nil {
		t.Fatal(err)
	}

	token, err := s.CreateSession(ctx, id, "192.0.2.1", "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	var stored string
	s.DB.QueryRow(`SELECT id FROM sessions`).Scan(&stored)
	if stored == token || stored != HashToken(token) {
		t.Fatal("session must be stored as a hash of the token")
	}

	u, err := s.SessionUser(ctx, token)
	if err != nil || u.Username != "alice" {
		t.Fatalf("lookup: %v %v", u, err)
	}
	if _, err := s.SessionUser(ctx, "wrong-token"); err != ErrNotFound {
		t.Fatalf("wrong token: %v", err)
	}

	// Sliding expiry: an old last_seen gets pushed forward.
	s.DB.Exec(`UPDATE sessions SET last_seen = last_seen - 3600, expires_at = expires_at - 3600`)
	var before int64
	s.DB.QueryRow(`SELECT expires_at FROM sessions`).Scan(&before)
	s.SessionUser(ctx, token)
	var after int64
	s.DB.QueryRow(`SELECT expires_at FROM sessions`).Scan(&after)
	if after <= before {
		t.Errorf("expiry did not slide: %d -> %d", before, after)
	}

	// Blocked user: session no longer valid.
	s.DB.Exec(`UPDATE users SET status = 'blocked'`)
	if _, err := s.SessionUser(ctx, token); err != ErrNotFound {
		t.Errorf("blocked user: %v", err)
	}
	s.DB.Exec(`UPDATE users SET status = 'active'`)

	// Expired session is rejected and removed.
	s.DB.Exec(`UPDATE sessions SET expires_at = 1`)
	if _, err := s.SessionUser(ctx, token); err != ErrNotFound {
		t.Errorf("expired: %v", err)
	}
	var n int
	s.DB.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n)
	if n != 0 {
		t.Errorf("expired session not deleted")
	}

	token2, _ := s.CreateSession(ctx, id, "", "")
	if err := s.DeleteSession(ctx, token2); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SessionUser(ctx, token2); err != ErrNotFound {
		t.Errorf("deleted session still valid: %v", err)
	}
}

func TestRegisterUniqueAndLockout(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	id, err := s.RegisterUser(ctx, "bob", "hash", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RegisterUser(ctx, "bob", "hash", ""); err != ErrUsernameTaken {
		t.Errorf("duplicate username: %v", err)
	}
	if _, err := s.RegisterUser(ctx, "bob2", "hash", "bob@example.com"); err != ErrEmailTaken {
		t.Errorf("duplicate email: %v", err)
	}
	if _, err := s.RegisterUser(ctx, "carol", "hash", ""); err != nil {
		t.Errorf("two users without email must be allowed: %v", err)
	}

	for i := 1; i < MaxFailedLogins; i++ {
		locked, err := s.RecordLoginFailure(ctx, id)
		if err != nil || locked {
			t.Fatalf("failure %d: locked=%v err=%v", i, locked, err)
		}
	}
	locked, err := s.RecordLoginFailure(ctx, id)
	if err != nil || !locked {
		t.Fatalf("failure %d should lock: locked=%v err=%v", MaxFailedLogins, locked, err)
	}
	u, _ := s.UserByUsername(ctx, "bob")
	if u.LockedUntil == 0 || u.FailedAttempts != 0 {
		t.Errorf("after lock: until=%d attempts=%d", u.LockedUntil, u.FailedAttempts)
	}

	if err := s.RecordLoginSuccess(ctx, id); err != nil {
		t.Fatal(err)
	}
	u, _ = s.UserByUsername(ctx, "bob")
	if u.LockedUntil != 0 || u.FailedAttempts != 0 {
		t.Errorf("after success: until=%d attempts=%d", u.LockedUntil, u.FailedAttempts)
	}
}
