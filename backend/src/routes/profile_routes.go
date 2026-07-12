package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/gin-gonic/gin"
)

func ProfileRoutes(router *gin.Engine) {
	// apply the auth middleware and then send to controllers
	profileGroup := router.Group("/v1/profile")
	profileGroup.Use(middleware.AuthMiddleware())
	{
		profileGroup.POST("", controllers.CreateProfile)
	}
}
