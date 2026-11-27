package routes

import (
	"techguild-backend/src/controllers"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterOAuthRoutes(api huma.API, c *controllers.OAuthController) {

	huma.Register(api, huma.Operation{
		OperationID: "google-login",
		Method:      "GET",
		Path:        "/oauth/google/login",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Login with Google",
		Security:    []map[string][]string{},
	}, c.GoogleLoginHandler)

	huma.Register(api, huma.Operation{
		OperationID: "google-callback",
		Method:      "GET",
		Path:        "/oauth/google/callback",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Google OAuth callback",
		Security:    []map[string][]string{},
	}, c.GoogleCallbackHandler)

	huma.Register(api, huma.Operation{
		OperationID: "github-login",
		Method:      "GET",
		Path:        "/oauth/github/login",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Login with GitHub",
		Security:    []map[string][]string{},
	}, c.GitHubLoginHandler)

	huma.Register(api, huma.Operation{
		OperationID: "github-callback",
		Method:      "GET",
		Path:        "/oauth/github/callback",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] GitHub OAuth callback",
		Security:    []map[string][]string{},
	}, c.GitHubCallbackHandler)
}
