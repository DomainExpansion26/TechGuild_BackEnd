package utils

import "strings"

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func NormalizeName(name string) string {
	return strings.TrimSpace(name)
}
