package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"
)

const (
	SessionTTL = 30 * 24 * time.Hour
	// A session's expiry slides forward at most once per touchInterval, so
	// forward-auth checks don't write to the database on every request.
	touchInterval = 60 // seconds
)

// NewToken returns 32 random bytes, base64url encoded.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken is what the database stores, so a leaked database holds no usable session cookies.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// CreateSession starts a session and returns the token for the cookie.
func (s *Store) CreateSession(ctx context.Context, userID int64, ip, userAgent string) (string, error) {
	token, err := NewToken()
	if err != nil {
		return "", err
	}
	if len(userAgent) > 255 {
		userAgent = userAgent[:255]
	}
	t := now()
	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, created_at, expires_at, last_seen, ip, user_agent) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		HashToken(token), userID, t, t+int64(SessionTTL/time.Second), t, ip, userAgent)
	if err != nil {
		return "", err
	}
	return token, nil
}

// SessionUser returns the active user behind a session token and slides the
// session's expiry. Expired sessions and blocked users give ErrNotFound.
func (s *Store) SessionUser(ctx context.Context, token string) (*User, error) {
	if token == "" {
		return nil, ErrNotFound
	}
	id := HashToken(token)
	var expires, lastSeen int64
	u, err := scanUser(s.DB.QueryRowContext(ctx,
		`SELECT `+userCols+`, s.expires_at, s.last_seen FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.id = ?`, id),
		&expires, &lastSeen)
	if err != nil {
		return nil, err
	}

	t := now()
	if expires <= t {
		if _, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id); err != nil {
			return nil, err
		}
		return nil, ErrNotFound
	}
	if u.Status != "active" {
		return nil, ErrNotFound
	}
	if t-lastSeen >= touchInterval {
		if _, err := s.DB.ExecContext(ctx,
			`UPDATE sessions SET last_seen = ?, expires_at = ? WHERE id = ?`,
			t, t+int64(SessionTTL/time.Second), id); err != nil {
			return nil, err
		}
	}
	return u, nil
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, HashToken(token))
	return err
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, now())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
