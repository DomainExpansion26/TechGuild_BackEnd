package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/gin-gonic/gin"
)

func ProfileRoutes(router *gin.Engine) {
	// Public routes (no auth required)
	router.GET("/v1/profile/check-slug", controllers.CheckSlug)
	router.GET("/v1/profile/:slug", controllers.GetPublicProfile)

	// Protected routes (auth required)
	profileGroup := router.Group("/v1/profile")
	profileGroup.Use(middleware.AuthMiddleware())
	{
		profileGroup.POST("/upload-resume", controllers.UploadResume)
		profileGroup.POST("/avatar", controllers.UploadAvatar) // renamed from /upload-avatar
		profileGroup.POST("/logo", controllers.UploadLogo)
		profileGroup.DELETE("/avatar", controllers.DeleteAvatar)
		profileGroup.DELETE("/logo", controllers.DeleteLogo)
		profileGroup.DELETE("/resume", controllers.DeleteResume)

		profileGroup.POST("", controllers.CreateOrUpdateProfile)
		profileGroup.PATCH("", controllers.CreateOrUpdateProfile)
		profileGroup.GET("", controllers.GetMyProfile)
		profileGroup.DELETE("", controllers.DeleteProfileAccount)

		profileGroup.GET("/points", controllers.GetUserPoints)
		profileGroup.POST("/export", controllers.ExportProfile)
	}

	settingsGroup := router.Group("/v1/settings")
	settingsGroup.Use(middleware.AuthMiddleware())
	{
		settingsGroup.PATCH("/account", controllers.UpdateAccountSettings)
		settingsGroup.PATCH("/notifications", controllers.UpdateNotifications)
		settingsGroup.PATCH("/privacy", controllers.UpdatePrivacySettings)
	}
}
