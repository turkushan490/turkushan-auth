package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestBackupDownload(t *testing.T) {
	h, st := newTestServer(t)
	admin := signedIn(t, h, st, "boss", true)
	helper := signedIn(t, h, st, "helper", false)
	g := mkGroup(t, admin, "Settings")
	admin.put(fmt.Sprintf("/api/admin/groups/%d/perms", g), map[string]any{"perms": []string{"settings", "users"}})
	addMember(t, admin, g, userID(t, st, "helper"), true)

	rec, _ := admin.do(http.MethodPost, "/api/admin/backup", map[string]any{}, nil)
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), "SQLite format 3\x00") {
		t.Fatalf("admin backup: %d, %d bytes", rec.Code, rec.Body.Len())
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Errorf("not a download: %q", cd)
	}
	if rec, _ := helper.do(http.MethodPost, "/api/admin/backup", map[string]any{}, nil); rec.Code != http.StatusForbidden {
		t.Errorf("non-admin backup: %d", rec.Code)
	}
	if rec, _ := newClient(t, h).do(http.MethodPost, "/api/admin/backup", map[string]any{}, nil); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous backup: %d", rec.Code)
	}
}
