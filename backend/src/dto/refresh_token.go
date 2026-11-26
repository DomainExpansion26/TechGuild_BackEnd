package dto

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token,omitempty" minLength:"10" maxLength:"4096" pattern:"^[A-Za-z0-9_-]+\\.[A-Za-z0-9_-]+\\.[A-Za-z0-9_-]+$" doc:"Optional when the refresh_token cookie is sent." example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"`
}

type RefreshResponse struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"`
	// RefreshToken string `json:"refresh_token"`
	ExpiresIn int `json:"expires_in" example:"3600"`
}
