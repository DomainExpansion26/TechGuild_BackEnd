// dto/two_factor.go
package dto

// ---------- Request/Response bodies ----------

type Setup2FAResponse struct {
	ProvisioningURI string `json:"provisioning_uri" example:"otpauth://totp/TechGuild:user@example.com?secret=XXXX&issuer=TechGuild"`
	QRCodeImage     string `json:"qr_code_image"` //base64 data uri - frontend can render this directly in an <img> tag
	ExpiresAt       string `json:"expires_at" example:"2026-09-07T14:30:00Z"`
}

type VerifySetup2FARequest struct {
	Code string `json:"code" binding:"required" example:"123456"`
}

type VerifySetup2FAResponse struct {
	Message       string   `json:"message" example:"2FA enabled"`
	RecoveryCodes []string `json:"recovery_codes"`
}

type VerifyLogin2FARequest struct {
	TemporaryToken string `json:"temporary_token" binding:"required"`
	Code           string `json:"code" binding:"required" example:"123456"`
}

type VerifyRecoveryCodeRequest struct {
	TemporaryToken string `json:"temporary_token" binding:"required"`
	Code           string `json:"code" binding:"required" example:"ABCD-EFGH"`
}

type Disable2FARequest struct {
	Password string `json:"password,omitempty" example:"test@123"`
	Code     string `json:"code" binding:"required" example:"123456"`
}

type RegenerateRecoveryCodesRequest struct {
	Password string `json:"password,omitempty" example:"test@123"`
	Code     string `json:"code,omitempty" example:"123456"`
}

type RegenerateRecoveryCodesResponse struct {
	Message       string   `json:"message" example:"Recovery codes regenerated"`
	RecoveryCodes []string `json:"recovery_codes"`
}

type Message2FAResponse struct {
	Message string `json:"message"`
}

// ---------- Huma operation I/O ----------

type Setup2FAInput struct{}
type Setup2FAOutput struct {
	Body Setup2FAResponse
}

type VerifySetup2FAInput struct {
	Body VerifySetup2FARequest
}
type VerifySetup2FAOutput struct {
	Body VerifySetup2FAResponse
}

type VerifyLogin2FAInput struct {
	Body VerifyLogin2FARequest
}
type VerifyLogin2FAOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      LoginResponse
}

type VerifyRecoveryCodeInput struct {
	Body VerifyRecoveryCodeRequest
}
type VerifyRecoveryCodeOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      LoginResponse
}

type Disable2FAInput struct {
	Body Disable2FARequest
}
type Disable2FAOutput struct {
	Body Message2FAResponse
}

type RegenerateRecoveryCodesInput struct {
	Body RegenerateRecoveryCodesRequest
}
type RegenerateRecoveryCodesOutput struct {
	Body RegenerateRecoveryCodesResponse
}
