package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func getEnv(key string) string {
	_ = godotenv.Load()
	return os.Getenv(key)
}

func getRedirectURL(key, providerDefaultPath string) string {
	val := strings.TrimSpace(getEnv(key))
	// Prevent cross-provider configuration mixups (e.g. GOOGLE_REDIRECT_URL pointing to github callback)
	if strings.Contains(providerDefaultPath, "google") && (val == "" || strings.Contains(val, "github")) {
		publicURL := strings.TrimRight(strings.TrimSpace(os.Getenv("SERVER_PUBLIC_URL")), "/")
		if publicURL == "" {
			publicURL = "https://api-preprod.techguild.in"
		}
		return publicURL + providerDefaultPath
	}
	if strings.Contains(providerDefaultPath, "github") && (val == "" || strings.Contains(val, "google")) {
		publicURL := strings.TrimRight(strings.TrimSpace(os.Getenv("SERVER_PUBLIC_URL")), "/")
		if publicURL == "" {
			publicURL = "https://api-preprod.techguild.in"
		}
		return publicURL + providerDefaultPath
	}
	return val
}

var (
	GoogleClientID     = getEnv("GOOGLE_CLIENT_ID")
	GoogleClientSecret = getEnv("GOOGLE_CLIENT_SECRET")
	GoogleRedirectURL  = getRedirectURL("GOOGLE_REDIRECT_URL", "/oauth/google/callback")
)

var (
	GitHubClientID     = getEnv("GITHUB_CLIENT_ID")
	GitHubClientSecret = getEnv("GITHUB_CLIENT_SECRET")
	GitHubRedirectURL  = getRedirectURL("GITHUB_REDIRECT_URL", "/oauth/github/callback")
)