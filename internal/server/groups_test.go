package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/turkushan490/turkushan-auth/internal/store"
)

// mkGroup creates a group through the API and returns its id.
func mkGroup(t *testing.T, admin *client, name string) int64 {
	t.Helper()
	code, out := admin.post("/api/admin/groups", map[string]string{"name": name})
	if code != http.StatusOK {
		t.Fatalf("create group %s: %d %v", name, code, out)
	}
	return int64(out["id"].(float64))
}

func userID(t *testing.T, st *store.Store, username string) int64 {
	t.Helper()
	u, err := st.UserByUsername(context.Background(), username)
	if err != nil {
		t.Fatalf("user %s: %v", username, err)
	}
	return u.ID
}

func addMember(t *testing.T, c *client, group, user int64, member bool) int {
	t.Helper()
	code, _ := c.put(fmt.Sprintf("/api/admin/groups/%d/members", group), map[string]any{"user_id": user, "member": member})
	return code
}

func TestGroupGivesSiteAccess(t *testing.T) {
	h, st := newTestServer(t)
	mangaID := addSite(t, st, "manga.example.com", true, false)
	admin := signedIn(t, h, st, "boss", true)
	alice := signedIn(t, h, st, "alice", false)
	aliceID := userID(t, st, "alice")
	const orig = "https://manga.example.com/"

	friends := mkGroup(t, admin, "Friends")
	if code, _ := admin.put(fmt.Sprintf("/api/admin/groups/%d/sites", friends), map[string]any{"site_id": mangaID, "can_open": true}); code != http.StatusOK {
		t.Fatalf("group site: %d", code)
	}
	if addMember(t, admin, friends, aliceID, true) != http.StatusOK {
		t.Fatal("add member failed")
	}

	// In the group: straight in, and no pending request is filed.
	if res := alice.forward(http.MethodGet, orig); res.StatusCode != http.StatusOK {
		t.Fatalf("group member: %d", res.StatusCode)
	}
	if n, _ := st.CountPending(context.Background()); n != 0 {
		t.Errorf("pending requests for a group member: %d", n)
	}

	// An explicit deny for this person wins over the group.
	admin.put("/api/admin/access", map[string]any{"user_id": aliceID, "site_id": mangaID, "status": "denied"})
	if res := alice.forward(http.MethodGet, orig); res.StatusCode != http.StatusForbidden {
		t.Errorf("denied but in group: %d", res.StatusCode)
	}
	admin.put("/api/admin/access", map[string]any{"user_id": aliceID, "site_id": mangaID, "status": "approved"})

	// Out of the group, and without personal approval: no access.
	st.DB.Exec(`DELETE FROM site_access`)
	if addMember(t, admin, friends, aliceID, false) != http.StatusOK {
		t.Fatal("remove member failed")
	}
	if res := alice.forward(http.MethodGet, orig); res.StatusCode != http.StatusForbidden {
		t.Errorf("after leaving the group: %d", res.StatusCode)
	}

	// Deleting a group takes its access away too.
	addMember(t, admin, friends, aliceID, true)
	st.DB.Exec(`DELETE FROM site_access`)
	if code, _ := admin.del(fmt.Sprintf("/api/admin/groups/%d", friends)); code != http.StatusOK {
		t.Fatalf("delete group: %d", code)
	}
	if res := alice.forward(http.MethodGet, orig); res.StatusCode != http.StatusForbidden {
		t.Errorf("after group deleted: %d", res.StatusCode)
	}
}

