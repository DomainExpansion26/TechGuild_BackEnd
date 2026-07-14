package dto

type UpdateAccountRequest struct {
	Email       string  `json:"email"`
	Phone       *string `json:"phone"`
	Password    string  `json:"password"` // For verification if changing email/password
	NewPassword string  `json:"new_password"`
}

type UpdateNotificationsRequest struct {
	Preferences map[string]interface{} `json:"preferences" binding:"required"`
}

type UpdatePrivacyRequest struct {
	ProfileVisibility string `json:"profile_visibility" binding:"required,oneof=public private"`
}
