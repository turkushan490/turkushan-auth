// Package db opens the SQLite database and applies schema migrations.
package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // pure-Go driver, registers "sqlite"
)

// Open opens (or creates) the SQLite database at path.
func Open(path string) (*sql.DB, error) {
	dsn := "file:" + path +
		"?_pragma=foreign_keys(1)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=busy_timeout(5000)"
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// One connection serializes writes and avoids SQLITE_BUSY; plenty for a home portal.
	// Never start a query on the *sql.DB while holding a transaction, or it deadlocks.
	d.SetMaxOpenConns(1)
	if err := d.Ping(); err != nil {
		d.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return d, nil
}

// Migrate applies every migration newer than the stored schema version.
func Migrate(ctx context.Context, d *sql.DB) error {
	if _, err := d.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return err
	}
	var current int
	if err := d.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&current); err != nil {
		return err
	}
	for i := current; i < len(migrations); i++ {
		tx, err := d.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_version (version) VALUES (?)`, i+1); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// migrations are append-only: never edit one that has shipped, add a new entry instead.
// All timestamps are unix seconds.
var migrations = []string{
	// 1: initial schema
	`
CREATE TABLE users (
  id                INTEGER PRIMARY KEY,
  username          TEXT    NOT NULL UNIQUE,
  password_hash     TEXT    NOT NULL,
  email             TEXT,
  email_verified_at INTEGER,
  is_admin          INTEGER NOT NULL DEFAULT 0,
  status            TEXT    NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'blocked')),
  failed_attempts   INTEGER NOT NULL DEFAULT 0,
  locked_until      INTEGER,
  created_at        INTEGER NOT NULL,
  last_login        INTEGER
);
CREATE UNIQUE INDEX users_email ON users (email) WHERE email IS NOT NULL;

CREATE TABLE sites (
  id               INTEGER PRIMARY KEY,
  name             TEXT    NOT NULL,
  host             TEXT    NOT NULL UNIQUE,
  upstream         TEXT    NOT NULL DEFAULT '',
  require_approval INTEGER NOT NULL DEFAULT 1,
  require_email    INTEGER NOT NULL DEFAULT 0,
  created_at       INTEGER NOT NULL
);

CREATE TABLE site_access (
  user_id      INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  site_id      INTEGER NOT NULL REFERENCES sites (id) ON DELETE CASCADE,
  status       TEXT    NOT NULL CHECK (status IN ('pending', 'approved', 'denied')),
  requested_at INTEGER NOT NULL,
  decided_at   INTEGER,
  decided_by   TEXT,
  PRIMARY KEY (user_id, site_id)
);
CREATE INDEX site_access_status ON site_access (status);

CREATE TABLE sessions (
  id         TEXT    PRIMARY KEY,
  user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  last_seen  INTEGER NOT NULL,
  ip         TEXT    NOT NULL DEFAULT '',
  user_agent TEXT    NOT NULL DEFAULT ''
);
CREATE INDEX sessions_user ON sessions (user_id);
CREATE INDEX sessions_expires ON sessions (expires_at);

CREATE TABLE tokens (
  id         TEXT    PRIMARY KEY,
  user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  kind       TEXT    NOT NULL CHECK (kind IN ('verify_email', 'reset_password')),
  email      TEXT,
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  used_at    INTEGER
);
CREATE INDEX tokens_user ON tokens (user_id);

CREATE TABLE audit_log (
  id         INTEGER PRIMARY KEY,
  actor      TEXT    NOT NULL,
  action     TEXT    NOT NULL,
  target     TEXT    NOT NULL DEFAULT '',
  detail     TEXT    NOT NULL DEFAULT '',
  ip         TEXT    NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL
);
CREATE INDEX audit_log_created ON audit_log (created_at);

CREATE TABLE settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
`,
	// 2: an email is only reserved once it's verified, so typing someone else's
	// address at sign-up can't block the real owner from using it.
	`
DROP INDEX users_email;
CREATE UNIQUE INDEX users_verified_email ON users (email) WHERE email_verified_at IS NOT NULL;
CREATE INDEX users_email ON users (email);
`,
}
