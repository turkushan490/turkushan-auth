package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/turkushan490/turkushan-auth/internal/auth"
)

var (
	ErrUsernameTaken = errors.New("username taken")
	ErrEmailTaken    = errors.New("email already in use")
)

const (
	MaxFailedLogins = 5
	LockDuration    = 15 * time.Minute
)

type User struct {
	ID             int64
	Username       string
	PasswordHash   string
	Email          string // "" when none
	EmailVerified  bool
	IsAdmin        bool
	Status         string // active | blocked
	FailedAttempts int
	LockedUntil    int64 // unix seconds, 0 when not locked
}

// userCols matches scanUser; queries alias the users table as u.
const userCols = `u.id, u.username, u.password_hash, COALESCE(u.email, ''), u.email_verified_at IS NOT NULL,
  u.is_admin, u.status, u.failed_attempts, COALESCE(u.locked_until, 0)`

type scanner interface{ Scan(dest ...any) error }

// scanUser scans userCols, followed by any extra columns into extra.
func scanUser(sc scanner, extra ...any) (*User, error) {
	u := &User{}
	dest := append([]any{
		&u.ID, &u.Username, &u.PasswordHash, &u.Email, &u.EmailVerified,
		&u.IsAdmin, &u.Status, &u.FailedAttempts, &u.LockedUntil,
	}, extra...)
	if err := sc.Scan(dest...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (s *Store) UserByUsername(ctx context.Context, username string) (*User, error) {
	return scanUser(s.DB.QueryRowContext(ctx, `SELECT `+userCols+` FROM users u WHERE u.username = ?`, username))
}

func (s *Store) CountAdmins(ctx context.Context) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE is_admin = 1`).Scan(&n)
	return n, err
}

// CreateUser inserts a user; username must already be normalized and validated.
func (s *Store) CreateUser(ctx context.Context, username, passwordHash string, isAdmin bool) (int64, error) {
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, is_admin, created_at) VALUES (?, ?, ?, ?)`,
		username, passwordHash, isAdmin, now())
	if err != nil {
		return 0, uniqueErr(err)
	}
	return res.LastInsertId()
}

// RegisterUser inserts a self-registered user. email may be "".
func (s *Store) RegisterUser(ctx context.Context, username, passwordHash, email string) (int64, error) {
	var emailArg any
	if email != "" {
		emailArg = email
	}
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, email, created_at) VALUES (?, ?, ?, ?)`,
		username, passwordHash, emailArg, now())
	if err != nil {
		return 0, uniqueErr(err)
	}
	return res.LastInsertId()
}

func uniqueErr(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "UNIQUE") && strings.Contains(msg, "users.username"):
		return ErrUsernameTaken
	case strings.Contains(msg, "UNIQUE") && strings.Contains(msg, "users.email"):
		return ErrEmailTaken
	}
	return err
}

// RecordLoginFailure counts a wrong password and locks the account for
// LockDuration after MaxFailedLogins in a row. It reports whether this
// failure caused a lock.
func (s *Store) RecordLoginFailure(ctx context.Context, id int64) (bool, error) {
	lockUntil := time.Now().Add(LockDuration).Unix()
	// SQLite evaluates every SET expression against the old row.
	_, err := s.DB.ExecContext(ctx, `
UPDATE users SET
  locked_until    = CASE WHEN failed_attempts + 1 >= ? THEN ? ELSE locked_until END,
  failed_attempts = CASE WHEN failed_attempts + 1 >= ? THEN 0 ELSE failed_attempts + 1 END
WHERE id = ?`, MaxFailedLogins, lockUntil, MaxFailedLogins, id)
	if err != nil {
		return false, err
	}
	var until int64
	err = s.DB.QueryRowContext(ctx, `SELECT COALESCE(locked_until, 0) FROM users WHERE id = ?`, id).Scan(&until)
	return until == lockUntil, err
}

func (s *Store) RecordLoginSuccess(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx,
		`UPDATE users SET failed_attempts = 0, locked_until = NULL, last_login = ? WHERE id = ?`, now(), id)
	return err
}

func (s *Store) userAdminFlag(ctx context.Context, username string) (id int64, isAdmin bool, err error) {
	err = s.DB.QueryRowContext(ctx, `SELECT id, is_admin FROM users WHERE username = ?`, username).Scan(&id, &isAdmin)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, ErrNotFound
	}
	return id, isAdmin, err
}

type BootstrapResult string

const (
	BootstrapCreated  BootstrapResult = "created"  // ADMIN_USER did not exist and was created
	BootstrapPromoted BootstrapResult = "promoted" // ADMIN_USER existed and was made admin
	BootstrapExisting BootstrapResult = "existing" // ADMIN_USER already an admin; password left alone
	BootstrapSkipped  BootstrapResult = "skipped"  // no ADMIN_USER set, but an admin exists
	BootstrapNoAdmin  BootstrapResult = "no-admin" // no ADMIN_USER set and no admin exists
)

// BootstrapAdmin makes sure the ADMIN_USER account exists and is an admin.
// The password is only used when the account is created, so changing it later
// in the panel is not undone by a container restart.
func (s *Store) BootstrapAdmin(ctx context.Context, rawUser, password string) (BootstrapResult, error) {
	username := auth.NormalizeUsername(rawUser)
	if username == "" {
		n, err := s.CountAdmins(ctx)
		if err != nil {
			return "", err
		}
		if n == 0 {
			return BootstrapNoAdmin, nil
		}
		return BootstrapSkipped, nil
	}
	if err := auth.ValidateUsername(username); err != nil {
		return "", fmt.Errorf("ADMIN_USER: %w", err)
	}

	id, isAdmin, err := s.userAdminFlag(ctx, username)
	switch {
	case errors.Is(err, ErrNotFound):
		if password == "" {
			return "", errors.New("ADMIN_PASSWORD is required to create the admin account")
		}
		if err := auth.ValidatePassword(password); err != nil {
			return "", fmt.Errorf("ADMIN_PASSWORD: %w", err)
		}
		hash, err := auth.HashPassword(password)
		if err != nil {
			return "", err
		}
		if _, err := s.CreateUser(ctx, username, hash, true); err != nil {
			return "", err
		}
		return BootstrapCreated, s.Audit(ctx, "system", "admin.bootstrap", username, "created from ADMIN_USER", "")
	case err != nil:
		return "", err
	case isAdmin:
		return BootstrapExisting, nil
	}

	if _, err := s.DB.ExecContext(ctx, `UPDATE users SET is_admin = 1 WHERE id = ?`, id); err != nil {
		return "", err
	}
	return BootstrapPromoted, s.Audit(ctx, "system", "admin.promote", username, "promoted from ADMIN_USER", "")
}
