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
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "2fa-setup",
		Method:      "POST",
		Path:        "/auth/2fa/setup",
		Tags:        []string{"Two-Factor Authentication"},
		Summary:     "Generate TOTP secret and QR provisioning URI",
		Middlewares: huma.Middlewares{middleware.AuthMiddlewareHuma(api)},
	}, controllers.Setup2FAHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "2fa-verify-setup",
		Method:      "POST",
		Path:        "/auth/2fa/verify-setup",
		Tags:        []string{"Two-Factor Authentication"},
		Summary:     "Verify initial TOTP code and enable 2FA",
		Middlewares: huma.Middlewares{middleware.AuthMiddlewareHuma(api)},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.VerifySetup2FARequest{Code: "123456"},
				},
			},
		},
	}, controllers.VerifySetup2FAHandler)

	huma.Register(api, huma.Operation{
		OperationID: "2fa-verify-login",
		Method:      "POST",
		Path:        "/auth/2fa/verify-login",
		Tags:        []string{"Two-Factor Authentication"},
		Summary:     "Verify TOTP and issue final tokens",
		Security:    []map[string][]string{}, // uses temporary token, not bearer auth
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.VerifyLogin2FARequest{
						TemporaryToken: "eyJhbGciOiJIUzI1NiIs...",
						Code:           "123456",
					},
				},
			},
		},
	}, controllers.VerifyLogin2FAHandler)

	huma.Register(api, huma.Operation{
		OperationID: "2fa-verify-recovery-code",
		Method:      "POST",
		Path:        "/auth/2fa/verify-recovery-code",
		Tags:        []string{"Two-Factor Authentication"},
		Summary:     "Authenticate using a single-use recovery code",
		Security:    []map[string][]string{},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.VerifyRecoveryCodeRequest{
						TemporaryToken: "eyJhbGciOiJIUzI1NiIs...",
						Code:           "ABCD-EFGH",
					},
				},
			},
		},
	}, controllers.VerifyRecoveryCodeHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "2fa-disable",
		Method:      "POST",
		Path:        "/auth/2fa/disable",
		Tags:        []string{"Two-Factor Authentication"},
		Summary:     "Disable 2FA after verifying password and TOTP",
		Middlewares: huma.Middlewares{middleware.AuthMiddlewareHuma(api)},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.Disable2FARequest{Password: "test@123", Code: "123456"},
				},
			},
		},
	}, controllers.Disable2FAHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "2fa-recovery-codes-regenerate",
		Method:      "POST",
		Path:        "/auth/2fa/recovery-codes/regenerate",
		Tags:        []string{"Two-Factor Authentication"},
		Summary:     "Invalidate old recovery codes and generate new ones",
		Middlewares: huma.Middlewares{middleware.AuthMiddlewareHuma(api)},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RegenerateRecoveryCodesRequest{Password: "test@123"},
				},
			},
		},
	}, controllers.RegenerateRecoveryCodesHandler)
}
