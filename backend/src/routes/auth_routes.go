package routes

import (
	"github.com/gin-gonic/gin"
	"techguild-backend/src/controllers"
)

func AuthRoutes(router *gin.Engine) {
	router.POST("/auth/register", controllers.Register)
	router.POST("/auth/login", controllers.Login)
}