func TestSiteModerator(t *testing.T) {
	h, st := newTestServer(t)
	ctx := context.Background()
	mangaID := addSite(t, st, "manga.example.com", true, false)
	stashID := addSite(t, st, "stash.example.com", true, false)
	admin := signedIn(t, h, st, "boss", true)
	mod := signedIn(t, h, st, "mod", false)
	signedIn(t, h, st, "carl", false)
	carlID := userID(t, st, "carl")
	st.RequestAccess(ctx, carlID, mangaID)
	st.RequestAccess(ctx, carlID, stashID)

	// Not staff yet.
	if rec, _ := mod.do(http.MethodGet, "/api/admin/data", nil, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("before rights: %d", rec.Code)
	}
	if u := mod.session()["user"].(map[string]any); u["staff"] != false {
		t.Errorf("staff flag before rights: %v", u)
	}

	mods := mkGroup(t, admin, "Manga mods")
	admin.put(fmt.Sprintf("/api/admin/groups/%d/sites", mods), map[string]any{"site_id": mangaID, "can_approve": true})
	addMember(t, admin, mods, userID(t, st, "mod"), true)

	if u := mod.session()["user"].(map[string]any); u["staff"] != true {
		t.Errorf("staff flag with rights: %v", u)
	}
	rec, data := mod.do(http.MethodGet, "/api/admin/data", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("moderator data: %d", rec.Code)
	}
	// Sees only manga's request, no users, no internal addresses, no mail settings.
	access := data["access"].([]any)
	if len(access) != 1 || access[0].(map[string]any)["site_host"] != "manga.example.com" {
		t.Errorf("moderator sees requests: %v", access)
	}
	if len(data["users"].([]any)) != 0 {
		t.Error("moderator sees the user list")
	}
	for _, s := range data["sites"].([]any) {
		if site := s.(map[string]any); site["upstream"] != "" || site["snippet"] != "" {
			t.Errorf("moderator sees internal address: %v", site)
		}
	}
	if _, has := data["settings"].(map[string]any)["smtp"]; has {
		t.Error("moderator sees mail settings")
	}

	// May approve manga, not stash.
	if code, _ := mod.put("/api/admin/access", map[string]any{"user_id": carlID, "site_id": mangaID, "status": "approved"}); code != http.StatusOK {
		t.Errorf("approve own site: %d", code)
	}
	if code, _ := mod.put("/api/admin/access", map[string]any{"user_id": carlID, "site_id": stashID, "status": "approved"}); code != http.StatusForbidden {
		t.Errorf("approve other site: %d", code)
	}

	// And nothing else.
	modID := userID(t, st, "mod")
	forbidden := []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/api/admin/sites", map[string]any{"name": "X", "host": "x.example.com", "upstream": "http://1.2.3.4"}},
		{http.MethodPut, "/api/admin/settings", map[string]any{"portal_url": "http://1.2.3.4:1"}},
		{http.MethodPut, "/api/admin/appearance", map[string]any{"site_name": "x"}},
		{http.MethodGet, "/api/admin/audit", nil},
		{http.MethodPost, "/api/admin/users", map[string]any{"username": "newbie", "password": "Secret!1"}},
		{http.MethodDelete, fmt.Sprintf("/api/admin/users/%d", carlID), nil},
		{http.MethodPut, fmt.Sprintf("/api/admin/users/%d/admin", modID), map[string]any{"admin": true}},
		{http.MethodPost, "/api/admin/groups", map[string]any{"name": "Mine"}},
		{http.MethodPut, fmt.Sprintf("/api/admin/groups/%d/perms", mods), map[string]any{"perms": []string{"users"}}},
		{http.MethodPut, fmt.Sprintf("/api/admin/groups/%d/members", mods), map[string]any{"user_id": carlID, "member": true}},
	}
	for _, f := range forbidden {
		if rec, _ := mod.do(f.method, f.path, f.body, nil); rec.Code != http.StatusForbidden {
			t.Errorf("%s %s: got %d, want 403", f.method, f.path, rec.Code)
		}
	}
}

