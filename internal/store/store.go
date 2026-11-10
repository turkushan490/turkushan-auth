// Package store holds all database queries.
package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	DB *sql.DB
}

func New(d *sql.DB) *Store { return &Store{DB: d} }

func now() int64 { return time.Now().Unix() }

// Backup writes a consistent copy of the whole database to path (which must
// not exist yet), while the portal keeps running. Copying the .db file by hand
// can miss what's still in the -wal file.
func (s *Store) Backup(ctx context.Context, path string) error {
	_, err := s.DB.ExecContext(ctx, `VACUUM INTO ?`, path)
	return err
}

// Audit records an admin or system action.
func (s *Store) Audit(ctx context.Context, actor, action, target, detail, ip string) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO audit_log (actor, action, target, detail, ip, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		actor, action, target, detail, ip, now())
	return err
}
