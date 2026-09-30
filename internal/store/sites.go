package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

var ErrHostTaken = errors.New("host already added")

type Site struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Host            string `json:"host"`     // e.g. manga.turkushan.com
	Upstream        string `json:"upstream"` // e.g. http://192.168.0.6:3000, used in the NPM snippet
	RequireApproval bool   `json:"require_approval"`
	RequireEmail    bool   `json:"require_email"`
	CreatedAt       int64  `json:"created_at"`
}

const siteCols = `id, name, host, upstream, require_approval, require_email, created_at`

func scanSite(sc scanner) (*Site, error) {
	st := &Site{}
	err := sc.Scan(&st.ID, &st.Name, &st.Host, &st.Upstream, &st.RequireApproval, &st.RequireEmail, &st.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return st, err
}

func (s *Store) ListSites(ctx context.Context) ([]Site, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+siteCols+` FROM sites ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sites := []Site{}
	for rows.Next() {
		st, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		sites = append(sites, *st)
	}
	return sites, rows.Err()
}

func (s *Store) SiteByHost(ctx context.Context, host string) (*Site, error) {
	return scanSite(s.DB.QueryRowContext(ctx, `SELECT `+siteCols+` FROM sites WHERE host = ?`, host))
}

func (s *Store) SiteByID(ctx context.Context, id int64) (*Site, error) {
	return scanSite(s.DB.QueryRowContext(ctx, `SELECT `+siteCols+` FROM sites WHERE id = ?`, id))
}

func (s *Store) CreateSite(ctx context.Context, st *Site) (int64, error) {
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO sites (name, host, upstream, require_approval, require_email, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		st.Name, st.Host, st.Upstream, st.RequireApproval, st.RequireEmail, now())
	if err != nil {
		return 0, siteUniqueErr(err)
	}
	return res.LastInsertId()
}

func (s *Store) UpdateSite(ctx context.Context, st *Site) error {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE sites SET name = ?, host = ?, upstream = ?, require_approval = ?, require_email = ? WHERE id = ?`,
		st.Name, st.Host, st.Upstream, st.RequireApproval, st.RequireEmail, st.ID)
	if err != nil {
		return siteUniqueErr(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSite also removes its access decisions (ON DELETE CASCADE).
func (s *Store) DeleteSite(ctx context.Context, id int64) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM sites WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func siteUniqueErr(err error) error {
	if msg := err.Error(); strings.Contains(msg, "UNIQUE") && strings.Contains(msg, "sites.host") {
		return ErrHostTaken
	}
	return err
}