func TestNoPrivilegeEscalation(t *testing.T) {
	h, st := newTestServer(t)
	mangaID := addSite(t, st, "manga.example.com", true, false)
	admin := signedIn(t, h, st, "boss", true)
	helper := signedIn(t, h, st, "helper", false)
	helperID := userID(t, st, "helper")
	bossID := userID(t, st, "boss")

	// helper may manage users and groups, nothing more.
	helpers := mkGroup(t, admin, "Helpers")
	if code, _ := admin.put(fmt.Sprintf("/api/admin/groups/%d/perms", helpers), map[string]any{"perms": []string{"users", "groups", "bogus"}}); code != http.StatusOK {
		t.Fatalf("set perms: %d", code)
	}
	addMember(t, admin, helpers, helperID, true)
	// A group with more rights than helper has, and one that opens a site.
	powerful := mkGroup(t, admin, "Site managers")
	admin.put(fmt.Sprintf("/api/admin/groups/%d/perms", powerful), map[string]any{"perms": []string{"sites", "settings"}})
	vip := mkGroup(t, admin, "VIP")
	admin.put(fmt.Sprintf("/api/admin/groups/%d/sites", vip), map[string]any{"site_id": mangaID, "can_open": true})

	_, data := helper.do(http.MethodGet, "/api/admin/data", nil, nil)
	for _, g := range data["groups"].([]any) {
		if g := g.(map[string]any); g["name"] == "Helpers" && fmt.Sprint(g["perms"]) != "[groups users]" {
			t.Errorf("unknown right was stored: %v", g["perms"])
		}
	}

	// Can't join or change a group that has rights they lack, or give themselves rights.
	if code := addMember(t, helper, powerful, helperID, true); code != http.StatusForbidden {
		t.Errorf("join a more powerful group: %d", code)
	}
	if code := addMember(t, helper, vip, helperID, true); code != http.StatusForbidden {
		t.Errorf("join a group that opens a site they can't approve: %d", code)
	}
	if code, _ := helper.put(fmt.Sprintf("/api/admin/groups/%d/perms", helpers), map[string]any{"perms": []string{"users", "groups", "settings"}}); code != http.StatusForbidden {
		t.Errorf("raise own group's rights: %d", code)
	}
	if code, _ := helper.put(fmt.Sprintf("/api/admin/groups/%d/sites", helpers), map[string]any{"site_id": mangaID, "can_approve": true}); code != http.StatusForbidden {
		t.Errorf("give own group site rights: %d", code)
	}
	if code, _ := helper.del(fmt.Sprintf("/api/admin/groups/%d", powerful)); code != http.StatusForbidden {
		t.Errorf("delete a more powerful group: %d", code)
	}
	if code, _ := helper.post(fmt.Sprintf("/api/admin/groups/%d/rules", powerful), map[string]any{"kind": "everyone"}); code != http.StatusForbidden {
		t.Errorf("add a rule to a more powerful group: %d", code)
	}
	if code, _ := helper.put(fmt.Sprintf("/api/admin/users/%d/admin", helperID), map[string]any{"admin": true}); code != http.StatusForbidden {
		t.Errorf("make self admin: %d", code)
	}
	if code, _ := helper.post("/api/admin/users", map[string]any{"username": "sneaky", "password": "Secret!1", "admin": true}); code != http.StatusForbidden {
		t.Errorf("create an admin: %d", code)
	}
	if code, _ := helper.post("/api/admin/users", map[string]any{"username": "sneaky", "password": "Secret!1", "group_ids": []int64{powerful}}); code != http.StatusForbidden {
		t.Errorf("create a user in a more powerful group: %d", code)
	}
	// Can't touch an admin account.
	if code, _ := helper.put(fmt.Sprintf("/api/admin/users/%d/password", bossID), map[string]string{"password": "Hacked!1"}); code != http.StatusForbidden {
		t.Errorf("reset an admin's password: %d", code)
	}
	if code, _ := helper.del(fmt.Sprintf("/api/admin/users/%d", bossID)); code != http.StatusForbidden {
		t.Errorf("delete an admin: %d", code)
	}

	// What helper IS allowed: groups without extra rights, and normal users.
	plain := mkGroup(t, helper, "Family")
	if code := addMember(t, helper, plain, helperID, true); code != http.StatusOK {
		t.Errorf("manage a plain group: %d", code)
	}
	if code, out := helper.post("/api/admin/users", map[string]any{"username": "gran", "password": "Secret!1", "group_ids": []int64{plain}}); code != http.StatusOK {
		t.Errorf("create a normal user: %d %v", code, out)
	}
	for _, name := range []string{"Admins", "admin", "  "} {
		if code, _ := helper.post("/api/admin/groups", map[string]any{"name": name}); code != http.StatusBadRequest {
			t.Errorf("group named %q: %d", name, code)
		}
	}
	if code, _ := helper.post("/api/admin/groups", map[string]any{"name": "family"}); code != http.StatusConflict {
		t.Errorf("duplicate group name: %d", code)
	}
}

