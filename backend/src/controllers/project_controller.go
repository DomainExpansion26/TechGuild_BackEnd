package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
)

// create a new project
func CreateProject(c *gin.Context) {

	clientID := c.GetString("user_id")

	var req dto.CreateProjectRequest

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	projectService := services.NewProjectService()

	res, err := projectService.CreateProject(clientID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, res)
}

// update a project
func UpdateProject(c *gin.Context) {

	clientID := c.GetString("user_id")
	projectID := c.Param("project_id")

	var req dto.UpdateProjectRequest

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	projectService := services.NewProjectService()

		res, err := projectService.UpdateProject(clientID, projectID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, res)
}

// delete a project
func DeleteProject(c *gin.Context) {

	clientID := c.GetString("user_id")
	projectID := c.Param("project_id")

	projectService := services.NewProjectService()

	err := projectService.DeleteProject(clientID, projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.DeleteProjectResponse{
		Message: "Project deleted successfully",
	})
}

// publish project
func PublishProject(c *gin.Context) {

	clientID := c.GetString("user_id")
	projectID := c.Param("project_id")

	projectService := services.NewProjectService()

	err := projectService.PublishProject(clientID, projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.PublishProjectResponse{
		Message: "Project published successfully",
	})
}

// close project
func CloseProject(c *gin.Context) {

	clientID := c.GetString("user_id")
	projectID := c.Param("project_id")

	projectService := services.NewProjectService()

	err := projectService.CloseProject(clientID, projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.CloseProjectResponse{
		Message: "Project closed successfully",
	})
}

// reopen project
func ReopenProject(c *gin.Context) {

	clientID := c.GetString("user_id")
	projectID := c.Param("project_id")

	projectService := services.NewProjectService()

	err := projectService.ReopenProject(clientID, projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.ReopenProjectResponse{
		Message: "Project reopened successfully",
	})
}

// get project by id
func GetProjectByID(c *gin.Context) {

	projectID := c.Param("project_id")

	projectService := services.NewProjectService()

	res, err := projectService.GetProjectByID(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, res)
}

// get my projects
func GetMyProjects(c *gin.Context) {

	clientID := c.GetString("user_id")

	projectService := services.NewProjectService()

	res, err := projectService.GetMyProjects(clientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, res)
}

// browse published projects
func BrowseProjects(c *gin.Context) {

	projectService := services.NewProjectService()

	res, err := projectService.BrowseProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, res)
}

// search projects
func SearchProjects(c *gin.Context) {

	var req dto.SearchProjectRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	projectService := services.NewProjectService()

	res, err := projectService.SearchProjects(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, res)
}
