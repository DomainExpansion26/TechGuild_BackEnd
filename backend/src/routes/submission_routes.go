package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterSubmissionRoutes(router *gin.Engine) {

	submissionController := controllers.NewSubmissionController()

	submissions := router.Group("/submissions")
	submissions.Use(middleware.AuthMiddleware())

	{
		// Submission CRUD
		submissions.POST("", submissionController.CreateSubmission)
		submissions.GET("/:id", submissionController.GetSubmissionByID)
		submissions.PUT("/:id", submissionController.UpdateSubmission)
		submissions.DELETE("/:id", submissionController.DeleteSubmission)

		// Get submissions of a milestone
		submissions.GET("/milestone/:milestone_id", submissionController.GetMilestoneSubmissions)

		// Client actions
		submissions.POST("/:id/approve", submissionController.ApproveSubmission)
		submissions.POST("/:id/reject", submissionController.RejectSubmission)
	}
}