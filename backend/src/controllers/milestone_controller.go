package controllers

import (
	"net/http"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	_ "techguild-backend/src/swagger"

	"github.com/gin-gonic/gin"
)

type MilestoneController struct {
	service *services.MilestoneService
}

func NewMilestoneController() *MilestoneController {
	return &MilestoneController{
		service: services.NewMilestoneService(),
	}
}

//create milestone 
func (c *MilestoneController) CreateMilestone(ctx *gin.Context) {

	clientID := ctx.GetString("userID")

	var req dto.CreateMilestoneRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	res, err := c.service.CreateMilestone(clientID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, res)
}

//update milestone 
func (c *MilestoneController) UpdateMilestone(ctx *gin.Context) {

	clientID := ctx.GetString("userID")
	milestoneID := ctx.Param("id")

	var req dto.UpdateMilestoneRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := c.service.UpdateMilestone(clientID, milestoneID, req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Milestone updated successfully",
	})
}

//delete milestone
func (c *MilestoneController) DeleteMilestone(ctx *gin.Context) {

	clientID := ctx.GetString("userID")
	milestoneID := ctx.Param("id")

	if err := c.service.DeleteMilestone(clientID, milestoneID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Milestone deleted successfully",
	})
}

//submit milestone
func (c *MilestoneController) SubmitMilestone(ctx *gin.Context) {

	freelancerID := ctx.GetString("userID")
	milestoneID := ctx.Param("id")

	if err := c.service.SubmitMilestone(freelancerID, milestoneID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Milestone submitted successfully",
	})
}

func (c *MilestoneController) ApproveMilestone(ctx *gin.Context) {

	clientID := ctx.GetString("userID")
	milestoneID := ctx.Param("id")

	if err := c.service.ApproveMilestone(clientID, milestoneID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Milestone approved successfully",
	})
}

func (c *MilestoneController) RejectMilestone(ctx *gin.Context) {

	clientID := ctx.GetString("userID")
	milestoneID := ctx.Param("id")

	if err := c.service.RejectMilestone(clientID, milestoneID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Milestone rejected successfully",
	})
}

//mark milestone paid
func (c *MilestoneController) MarkMilestonePaid(ctx *gin.Context) {

	clientID := ctx.GetString("userID")
	milestoneID := ctx.Param("id")

	if err := c.service.MarkMilestonePaid(clientID, milestoneID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Milestone marked as paid successfully",
	})
}

//get milestone by ID
func (c *MilestoneController) GetMilestoneByID(ctx *gin.Context) {

	milestoneID := ctx.Param("id")

	response, err := c.service.GetMilestoneByID(milestoneID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

//Get contract milestone 
func (c *MilestoneController) GetContractMilestones(ctx *gin.Context) {

	contractID := ctx.Param("contract_id")

	response, err := c.service.GetContractMilestones(contractID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}
