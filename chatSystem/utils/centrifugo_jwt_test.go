package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateCentrifugoToken(t *testing.T) {
	secret := "test-secret-key-12345"
	userID := "user-uuid-123-abc"
	ttl := 15 * time.Minute

	tokenStr, exp, err := GenerateCentrifugoToken(userID, secret, ttl)
	if err != nil {
		t.Fatalf("unexpected error generating centrifugo token: %v", err)
	}

	if tokenStr == "" {
		t.Fatal("expected non-empty token string")
	}

	if exp <= time.Now().Unix() {
		t.Fatalf("expected expiration in the future, got: %d", exp)
	}

	// Verify token signature and claims
	parsedToken, err := jwt.ParseWithClaims(tokenStr, &CentrifugoClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	if !parsedToken.Valid {
		t.Fatal("expected parsed token to be valid")
	}

	claims, ok := parsedToken.Claims.(*CentrifugoClaims)
	if !ok {
		t.Fatal("expected claims to be *CentrifugoClaims")
	}

	if claims.Subject != userID {
		t.Errorf("expected subject %s, got %s", userID, claims.Subject)
	}
}

func TestGenerateCentrifugoToken_EmptySecret(t *testing.T) {
	_, _, err := GenerateCentrifugoToken("user-1", "", 10*time.Minute)
	if err == nil {
		t.Fatal("expected error when secret is empty")
	}
}

