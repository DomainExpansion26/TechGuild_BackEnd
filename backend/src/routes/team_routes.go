package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/gin-gonic/gin"
)

func TeamRoutes(router *gin.Engine) {

	teamController := controllers.NewTeamController()

	team := router.Group("/teams")
	team.Use(middleware.AuthMiddleware())
	{
		// Team
		team.POST("", teamController.CreateTeam)
		team.PUT("/:team_id", teamController.UpdateTeam)
		team.DELETE("/:team_id", teamController.DeleteTeam)

		team.GET("/:team_id", teamController.GetTeam)
		team.GET("/my", teamController.GetMyTeams)

		// Members
		team.POST("/:team_id/invite", teamController.InviteMember)
		team.POST("/invitation/:invitation_id/accept", teamController.AcceptInvitation)
		team.POST("/invitation/:invitation_id/reject", teamController.RejectInvitation)

		team.DELETE("/:team_id/member/:member_id", teamController.RemoveMember)
		team.POST("/:team_id/leave", teamController.LeaveTeam)

		// Portfolio
		team.POST("/:team_id/portfolio", teamController.CreatePortfolio)
		team.PUT("/portfolio/:portfolio_id", teamController.UpdatePortfolio)
		team.DELETE("/portfolio/:portfolio_id", teamController.DeletePortfolio)

		// Skills
		team.POST("/:team_id/skills", teamController.AddSkill)
		team.PUT("/skills/:skill_id", teamController.UpdateSkill)
		team.DELETE("/skills/:skill_id", teamController.DeleteSkill)
	}
}