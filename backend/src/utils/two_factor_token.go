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
			ID:        uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(Temporary2FATokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(temp2FATokenSecret)
}

func ValidateTemporary2FATokenWithJTI(tokenStr string) (uuid.UUID, string, time.Time, error) {
	claims := &temp2FAClaims{}
	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			return temp2FATokenSecret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || !token.Valid {
		return uuid.Nil, "", time.Time{}, errors.New("invalid or expired token")
	}
	if claims.Scope != "2fa_pending" {
		return uuid.Nil, "", time.Time{}, errors.New("invalid token scope")
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, "", time.Time{}, errors.New("invalid token payload")
	}
	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}
	return userID, claims.ID, expiresAt, nil
}

func ValidateTemporary2FAToken(tokenStr string) (uuid.UUID, error) {
	userID, _, _, err := ValidateTemporary2FATokenWithJTI(tokenStr)
	return userID, err
}
