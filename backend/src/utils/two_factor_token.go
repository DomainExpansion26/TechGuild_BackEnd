package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var temp2FATokenSecret = []byte(getEnvOrPanic("TWO_FACTOR_TEMP_TOKEN_SECRET")) // separate secret from access/refresh

const Temporary2FATokenTTL = 5 * time.Minute

type temp2FAClaims struct {
	UserID string `json:"user_id"`
	Scope  string `json:"scope"` // always "2fa_pending"
	jwt.RegisteredClaims
}

func GenerateTemporary2FAToken(userID string) (string, error) {
	claims := temp2FAClaims{
		UserID: userID,
		Scope:  "2fa_pending",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(Temporary2FATokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(temp2FATokenSecret)
}

func ValidateTemporary2FAToken(tokenStr string) (uuid.UUID, error) {
	claims := &temp2FAClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return temp2FATokenSecret, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, errors.New("invalid or expired token")
	}
	if claims.Scope != "2fa_pending" {
		return uuid.Nil, errors.New("invalid token scope")
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, errors.New("invalid token payload")
	}
	return userID, nil
}
