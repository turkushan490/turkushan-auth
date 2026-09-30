package auth

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	MinPasswordLen = 10
	MaxPasswordLen = 256 // bytes; keeps hashing cost bounded
)

var usernameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,31}$`)

var (
	ErrUsernameInvalid  = errors.New("username must be 3-32 characters (letters, digits, . _ -) and start with a letter or digit")
	ErrPasswordTooShort = errors.New("password must be at least 10 characters")
	ErrPasswordTooLong  = errors.New("password is too long")
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

func ValidatePassword(p string) error {
	if utf8.RuneCountInString(p) < MinPasswordLen {
		return ErrPasswordTooShort
	}
	if len(p) > MaxPasswordLen {
		return ErrPasswordTooLong
	}
	return nil
}
