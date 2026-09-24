package identity

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"gestor-documental/internal/domain"
	"golang.org/x/crypto/argon2"
)

type PasswordHasher struct {
	memoryKiB  uint32
	iterations uint32
	slots      chan struct{}
	dummyHash  string
}

func NewPasswordHasher() (*PasswordHasher, error) { return newPasswordHasher(64*1024, 3) }

func newPasswordHasher(memoryKiB, iterations uint32) (*PasswordHasher, error) {
	hasher := &PasswordHasher{memoryKiB: memoryKiB, iterations: iterations, slots: make(chan struct{}, 2)}
	encoded, err := hasher.Hash(context.Background(), domain.RandomToken())
	if err != nil {
		return nil, err
	}
	hasher.dummyHash = encoded
	return hasher, nil
}

func ValidatePassword(password string) error {
	length := utf8.RuneCountInString(password)
	if !utf8.ValidString(password) || length < 6 || length > 128 {
		return domain.Failure("PASSWORD_POLICY", "La contraseña debe tener entre 6 y 128 caracteres.", 422)
	}
	return nil
}

func (hasher *PasswordHasher) acquire(ctx context.Context) error {
	select {
	case hasher.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (hasher *PasswordHasher) Hash(ctx context.Context, password string) (string, error) {
	if err := hasher.acquire(ctx); err != nil {
		return "", err
	}
	defer func() { <-hasher.slots }()
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	derived := argon2.IDKey([]byte(password), salt, hasher.iterations, hasher.memoryKiB, 1, 32)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=1$%s$%s", hasher.memoryKiB, hasher.iterations,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(derived)), nil
}

func (hasher *PasswordHasher) Verify(ctx context.Context, password, encoded string) (bool, error) {
	if encoded == "" {
		encoded = hasher.dummyHash
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false, fmt.Errorf("unsupported password hash")
	}
	parameters := strings.Split(parts[3], ",")
	if len(parameters) != 3 {
		return false, fmt.Errorf("invalid password parameters")
	}
	values := make([]uint64, 3)
	for index, prefix := range []string{"m=", "t=", "p="} {
		if !strings.HasPrefix(parameters[index], prefix) {
			return false, fmt.Errorf("invalid password parameters")
		}
		value, err := strconv.ParseUint(strings.TrimPrefix(parameters[index], prefix), 10, 32)
		if err != nil {
			return false, err
		}
		values[index] = value
	}
	if values[0] < 8192 || values[0] > 131072 || values[1] < 1 || values[1] > 6 || values[2] != 1 {
		return false, fmt.Errorf("password parameters out of bounds")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) != 16 {
		return false, fmt.Errorf("invalid salt")
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) != 32 {
		return false, fmt.Errorf("invalid digest")
	}
	if err := hasher.acquire(ctx); err != nil {
		return false, err
	}
	defer func() { <-hasher.slots }()
	actual := argon2.IDKey([]byte(password), salt, uint32(values[1]), uint32(values[0]), 1, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}
