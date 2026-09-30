// Package config reads the portal settings from environment variables.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Port              int
	AppURL            *url.URL // public portal URL, e.g. https://auth.example.com
	CookieDomain      string   // e.g. .example.com
	DataDir           string
	AdminUser         string
	AdminPassword     string
	TrustedProxies    []netip.Prefix // only these peers may set x-real-ip / x-forwarded-for
	PortalInternalURL string         // how NPM reaches the portal, used in generated snippets
	SessionSecret     []byte
	PUID, PGID        int // user/group to drop to when started as root (Unraid: nobody:users)
}

func Load() (*Config, error) {
	c := &Config{
		DataDir:           env("DATA_DIR", "/data"),
		AdminUser:         os.Getenv("ADMIN_USER"),
		AdminPassword:     os.Getenv("ADMIN_PASSWORD"),
		PortalInternalURL: strings.TrimRight(env("PORTAL_INTERNAL_URL", ""), "/"),
	}

	var err error
	if c.Port, err = envInt("PORT", 3010); err != nil {
		return nil, err
	}
	if c.PUID, err = envInt("PUID", 99); err != nil {
		return nil, err
	}
	if c.PGID, err = envInt("PGID", 100); err != nil {
		return nil, err
	}

	u, err := url.Parse(env("APP_URL", ""))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, errors.New("APP_URL is required and must be a full URL like https://auth.example.com")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	c.AppURL = u
	c.CookieDomain = env("COOKIE_DOMAIN", DefaultCookieDomain(u.Hostname()))

	if c.TrustedProxies, err = ParsePrefixes(env("TRUSTED_PROXIES", "172.16.0.0/12")); err != nil {
		return nil, fmt.Errorf("TRUSTED_PROXIES: %w", err)
	}
	return c, nil
}

// DefaultCookieDomain shares the cookie with the portal's sibling subdomains:
// auth.example.com → .example.com. A bare domain is used as is. For localhost
// or an IP it returns "" (a host-only cookie), since browsers reject those as a domain.
func DefaultCookieDomain(host string) string {
	if _, err := netip.ParseAddr(host); err == nil || !strings.Contains(host, ".") {
		return ""
	}
	labels := strings.Split(host, ".")
	if len(labels) >= 3 {
		return "." + strings.Join(labels[1:], ".")
	}
	return "." + host
}

// ParsePrefixes parses a comma-separated list of IPs and CIDRs.
func ParsePrefixes(s string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "/") {
			p, err := netip.ParsePrefix(part)
			if err != nil {
				return nil, fmt.Errorf("%q: %w", part, err)
			}
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(part)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", part, err)
		}
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
}

// LoadSecret sets SessionSecret from SESSION_SECRET, or from DataDir/secret,
// generating that file on first start.
func (c *Config) LoadSecret() error {
	if s := os.Getenv("SESSION_SECRET"); s != "" {
		if len(s) < 32 {
			return errors.New("SESSION_SECRET must be at least 32 characters")
		}
		c.SessionSecret = []byte(s)
		return nil
	}

	path := filepath.Join(c.DataDir, "secret")
	b, err := os.ReadFile(path)
	if err == nil {
		s := strings.TrimSpace(string(b))
		if len(s) < 32 {
			return fmt.Errorf("%s is too short; delete it to regenerate", path)
		}
		c.SessionSecret = []byte(s)
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	s := hex.EncodeToString(buf)
	if err := os.WriteFile(path, []byte(s+"\n"), 0o600); err != nil {
		return err
	}
	c.SessionSecret = []byte(s)
	return nil
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) (int, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number, got %q", key, v)
	}
	return n, nil
}
