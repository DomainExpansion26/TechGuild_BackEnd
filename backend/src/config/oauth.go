package config

import (
	"os"

	"github.com/joho/godotenv"
)

func getEnv(key string) string {
	_ = godotenv.Load()
	return os.Getenv(key)
}

var (
	GoogleClientID     = getEnv("GOOGLE_CLIENT_ID")
	GoogleClientSecret = getEnv("GOOGLE_CLIENT_SECRET")
	GoogleRedirectURL  = getEnv("GOOGLE_REDIRECT_URL")
)

var (
	GitHubClientID     = getEnv("GITHUB_CLIENT_ID")
	GitHubClientSecret = getEnv("GITHUB_CLIENT_SECRET")
	GitHubRedirectURL  = getEnv("GITHUB_REDIRECT_URL")
)
