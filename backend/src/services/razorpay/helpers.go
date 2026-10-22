package razorpay

import (
	"strings"

	"github.com/google/uuid"
)

// shortID generates a 12-char compact ID.
func shortID() string {
	id := uuid.New().String()
	clean := strings.ReplaceAll(id, "-", "")
	if len(clean) > 12 {
		clean = clean[:12]
	}
	return clean
}

// toInt64 safely converts interface{} to int64.
func toInt64(v interface{}) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	default:
		return 0
	}
}

// toStr safely converts interface{} to string.
func toStr(v interface{}) string {
	s, _ := v.(string)
	return s
}

// maskAccount returns XXXX + last 4 digits.
func maskAccount(account string) string {
	if len(account) <= 4 {
		return "XXXX"
	}
	return "XXXX" + account[len(account)-4:]
}

// maskVPA partially hides a UPI VPA. Example: "john@okhdfc" -> "jo**@okhdfc"
func maskVPA(vpa string) string {
	parts := strings.SplitN(vpa, "@", 2)
	if len(parts) != 2 {
		return vpa
	}
	local := parts[0]
	if len(local) <= 2 {
		return vpa
	}
	return local[:2] + strings.Repeat("*", len(local)-2) + "@" + parts[1]
}
