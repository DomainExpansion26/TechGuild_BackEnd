package routes

import (
	"github.com/gin-gonic/gin"

	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"
)

func VerificationRoutes(router *gin.Engine) {

	verification := router.Group("/v1/verification")
	verification.Use(middleware.AuthMiddleware())
	{
		// Tier 1 Individual Verification
		verification.POST("/identity/submit", controllers.SubmitIdentityVerification)
		verification.GET("/identity/status", controllers.GetIdentityVerificationStatus)

		// Tier 2 Business Verification
		verification.POST("/business/submit", controllers.SubmitBusinessVerification)

		// Generic Verification Status
		verification.GET("/status", controllers.GetVerificationStatus)

		// Resubmit Verification
		verification.POST("/resubmit/:record_id", controllers.ResubmitVerification)
	}

	admin := router.Group("/v1/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware())
	{
		// Admin Queue
		admin.GET("/verification/queue", controllers.GetVerificationQueue)

		// Admin Actions
		admin.POST("/verification/:id/approve", controllers.ApproveVerification)
		admin.POST("/verification/:id/reject", controllers.RejectVerification)
	}
}