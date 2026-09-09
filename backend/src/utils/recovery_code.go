// utils/recovery_code.go
package utils

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// GenerateRecoveryCodes returns plain codes (shown once) - hash separately for storage
func GenerateRecoveryCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		b := make([]byte, 5)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		codes[i] = fmt.Sprintf("%X-%X", b[:2], b[2:5]) // e.g. AB12-CD34E
	}
	return codes, nil
}

func HashRecoveryCode(code string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	return string(hash), err
}

func CompareRecoveryCode(hash, code string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)) == nil
}
