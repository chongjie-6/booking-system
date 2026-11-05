package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

type UserInfo struct {
	UserID   string
	UserRole string
}

type Claims struct {
	jwt.RegisteredClaims
	User *UserInfo `json:"user"`
}

type Argon2Configuration struct {
	TimeCost   uint32
	MemoryCost uint32
	Threads    uint8
	KeyLength  uint32
}

// Parsed on first use, after main has loaded .env
var privateKey = sync.OnceValues(func() (*ecdsa.PrivateKey, error) {
	return jwt.ParseECPrivateKeyFromPEM([]byte(os.Getenv("JWT_PRIVATE_KEY")))
})

var ErrPasswordMismatch = errors.New("password does not match")

const (
	AccessTTL  = 15 * time.Minute
	RefreshTTL = 7 * 24 * time.Hour
	DeviceTTL  = 365 * 24 * time.Hour
)

// NewAccessToken signs an ES256 JWT carrying the user's ID and role that expires after 15 minutes.
// Takes the user ID and role; returns the signed token, or an error if the private key can't be loaded.
func NewAccessToken(userID string, userRole string) (string, error) {
	key, err := privateKey()
	if err != nil {
		return "", err
	}

	t := jwt.New(jwt.SigningMethodES256)
	t.Claims = &Claims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTTL)),
		User: &UserInfo{
			UserID:   userID,
			UserRole: userRole,
		},
	}
	s, err := t.SignedString(key)
	return s, err
}

// NewRefreshToken makes a random opaque token; it isn't a JWT, so it can't be passed off as an access token.
// Takes nothing; returns the token and its expiry a week from now.
func NewRefreshToken() (string, time.Time) {
	return rand.Text(), time.Now().Add(RefreshTTL)
}

// MintTokens creates a 15-minute access token and a week-long refresh token.
// Takes the user ID and role; returns (access, refresh, refresh expiry, err).
func MintTokens(userID string, userRole string) (string, string, time.Time, error) {
	refresh, refreshExp := NewRefreshToken()

	access, accessErr := NewAccessToken(userID, userRole)
	if accessErr != nil {
		return "", "", time.Time{}, accessErr
	}
	return access, refresh, refreshExp, nil
}

// ParseJWT verifies an access token's ES256 signature and expiry.
// Takes the raw token; returns its Claims, or an error if it's invalid or expired.
func ParseJWT(token string) (*Claims, error) {
	key, err := privateKey()
	if err != nil {
		return nil, err
	}

	claims := &Claims{}
	_, err = jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return &key.PublicKey, nil
	}, jwt.WithValidMethods([]string{"ES256"}), jwt.WithExpirationRequired())
	return claims, err
}

// generateSalt makes a salt from crypto/rand. Takes the size in bytes; returns that many random bytes.
func generateSalt(saltSize uint32) []byte {
	salt := make([]byte, saltSize)
	_, _ = rand.Read(salt)
	return salt
}

// argonSlots caps how many argon2 hashes run at once. Each holds 64 MiB and up to 4 threads, so a burst of logins
// queues here instead of running the server out of memory. One slot per CPU (GOMAXPROCS) keeps every core busy.
var argonSlots = make(chan struct{}, runtime.GOMAXPROCS(0))

// argonHash runs argon2.IDKey once a slot in argonSlots is free. Takes the request context and the argon2 inputs; returns the key,
// or ctx.Err() if the context ends first (e.g. the client hung up), so abandoned requests leave the queue without hashing.
func argonHash(ctx context.Context, password string, salt []byte, timeCost, memoryCost uint32, threads uint8, keyLen uint32) ([]byte, error) {
	select {
	case argonSlots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-argonSlots }()
	return argon2.IDKey([]byte(password), salt, timeCost, memoryCost, threads, keyLen), nil
}

// HashPassword hashes a password with argon2id and a random salt; it's memory-hard, so leaked hashes are costly to crack.
// Takes the request context and plain password; returns the encoded "$argon2id$v=..$m=..,t=..,p=..$salt$hash" string,
// or ctx.Err() if the context ends while waiting for an argon2 slot.
func HashPassword(ctx context.Context, password string) (string, error) {
	argonConfig := &Argon2Configuration{
		TimeCost:   2,
		MemoryCost: 64 * 1024,
		Threads:    4,
		KeyLength:  32,
	}

	salt := generateSalt(16)
	hash, err := argonHash(ctx, password, salt, argonConfig.TimeCost, argonConfig.MemoryCost, argonConfig.Threads, argonConfig.KeyLength)
	if err != nil {
		return "", err
	}
	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonConfig.MemoryCost,
		argonConfig.TimeCost,
		argonConfig.Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encodedHash, nil
}

// VerifyPassword re-hashes the password with the salt and settings stored in encodedHash and compares in constant time.
// Takes the request context, plain password and encoded hash; returns nil on a match, ErrPasswordMismatch if it differs,
// ctx.Err() if the context ends while waiting for an argon2 slot, or an error if the hash is malformed.
func VerifyPassword(ctx context.Context, password, encodedHash string) error {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return errors.New("invalid argon2id hash")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return errors.New("unsupported argon2 version")
	}

	var memoryCost, timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memoryCost, &timeCost, &threads); err != nil {
		return err
	}

	// decode salt
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return err
	}

	// decode hash
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return err
	}

	// constant-time compare, so response time doesn't reveal how much of the hash matched
	got, err := argonHash(ctx, password, salt, timeCost, memoryCost, threads, uint32(len(want)))
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return ErrPasswordMismatch
	}
	return nil
}

// HashRefreshToken hashes a refresh token with SHA-256; its high entropy makes a slow hash like argon2 unnecessary.
// Takes the raw token; returns the hex-encoded hash.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
