package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

var ErrGroupNameTaken = errors.New("group name already used")

// Global rights a group can carry. Admins (users.is_admin) have all of them.
const (
	PermApprove    = "approve"    // approve/deny requests for every site
	PermUsers      = "users"      // create, block, reset, delete users
	PermGroups     = "groups"     // groups, their members and auto-add rules
	PermSites      = "sites"      // add/edit/remove sites
	PermSettings   = "settings"   // email, Discord, login methods, portal address
	PermAppearance = "appearance" // logo, colors, background
	PermAudit      = "audit"      // read the audit log
)

var AllPerms = []string{PermApprove, PermUsers, PermGroups, PermSites, PermSettings, PermAppearance, PermAudit}

func validPerm(p string) bool {
	for _, k := range AllPerms {
		if k == p {
			return true
		}
	}
	return false
}

type Group struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Perms       []string `json:"perms"`
	CreatedAt   int64    `json:"created_at"`
}

// cleanPerms keeps known rights only, sorted, without duplicates.
func cleanPerms(perms []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, p := range perms {
		if validPerm(p) && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

func splitPerms(s string) []string {
	if s == "" {
		return []string{}
	}
	return cleanPerms(strings.Split(s, ","))
}

func (s *Store) ListGroups(ctx context.Context) ([]Group, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, name, description, perms, created_at FROM groups ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Group{}
	for rows.Next() {
		var g Group
		var perms string
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &perms, &g.CreatedAt); err != nil {
			return nil, err
		}
		g.Perms = splitPerms(perms)
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) GroupByID(ctx context.Context, id int64) (*Group, error) {
	var g Group
	var perms string
	err := s.DB.QueryRowContext(ctx, `SELECT id, name, description, perms, created_at FROM groups WHERE id = ?`, id).
		Scan(&g.ID, &g.Name, &g.Description, &perms, &g.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	g.Perms = splitPerms(perms)
	return &g, err
}

func groupUniqueErr(err error) error {
	if err != nil && strings.Contains(err.Error(), "UNIQUE") && strings.Contains(err.Error(), "groups.name") {
		return ErrGroupNameTaken
	}
	return err
}

func (s *Store) CreateGroup(ctx context.Context, name, description string) (int64, error) {
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO groups (name, description, created_at) VALUES (?, ?, ?)`, name, description, now())
	if err != nil {
		return 0, groupUniqueErr(err)
	}
	return res.LastInsertId()
}

func (s *Store) UpdateGroup(ctx context.Context, id int64, name, description string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE groups SET name = ?, description = ? WHERE id = ?`, name, description, id)
	return groupUniqueErr(err)
}

func (s *Store) SetGroupPerms(ctx context.Context, id int64, perms []string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE groups SET perms = ? WHERE id = ?`, strings.Join(cleanPerms(perms), ","), id)
	return err
}

// DeleteGroup also removes its memberships, site rights and rules (ON DELETE CASCADE).
func (s *Store) DeleteGroup(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM groups WHERE id = ?`, id)
	return err
}

type GroupSite struct {
	GroupID    int64 `json:"group_id"`
	SiteID     int64 `json:"site_id"`
	CanOpen    bool  `json:"can_open"`
	CanApprove bool  `json:"can_approve"`
}

func (s *Store) ListGroupSites(ctx context.Context) ([]GroupSite, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT group_id, site_id, can_open, can_approve FROM group_sites`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []GroupSite{}
	for rows.Next() {
		var gs GroupSite
		if err := rows.Scan(&gs.GroupID, &gs.SiteID, &gs.CanOpen, &gs.CanApprove); err != nil {
			return nil, err
		}
		out = append(out, gs)
	}
	return out, rows.Err()
}

// SetGroupSite sets what a group may do with a site; both false removes the row.
func (s *Store) SetGroupSite(ctx context.Context, groupID, siteID int64, canOpen, canApprove bool) error {
	if !canOpen && !canApprove {
		_, err := s.DB.ExecContext(ctx, `DELETE FROM group_sites WHERE group_id = ? AND site_id = ?`, groupID, siteID)
		return err
	}
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO group_sites (group_id, site_id, can_open, can_approve) VALUES (?, ?, ?, ?)
ON CONFLICT (group_id, site_id) DO UPDATE SET can_open = excluded.can_open, can_approve = excluded.can_approve`,
		groupID, siteID, canOpen, canApprove)
	return err
}

