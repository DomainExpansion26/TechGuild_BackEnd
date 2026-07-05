package routes

import (
	"github.com/gin-gonic/gin"

	"techguild-backend/src/controllers"
)

func AuthRoutes(router *gin.Engine) {

	auth := router.Group("/auth")
	{
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)

		auth.POST("/verify-email", controllers.VerifyEmail)
		auth.POST("/resend-otp", controllers.ResendOTP)
		auth.POST("/refresh", controllers.RefreshToken)
	}
}