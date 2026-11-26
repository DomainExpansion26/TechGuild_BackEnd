package utils

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestVerifyRecoveryCode(t *testing.T) {
	svc := NewTOTPService("TechGuild", make([]byte, 32))

	plainCodes, hashes, err := svc.GenerateRecoveryCodes(1)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes failed: %v", err)
	}
	newCode := plainCodes[0]
	newHash := hashes[0]

	// Legacy format: mixed-case base64url with dash hashed directly
	legacyCode := "aB3d-Ef9h"
	legacyHashBytes, err := bcrypt.GenerateFromPassword([]byte(legacyCode), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt failed: %v", err)
	}
	legacyHash := string(legacyHashBytes)

	tests := []struct {
		name string
		code string
		hash string
		want bool
	}{
		{"new exact", newCode, newHash, true},
		{"new lowercase", strings.ToLower(newCode), newHash, true},
		{"new dashless", strings.ReplaceAll(newCode, "-", ""), newHash, true},
		{"new lowercase dashless with spaces", " " + strings.ToLower(strings.ReplaceAll(newCode, "-", " ")) + " ", newHash, true},
		{"new wrong code", "ZZZZ-ZZZZ", newHash, false},
		{"legacy dashed mixed-case", legacyCode, legacyHash, true},
		{"legacy trimmed", "  " + legacyCode + " ", legacyHash, true},
		{"legacy wrong case fails", strings.ToUpper(legacyCode), legacyHash, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := svc.VerifyRecoveryCode(tc.code, tc.hash); got != tc.want {
				t.Errorf("VerifyRecoveryCode(%q) = %v, want %v", tc.code, got, tc.want)
			}
		})
	}
}