type Membership struct {
	UserID  int64  `json:"user_id"`
	GroupID int64  `json:"group_id"`
	Source  string `json:"source"` // manual | rule
}

func (s *Store) ListMemberships(ctx context.Context) ([]Membership, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT user_id, group_id, source FROM user_groups`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Membership{}
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.UserID, &m.GroupID, &m.Source); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// AddMember puts a user in a group by hand. An existing rule-made membership becomes manual, so a later rule change keeps it.
func (s *Store) AddMember(ctx context.Context, userID, groupID int64) error {
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO user_groups (user_id, group_id, source, added_at) VALUES (?, ?, 'manual', ?)
ON CONFLICT (user_id, group_id) DO UPDATE SET source = 'manual'`, userID, groupID, now())
	return err
}

func (s *Store) RemoveMember(ctx context.Context, userID, groupID int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM user_groups WHERE user_id = ? AND group_id = ?`, userID, groupID)
	return err
}

// Rights is everything a user may do in the admin panel, from all their groups together.
type Rights struct {
	Admin        bool
	Global       map[string]bool
	ApproveSites map[int64]bool // sites whose requests they may approve (when lacking the global right)
}

func (r Rights) Has(perm string) bool { return r.Admin || r.Global[perm] }

func (r Rights) CanApprove(siteID int64) bool {
	return r.Has(PermApprove) || r.ApproveSites[siteID]
}

// Staff reports whether the user may enter the admin panel at all.
func (r Rights) Staff() bool {
	return r.Admin || len(r.Global) > 0 || len(r.ApproveSites) > 0
}

func (s *Store) UserRights(ctx context.Context, u *User) (Rights, error) {
	r := Rights{Admin: u.IsAdmin, Global: map[string]bool{}, ApproveSites: map[int64]bool{}}
	if u.IsAdmin {
		return r, nil
	}
	rows, err := s.DB.QueryContext(ctx,
		`SELECT g.perms FROM groups g JOIN user_groups ug ON ug.group_id = g.id WHERE ug.user_id = ?`, u.ID)
	if err != nil {
		return r, err
	}
	for rows.Next() {
		var perms string
		if err := rows.Scan(&perms); err != nil {
			rows.Close()
			return r, err
		}
		for _, p := range splitPerms(perms) {
			r.Global[p] = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return r, err
	}

	rows, err = s.DB.QueryContext(ctx, `
SELECT DISTINCT gs.site_id FROM group_sites gs
JOIN user_groups ug ON ug.group_id = gs.group_id
WHERE ug.user_id = ? AND gs.can_approve = 1`, u.ID)
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return r, err
		}
		r.ApproveSites[id] = true
	}
	return r, rows.Err()
}

// GroupOpens reports whether one of the user's groups may open the site.
func (s *Store) GroupOpens(ctx context.Context, userID, siteID int64) (bool, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `
SELECT COUNT(*) FROM group_sites gs
JOIN user_groups ug ON ug.group_id = gs.group_id
WHERE ug.user_id = ? AND gs.site_id = ? AND gs.can_open = 1`, userID, siteID).Scan(&n)
	return n > 0, err
}

// SetAdmin puts a user in (or takes them out of) the built-in Admins group.
func (s *Store) SetAdmin(ctx context.Context, userID int64, admin bool) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET is_admin = ? WHERE id = ?`, admin, userID)
	return err
}

// --- auto-add rules ---

const (
	RuleEveryone      = "everyone"       // every account
	RuleMethod        = "method"         // value: password | discord | google
	RuleEmail         = "email"          // value: a verified address
	RuleDomain        = "domain"         // value: example.com (verified addresses only)
	RuleDiscordServer = "discord_server" // value: server (guild) id
	RuleDiscordRole   = "discord_role"   // value: serverID:roleID
)

type Rule struct {
	ID      int64  `json:"id"`
	GroupID int64  `json:"group_id"`
	Kind    string `json:"kind"`
	Value   string `json:"value"`
}

