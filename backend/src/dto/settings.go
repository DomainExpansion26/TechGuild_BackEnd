package dto

import "time"

type GetAccountSettingsInput struct{}

type GetAccountSettingsOutput struct {
	Body GetAccountSettingsResponse
}

type GetAccountSettingsResponse struct {
	Email         string `json:"email" example:"user@example.com"`
	EmailVerified bool   `json:"email_verified" example:"true"`
}

type GetPrivacySettingsInput struct{}

type GetPrivacySettingsOutput struct {
	Body GetPrivacySettingsResponse `json:"body" example:"{\"profile_visibility\":\"public\"}"`
}

type GetPrivacySettingsResponse struct {
	ProfileVisibility string `json:"profile_visibility" required:"true" enum:"public;private" example:"public"`
}

type GetBillingSettingsInput struct{}

type GetBillingSettingsOutput struct {
	Body GetBillingSettingsResponse `json:"body"`
}

type GetBillingSettingsResponse struct {
	PayoutMethod   string `json:"payout_method" required:"true" enum:"paypal;bank_transfer" example:"paypal"`
	PayoutSchedule string `json:"payout_schedule" required:"true" enum:"weekly;monthly" example:"weekly"`
}

type UpdatePayoutMethodInput struct {
	Body UpdatePayoutMethodRequest `json:"body"`
}

type UpdatePayoutMethodRequest struct {
	PayoutMethod string `json:"payout_method" required:"true" enum:"paypal;bank_transfer" example:"paypal"`
}

type UpdatePayoutMethodOutput struct {
	Body SettingsUpdateResponse `json:"body"`
}

type UpdatePayoutScheduleInput struct {
	Body UpdatePayoutScheduleRequest `json:"body"`
}

type UpdatePayoutScheduleRequest struct {
	PayoutSchedule string `json:"payout_schedule" required:"true" enum:"weekly;monthly" example:"weekly"`
}

type UpdatePayoutScheduleOutput struct {
	Body SettingsUpdateResponse `json:"body"`
}

type GetNotificationsInput struct{}

type GetNotificationsOutput struct {
	Body GetNotificationsResponse `json:"body"`
}

type GetNotificationsResponse struct {
	ProjectWork NotificationPreference `json:"project_work"`
	Messages    NotificationPreference `json:"messages"`
	Payments    NotificationPreference `json:"payments"`
	Reviews     NotificationPreference `json:"reviews"`
	Marketing   NotificationPreference `json:"marketing"`
}

type NotificationPreference struct {
	Email bool `json:"email"`
	Push  bool `json:"push"`
}

type DeactivateAccountInput struct {
	Body DeactivateAccountRequest
}

type DeactivateAccountRequest struct {
	Password     string `json:"password" required:"true" example:"test@123"`
	Confirmation bool   `json:"confirmation" required:"true" example:"true"`
}

type DeactivateAccountOutput struct {
	Body SettingsUpdateResponse
}

type DeleteAccountRequest struct {
	Password string `json:"password" required:"true" example:"test@123"`
}

type DeleteAccountInput struct {
	Body DeleteAccountRequest
}

type DeleteAccountOutput struct {
	Body SettingsUpdateResponse
}

type GetSessionsInput struct{}

type GetSessionsOutput struct {
	Body GetSessionsResponse `json:"body"`
}

type GetSessionsResponse struct {
	Sessions []SessionResponse `json:"sessions"`
}

type SessionResponse struct {
	ID           string    `json:"id"`
	Device       string    `json:"device"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	LastActiveAt time.Time `json:"last_active_at"`
	CreatedAt    time.Time `json:"created_at"`
	IsCurrent    bool      `json:"is_current"`
}

type RevokeSessionInput struct {
	ID string `path:"id" format:"uuid"`
}

type RevokeSessionOutput struct {
	Body SettingsUpdateResponse
}

type SignOtherSessionsInput struct{}

type SignOtherSessionsOutput struct {
	Body SettingsUpdateResponse
}

type UpdateAccountRequest struct {
	Password        string `json:"password" required:"true" example:"test@123"` // For verification if changing email/password
	NewPassword     string `json:"new_password" required:"true" minLength:"8" example:"newpass@123"`
	ConfirmPassword string `json:"confirm_password" required:"true" minLength:"8" eqfield:"new_password" example:"newpass@123"`
}

type UpdateNotificationsRequest struct {
	Preferences NotificationPreference `json:"preferences" required:"true"`
}

type UpdatePrivacyRequest struct {
	ProfileVisibility string `json:"profile_visibility" required:"true" enum:"public;private" example:"public"`
}

type SettingsUpdateResponse struct {
	Message string `json:"message" example:"settings updated successfully"`
}

type UpdateAccountSettingsInput struct {
	Body UpdateAccountRequest
}

type UpdateAccountSettingsOutput struct {
	Body SettingsUpdateResponse
}

type UpdateNotificationsInput struct {
	Body UpdateNotificationsRequest
}

type UpdateNotificationsOutput struct {
	Body SettingsUpdateResponse
}

type UpdatePrivacyInput struct {
	Body UpdatePrivacyRequest
}

type UpdatePrivacyOutput struct {
	Body SettingsUpdateResponse
}
