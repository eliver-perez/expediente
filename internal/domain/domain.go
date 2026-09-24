// Package domain contains the small value types shared by H2 modules.
package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

const TimeLayout = "2006-01-02T15:04:05.000000Z"

func Timestamp(instant time.Time) string { return instant.UTC().Format(TimeLayout) }

func NewID() string {
	var identifier [16]byte
	if _, err := rand.Read(identifier[:]); err != nil {
		panic("system random source unavailable")
	}
	identifier[6] = (identifier[6] & 0x0f) | 0x40
	identifier[8] = (identifier[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", identifier[:4], identifier[4:6], identifier[6:8], identifier[8:10], identifier[10:])
}

func RandomToken() string {
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		panic("system random source unavailable")
	}
	return base64.RawURLEncoding.EncodeToString(token[:])
}

func Digest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

type Error struct {
	Code    string
	Message string
	Status  int
}

func (err *Error) Error() string                     { return err.Code }
func Failure(code, message string, status int) error { return &Error{code, message, status} }

type RequestMetadata struct {
	RequestID  string
	ObservedIP string
	UserAgent  string
}

type User struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Disabled    bool     `json:"disabled"`
	Revision    int64    `json:"revision"`
	CreatedAt   string   `json:"created_at"`
	RoleIDs     []string `json:"role_ids"`
	Permissions []string `json:"permissions"`
}

type Principal struct {
	User              User
	SessionID         string
	TokenDigest       string
	CredentialVersion int64
	CSRFTokenDigest   string
	ExpiresAt         string
}

func (principal Principal) Can(permission string) bool {
	for _, granted := range principal.User.Permissions {
		if granted == permission {
			return true
		}
	}
	return false
}
