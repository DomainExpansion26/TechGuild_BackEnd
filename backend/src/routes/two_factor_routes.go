package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

// ---------- Two-Factor Authentication routes ----------

func RegisterTwoFactorRoutes(api huma.API) {

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "2fa-setup",
		Method:      "POST",
		Path:        "/auth/2fa/setup",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Generate TOTP secret and QR provisioning URI",
		Description: "Takes no request body. Returns the TOTP provisioning URI and secret to enroll in an authenticator app.",
		Middlewares: huma.Middlewares{middleware.AuthMiddlewareHuma(api)},
	}, controllers.Setup2FAHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "2fa-verify-setup",
		Method:      "POST",
		Path:        "/auth/2fa/verify-setup",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Verify initial TOTP code and enable 2FA",
		Middlewares: huma.Middlewares{middleware.AuthMiddlewareHuma(api)},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.VerifySetup2FARequest{Code: "482915"},
				},
			},
		},
	}, controllers.VerifySetup2FAHandler)

	huma.Register(api, huma.Operation{
		OperationID: "2fa-verify-login",
		Method:      "POST",
		Path:        "/auth/2fa/verify-login",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Verify TOTP and issue final tokens",
		Security:    []map[string][]string{}, // uses temporary token, not bearer auth
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.VerifyLogin2FARequest{
						TemporaryToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI3ZjNhOWMyZS0xYjRkLTQ2OGYtOWMwMi0xZDNlNWY3YTliMDEiLCJ0b2tlbl90eXBlIjoiMmZhX3RtcCIsImlhdCI6MTc2NDIzMDQwMH0.9rT4wYq2Lm8Np0XsVb3cKd6EgHj1UiOaPqRsTuVwXyZ",
						Code:           "482915",
					},
				},
			},
		},
	}, controllers.VerifyLogin2FAHandler)

	huma.Register(api, huma.Operation{
		OperationID: "2fa-verify-recovery-code",
		Method:      "POST",
		Path:        "/auth/2fa/verify-recovery-code",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Authenticate using a single-use recovery code",
		Security:    []map[string][]string{},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.VerifyRecoveryCodeRequest{
						TemporaryToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI3ZjNhOWMyZS0xYjRkLTQ2OGYtOWMwMi0xZDNlNWY3YTliMDEiLCJ0b2tlbl90eXBlIjoiMmZhX3RtcCIsImlhdCI6MTc2NDIzMDQwMH0.9rT4wYq2Lm8Np0XsVb3cKd6EgHj1UiOaPqRsTuVwXyZ",
						Code:           "K7QM-3XPZ",
					},
				},
			},
		},
	}, controllers.VerifyRecoveryCodeHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "2fa-disable",
		Method:      "POST",
		Path:        "/auth/2fa/disable",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Disable 2FA after verifying password and TOTP",
		Middlewares: huma.Middlewares{middleware.AuthMiddlewareHuma(api)},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.Disable2FARequest{Password: "Pune@2026x", Code: "482915"},
				},
			},
		},
	}, controllers.Disable2FAHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "2fa-recovery-codes-regenerate",
		Method:      "POST",
		Path:        "/auth/2fa/recovery-codes/regenerate",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Invalidate old recovery codes and generate new ones",
		Description: "Password-based accounts send {\"password\":\"Pune@2026x\"}. OAuth-only accounts send {\"code\":\"482915\"} instead.",
		Middlewares: huma.Middlewares{middleware.AuthMiddlewareHuma(api)},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RegenerateRecoveryCodesRequest{Password: "Pune@2026x"},
				},
			},
		},
	}, controllers.RegenerateRecoveryCodesHandler)
}
