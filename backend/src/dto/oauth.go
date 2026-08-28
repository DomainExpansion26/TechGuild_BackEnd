package dto

import "context"

type GoogleLoginRequest struct {
	GoogleID string `json:"google_id" example:"110234567890123456789"`
	Email    string `json:"email" example:"test@example.com"`
	FullName string `json:"full_name" example:"John Doe"`
	Picture  string `json:"picture" example:"https://lh3.googleusercontent.com/a-/default-photo"`
}

type GoogleLoginResponse struct {
	Message     string `json:"message" example:"Login successful"`
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"`
	ExpiresIn   int    `json:"expires_in" example:"3600"`
}

type GitHubLoginRequest struct {
	GitHubID string `json:"github_id" example:"12345678"`
	Email    string `json:"email" example:"test@example.com"`
	FullName string `json:"full_name" example:"John Doe"`
	Avatar   string `json:"avatar" example:"https://avatars.githubusercontent.com/u/12345678"`
}

type GitHubLoginResponse struct {
	Message     string `json:"message" example:"Login successful"`
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"`
	ExpiresIn   int    `json:"expires_in" example:"3600"`
}

// ---------- Huma Input/Output wrapper structs ----------

type GoogleLoginInput struct{}
type GoogleLoginOutput struct {
	Status    int    `json:"-"`
	Location  string `header:"Location"`
	SetCookie string `header:"Set-Cookie"`
}

type GoogleCallbackInput struct {
	Code             string `query:"code"`
	State            string `query:"state"`
	OauthStateCookie string `cookie:"oauth_state"`
}
type GoogleCallbackOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      GoogleLoginResponse
}

type GitHubLoginInput struct{}
type GitHubLoginOutput struct {
	Status    int    `json:"-"`
	Location  string `header:"Location"`
	SetCookie string `header:"Set-Cookie"`
}

type GitHubCallbackInput struct {
	Code             string `query:"code"`
	State            string `query:"state"`
	OauthStateCookie string `cookie:"oauth_state"`
}
type GitHubCallbackOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      GitHubLoginResponse
}

var _ = context.Background // placeholder to avoid unused import if needed elsewhere
