package dto

type GoogleLoginRequest struct {
	GoogleID string `json:"google_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Picture  string `json:"picture"`
}

type GoogleLoginResponse struct {
	Message     string `json:"message"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}
type GitHubLoginRequest struct {
	GitHubID string `json:"github_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Avatar   string `json:"avatar"`
}

type GitHubLoginResponse struct {
	Message     string `json:"message"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}
