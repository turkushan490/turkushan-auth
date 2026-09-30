package store

import (
	"context"
	"testing"
	"time"
)

func TestTokens(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	id, _ := s.RegisterUser(ctx, "alice", "hash", "alice@example.com")

	first, _ := s.CreateToken(ctx, id, TokenVerifyEmail, "alice@example.com", time.Hour)
	second, err := s.CreateToken(ctx, id, TokenVerifyEmail, "alice@example.com", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.ConsumeToken(ctx, first, TokenVerifyEmail); err != ErrNotFound {
		t.Errorf("older token should be invalid after a new one: %v", err)
	}
	if _, _, err := s.ConsumeToken(ctx, second, TokenResetPassword); err != ErrNotFound {
		t.Errorf("token used for the wrong purpose: %v", err)
	}
	uid, email, err := s.ConsumeToken(ctx, second, TokenVerifyEmail)
	if err != nil || uid != id || email != "alice@example.com" {
		t.Fatalf("consume: %d %q %v", uid, email, err)
	}
	if _, _, err := s.ConsumeToken(ctx, second, TokenVerifyEmail); err != ErrNotFound {
		t.Errorf("token worked twice: %v", err)
	}

	expired, _ := s.CreateToken(ctx, id, TokenResetPassword, "", -time.Minute)
	if _, _, err := s.ConsumeToken(ctx, expired, TokenResetPassword); err != ErrNotFound {
		t.Errorf("expired token accepted: %v", err)
	}

	// A link for an address the user has since changed does nothing.
	s.SetEmail(ctx, id, "new@example.com")
	if ok, _ := s.VerifyEmail(ctx, id, "alice@example.com"); ok {
		t.Error("verified an old address")
	}
	if ok, _ := s.VerifyEmail(ctx, id, "new@example.com"); !ok {
		t.Error("could not verify the current address")
	}
	u, err := s.UserByVerifiedEmail(ctx, "new@example.com")
	if err != nil || u.ID != id || !u.EmailVerified {
		t.Errorf("by verified email: %+v %v", u, err)
	}
}
