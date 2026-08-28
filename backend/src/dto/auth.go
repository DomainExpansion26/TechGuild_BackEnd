package dto

type RegisterRequest struct {
	FirstName string `json:"first_name" binding:"required" example:"John"`
	LastName  string `json:"last_name" binding:"required" example:"Doe"`
	Email     string `json:"email" binding:"required,email" example:"test@example.com"`
	Password  string `json:"password" binding:"required,min=8" example:"test@123"`
}

type RegisterResponse struct {
	Message string `json:"message" example:"Registration successful"`
	UserID  string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"test@example.com"`
	Password string `json:"password" binding:"required" example:"test@123"`
}

type LoginResponse struct {
	Message     string `json:"message" example:"Login successful"`
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"`
	// RefreshToken string `json:"refresh_token"`
	ExpiresIn int `json:"expires_in" example:"3600"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required" example:"abc123def456ghi789"`
}

type VerifyEmailResponse struct {
	Message string `json:"message" example:"Email verified successfully"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email" example:"test@example.com"`
}

type ResendVerificationResponse struct {
	Message string `json:"message" example:"Verification email sent"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"` //if mobile app support need remove bidning
}

type LogoutResponse struct {
	Message string `json:"message" example:"Logged out successfully"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email" example:"test@example.com"`
}

type ForgotPasswordResponse struct {
	Message string `json:"message" example:"Password reset email sent"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" example:"abc123def456ghi789"`
	NewPassword string `json:"new_password" binding:"required,min=8" example:"newpass@123"`
}

type ResetPasswordResponse struct {
	Message string `json:"message" example:"Password reset successfully"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required" example:"test@123"`
	NewPassword string `json:"new_password" binding:"required,min=8" example:"newpass@123"`
}

type ChangePasswordResponse struct {
	Message string `json:"message" example:"Password changed successfully"`
}

// ---------- Huma operation I/O (auth) ----------

type RegisterInput struct {
	Body RegisterRequest
}
type RegisterOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

type LoginInput struct {
	Body LoginRequest
}
type LoginOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      LoginResponse
}

type VerifyEmailInput struct {
	Token string `query:"token" required:"true" doc:"Email verification token"`
}
type VerifyEmailOutput struct {
	Body VerifyEmailResponse
}

type ResendVerificationInput struct {
	Body ResendVerificationRequest
}
type ResendVerificationOutput struct {
	Body ResendVerificationResponse
}

type LogoutInput struct {
	RefreshTokenCookie string `cookie:"refresh_token"`
	Authorization      string `header:"Authorization"`
	Body               LogoutRequest
}
type LogoutOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      LogoutResponse
}

type RefreshTokenInput struct {
	RefreshTokenCookie string `cookie:"refresh_token"`
	Body               RefreshRequest
}
type RefreshTokenOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      RefreshResponse
}

type ResetPasswordInput struct {
	Token string `query:"token" required:"true" doc:"Password reset token"`
	Body  ResetPasswordRequest
}
type ResetPasswordOutput struct {
	Body ResetPasswordResponse
}

type ForgotPasswordInput struct {
	Body ForgotPasswordRequest
}
type ForgotPasswordOutput struct {
	Body ForgotPasswordResponse
}

type ChangePasswordInput struct {
	Body ChangePasswordRequest
}
type ChangePasswordOutput struct {
	Body ChangePasswordResponse
}

type DeleteAccountInput struct{}
type DeleteAccountOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}
