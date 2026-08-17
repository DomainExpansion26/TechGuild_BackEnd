package routes

import (
	"techguild-backend/src/controllers"

	"github.com/gin-gonic/gin"
)

func OAuthRoutes(router *gin.Engine) {

	auth := router.Group("/oauth")
	{
		auth.GET("/google/login", controllers.GoogleLogin)
		auth.GET("/google/callback", controllers.GoogleCallback)
		auth.GET("/github/login", controllers.GitHubLogin)
		auth.GET("/github/callback", controllers.GitHubCallback)
	}
}
