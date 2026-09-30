package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	h, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("unexpected hash format: %s", h)
	}
	if strings.Contains(h, "correct horse") {
		t.Fatal("hash contains the plain password")
	}

	ok, err := VerifyPassword("correct horse battery", h)
	if err != nil || !ok {
		t.Fatalf("correct password rejected: ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword("wrong horse battery", h)
	if err != nil || ok {
		t.Fatalf("wrong password accepted: ok=%v err=%v", ok, err)
	}

	h2, _ := HashPassword("correct horse battery")
	if h == h2 {
		t.Error("two hashes of the same password are identical (salt not random)")
	}
}

func TestVerifyMalformed(t *testing.T) {
	for _, bad := range []string{
		"",
		"plain",
		"$argon2i$v=19$m=65536,t=3,p=2$c2FsdA$a2V5",
		"$argon2id$v=18$m=65536,t=3,p=2$c2FsdA$a2V5",
		"$argon2id$v=19$m=0,t=3,p=2$c2FsdA$a2V5",
		"$argon2id$v=19$m=65536,t=3,p=2$!!!$a2V5",
		"$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$",
	} {
		if ok, err := VerifyPassword("x", bad); ok || err == nil {
			t.Errorf("%q: expected ErrMalformedHash, got ok=%v err=%v", bad, ok, err)
		}
	}
}

func TestUsername(t *testing.T) {
	if got := NormalizeUsername("  Turkushan "); got != "turkushan" {
		t.Errorf("normalize: got %q", got)
	}
	for _, good := range []string{"abc", "turk.ushan", "a_b-c", "9lives", strings.Repeat("a", 32)} {
		if err := ValidateUsername(good); err != nil {
			t.Errorf("%q rejected: %v", good, err)
		}
	}
	for _, bad := range []string{"", "ab", "-abc", ".abc", "Abc", "a b c", "abc@x", strings.Repeat("a", 33)} {
		if err := ValidateUsername(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestPassword(t *testing.T) {
	for pw, want := range map[string]error{
		"Ab!12":                         ErrPasswordTooShort,
		"Ab!123":                        nil,
		"Wacht#woord":                   nil,
		"abc!123":                       ErrPasswordNoUpper,
		"Abc1234":                       ErrPasswordNoSymbol,
		"Abc 1234":                      ErrPasswordNoSymbol, // a space is not a symbol
		"A!" + strings.Repeat("x", 255): ErrPasswordTooLong,
	} {
		if got := ValidatePassword(pw); got != want {
			t.Errorf("%q: got %v, want %v", pw, got, want)
		}
	}
}
