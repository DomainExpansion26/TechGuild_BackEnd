package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/gin-gonic/gin"
)

func ProfileRoutes(router *gin.Engine) {
	// Public routes (no auth required)
	router.GET("/v1/profile/:slug", controllers.GetPublicProfile)

	// Protected routes (auth required)
	profileGroup := router.Group("/v1/profile")
	profileGroup.Use(middleware.AuthMiddleware())
	{
		profileGroup.POST("/upload-resume", controllers.UploadResume)
		profileGroup.POST("/upload-avatar", controllers.UploadAvatar)
		profileGroup.DELETE("/avatar", controllers.DeleteAvatar)
		profileGroup.DELETE("/resume", controllers.DeleteResume)
		profileGroup.POST("/individual", controllers.CreateOrUpdateIndividualProfile)
		profileGroup.PUT("/individual", controllers.CreateOrUpdateIndividualProfile)
		profileGroup.POST("/agency", controllers.CreateOrUpdateAgencyProfile)
		profileGroup.POST("/client", controllers.CreateOrUpdateClientProfile)
		profileGroup.GET("/me", controllers.GetMyProfile)
		profileGroup.GET("/points", controllers.GetUserPoints)
		profileGroup.POST("/export", controllers.ExportProfile)
	}
}
