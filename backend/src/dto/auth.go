package dto

type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	FullName    string `json:"full_name" binding:"required"`
	AccountType string `json:"account_type" binding:"required,oneof=individual agency_admin client_admin"`
	Phone       string `json:"phone" binding:"required"`
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
	Message      string `json:"message"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}
type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=6"`
}

type VerifyEmailResponse struct {
	Message string `json:"message"`
}

type ResendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResendOTPResponse struct {
	Message string `json:"message"`
}