package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/gin-gonic/gin"
)

func ProjectApplicationRoutes(router *gin.Engine) {

	applicationGroup := router.Group("/v1/applications")
	applicationGroup.Use(middleware.AuthMiddleware())
	{
		// Freelancer / Agency
		applicationGroup.GET("/my", controllers.GetMyApplications)
		applicationGroup.DELETE("/:application_id", controllers.WithdrawApplication)

		// Client
		applicationGroup.POST("/:application_id/accept", controllers.AcceptApplication)
		applicationGroup.POST("/:application_id/reject", controllers.RejectApplication)
		applicationGroup.POST("/:application_id/shortlist", controllers.ShortlistApplication)
	}

	projectGroup := router.Group("/v1/projects")
	projectGroup.Use(middleware.AuthMiddleware())
	{
		projectGroup.POST("/:project_id/apply", controllers.ApplyProject)
		projectGroup.GET("/:project_id/applications", controllers.GetProjectApplications)
	}
}