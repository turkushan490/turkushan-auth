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

// Audit records an admin or system action.
func (s *Store) Audit(ctx context.Context, actor, action, target, detail, ip string) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO audit_log (actor, action, target, detail, ip, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		actor, action, target, detail, ip, now())
	return err
}
