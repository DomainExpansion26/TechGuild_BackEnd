package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

// ---------- Huma routes (naye, migrated handlers) ----------

func RegisterAuthRoutes(api huma.API, c *controllers.AuthController) {

	// Public routes
	huma.Register(api, huma.Operation{
		OperationID: "register",
		Method:      "POST",
		Path:        "/auth/register",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Register a new user",
		Security:    []map[string][]string{},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RegisterRequest{
						FirstName: "Aarav",
						LastName:  "Sharma",
						Email:     "aarav.sharma@example.in",
						Password:  "Bengaluru@2026",
					},
				},
			},
		},
	}, c.RegisterHandler)

	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      "POST",
		Path:        "/auth/login",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] User Login",
		Security:    []map[string][]string{},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.LoginRequest{
						Email:    "aarav.sharma@example.in",
						Password: "Bengaluru@2026",
					},
				},
			},
		},
	}, c.LoginHandler)

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      "POST",
		Path:        "/auth/logout",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Logout user",
		Security:    []map[string][]string{},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.LogoutRequest{
						RefreshToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
					},
				},
			},
		},
	}, c.LogoutHandler)

	huma.Register(api, huma.Operation{
		OperationID: "refresh-token",
		Method:      "POST",
		Path:        "/auth/refresh-token",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Refresh access token",
		Security:    []map[string][]string{},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RefreshRequest{
						RefreshToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
					},
				},
			},
		},
	}, c.RefreshTokenHandler)

	huma.Register(api, huma.Operation{
		OperationID: "verify-email",
		Method:      "GET",
		Path:        "/auth/verify-email",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Verify email",
		Security:    []map[string][]string{},
	}, c.VerifyEmailHandler)

	huma.Register(api, huma.Operation{
		OperationID: "resend-verification",
		Method:      "POST",
		Path:        "/auth/resend-verification",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Resend verification email",
		Security:    []map[string][]string{},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ResendVerificationRequest{
						Email: "aarav.sharma@example.in",
					},
				},
			},
		},
	}, c.ResendVerificationEmailHandler)

	huma.Register(api, huma.Operation{
		OperationID: "forgot-password",
		Method:      "POST",
		Path:        "/auth/forgot-password",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Forgot password",
		Security:    []map[string][]string{},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ForgotPasswordRequest{
						Email: "aarav.sharma@example.in",
					},
				},
			},
		},
	}, c.ForgotPasswordHandler)

	huma.Register(api, huma.Operation{
		OperationID: "reset-password",
		Method:      "POST",
		Path:        "/auth/reset-password",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Reset password",
		Description: "Completes a password reset. A valid reset token is ALSO required as the ?token=<reset token> query parameter (in addition to this JSON body).",
		Security:    []map[string][]string{},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ResetPasswordRequest{
						NewPassword: "Mumbai@2026x",
					},
				},
			},
		},
	}, c.ResetPasswordHandler)

	// Protected routes
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "change-password",
		Method:      "POST",
		Path:        "/auth/change-password",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Change password",
		Middlewares: huma.Middlewares{middleware.AuthMiddlewareHuma(api)},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ChangePasswordRequest{
						OldPassword:     "Bengaluru@2026",
						NewPassword:     "Pune@2026x",
						ConfirmPassword: "Pune@2026x",
					},
				},
			},
		},
	}, c.ChangePasswordHandler)

	// ---------- for oauth ----------
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "set-account-type-authenticated",
		Method:      "POST",
		Path:        "/auth/account-type",
		Tags:        []string{"Common / Authentication"},
		Summary:     "[Common] Set account type for an already-logged-in user (e.g. OAuth signup)",
		Middlewares: huma.Middlewares{middleware.AuthMiddlewareHuma(api)},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.SetAccountTypeAuthRequest{AccountType: "individual"},
				},
			},
		},
	}, c.SetAccountTypeAuthenticatedHandler)
}
