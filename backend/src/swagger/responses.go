package swagger

type RegisterSuccessResponse struct {
	Message string `json:"message" example:"Registration successful. Please check your email to verify your account."`
}

type RegisterErrorResponse struct {
	Error string `json:"error" example:"Email already exists"`
}

type LoginSuccessResponse struct {
	Message      string `json:"message" example:"Login successful"`
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresIn    int    `json:"expires_in" example:"3600"`
}

type LoginBadRequestResponse struct {
	Error string `json:"error" example:"Invalid request body"`
}

type LoginUnauthorizedResponse struct {
	Error string `json:"error" example:"Invalid email or password"`
}


type UploadAvatarResponse struct {
	Message   string `json:"message" example:"avatar uploaded successfully"`
	AvatarURL string `json:"avatar_url" example:"https://res.cloudinary.com/demo/image/upload/v123456/avatar.jpg"`
}

type UploadLogoResponse struct {
	Message string `json:"message" example:"logo uploaded successfully"`
	LogoURL string `json:"logo_url" example:"https://res.cloudinary.com/demo/image/upload/v123456/logo.png"`
}

type DeleteAvatarResponse struct {
	Message string `json:"message" example:"avatar deleted successfully"`
}

type UpdateAccountSettingsResponse struct {
	Message string `json:"message" example:"account settings updated successfully"`
}

type UpdatePrivacySettingsResponse struct {
	Message string `json:"message" example:"privacy settings updated successfully"`
}
type UploadResumeResponse struct {
	Message   string `json:"message" example:"resume uploaded successfully"`
	ResumeURL string `json:"resume_url" example:"https://example.com/resume.pdf"`
}

type DeleteResumeResponse struct {
    Message string `json:"message" example:"resume deleted successfully"`
}
type DeleteLogoResponse struct {
    Message string `json:"message" example:"logo deleted successfully"`
}
type DeleteProfileAccountResponse struct {
	Message string `json:"message" example:"account successfully scheduled for deletion"`
}
type UpdateAccountResponse struct {
	Message string `json:"message" example:"account settings updated successfully"`
}

type UpdateNotificationsResponse struct {
	Message string `json:"message" example:"notifications updated successfully"`
}

type UpdatePrivacySettingsResponsestruct struct {
	Message string `json:"message" example:"privacy settings updated successfully"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"error message"`
}
