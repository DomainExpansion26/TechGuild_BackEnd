package routes

import (
	"context"
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
)

func RegisterAuthRoutes(api huma.API) {
	huma.Get(api, "/health-huma", func(ctx context.Context, input *struct{}) (*struct{ Body string }, error) {
		return &struct{ Body string }{Body: "ok"}, nil
	})
}

func AuthRoutes(router *gin.Engine) {

	auth := router.Group("/auth")
	{
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)
		auth.POST("/logout", controllers.Logout)
		auth.POST("/refresh-token", controllers.RefreshToken)
		auth.GET("/verify-email", controllers.VerifyEmail)
		auth.POST("/resend-verification", controllers.ResendVerificationEmail)
		auth.POST("/forgot-password", controllers.ForgotPassword)
		auth.POST("/reset-password", controllers.ResetPassword)
		auth.POST("/register/account-type", controllers.SetAccountType)
	}

	authProtected := router.Group("/auth")
	authProtected.Use(middleware.AuthMiddleware())
	{
		authProtected.POST("/change-password", controllers.ChangePassword)
		authProtected.DELETE("/account", controllers.DeleteAccount)
	}
}
