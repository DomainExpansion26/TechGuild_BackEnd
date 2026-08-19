package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
)

// ---------- Huma routes (naye, migrated handlers) ----------

func RegisterAuthRoutes(api huma.API) {

	// Public routes
	huma.Register(api, huma.Operation{
		OperationID: "register",
		Method:      "POST",
		Path:        "/auth/register",
		Tags:        []string{"Authentication"},
		Summary:     "Register a new user",
	}, controllers.RegisterHandler)

	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      "POST",
		Path:        "/auth/login",
		Tags:        []string{"Authentication"},
		Summary:     "User Login",
	}, controllers.LoginHandler)

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      "POST",
		Path:        "/auth/logout",
		Tags:        []string{"Authentication"},
		Summary:     "Logout user",
	}, controllers.LogoutHandler)

	huma.Register(api, huma.Operation{
		OperationID: "refresh-token",
		Method:      "POST",
		Path:        "/auth/refresh-token",
		Tags:        []string{"Authentication"},
		Summary:     "Refresh access token",
	}, controllers.RefreshTokenHandler)

	huma.Register(api, huma.Operation{
		OperationID: "verify-email",
		Method:      "GET",
		Path:        "/auth/verify-email",
		Tags:        []string{"Authentication"},
		Summary:     "Verify email",
	}, controllers.VerifyEmailHandler)

	huma.Register(api, huma.Operation{
		OperationID: "resend-verification",
		Method:      "POST",
		Path:        "/auth/resend-verification",
		Tags:        []string{"Authentication"},
		Summary:     "Resend verification email",
	}, controllers.ResendVerificationEmailHandler)

	huma.Register(api, huma.Operation{
		OperationID: "forgot-password",
		Method:      "POST",
		Path:        "/auth/forgot-password",
		Tags:        []string{"Authentication"},
		Summary:     "Forgot password",
	}, controllers.ForgotPasswordHandler)

	huma.Register(api, huma.Operation{
		OperationID: "reset-password",
		Method:      "POST",
		Path:        "/auth/reset-password",
		Tags:        []string{"Authentication"},
		Summary:     "Reset password",
	}, controllers.ResetPasswordHandler)

	// Protected routes
	huma.Register(api, huma.Operation{
		OperationID: "change-password",
		Method:      "POST",
		Path:        "/auth/change-password",
		Tags:        []string{"Authentication"},
		Summary:     "Change password",
		Middlewares: huma.Middlewares{middleware.AuthMiddlewareHuma(api)},
	}, controllers.ChangePasswordHandler)

	huma.Register(api, huma.Operation{
		OperationID: "delete-account",
		Method:      "DELETE",
		Path:        "/auth/account",
		Tags:        []string{"Authentication"},
		Summary:     "Delete account",
		Middlewares: huma.Middlewares{middleware.AuthMiddlewareHuma(api)},
	}, controllers.DeleteAccountHandler)
}

// ---------- Old Gin routes (sirf SetAccountType — abhi migrate nahi hua) ----------

func AuthRoutes(router *gin.Engine) {
	auth := router.Group("/auth")
	{
		auth.POST("/register/account-type", controllers.SetAccountType)
	}
}
