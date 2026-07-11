package routes

import (
	"github.com/gin-gonic/gin"

	"techguild-backend/src/controllers"
)

func AuthRoutes(router *gin.Engine) {

	auth := router.Group("/auth")
	{
		// Authentication
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)
		auth.POST("/logout", controllers.Logout)
		auth.POST("/refresh-token", controllers.RefreshToken)
		auth.GET("/verify-email", controllers.VerifyEmail)
		auth.POST("/resend-verification", controllers.ResendVerificationEmail)
		auth.POST("/forgot-password", controllers.ForgotPassword)
		auth.POST("/reset-password", controllers.ResetPassword)
		auth.POST("/change-password", controllers.ChangePassword)
	}
}