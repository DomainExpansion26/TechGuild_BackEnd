package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
)
//apply for a project
func ApplyProject(c *gin.Context) {

	applicantID := c.GetString("user_id")
	projectID := c.Param("project_id")

	var req dto.ApplyProjectRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	projectApplicationService := services.NewProjectApplicationService()

	res, err := projectApplicationService.ApplyProject(
		applicantID,
		projectID,
		req,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, res)
}

//withdraw application
func WithdrawApplication(c *gin.Context) {

	applicantID := c.GetString("user_id")
	applicationID := c.Param("application_id")

	projectApplicationService := services.NewProjectApplicationService()

	err := projectApplicationService.WithdrawApplication(
		applicantID,
		applicationID,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.WithdrawApplicationResponse{
		Message: "Application withdrawn successfully",
	})
}

//accept and reject application
func AcceptApplication(c *gin.Context) {

	applicationID := c.Param("application_id")

	projectApplicationService := services.NewProjectApplicationService()

	err := projectApplicationService.AcceptApplication(applicationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.AcceptApplicationResponse{
		Message: "Application accepted successfully",
	})
}
func RejectApplication(c *gin.Context) {

	applicationID := c.Param("application_id")

	projectApplicationService := services.NewProjectApplicationService()

	err := projectApplicationService.RejectApplication(applicationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.RejectApplicationResponse{
		Message: "Application rejected successfully",
	})
}

//shortlist application

func ShortlistApplication(c *gin.Context) {

	applicationID := c.Param("application_id")

	projectApplicationService := services.NewProjectApplicationService()

	err := projectApplicationService.ShortlistApplication(applicationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.ShortlistApplicationResponse{
		Message: "Application shortlisted successfully",
	})
}

//get my applications
func GetMyApplications(c *gin.Context) {

	applicantID := c.GetString("user_id")

	projectApplicationService := services.NewProjectApplicationService()

	response, err := projectApplicationService.GetMyApplications(applicantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

//get applications for a project

func GetProjectApplications(c *gin.Context) {

	projectID := c.Param("project_id")

	projectApplicationService := services.NewProjectApplicationService()

	response, err := projectApplicationService.GetProjectApplications(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}