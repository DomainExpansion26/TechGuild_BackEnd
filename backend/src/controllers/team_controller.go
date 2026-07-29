package controllers

import (
	"net/http"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"

	"github.com/gin-gonic/gin"
)

type TeamController struct {
	service *services.TeamService
}

func NewTeamController() *TeamController {
	return &TeamController{
		service: services.NewTeamService(),
	}
}

//to create the team 
//calles the service create team
func (c *TeamController) CreateTeam(ctx *gin.Context) {

	userID := ctx.GetString("user_id")

	var req dto.CreateTeamRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	response, err := c.service.CreateTeam(userID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

//update the team 

func (c *TeamController) UpdateTeam(ctx *gin.Context) {

	userID := ctx.GetString("user_id")
	teamID := ctx.Param("team_id")

	var req dto.UpdateTeamRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := c.service.UpdateTeam(userID, teamID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Team updated successfully",
	})
}

//delete the team
func (c *TeamController) DeleteTeam(ctx *gin.Context) {

	userID := ctx.GetString("user_id")
	teamID := ctx.Param("team_id")

	err := c.service.DeleteTeam(userID, teamID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Team deleted successfully",
	})
}

//get team 
func (c *TeamController) GetTeam(ctx *gin.Context) {

	teamID := ctx.Param("team_id")

	response, err := c.service.GetTeam(teamID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

//get my team

func (c *TeamController) GetMyTeams(ctx *gin.Context) {

	userID := ctx.GetString("user_id")

	response, err := c.service.GetMyTeams(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

//invite the member
func (c *TeamController) InviteMember(ctx *gin.Context) {

	userID := ctx.GetString("user_id")
	teamID := ctx.Param("team_id")

	var req dto.InviteMemberRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := c.service.InviteMember(userID, teamID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Invitation sent successfully",
	})
}

//accept and reject the invite
func (c *TeamController) AcceptInvitation(ctx *gin.Context) {

	userID := ctx.GetString("user_id")
	invitationID := ctx.Param("invitation_id")

	err := c.service.AcceptInvitation(userID, invitationID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Invitation accepted successfully",
	})
}

func (c *TeamController) RejectInvitation(ctx *gin.Context) {

	userID := ctx.GetString("user_id")
	invitationID := ctx.Param("invitation_id")

	err := c.service.RejectInvitation(userID, invitationID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Invitation rejected successfully",
	})
}

//remove the member
func (c *TeamController) RemoveMember(ctx *gin.Context) {

	userID := ctx.GetString("user_id")
	teamID := ctx.Param("team_id")
	memberID := ctx.Param("member_id")

	err := c.service.RemoveMember(userID, teamID, memberID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Member removed successfully",
	})
}

//leave the team 
func (c *TeamController) LeaveTeam(ctx *gin.Context) {

	userID := ctx.GetString("user_id")
	teamID := ctx.Param("team_id")

	err := c.service.LeaveTeam(userID, teamID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Left team successfully",
	})
}

//create portfolio
func (c *TeamController) CreatePortfolio(ctx *gin.Context) {

	userID := ctx.GetString("user_id")
	teamID := ctx.Param("team_id")

	var req dto.CreatePortfolioRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := c.service.CreatePortfolio(userID, teamID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Portfolio created successfully",
	})
}

//update  portfolio
func (c *TeamController) UpdatePortfolio(ctx *gin.Context) {

	userID := ctx.GetString("user_id")
	portfolioID := ctx.Param("portfolio_id")

	var req dto.UpdatePortfolioRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := c.service.UpdatePortfolio(userID, portfolioID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Portfolio updated successfully",
	})
}

//delete the potfolio
func (c *TeamController) DeletePortfolio(ctx *gin.Context) {

	userID := ctx.GetString("user_id")
	portfolioID := ctx.Param("portfolio_id")

	err := c.service.DeletePortfolio(userID, portfolioID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Portfolio deleted successfully",
	})
}

//add update and delete the skill 
func (c *TeamController) AddSkill(ctx *gin.Context) {

	userID := ctx.GetString("user_id")
	teamID := ctx.Param("team_id")

	var req dto.AddSkillRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := c.service.AddSkill(userID, teamID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Skill added successfully",
	})
}

func (c *TeamController) UpdateSkill(ctx *gin.Context) {

	userID := ctx.GetString("user_id")
	skillID := ctx.Param("skill_id")

	var req dto.AddSkillRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := c.service.UpdateSkill(userID, skillID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Skill updated successfully",
	})
}

func (c *TeamController) DeleteSkill(ctx *gin.Context) {

	userID := ctx.GetString("user_id")
	skillID := ctx.Param("skill_id")

	err := c.service.DeleteSkill(userID, skillID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Skill deleted successfully",
	})
}