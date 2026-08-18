package config

import (
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var GoogleOAuthConfig = &oauth2.Config{
	ClientID:     GoogleClientID,
	ClientSecret: GoogleClientSecret,
	RedirectURL:  GoogleRedirectURL,
	Scopes: []string{
		"openid",
		"profile",
		"email",
	},
	Endpoint: google.Endpoint,
}
