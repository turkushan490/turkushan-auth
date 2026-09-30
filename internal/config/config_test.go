package config

import (
	"net/netip"
	"os"
	"path/filepath"
	"testing"
)

func TestParsePrefixes(t *testing.T) {
	got, err := ParsePrefixes(" 172.16.0.0/12, 192.168.0.6 ,,10.1.2.3/8")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"172.16.0.0/12", "192.168.0.6/32", "10.0.0.0/8"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != netip.MustParsePrefix(want[i]) {
			t.Errorf("prefix %d: got %v, want %s", i, got[i], want[i])
		}
	}

	if _, err := ParsePrefixes("not-an-ip"); err == nil {
		t.Error("expected error for invalid entry")
	}
}

func TestLoadSecretGeneratesAndReuses(t *testing.T) {
	t.Setenv("SESSION_SECRET", "")
	c := &Config{DataDir: t.TempDir()}
	if err := c.LoadSecret(); err != nil {
		t.Fatal(err)
	}
	first := string(c.SessionSecret)
	if len(first) < 32 {
		t.Fatalf("secret too short: %d", len(first))
	}
	if _, err := os.Stat(filepath.Join(c.DataDir, "secret")); err != nil {
		t.Fatalf("secret file not written: %v", err)
	}

	c2 := &Config{DataDir: c.DataDir}
	if err := c2.LoadSecret(); err != nil {
		t.Fatal(err)
	}
	if string(c2.SessionSecret) != first {
		t.Error("secret changed between starts")
	}
}

func TestLoadSecretFromEnv(t *testing.T) {
	t.Setenv("SESSION_SECRET", "short")
	c := &Config{DataDir: t.TempDir()}
	if err := c.LoadSecret(); err == nil {
		t.Error("expected error for short SESSION_SECRET")
	}
}
