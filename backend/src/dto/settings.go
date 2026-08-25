package dto

type UpdateAccountRequest struct {
	Email       string  `json:"email" example:"test@example.com"`
	Phone       *string `json:"phone" example:"+1234567890"`
	Password    string  `json:"password" example:"test@123"` // For verification if changing email/password
	NewPassword string  `json:"new_password" example:"newpass@123"`
}

type UpdateNotificationsRequest struct {
	Preferences map[string]any `json:"preferences" huma:"required"`
}

type UpdatePrivacyRequest struct {
	ProfileVisibility string `json:"profile_visibility" huma:"required,enum=public;private" example:"public"`
}