func (s *Store) ListRules(ctx context.Context) ([]Rule, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, group_id, kind, value FROM group_rules ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Rule{}
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.ID, &r.GroupID, &r.Kind, &r.Value); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) RuleByID(ctx context.Context, id int64) (*Rule, error) {
	var r Rule
	err := s.DB.QueryRowContext(ctx, `SELECT id, group_id, kind, value FROM group_rules WHERE id = ?`, id).
		Scan(&r.ID, &r.GroupID, &r.Kind, &r.Value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &r, err
}

func (s *Store) CreateRule(ctx context.Context, groupID int64, kind, value string) (int64, error) {
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO group_rules (group_id, kind, value, created_at) VALUES (?, ?, ?, ?)`, groupID, kind, value, now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) DeleteRule(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM group_rules WHERE id = ?`, id)
	return err
}

// Identity is a Discord or Google account linked to a user.
type Identity struct {
	Provider string `json:"provider"`
	Subject  string `json:"-"`
	UserID   int64  `json:"user_id"`
	Email    string `json:"email"`
	Display  string `json:"display"`
	Data     string `json:"-"` // provider extras as JSON; Discord: {"guilds": {"<server id>": ["<role id>", ...]}}
}

type discordData struct {
	Guilds map[string][]string `json:"guilds"`
}

// userFacts is what rules are matched against.
type userFacts struct {
	verifiedEmail string
	methods       map[string]bool
	guildRoles    map[string][]string
}

func (f userFacts) matches(r Rule) bool {
	switch r.Kind {
	case RuleEveryone:
		return true
	case RuleMethod:
		return f.methods[r.Value]
	case RuleEmail:
		return f.verifiedEmail != "" && strings.EqualFold(f.verifiedEmail, r.Value)
	case RuleDomain:
		at := strings.LastIndex(f.verifiedEmail, "@")
		return at >= 0 && strings.EqualFold(f.verifiedEmail[at+1:], r.Value)
	case RuleDiscordServer:
		_, in := f.guildRoles[r.Value]
		return in
	case RuleDiscordRole:
		guild, role, ok := strings.Cut(r.Value, ":")
		if !ok {
			return false
		}
		for _, have := range f.guildRoles[guild] {
			if have == role {
				return true
			}
		}
	}
	return false
}

func (s *Store) userFacts(ctx context.Context, userID int64) (userFacts, error) {
	f := userFacts{methods: map[string]bool{}, guildRoles: map[string][]string{}}
	u, err := s.UserByID(ctx, userID)
	if err != nil {
		return f, err
	}
	if u.EmailVerified {
		f.verifiedEmail = u.Email
	}
	if u.PasswordHash != "" {
		f.methods["password"] = true
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT provider, data FROM identities WHERE user_id = ?`, userID)
	if err != nil {
		return f, err
	}
	defer rows.Close()
	for rows.Next() {
		var provider, data string
		if err := rows.Scan(&provider, &data); err != nil {
			return f, err
		}
		f.methods[provider] = true
		if provider == "discord" && data != "" {
			var d discordData
			if json.Unmarshal([]byte(data), &d) == nil {
				for guild, roles := range d.Guilds {
					f.guildRoles[guild] = roles
				}
			}
		}
	}
	return f, rows.Err()
}

// SyncRuleGroups makes the user's rule-made memberships match the rules:
// adds the groups they now qualify for and drops rule-made ones they no longer
// do. Memberships added by hand are never touched.
func (s *Store) SyncRuleGroups(ctx context.Context, userID int64) error {
	facts, err := s.userFacts(ctx, userID)
	if err != nil {
		return err
	}
	rules, err := s.ListRules(ctx)
	if err != nil {
		return err
	}
	want := map[int64]bool{}
	for _, r := range rules {
		if facts.matches(r) {
			want[r.GroupID] = true
		}
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `SELECT group_id FROM user_groups WHERE user_id = ? AND source = 'rule'`, userID)
	if err != nil {
		return err
	}
	var drop []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		if !want[id] {
			drop = append(drop, id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range drop {
		if _, err := tx.ExecContext(ctx, `DELETE FROM user_groups WHERE user_id = ? AND group_id = ? AND source = 'rule'`, userID, id); err != nil {
			return err
		}
	}
	for id := range want {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO user_groups (user_id, group_id, source, added_at) VALUES (?, ?, 'rule', ?)`, userID, id, now()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SyncAllRuleGroups re-applies the rules to every user, after a rule changed.
func (s *Store) SyncAllRuleGroups(ctx context.Context) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT id FROM users`)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if err := s.SyncRuleGroups(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
