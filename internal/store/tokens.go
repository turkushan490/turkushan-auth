package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

const (
	TokenVerifyEmail   = "verify_email"
	TokenResetPassword = "reset_password"
)

// CreateToken makes a one-time link token (stored hashed) and invalidates the
// user's older tokens of the same kind. email is the address it was sent to.
func (s *Store) CreateToken(ctx context.Context, userID int64, kind, email string, ttl time.Duration) (string, error) {
	token, err := NewToken()
	if err != nil {
		return "", err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM tokens WHERE user_id = ? AND kind = ?`, userID, kind); err != nil {
		return "", err
	}
	t := now()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO tokens (id, user_id, kind, email, created_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)`,
		HashToken(token), userID, kind, email, t, t+int64(ttl/time.Second)); err != nil {
		return "", err
	}
	return token, tx.Commit()
}

// ConsumeToken marks a valid, unused, unexpired token as used and returns who
// it belongs to. Anything else gives ErrNotFound.
func (s *Store) ConsumeToken(ctx context.Context, token, kind string) (userID int64, email string, err error) {
	t := now()
	err = s.DB.QueryRowContext(ctx, `
UPDATE tokens SET used_at = ?
WHERE id = ? AND kind = ? AND used_at IS NULL AND expires_at > ?
RETURNING user_id, COALESCE(email, '')`, t, HashToken(token), kind, t).Scan(&userID, &email)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", ErrNotFound
	}
	return userID, email, err
}

// DeleteExpiredTokens clears tokens that can no longer be used.
func (s *Store) DeleteExpiredTokens(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM tokens WHERE expires_at <= ? OR used_at IS NOT NULL`, now())
	return err
}

// EmailInUse reports whether another account has already verified this address.
func (s *Store) EmailInUse(ctx context.Context, email string, exceptUserID int64) (bool, error) {
	var n int
	err := s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE email = ? AND email_verified_at IS NOT NULL AND id != ?`,
		email, exceptUserID).Scan(&n)
	return n > 0, err
}

// SetEmail changes a user's address (unverified again); "" removes it.
func (s *Store) SetEmail(ctx context.Context, userID int64, email string) error {
	var arg any
	if email != "" {
		arg = email
	}
	_, err := s.DB.ExecContext(ctx,
		`UPDATE users SET email = ?, email_verified_at = NULL WHERE id = ?`, arg, userID)
	return uniqueErr(err)
}

// VerifyEmail marks the address as verified, but only if it's still the
// user's current address (a link for an old address does nothing).
func (s *Store) VerifyEmail(ctx context.Context, userID int64, email string) (bool, error) {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE users SET email_verified_at = ? WHERE id = ? AND email = ?`, now(), userID, email)
	if err != nil {
		return false, uniqueErr(err)
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// UserByVerifiedEmail finds the account that verified this address.
func (s *Store) UserByVerifiedEmail(ctx context.Context, email string) (*User, error) {
	return scanUser(s.DB.QueryRowContext(ctx,
		`SELECT `+userCols+` FROM users u WHERE u.email = ? AND u.email_verified_at IS NOT NULL`, email))
}
