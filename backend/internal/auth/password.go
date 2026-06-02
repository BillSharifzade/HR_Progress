package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/sethvargo/go-password/password"
	"golang.org/x/crypto/argon2"
)

// argon2id parameters — chosen for tens-of-users scale, balanced for 1-2 logins/sec.
const (
	argonTime    = 2
	argonMemory  = 64 * 1024
	argonThreads = 2
	argonKeyLen  = 32
	argonSaltLen = 16
)

var ErrInvalidHashFormat = errors.New("invalid argon2id hash format")

// passwordGenerator is the shared generator from sethvargo/go-password.
// The library refuses to repeat characters from its ambiguous-symbol set
// (0/O, 1/l/I, etc.), so passwords stay readable when handed off verbally.
var passwordGenerator = func() *password.Generator {
	g, err := password.NewGenerator(&password.GeneratorInput{
		Symbols: "!@#$%^&*-_=+?",
	})
	if err != nil {
		panic(fmt.Errorf("password generator init: %w", err))
	}
	return g
}()

// GenerateTempPassword returns a 16-char crypto-random password:
//   - 4 digits, 2 symbols, the rest letters,
//   - no repeats, no ambiguous chars (0/O/1/l/I),
//   - rejection-sampled (no modulo bias) by the upstream library.
//
// Suitable for handing to a new user. Backed by sethvargo/go-password.
func GenerateTempPassword() (string, error) {
	return passwordGenerator.Generate(16, 4, 2, false, false)
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrInvalidHashFormat
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, ErrInvalidHashFormat
	}
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, ErrInvalidHashFormat
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrInvalidHashFormat
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ErrInvalidHashFormat
	}
	candidate := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(key)))
	return subtle.ConstantTimeCompare(key, candidate) == 1, nil
}
