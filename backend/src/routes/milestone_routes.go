package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterMilestoneRoutes(router *gin.Engine) {

	milestoneController := controllers.NewMilestoneController()

	milestones := router.Group("/milestones")
	milestones.Use(middleware.AuthMiddleware())

	{
		// Milestone CRUD
		milestones.POST("", milestoneController.CreateMilestone)
		milestones.GET("/:id", milestoneController.GetMilestoneByID)
		milestones.PUT("/:id", milestoneController.UpdateMilestone)
		milestones.DELETE("/:id", milestoneController.DeleteMilestone)

		// Contract milestones
		milestones.GET("/contract/:contract_id", milestoneController.GetContractMilestones)

		// Freelancer actions
		milestones.POST("/:id/submit", milestoneController.SubmitMilestone)

		// Client actions
		milestones.POST("/:id/approve", milestoneController.ApproveMilestone)
		milestones.POST("/:id/reject", milestoneController.RejectMilestone)
		milestones.POST("/:id/pay", milestoneController.MarkMilestonePaid)
	}
}