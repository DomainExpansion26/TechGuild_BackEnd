package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/gin-gonic/gin"
)

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
	}

	authProtected := router.Group("/auth")
	authProtected.Use(middleware.AuthMiddleware())
	{
		authProtected.POST("/register/account-type", controllers.SetAccountType)
		authProtected.POST("/change-password", controllers.ChangePassword)
		authProtected.DELETE("/account", controllers.DeleteAccount)
	}
}
