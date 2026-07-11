package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"github.com/golang-jwt/jwt/v5"
	"time"
)
var verificationSecret = []byte(os.Getenv("JWT_SECRET"))
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
		return []byte(verificationSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims := token.Claims.(jwt.MapClaims)

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

	return token.SignedString([]byte(verificationSecret))
}
func ValidateResetPasswordToken(tokenStr string) (jwt.MapClaims, error) {

	token, err := jwt.ParseWithClaims(tokenStr, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(verificationSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims := token.Claims.(jwt.MapClaims)

	if claims["type"] != "password_reset" {
		return nil, errors.New("invalid reset token")
	}

	return claims, nil
}