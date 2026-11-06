package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CentrifugoClaims defines standard JWT claims expected by Centrifugo
type CentrifugoClaims struct {
	jwt.RegisteredClaims
}

// GenerateCentrifugoToken generates a signed HMAC-SHA256 connection JWT for Centrifugo
func GenerateCentrifugoToken(userID string, hmacSecret string, ttl time.Duration) (string, int64, error) {
	if hmacSecret == "" {
		return "", 0, fmt.Errorf("centrifugo token_hmac_secret_key is empty")
	}

	now := time.Now().UTC()
	exp := now.Add(ttl)

	claims := CentrifugoClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(hmacSecret))
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign centrifugo token: %w", err)
	}

	return tokenString, exp.Unix(), nil
}

