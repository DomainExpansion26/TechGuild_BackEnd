package config

import (
	"os"
)

var (
	GoogleClientID     = os.Getenv("GOOGLE_CLIENT_ID")
	GoogleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	GoogleRedirectURL  = os.Getenv("GOOGLE_REDIRECT_URL")
)
var (
	GitHubClientID     = os.Getenv("GITHUB_CLIENT_ID")
	GitHubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
	GitHubRedirectURL  = os.Getenv("GITHUB_REDIRECT_URL")
)