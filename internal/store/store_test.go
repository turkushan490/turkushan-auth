package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/turkushan490/turkushan-auth/internal/auth"
	"github.com/turkushan490/turkushan-auth/internal/db"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if err := db.Migrate(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	return New(d)
}

func TestBootstrapAdmin(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if res, err := s.BootstrapAdmin(ctx, "", ""); err != nil || res != BootstrapNoAdmin {
		t.Fatalf("empty db, no env: got %q %v", res, err)
	}
	if _, err := s.BootstrapAdmin(ctx, "Turkushan", ""); err == nil {
		t.Fatal("expected error without ADMIN_PASSWORD")
	}
	if _, err := s.BootstrapAdmin(ctx, "Turkushan", "short"); err == nil {
		t.Fatal("expected error for short ADMIN_PASSWORD")
	}

	res, err := s.BootstrapAdmin(ctx, " Turkushan ", "first-password-123")
	if err != nil || res != BootstrapCreated {
		t.Fatalf("create: got %q %v", res, err)
	}

	var hash string
	var isAdmin bool
	if err := s.DB.QueryRow(`SELECT password_hash, is_admin FROM users WHERE username = 'turkushan'`).Scan(&hash, &isAdmin); err != nil {
		t.Fatal(err)
	}
	if !isAdmin {
		t.Error("bootstrapped user is not admin")
	}
	if ok, _ := auth.VerifyPassword("first-password-123", hash); !ok {
		t.Error("stored hash does not verify")
	}

	// Restart with a different password: the account and its password stay as they were.
	res, err = s.BootstrapAdmin(ctx, "turkushan", "other-password-456")
	if err != nil || res != BootstrapExisting {
		t.Fatalf("second start: got %q %v", res, err)
	}
	var hash2 string
	s.DB.QueryRow(`SELECT password_hash FROM users WHERE username = 'turkushan'`).Scan(&hash2)
	if hash2 != hash {
		t.Error("password was overwritten on restart")
	}

	if res, err := s.BootstrapAdmin(ctx, "", ""); err != nil || res != BootstrapSkipped {
		t.Fatalf("admin exists, no env: got %q %v", res, err)
	}

	var audits int
	s.DB.QueryRow(`SELECT COUNT(*) FROM audit_log WHERE action = 'admin.bootstrap'`).Scan(&audits)
	if audits != 1 {
		t.Errorf("audit entries: got %d, want 1", audits)
	}
}

func TestBootstrapPromotesExistingUser(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	hash, _ := auth.HashPassword("some-password-1")
	if _, err := s.CreateUser(ctx, "alice", hash, false); err != nil {
		t.Fatal(err)
	}
	res, err := s.BootstrapAdmin(ctx, "alice", "")
	if err != nil || res != BootstrapPromoted {
		t.Fatalf("got %q %v", res, err)
	}
	if n, _ := s.CountAdmins(ctx); n != 1 {
		t.Errorf("admins: got %d, want 1", n)
	}
}
