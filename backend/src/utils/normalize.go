package utils

import "strings"

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func NormalizeName(name string) string {
	return strings.TrimSpace(name)
}

func NormalizeOptionalPhone(phone *string) *string {
	if phone == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*phone)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
