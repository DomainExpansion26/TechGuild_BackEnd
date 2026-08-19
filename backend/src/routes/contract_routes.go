package routes

import (
	"github.com/gin-gonic/gin"

	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"
)

func ContractRoutes(router *gin.Engine) {

	contractController := controllers.NewContractController()

	contracts := router.Group("/contracts")
	contracts.Use(middleware.AuthMiddleware())

	{
		contracts.POST("/", contractController.CreateContract)

		contracts.PUT("/:id/sign", contractController.SignContract)

		contracts.PUT("/:id/complete", contractController.CompleteContract)

		contracts.PUT("/:id/cancel", contractController.CancelContract)

		contracts.GET("/:id", contractController.GetContractByID)

		contracts.GET("/client", contractController.GetClientContracts)

		contracts.GET("/freelancer", contractController.GetFreelancerContracts)
	}
}
