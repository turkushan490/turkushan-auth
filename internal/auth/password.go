// Package auth holds password hashing and credential validation.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type argonParams struct {
	memory  uint32 // KiB
	time    uint32
	threads uint8
	keyLen  uint32
}

var defaultParams = argonParams{memory: 64 * 1024, time: 3, threads: 2, keyLen: 32}

var ErrMalformedHash = errors.New("malformed password hash")

// Each hash takes 64 MiB of RAM. Capping how many run at once keeps a burst of
// login attempts from exhausting the server's memory; extra requests just wait.
var hashSlots = make(chan struct{}, 4)

func idKey(password, salt []byte, p argonParams, keyLen uint32) []byte {
	hashSlots <- struct{}{}
	defer func() { <-hashSlots }()
	return argon2.IDKey(password, salt, p.time, p.memory, p.threads, keyLen)
}

// HashPassword returns an argon2id hash in PHC string format.
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	p := defaultParams
	key := idKey([]byte(password), salt, p, p.keyLen)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.memory, p.time, p.threads,
		b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword checks password against a hash made by HashPassword.
func VerifyPassword(password, encoded string) (bool, error) {
	// "$argon2id$v=19$m=..,t=..,p=..$salt$key" splits into 6 parts, the first empty.
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrMalformedHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, ErrMalformedHash
	}
	var p argonParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return false, ErrMalformedHash
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, ErrMalformedHash
	}
	key, err := b64.DecodeString(parts[5])
	if err != nil || len(key) == 0 || p.memory == 0 || p.time == 0 || p.threads == 0 {
		return false, ErrMalformedHash
	}

	// Refuse absurd parameters from a tampered hash instead of allocating them.
	if p.memory > 1<<20 || p.time > 20 || len(key) > 128 {
		return false, ErrMalformedHash
	}
	got := idKey([]byte(password), salt, p, uint32(len(key)))
	return subtle.ConstantTimeCompare(got, key) == 1, nil
}
