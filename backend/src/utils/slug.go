package utils

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var slugInvalidChars = regexp.MustCompile(`[^a-z0-9]+`)

// GenerateSlug ek unique, URL-safe slug banata hai (jaise "fujel-a1b2c3d4")
func GenerateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = slugInvalidChars.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")

	if slug == "" {
		slug = "user"
	}

	// Random suffix taaki collision practically impossible ho
	suffix := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	return slug + "-" + suffix
}
