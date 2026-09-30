package auth

import (
	"errors"
	"net/mail"
	"net/url"
	"strings"
	"sync"
)

// SafeRedirect returns rd if it points at baseDomain or one of its subdomains,
// otherwise "". This is the open-redirect guard for the ?rd= parameter.
func SafeRedirect(rd, baseDomain string, allowHTTP bool) string {
	if rd == "" || len(rd) > 2048 || strings.ContainsAny(rd, "\\\r\n\t") {
		return ""
	}
	u, err := url.Parse(rd)
	if err != nil || u.User != nil || u.Host == "" {
		return ""
	}
	if u.Scheme != "https" && !(allowHTTP && u.Scheme == "http") {
		return ""
	}
	base := strings.ToLower(strings.TrimPrefix(baseDomain, "."))
	host := strings.ToLower(u.Hostname())
	if base == "" || (host != base && !strings.HasSuffix(host, "."+base)) {
		return ""
	}
	return u.String()
}

var ErrEmailInvalid = errors.New("that email address doesn't look right")

// NormalizeEmail lowercases and checks an optional email address; "" stays "".
func NormalizeEmail(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", nil
	}
	if len(s) > 254 {
		return "", ErrEmailInvalid
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s {
		return "", ErrEmailInvalid
	}
	if domain := s[strings.LastIndex(s, "@")+1:]; !strings.Contains(domain, ".") {
		return "", ErrEmailInvalid
	}
	return s, nil
}

var dummyHash = sync.OnceValue(func() string {
	h, _ := HashPassword("timing-equalizer-Password!")
	return h
})

// DummyVerify costs as much as a real password check, so a login for an
// unknown username takes as long as one for a known username.
func DummyVerify(password string) {
	VerifyPassword(password, dummyHash())
}
