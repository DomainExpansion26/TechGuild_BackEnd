package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	verificationSecret     []byte
	verificationSecretOnce sync.Once
)

func getVerificationSecret() []byte {
	verificationSecretOnce.Do(func() {
		v := os.Getenv("JWT_SECRET")
		if v == "" {
			log.Fatalf("missing requried env var : JWT_SECRET")
		}
		verificationSecret = []byte(v)
	})
	return verificationSecret
}

func GenerateVerificationToken(userID string) (string, error) {

	bytes := make([]byte, 32)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
func ValidateVerificationToken(tokenStr string) (jwt.MapClaims, error) {

	token, err := jwt.ParseWithClaims(tokenStr, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return getVerificationSecret(), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	if claims["type"] != "email_verification" {
		return nil, errors.New("invalid verification token")
	}

	return claims, nil
}
func GenerateResetPasswordToken(userID string) (string, error) {

	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    "password_reset",
		"exp":     time.Now().Add(30 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(getVerificationSecret())
}
func ValidateResetPasswordToken(tokenStr string) (jwt.MapClaims, error) {

	token, err := jwt.ParseWithClaims(tokenStr, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Unexpected signing method")
		}
		return getVerificationSecret(), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	if claims["type"] != "password_reset" {
		return nil, errors.New("invalid reset token")
	}

	return claims, nil
}
