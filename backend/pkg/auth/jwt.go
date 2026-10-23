package auth

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
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
	Sub *UserInfo `json:"sub"`
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

func NewJWT(userID string, userRole string) (string, error) {
	key, err := privateKey()
	if err != nil {
		return "", err
	}

	t := jwt.New(jwt.SigningMethodES256)

	t.Claims = &Claims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 1)),
		Sub: &UserInfo{
			UserID:   userID,
			UserRole: userRole,
		},
	}
	return t.SignedString(key)
}

// ParseJWT verifies the signature and expiry and returns the claims
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

// Uses crypto's random package to generate random salt
func generateSalt(saltSize uint32) []byte {
	salt := make([]byte, saltSize)
	_, _ = rand.Read(salt)
	return salt
}

// Uses argon2 to hash the password. If hashes ever get compromised, intensive resources must be used to decode as opposed to bcrypt.
func HashPassword(password string) string {
	argonConfig := &Argon2Configuration{
		TimeCost:   2,
		MemoryCost: 64 * 1024,
		Threads:    4,
		KeyLength:  32,
	}

	salt := generateSalt(16)
	hash := argon2.IDKey([]byte(password), salt, argonConfig.TimeCost, argonConfig.MemoryCost, argonConfig.Threads, argonConfig.KeyLength)
	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonConfig.MemoryCost,
		argonConfig.TimeCost,
		argonConfig.Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encodedHash
}

// Re-hashes password with the salt and settings stored in encodedHash and compares the results.
func VerifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("invalid argon2id hash")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errors.New("unsupported argon2 version")
	}

	var memoryCost, timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memoryCost, &timeCost, &threads); err != nil {
		return false, err
	}

	// decode salt
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	// decode hash
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	// constant-time compare, so response time doesn't reveal how much of the hash matched
	got := argon2.IDKey([]byte(password), salt, timeCost, memoryCost, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