func TestAdmins(t *testing.T) {
	h, st := newTestServer(t)
	admin := signedIn(t, h, st, "boss", true)
	dana := signedIn(t, h, st, "dana", false)
	danaID, bossID := userID(t, st, "dana"), userID(t, st, "boss")

	if code, _ := admin.put(fmt.Sprintf("/api/admin/users/%d/admin", bossID), map[string]any{"admin": false}); code != http.StatusForbidden {
		t.Errorf("demote self: %d", code)
	}
	if code, _ := admin.put(fmt.Sprintf("/api/admin/users/%d/admin", danaID), map[string]any{"admin": true}); code != http.StatusOK {
		t.Fatalf("promote: %d", code)
	}
	if rec, data := dana.do(http.MethodGet, "/api/admin/data", nil, nil); rec.Code != http.StatusOK || data["me"].(map[string]any)["admin"] != true {
		t.Errorf("new admin: %d %v", rec.Code, data["me"])
	}
	// A second admin can demote the first; then the first is locked out of the panel.
	if code, _ := dana.put(fmt.Sprintf("/api/admin/users/%d/admin", bossID), map[string]any{"admin": false}); code != http.StatusOK {
		t.Errorf("demote other admin: %d", code)
	}
	if rec, _ := admin.do(http.MethodGet, "/api/admin/data", nil, nil); rec.Code != http.StatusForbidden {
		t.Errorf("demoted admin still in the panel: %d", rec.Code)
	}
	if n, _ := st.CountAdmins(context.Background()); n != 1 {
		t.Errorf("admins left: %d", n)
	}
}

func TestCreateUser(t *testing.T) {
	h, st := newTestServer(t)
	srv := withMail(t, st)
	admin := signedIn(t, h, st, "boss", true)
	friends := mkGroup(t, admin, "Friends")

	// With a password set by the admin.
	if code, out := admin.post("/api/admin/users", map[string]any{"username": "Piet", "password": "Secret!1", "group_ids": []int64{friends}}); code != http.StatusOK {
		t.Fatalf("create with password: %d %v", code, out)
	}
	if code, _ := newClient(t, h).post("/api/login", map[string]string{"username": "piet", "password": "Secret!1"}); code != http.StatusOK {
		t.Errorf("login as created user: %d", code)
	}
	var inGroup int
	st.DB.QueryRow(`SELECT COUNT(*) FROM user_groups WHERE user_id = ? AND group_id = ?`, userID(t, st, "piet"), friends).Scan(&inGroup)
	if inGroup != 1 {
		t.Error("created user is not in the chosen group")
	}

	for _, bad := range []map[string]any{
		{"username": "piet", "password": "Secret!1"},                  // taken
		{"username": "x", "password": "Secret!1"},                     // bad username
		{"username": "klaas", "password": "weak"},                     // weak password
		{"username": "klaas", "invite": true},                         // invite without email
		{"username": "klaas", "password": "Secret!1", "email": "nope"}, // bad email
	} {
		if code, _ := admin.post("/api/admin/users", bad); code < 400 || code >= 500 {
			t.Errorf("create %v: %d", bad, code)
		}
	}

	// Invite: no password until they use the link; the link also verifies their address.
	before := srv.Count()
	if code, out := admin.post("/api/admin/users", map[string]any{"username": "anna", "email": "Anna@Example.com", "invite": true}); code != http.StatusOK || out["warning"] != nil {
		t.Fatalf("invite: %d %v", code, out)
	}
	if srv.Count() != before+1 {
		t.Fatal("no invite mail sent")
	}
	to, text := srv.Last(t)
	if !strings.Contains(to, "anna@example.com") || !strings.Contains(text, "username is anna") {
		t.Errorf("invite mail: to %q\n%s", to, text)
	}
	if code, _ := newClient(t, h).post("/api/login", map[string]string{"username": "anna", "password": ""}); code != http.StatusUnauthorized {
		t.Errorf("login to an account without a password: %d", code)
	}
	token := tokenRe.FindStringSubmatch(text)[1]
	c := newClient(t, h)
	if code, out := c.post("/api/password/reset", map[string]string{"token": token, "password": "MyOwn!pass1"}); code != http.StatusOK {
		t.Fatalf("accept invite: %d %v", code, out)
	}
	if code, _ := c.post("/api/login", map[string]string{"username": "anna", "password": "MyOwn!pass1"}); code != http.StatusOK {
		t.Fatalf("login after invite: %d", code)
	}
	if u := c.session()["user"].(map[string]any); u["email_verified"] != true || u["has_password"] != true {
		t.Errorf("after invite: %v", u)
	}
}

