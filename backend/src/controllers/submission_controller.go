package controllers

import (
	"net/http"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	_ "techguild-backend/src/swagger"

	"github.com/gin-gonic/gin"
)

type SubmissionController struct {
	service *services.SubmissionService
}

func NewSubmissionController() *SubmissionController {
	return &SubmissionController{
		service: services.NewSubmissionService(),
	}
}

//create the sumission 
func (c *SubmissionController) CreateSubmission(ctx *gin.Context) {

	freelancerID := ctx.GetString("userID")

	var req dto.CreateSubmissionRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	response, err := c.service.CreateSubmission(freelancerID, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

//update the submission
func (c *SubmissionController) UpdateSubmission(ctx *gin.Context) {

	freelancerID := ctx.GetString("userID")
	submissionID := ctx.Param("id")

	var req dto.UpdateSubmissionRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := c.service.UpdateSubmission(freelancerID, submissionID, req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Submission updated successfully",
	})
}

//delete submission
func (c *SubmissionController) DeleteSubmission(ctx *gin.Context) {

	freelancerID := ctx.GetString("userID")
	submissionID := ctx.Param("id")

	if err := c.service.DeleteSubmission(freelancerID, submissionID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Submission deleted successfully",
	})
}

//approve submission
func (c *SubmissionController) ApproveSubmission(ctx *gin.Context) {

	clientID := ctx.GetString("userID")
	submissionID := ctx.Param("id")

	if err := c.service.ApproveSubmission(clientID, submissionID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Submission approved successfully",
	})
}
func (c *SubmissionController) RejectSubmission(ctx *gin.Context) {

	clientID := ctx.GetString("userID")
	submissionID := ctx.Param("id")

	if err := c.service.RejectSubmission(clientID, submissionID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Submission rejected successfully",
	})
}

//get submission by ID
func (c *SubmissionController) GetSubmissionByID(ctx *gin.Context) {

	submissionID := ctx.Param("id")

	response, err := c.service.GetSubmissionByID(submissionID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

//get milestone Submission
func (c *SubmissionController) GetMilestoneSubmissions(ctx *gin.Context) {

	milestoneID := ctx.Param("milestone_id")

	response, err := c.service.GetMilestoneSubmissions(milestoneID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, response)
}