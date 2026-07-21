package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/gin-gonic/gin"
)

func ProjectRoutes(router *gin.Engine) {

	// Public Routes

	project := router.Group("/v1/projects")
	{
		project.GET("", controllers.BrowseProjects)
		project.GET("/search", controllers.SearchProjects)
		project.GET("/:project_id", controllers.GetProjectByID)
	}

	// Protected Routes


	projectProtected := router.Group("/v1/projects")
	projectProtected.Use(middleware.AuthMiddleware())
	{
		projectProtected.POST("", controllers.CreateProject)
		projectProtected.PATCH("/:project_id", controllers.UpdateProject)
		projectProtected.DELETE("/:project_id", controllers.DeleteProject)

		projectProtected.POST("/:project_id/publish", controllers.PublishProject)
		projectProtected.POST("/:project_id/close", controllers.CloseProject)
		projectProtected.POST("/:project_id/reopen", controllers.ReopenProject)

		projectProtected.GET("/my", controllers.GetMyProjects)
	}
}