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
		Tags:        []string{"Authentication"},
		Summary:     "Register a new user",
		Security:    []map[string][]string{},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RegisterRequest{
						FirstName: "John",
						LastName:  "Doe",
						Email:     "test@example.com",
						Password:  "test@123",
					},
				},
			},
		},
	}, c.RegisterHandler)

	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      "POST",
		Path:        "/auth/login",
		Tags:        []string{"Authentication"},
		Summary:     "User Login",
		Security:    []map[string][]string{},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.LoginRequest{
						Email:    "test@example.com",
						Password: "test@123",
					},
				},
			},
		},
	}, c.LoginHandler)

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      "POST",
		Path:        "/auth/logout",
		Tags:        []string{"Authentication"},
		Summary:     "Logout user",
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
		Tags:        []string{"Authentication"},
		Summary:     "Refresh access token",
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
		Tags:        []string{"Authentication"},
		Summary:     "Verify email",
		Security:    []map[string][]string{},
	}, c.VerifyEmailHandler)

	huma.Register(api, huma.Operation{
		OperationID: "resend-verification",
		Method:      "POST",
		Path:        "/auth/resend-verification",
		Tags:        []string{"Authentication"},
		Summary:     "Resend verification email",
		Security:    []map[string][]string{},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ResendVerificationRequest{
						Email: "test@example.com",
					},
				},
			},
		},
	}, c.ResendVerificationEmailHandler)

	huma.Register(api, huma.Operation{
		OperationID: "forgot-password",
		Method:      "POST",
		Path:        "/auth/forgot-password",
		Tags:        []string{"Authentication"},
		Summary:     "Forgot password",
		Security:    []map[string][]string{},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ForgotPasswordRequest{
						Email: "test@example.com",
					},
				},
			},
		},
	}, c.ForgotPasswordHandler)

	huma.Register(api, huma.Operation{
		OperationID: "reset-password",
		Method:      "POST",
		Path:        "/auth/reset-password",
		Tags:        []string{"Authentication"},
		Summary:     "Reset password",
		Security:    []map[string][]string{},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ResetPasswordRequest{
						NewPassword: "newpass@123",
					},
				},
			},
		},
	}, c.ResetPasswordHandler)

	// Protected routes
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "change-password",
		Method:      "POST",
		Path:        "/auth/change-password",
		Tags:        []string{"Authentication"},
		Summary:     "Change password",
		Middlewares: huma.Middlewares{middleware.AuthMiddlewareHuma(api)},
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ChangePasswordRequest{
						OldPassword:     "test@123",
						NewPassword:     "newpass@123",
						ConfirmPassword: "newpass@123",
					},
				},
			},
		},
	}, c.ChangePasswordHandler)

	// huma.Register(api, huma.Operation{
	// 	OperationID: "set-account-type",
	// 	Method:      "POST",
	// 	Path:        "/auth/register/account-type",
	// 	Tags:        []string{"Authentication"},
	// 	Summary:     "Set account type",
	// 	Security:    []map[string][]string{},
	// 	RequestBody: &huma.RequestBody{
	// 		Content: map[string]*huma.MediaType{
	// 			"application/json": {
	// 				Example: dto.SetAccountTypeRequest{
	// 					Email:       "test@example.com",
	// 					Password:    "test@123",
	// 					AccountType: "individual",
	// 				},
	// 			},
	// 		},
	// 	},
	// }, pc.SetAccountTypeHandler)

	// ---------- for oauth ----------
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "set-account-type-authenticated",
		Method:      "POST",
		Path:        "/auth/account-type",
		Tags:        []string{"Authentication"},
		Summary:     "Set account type for an already-logged-in user (e.g. OAuth signup)",
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
