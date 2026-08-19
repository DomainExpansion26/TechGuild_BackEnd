package utils

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	accessSecret  = loadSecret("JWT_ACCESS_SECRET")
	refreshSecret = loadSecret("JWT_REFRESH_SECRET")
)

func loadSecret(envKey string) []byte {
	v := os.Getenv(envKey)
	if v == "" {
		log.Fatalf("Environment variable %s is not set", envKey)
	}
	return []byte(v)
}

const AccessTokenTTL = 15 * time.Minute
const RefreshTokenTTL = 15 * 24 * time.Hour

func GenerateRefreshToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(RefreshTokenTTL).Unix(),
		"type":    "refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(refreshSecret))
}

func GenerateAccessToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(AccessTokenTTL).Unix(),
		"type":    "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(accessSecret))
}

// FIX: exported so middleware can reuse the SAME secret instead of hardcoding its own copy.
func GetAccessSecret() []byte {
	return accessSecret
}

func ValidateRefreshToken(tokenStr string) (jwt.MapClaims, error) {

	token, err := jwt.ParseWithClaims(tokenStr, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return refreshSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return nil, errors.New("not a refresh token")
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		return nil, errors.New("user id missing in token")
	}

	return claims, nil
}

// FIX (Bug 7 support): parse claims (already-valid token) just to read exp,
// used only to compute blacklist TTL — not a substitute for full validation.
func ParseAccessTokenUnverifiedExpiry(tokenStr string) (*jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(tokenStr, claims)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
