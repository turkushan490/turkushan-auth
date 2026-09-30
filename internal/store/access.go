package store

import (
	"context"
	"database/sql"
	"errors"
)

const (
	AccessPending  = "pending"
	AccessApproved = "approved"
	AccessDenied   = "denied"
)

// AccessStatus returns the user's status for a site, or "" when they never asked.
func (s *Store) AccessStatus(ctx context.Context, userID, siteID int64) (string, error) {
	var status string
	err := s.DB.QueryRowContext(ctx,
		`SELECT status FROM site_access WHERE user_id = ? AND site_id = ?`, userID, siteID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return status, err
}

// RequestAccess files a pending request unless one (or a decision) exists.
// It reports whether a new request was created.
func (s *Store) RequestAccess(ctx context.Context, userID, siteID int64) (bool, error) {
	res, err := s.DB.ExecContext(ctx,
		`INSERT OR IGNORE INTO site_access (user_id, site_id, status, requested_at) VALUES (?, ?, 'pending', ?)`,
		userID, siteID, now())
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

// SetAccess records an admin decision, creating the row if the user never asked.
func (s *Store) SetAccess(ctx context.Context, userID, siteID int64, status, decidedBy string) error {
	t := now()
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO site_access (user_id, site_id, status, requested_at, decided_at, decided_by)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (user_id, site_id) DO UPDATE SET
  status = excluded.status, decided_at = excluded.decided_at, decided_by = excluded.decided_by`,
		userID, siteID, status, t, t, decidedBy)
	return err
}

type AccessEntry struct {
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	SiteID      int64  `json:"site_id"`
	SiteName    string `json:"site_name"`
	SiteHost    string `json:"site_host"`
	Status      string `json:"status"`
	RequestedAt int64  `json:"requested_at"`
	DecidedAt   int64  `json:"decided_at"` // 0 when undecided
	DecidedBy   string `json:"decided_by"`
}

// ListAccess returns pending requests first, then decisions, newest first.
func (s *Store) ListAccess(ctx context.Context) ([]AccessEntry, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT a.user_id, u.username, a.site_id, st.name, st.host, a.status, a.requested_at,
       COALESCE(a.decided_at, 0), COALESCE(a.decided_by, '')
FROM site_access a
JOIN users u ON u.id = a.user_id
JOIN sites st ON st.id = a.site_id
ORDER BY CASE a.status WHEN 'pending' THEN 0 ELSE 1 END,
         COALESCE(a.decided_at, a.requested_at) DESC
LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AccessEntry{}
	for rows.Next() {
		var e AccessEntry
		if err := rows.Scan(&e.UserID, &e.Username, &e.SiteID, &e.SiteName, &e.SiteHost, &e.Status,
			&e.RequestedAt, &e.DecidedAt, &e.DecidedBy); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) UsernameByID(ctx context.Context, id int64) (string, error) {
	var name string
	err := s.DB.QueryRowContext(ctx, `SELECT username FROM users WHERE id = ?`, id).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return name, err
}
