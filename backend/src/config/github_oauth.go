package config

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var GitHubOAuthConfig = &oauth2.Config{
	ClientID:     GitHubClientID,
	ClientSecret: GitHubClientSecret,
	RedirectURL:  GitHubRedirectURL,
	Scopes: []string{
		"user:email",
	},
	Endpoint: github.Endpoint,
}