func TestAutoAddRules(t *testing.T) {
	h, st := newTestServer(t)
	srv := withMail(t, st)
	ctx := context.Background()
	admin := signedIn(t, h, st, "boss", true)
	old := signedIn(t, h, st, "oldtimer", false)
	_ = old
	everyone := mkGroup(t, admin, "Everyone")
	staff := mkGroup(t, admin, "Company")

	inGroup := func(username string, group int64) string {
		var source string
		st.DB.QueryRow(`SELECT source FROM user_groups WHERE user_id = ? AND group_id = ?`, userID(t, st, username), group).Scan(&source)
		return source
	}
	rule := func(group int64, kind, value string) (int, map[string]any) {
		return admin.post(fmt.Sprintf("/api/admin/groups/%d/rules", group), map[string]any{"kind": kind, "value": value})
	}

	for _, bad := range [][2]string{{"method", "telepathy"}, {"email", "nope"}, {"domain", "not a domain"}, {"discord_server", "abc"}, {"discord_role", "123"}, {"magic", ""}} {
		if code, _ := rule(everyone, bad[0], bad[1]); code != http.StatusBadRequest {
			t.Errorf("rule %v: %d", bad, code)
		}
	}

	// "Everyone" applies to existing accounts right away, and to new ones.
	if code, out := rule(everyone, "everyone", ""); code != http.StatusOK {
		t.Fatalf("everyone rule: %d %v", code, out)
	}
	if inGroup("oldtimer", everyone) != "rule" {
		t.Error("existing user not added by the everyone rule")
	}
	if code, _ := rule(staff, "domain", "@Company.example"); code != http.StatusOK {
		t.Fatal("domain rule failed")
	}

	c := newClient(t, h)
	c.post("/api/register", map[string]string{"username": "nina", "password": "Secret!1", "email": "nina@company.example"})
	if inGroup("nina", everyone) != "rule" {
		t.Error("new user not added by the everyone rule")
	}
	// The domain rule waits until the address is verified.
	if inGroup("nina", staff) != "" {
		t.Error("domain rule applied to an unverified address")
	}
	c.post("/api/verify-email", map[string]string{"token": linkToken(t, srv, "nina@company.example", "/verify")})
	if inGroup("nina", staff) != "rule" {
		t.Error("domain rule not applied after verifying")
	}

	// Changing the address makes it unverified again, so the rule no longer holds.
	c.post("/api/account/email", map[string]string{"email": "nina@elsewhere.example", "password": "Secret!1"})
	if inGroup("nina", staff) != "" {
		t.Error("still in the domain group after changing address")
	}

	// A manual membership survives removing the rule; a rule-made one doesn't.
	ninaID := userID(t, st, "nina")
	addMember(t, admin, everyone, ninaID, true)
	rules, _ := st.ListRules(ctx)
	for _, r := range rules {
		if r.GroupID == everyone {
			if code, _ := admin.del(fmt.Sprintf("/api/admin/rules/%d", r.ID)); code != http.StatusOK {
				t.Fatalf("delete rule: %d", code)
			}
		}
	}
	if inGroup("nina", everyone) != "manual" {
		t.Error("manual membership lost when the rule was removed")
	}
	if inGroup("oldtimer", everyone) != "" {
		t.Error("rule-made membership kept after the rule was removed")
	}
}
