package domain

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	passwordMinLength = 8
	passwordMaxLength = 1000
	argon2Memory      = 64 * 1024
	argon2Iterations  = 3
	argon2Parallelism = 4
	argon2SaltLength  = 16
	argon2KeyLength   = 32
)

var (
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidPassword = errors.New("invalid password")
)

func NormalizeEmail(value string) (string, error) {
	email := strings.TrimSpace(strings.ToLower(value))
	if email == "" || len(email) > 255 {
		return "", ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", ErrInvalidEmail
	}
	return email, nil
}

func ValidatePassword(value string) error {
	if len(value) < passwordMinLength {
		return fmt.Errorf("password must be at least %d characters", passwordMinLength)
	}
	if len(value) > passwordMaxLength {
		return fmt.Errorf("password must be at most %d characters", passwordMaxLength)
	}
	if !hasUppercase(value) {
		return errors.New("password must include an uppercase letter")
	}
	if !hasNumber(value) {
		return errors.New("password must include a number")
	}
	if !hasSpecial(value) {
		return errors.New("password must include a special character")
	}
	if isCommonPassword(value) {
		return errors.New("password is too common")
	}
	return nil
}

func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argon2Iterations, argon2Memory, argon2Parallelism, argon2KeyLength)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", argon2Memory, argon2Iterations, argon2Parallelism, b64Salt, b64Hash)
	return encoded, nil
}

func VerifyPassword(password, encoded string) (bool, error) {
	params, salt, hash, err := decodeArgon2idHash(encoded)
	if err != nil {
		return false, err
	}
	computed := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallelism, uint32(len(hash)))
	if subtle.ConstantTimeCompare(computed, hash) == 1 {
		return true, nil
	}
	return false, nil
}

func FakePasswordHash(password string) {
	salt := make([]byte, argon2SaltLength)
	_, _ = rand.Read(salt)
	_ = argon2.IDKey([]byte(password), salt, argon2Iterations, argon2Memory, argon2Parallelism, argon2KeyLength)
}

type argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func decodeArgon2idHash(encoded string) (argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argon2Params{}, nil, nil, ErrInvalidPassword
	}

	versionParts := strings.Split(parts[2], "=")
	if len(versionParts) != 2 || versionParts[0] != "v" || versionParts[1] != "19" {
		return argon2Params{}, nil, nil, ErrInvalidPassword
	}

	paramsPart := strings.Split(parts[3], ",")
	if len(paramsPart) != 3 {
		return argon2Params{}, nil, nil, ErrInvalidPassword
	}

	memory, err := parseArgon2Param(paramsPart[0], "m")
	if err != nil {
		return argon2Params{}, nil, nil, ErrInvalidPassword
	}
	iterations, err := parseArgon2Param(paramsPart[1], "t")
	if err != nil {
		return argon2Params{}, nil, nil, ErrInvalidPassword
	}
	parallelism, err := parseArgon2Param(paramsPart[2], "p")
	if err != nil {
		return argon2Params{}, nil, nil, ErrInvalidPassword
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return argon2Params{}, nil, nil, ErrInvalidPassword
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(hash) == 0 {
		return argon2Params{}, nil, nil, ErrInvalidPassword
	}

	return argon2Params{
		memory:      uint32(memory),
		iterations:  uint32(iterations),
		parallelism: uint8(parallelism),
	}, salt, hash, nil
}

func parseArgon2Param(input, label string) (int, error) {
	parts := strings.Split(input, "=")
	if len(parts) != 2 || parts[0] != label {
		return 0, ErrInvalidPassword
	}
	value, err := strconv.Atoi(parts[1])
	if err != nil || value <= 0 {
		return 0, ErrInvalidPassword
	}
	return value, nil
}

func isCommonPassword(value string) bool {
	candidate := strings.ToLower(value)
	_, ok := commonPasswords[candidate]
	return ok
}

func hasUppercase(value string) bool {
	for _, r := range value {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func hasNumber(value string) bool {
	for _, r := range value {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func hasSpecial(value string) bool {
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return true
	}
	return false
}

var commonPasswords = map[string]struct{}{
	"password1!":   {},
	"password1@":   {},
	"password1#":   {},
	"password1$":   {},
	"password12!":  {},
	"password123!": {},
	"welcome1!":    {},
	"welcome123!":  {},
	"welcome2024!": {},
	"welcome2025!": {},
	"qwerty123!":   {},
	"qwerty123@":   {},
	"qwerty123#":   {},
	"qwerty123$":   {},
	"qwerty12!":    {},
	"admin123!":    {},
	"admin123@":    {},
	"admin123#":    {},
	"admin123$":    {},
	"letmein1!":    {},
	"letmein123!":  {},
	"letmein123@":  {},
	"iloveyou1!":   {},
	"iloveyou123!": {},
	"monk3y123!":   {},
	"dragon123!":   {},
	"princess1!":   {},
	"sunshine1!":   {},
	"football1!":   {},
	"baseball1!":   {},
	"starwars1!":   {},
	"trustno1!":    {},
	"shadow123!":   {},
	"master123!":   {},
	"login123!":    {},
	"passw0rd1!":   {},
	"passw0rd1@":   {},
	"passw0rd1#":   {},
	"c0mputer1!":   {},
	"c0mputer123!": {},
	"n1nja123!":    {},
	"n1nja2024!":   {},
	"s0ccer123!":   {},
	"hockey123!":   {},
	"p@ssw0rd1":    {},
	"p@ssword1":    {},
	"p@ssword1!":   {},
	"p@ssword123!": {},
	"ch@ngeme1!":   {},
	"default1!":    {},
	"temppass1!":   {},
	"temppass2@":   {},
	"test1234!":    {},
	"test12345!":   {},
	"welcome12!":   {},
	"welcome1234!": {},
	"qwerty12@":    {},
	"qwerty1234!":  {},
	"admin2024!":   {},
	"admin2025!":   {},
	"user1234!":    {},
	"user12345!":   {},
	"user2024!":    {},
	"user2025!":    {},
}
