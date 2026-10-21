package auth

import (
	"crypto/ecdsa"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserInfo struct {
	UserID   string
	UserRole string
}

type Claims struct {
	jwt.RegisteredClaims
	Sub *UserInfo `json:"sub"`
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
