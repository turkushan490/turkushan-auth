package store

import (
	"context"
)

type UserSummary struct {
	ID            int64  `json:"id"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	IsAdmin       bool   `json:"is_admin"`
	Status        string `json:"status"`
	Locked        bool   `json:"locked"` // temporarily locked after wrong passwords
	CreatedAt     int64  `json:"created_at"`
	LastLogin     int64  `json:"last_login"` // 0 when never
}

func (s *Store) ListUsers(ctx context.Context) ([]UserSummary, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT id, username, COALESCE(email, ''), email_verified_at IS NOT NULL, is_admin, status,
       COALESCE(locked_until, 0) > ?, created_at, COALESCE(last_login, 0)
FROM users ORDER BY is_admin DESC, username`, now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []UserSummary{}
	for rows.Next() {
		var u UserSummary
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.EmailVerified, &u.IsAdmin, &u.Status,
			&u.Locked, &u.CreatedAt, &u.LastLogin); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) UserByID(ctx context.Context, id int64) (*User, error) {
	return scanUser(s.DB.QueryRowContext(ctx, `SELECT `+userCols+` FROM users u WHERE u.id = ?`, id))
}

// SetUserStatus blocks or unblocks a user. Blocking ends all their sessions;
// unblocking also lifts a wrong-password lock.
func (s *Store) SetUserStatus(ctx context.Context, id int64, status string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET status = ?, failed_attempts = 0, locked_until = NULL WHERE id = ?`, status, id); err != nil {
		return err
	}
	if status != "active" {
		if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetPassword replaces the password, lifts any lock and signs the user out everywhere.
func (s *Store) SetPassword(ctx context.Context, id int64, hash string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, failed_attempts = 0, locked_until = NULL WHERE id = ?`, hash, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteUser removes the user with their sessions and access (ON DELETE CASCADE).
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CountPending(ctx context.Context) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM site_access WHERE status = 'pending'`).Scan(&n)
	return n, err
}

type AuditEntry struct {
	ID        int64  `json:"id"`
	Actor     string `json:"actor"`
	Action    string `json:"action"`
	Target    string `json:"target"`
	Detail    string `json:"detail"`
	IP        string `json:"ip"`
	CreatedAt int64  `json:"created_at"`
}

func (s *Store) ListAudit(ctx context.Context, limit int) ([]AuditEntry, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, actor, action, target, detail, ip, created_at FROM audit_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.Actor, &e.Action, &e.Target, &e.Detail, &e.IP, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
