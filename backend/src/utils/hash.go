package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
)

// HashGovtID generates a SHA-256 hash for a government ID using a secret salt.
func HashGovtID(govtID string) string {

	salt := os.Getenv("HASH_SALT")

	hash := sha256.Sum256([]byte(govtID + salt))

	return hex.EncodeToString(hash[:])
}

// HashBusinessPAN generates a SHA-256 hash for a business PAN using a secret salt.
func HashBusinessPAN(pan string) string {

	salt := os.Getenv("HASH_SALT")

	hash := sha256.Sum256([]byte(pan + salt))

	return hex.EncodeToString(hash[:])
}