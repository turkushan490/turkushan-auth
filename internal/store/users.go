package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/turkushan490/turkushan-auth/internal/auth"
)

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
		return 0, err
	}
	return res.LastInsertId()
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
