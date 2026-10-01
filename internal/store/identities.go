package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// IdentityBySubject finds who a Discord/Google account is linked to.
func (s *Store) IdentityBySubject(ctx context.Context, provider, subject string) (*Identity, error) {
	var i Identity
	err := s.DB.QueryRowContext(ctx,
		`SELECT provider, subject, user_id, email, display, data FROM identities WHERE provider = ? AND subject = ?`,
		provider, subject).Scan(&i.Provider, &i.Subject, &i.UserID, &i.Email, &i.Display, &i.Data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &i, err
}

// SaveIdentity links a provider account to a user, or refreshes what we know about it.
func (s *Store) SaveIdentity(ctx context.Context, i *Identity) error {
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO identities (provider, subject, user_id, email, display, data, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (provider, subject) DO UPDATE SET email = excluded.email, display = excluded.display, data = excluded.data`,
		i.Provider, i.Subject, i.UserID, i.Email, i.Display, i.Data, now())
	return err
}

func (s *Store) ListIdentities(ctx context.Context, userID int64) ([]Identity, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT provider, subject, user_id, email, display, data FROM identities WHERE user_id = ? ORDER BY provider`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Identity{}
	for rows.Next() {
		var i Identity
		if err := rows.Scan(&i.Provider, &i.Subject, &i.UserID, &i.Email, &i.Display, &i.Data); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// ListAllIdentities returns every connected Discord/Google account (without provider data).
func (s *Store) ListAllIdentities(ctx context.Context) ([]Identity, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT provider, user_id, email, display FROM identities ORDER BY provider`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Identity{}
	for rows.Next() {
		var i Identity
		if err := rows.Scan(&i.Provider, &i.UserID, &i.Email, &i.Display); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *Store) DeleteIdentity(ctx context.Context, userID int64, provider string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM identities WHERE user_id = ? AND provider = ?`, userID, provider)
	return err
}

// Pending is a first Discord/Google sign-in that still has to pick a username.
type Pending struct {
	Provider string
	Subject  string
	Email    string
	Verified bool
	Display  string
	Data     string
	RD       string
}

// CreatePending stores a half-finished sign-up for a short time and returns its one-time token.
func (s *Store) CreatePending(ctx context.Context, p *Pending, ttl time.Duration) (string, error) {
	token, err := NewToken()
	if err != nil {
		return "", err
	}
	_, err = s.DB.ExecContext(ctx, `
INSERT INTO oauth_pending (id, provider, subject, email, verified, display, data, rd, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		HashToken(token), p.Provider, p.Subject, p.Email, p.Verified, p.Display, p.Data, p.RD, now()+int64(ttl/time.Second))
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) GetPending(ctx context.Context, token string) (*Pending, error) {
	var p Pending
	err := s.DB.QueryRowContext(ctx, `
SELECT provider, subject, email, verified, display, data, rd FROM oauth_pending WHERE id = ? AND expires_at > ?`,
		HashToken(token), now()).Scan(&p.Provider, &p.Subject, &p.Email, &p.Verified, &p.Display, &p.Data, &p.RD)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &p, err
}

func (s *Store) DeletePending(ctx context.Context, token string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM oauth_pending WHERE id = ?`, HashToken(token))
	return err
}

func (s *Store) DeleteExpiredPending(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM oauth_pending WHERE expires_at <= ?`, now())
	return err
}

// UsernameTaken reports whether a (normalized) username is in use.
func (s *Store) UsernameTaken(ctx context.Context, username string) (bool, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE username = ?`, username).Scan(&n)
	return n > 0, err
}
