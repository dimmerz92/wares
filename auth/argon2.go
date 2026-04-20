package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHash         = errors.New("invalid hash")
	ErrIncompatibleVersion = errors.New("incompatible version")
)

type Argon2 struct {
	memory     uint32
	time       uint32
	threads    uint8
	saltLength uint32
	keyLength  uint32
}

// Argon2Option provides a functional way to configure the argon2 password provider
type Argon2Option func(*Argon2)

// WithMemory sets the memory cost. The value 0 is ignored.
func WithMemory(memory uint32) Argon2Option {
	return func(a *Argon2) {
		if memory > 0 {
			a.memory = memory
		}
	}
}

// WithTime sets the time cost. The value 0 is ignored.
func WithTime(time uint32) Argon2Option {
	return func(a *Argon2) {
		if time > 0 {
			a.time = time
		}
	}
}

// WithThreads sets the number of threads to be used. The value 0 is ignored.
func WithThreads(threads uint8) Argon2Option {
	return func(a *Argon2) {
		if threads > 0 {
			a.threads = threads
		}
	}
}

// WithSaltLength sets the length of the raw salt to be generated. The value 0 is ignored.
func WithSaltLength(length uint32) Argon2Option {
	return func(a *Argon2) {
		if length > 0 {
			a.saltLength = length
		}
	}
}

// WithKeyLength sets the length of the raw key to be generated. The value 0 is ignored.
func WithKeyLength(length uint32) Argon2Option {
	return func(a *Argon2) {
		if length > 0 {
			a.keyLength = length
		}
	}
}

// NewArgon2 returns a new instance of the Argon2 password provider.
//
// Defaults:
//   - memory:      64MB
//   - time:        3
//   - threads:     2
//   - salt length: 16
//   - key length:  32
func NewArgon2(opts ...Argon2Option) *Argon2 {
	argon2 := &Argon2{
		memory:     64 << 10,
		time:       3,
		threads:    2,
		saltLength: 16,
		keyLength:  32,
	}

	for _, opt := range opts {
		opt(argon2)
	}

	return argon2
}

// GenerateFromPassword hashes the given password and returns the hash.
func (a *Argon2) GenerateFromPassword(password string) string {
	salt := GenerateNonce(uint(a.saltLength))
	hash := argon2.IDKey([]byte(password), salt, a.time, a.memory, a.threads, a.keyLength)

	b64Salt := base64.StdEncoding.Strict().EncodeToString(salt)
	b64Hash := base64.StdEncoding.Strict().EncodeToString(hash)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, a.memory, a.time, a.threads, b64Salt, b64Hash,
	)
}

// ComparePasswordAndHash compares the given password and hash to see if they match.
// Errors are only returned if the hash is malformed, or the argon2 version specified in the hash is incorrect.
func (a *Argon2) ComparePasswordAndHash(password, hash string) (bool, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("%w: %s", ErrInvalidHash, hash)
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrInvalidHash, err)
	}

	if version != argon2.Version {
		return false, fmt.Errorf("%w: argon2id version %d", ErrInvalidHash, version)
	}

	var threads uint8
	var memory, time uint32
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrInvalidHash, err)
	}

	decodedSalt, err := base64.StdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrInvalidHash, err)
	}

	decodedHash, err := base64.StdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrInvalidHash, err)
	}

	otherHash := argon2.IDKey([]byte(password), decodedSalt, time, memory, threads, uint32(len(decodedHash)))

	if subtle.ConstantTimeCompare(decodedHash, otherHash) == 1 {
		return true, nil
	}

	return false, nil
}
