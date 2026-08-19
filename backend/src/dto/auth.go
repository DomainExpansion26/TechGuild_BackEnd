package dto

type RegisterRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
}

type RegisterResponse struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Message     string `json:"message"`
	AccessToken string `json:"access_token"`
	// RefreshToken string `json:"refresh_token"`
	ExpiresIn int `json:"expires_in"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

type VerifyEmailResponse struct {
	Message string `json:"message"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResendVerificationResponse struct {
	Message string `json:"message"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"` //if mobile app support need remove bidning
}

type LogoutResponse struct {
	Message string `json:"message"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	// RefreshToken string `json:"refresh_token"`
	ExpiresIn int `json:"expires_in"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ForgotPasswordResponse struct {
	Message string `json:"message"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type ResetPasswordResponse struct {
	Message string `json:"message"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type ChangePasswordResponse struct {
	Message string `json:"message"`
}
