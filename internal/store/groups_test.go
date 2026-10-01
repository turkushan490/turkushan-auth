package store

import (
	"context"
	"testing"
)

func TestRightsFromGroups(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	uid, _ := s.RegisterUser(ctx, "mod", "hash", "")
	site1, _ := s.CreateSite(ctx, &Site{Name: "A", Host: "a.example.com"})
	site2, _ := s.CreateSite(ctx, &Site{Name: "B", Host: "b.example.com"})
	g1, _ := s.CreateGroup(ctx, "Mods", "")
	g2, _ := s.CreateGroup(ctx, "Helpers", "")
	s.SetGroupPerms(ctx, g1, []string{PermUsers, "nonsense", PermUsers})
	s.SetGroupSite(ctx, g1, site1, true, true)
	s.SetGroupPerms(ctx, g2, []string{PermAudit})
	s.SetGroupSite(ctx, g2, site2, true, false)

	u, _ := s.UserByID(ctx, uid)
	r, err := s.UserRights(ctx, u)
	if err != nil || r.Staff() {
		t.Fatalf("no groups yet: %+v %v", r, err)
	}

	s.AddMember(ctx, uid, g1)
	s.AddMember(ctx, uid, g2)
	r, _ = s.UserRights(ctx, u)
	if !r.Staff() || !r.Has(PermUsers) || !r.Has(PermAudit) || r.Has(PermSettings) || r.Has("nonsense") {
		t.Errorf("combined global rights: %+v", r)
	}
	if !r.CanApprove(site1) || r.CanApprove(site2) {
		t.Errorf("site approve rights: %+v", r.ApproveSites)
	}
	for site, want := range map[int64]bool{site1: true, site2: true} {
		if got, _ := s.GroupOpens(ctx, uid, site); got != want {
			t.Errorf("GroupOpens(%d) = %v", site, got)
		}
	}

	// Both rights off removes the row.
	s.SetGroupSite(ctx, g2, site2, false, false)
	if got, _ := s.GroupOpens(ctx, uid, site2); got {
		t.Error("site right not removed")
	}

	// Admins have everything without any group.
	s.SetAdmin(ctx, uid, true)
	u, _ = s.UserByID(ctx, uid)
	r, _ = s.UserRights(ctx, u)
	if !r.Admin || !r.Has(PermSettings) || !r.CanApprove(site2) {
		t.Errorf("admin rights: %+v", r)
	}
}

func TestRulesByLoginMethodAndDiscord(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	uid, _ := s.RegisterUser(ctx, "gamer", "", "") // no password: signs in with Discord
	groups := map[string]int64{}
	for _, name := range []string{"Discord users", "Password users", "My server", "VIP role", "Other role", "Google users"} {
		groups[name], _ = s.CreateGroup(ctx, name, "")
	}
	s.CreateRule(ctx, groups["Discord users"], RuleMethod, "discord")
	s.CreateRule(ctx, groups["Password users"], RuleMethod, "password")
	s.CreateRule(ctx, groups["Google users"], RuleMethod, "google")
	s.CreateRule(ctx, groups["My server"], RuleDiscordServer, "111111")
	s.CreateRule(ctx, groups["VIP role"], RuleDiscordRole, "111111:222222")
	s.CreateRule(ctx, groups["Other role"], RuleDiscordRole, "111111:999999")

	member := func(name string) bool {
		var n int
		s.DB.QueryRow(`SELECT COUNT(*) FROM user_groups WHERE user_id = ? AND group_id = ?`, uid, groups[name]).Scan(&n)
		return n == 1
	}
	check := func(when string, want ...string) {
		t.Helper()
		if err := s.SyncRuleGroups(ctx, uid); err != nil {
			t.Fatal(err)
		}
		wanted := map[string]bool{}
		for _, w := range want {
			wanted[w] = true
		}
		for name := range groups {
			if member(name) != wanted[name] {
				t.Errorf("%s: member of %q = %v, want %v", when, name, member(name), wanted[name])
			}
		}
	}

	check("no logins linked")

	s.DB.Exec(`INSERT INTO identities (provider, subject, user_id, data, created_at) VALUES ('discord', '42', ?, ?, 1)`,
		uid, `{"guilds": {"111111": ["222222", "333333"]}}`)
	check("discord linked, in server with VIP role", "Discord users", "My server", "VIP role")

	// Role taken away in Discord: seen at the next Discord login.
	s.DB.Exec(`UPDATE identities SET data = ? WHERE user_id = ?`, `{"guilds": {"111111": []}}`, uid)
	check("role removed", "Discord users", "My server")

	// Left the server.
	s.DB.Exec(`UPDATE identities SET data = '{"guilds": {}}' WHERE user_id = ?`, uid)
	check("left the server", "Discord users")

	// Sets a password later.
	s.SetPassword(ctx, uid, "somehash")
	check("password set", "Discord users", "Password users")
}
