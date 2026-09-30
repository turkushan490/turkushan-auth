package server

import (
	"net/http/httptest"
	"testing"

	"github.com/turkushan490/turkushan-auth/internal/config"
)

func TestClientIP(t *testing.T) {
	trusted, _ := config.ParsePrefixes("172.16.0.0/12")
	cases := []struct {
		name, remote, realIP, xff, want string
	}{
		{"direct, headers ignored", "203.0.113.9:5555", "1.1.1.1", "2.2.2.2", "203.0.113.9"},
		{"via NPM, X-Real-IP", "172.17.0.5:4444", "198.51.100.7", "", "198.51.100.7"},
		{"via NPM, XFF only", "172.17.0.5:4444", "", "198.51.100.7", "198.51.100.7"},
		{"via NPM, spoofed XFF prefix", "172.17.0.5:4444", "", "6.6.6.6, 198.51.100.7", "198.51.100.7"},
		{"via NPM, chain of trusted proxies", "172.17.0.5:4444", "", "198.51.100.7, 172.18.0.2", "198.51.100.7"},
		{"via NPM, no headers", "172.17.0.5:4444", "", "", "172.17.0.5"},
		{"ipv6 peer", "[2001:db8::1]:443", "", "", "2001:db8::1"},
	}
	for _, tc := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = tc.remote
		if tc.realIP != "" {
			r.Header.Set("X-Real-IP", tc.realIP)
		}
		if tc.xff != "" {
			r.Header.Set("X-Forwarded-For", tc.xff)
		}
		if got := clientIP(r, trusted); got != tc.want {
			t.Errorf("%s: got %s, want %s", tc.name, got, tc.want)
		}
	}
}
