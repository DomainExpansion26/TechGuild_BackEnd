package routes

import (
	"techguild-backend/src/controllers"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterOAuthRoutes(api huma.API) {

	huma.Register(api, huma.Operation{
		OperationID: "google-login",
		Method:      "GET",
		Path:        "/oauth/google/login",
		Tags:        []string{"OAuth"},
		Summary:     "Login with Google",
		Security:    []map[string][]string{},
	}, controllers.GoogleLoginHandler)

	huma.Register(api, huma.Operation{
		OperationID: "google-callback",
		Method:      "GET",
		Path:        "/oauth/google/callback",
		Tags:        []string{"OAuth"},
		Summary:     "Google OAuth callback",
		Security:    []map[string][]string{},
	}, controllers.GoogleCallbackHandler)

	huma.Register(api, huma.Operation{
		OperationID: "github-login",
		Method:      "GET",
		Path:        "/oauth/github/login",
		Tags:        []string{"OAuth"},
		Summary:     "Login with GitHub",
		Security:    []map[string][]string{},
	}, controllers.GitHubLoginHandler)

	huma.Register(api, huma.Operation{
		OperationID: "github-callback",
		Method:      "GET",
		Path:        "/oauth/github/callback",
		Tags:        []string{"OAuth"},
		Summary:     "GitHub OAuth callback",
		Security:    []map[string][]string{},
	}, controllers.GitHubCallbackHandler)
}
