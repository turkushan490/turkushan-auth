package auth

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MinPasswordLen = 6
	MaxPasswordLen = 256 // bytes; keeps hashing cost bounded
)

var usernameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,31}$`)

var (
	ErrUsernameInvalid  = errors.New("username must be 3-32 characters (letters, digits, . _ -) and start with a letter or digit")
	ErrPasswordTooShort = errors.New("password must be at least 6 characters")
	ErrPasswordTooLong  = errors.New("password is too long")
	ErrPasswordNoUpper  = errors.New("password needs at least 1 capital letter")
	ErrPasswordNoSymbol = errors.New("password needs at least 1 symbol, like ! @ # ?")
)

// NormalizeUsername trims and lowercases a username so lookups are case-insensitive.
func NormalizeUsername(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ValidateUsername expects an already normalized username.
func ValidateUsername(u string) error {
	if !usernameRe.MatchString(u) {
		return ErrUsernameInvalid
	}
	return nil
}

// ValidatePassword requires at least 6 characters, a capital letter and a symbol.
func ValidatePassword(p string) error {
	if utf8.RuneCountInString(p) < MinPasswordLen {
		return ErrPasswordTooShort
	}
	if len(p) > MaxPasswordLen {
		return ErrPasswordTooLong
	}
	var upper, symbol bool
	for _, r := range p {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r):
			symbol = true
		}
	}
	if !upper {
		return ErrPasswordNoUpper
	}
	if !symbol {
		return ErrPasswordNoSymbol
	}
	return nil
}
