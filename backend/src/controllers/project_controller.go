package controllers

import (
	"context"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
)

// ---------- CreateProject ----------

func CreateProjectHandler(ctx context.Context, input *dto.CreateProjectInput) (*dto.CreateProjectOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	projectService := services.NewProjectService()
	res, err := projectService.CreateProject(clientID, input.Body)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.CreateProjectOutput{Body: *res}, nil
}

// ---------- UpdateProject ----------

func UpdateProjectHandler(ctx context.Context, input *dto.UpdateProjectInput) (*dto.UpdateProjectOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	projectService := services.NewProjectService()
	res, err := projectService.UpdateProject(clientID, input.ID, input.Body)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.UpdateProjectOutput{Body: *res}, nil
}

// ---------- DeleteProject ----------

func DeleteProjectHandler(ctx context.Context, input *dto.DeleteProjectInput) (*dto.DeleteProjectOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	projectService := services.NewProjectService()
	if err := projectService.DeleteProject(clientID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.DeleteProjectOutput{
		Body: dto.DeleteProjectResponse{Message: "Project deleted successfully"},
	}, nil
}

// ---------- PublishProject ----------

func PublishProjectHandler(ctx context.Context, input *dto.PublishProjectInput) (*dto.PublishProjectOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	projectService := services.NewProjectService()
	if err := projectService.PublishProject(clientID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.PublishProjectOutput{
		Body: dto.PublishProjectResponse{Message: "Project published successfully"},
	}, nil
}

// ---------- CloseProject ----------

func CloseProjectHandler(ctx context.Context, input *dto.CloseProjectInput) (*dto.CloseProjectOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	projectService := services.NewProjectService()
	if err := projectService.CloseProject(clientID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.CloseProjectOutput{
		Body: dto.CloseProjectResponse{Message: "Project closed successfully"},
	}, nil
}

// ---------- ReopenProject ----------

func ReopenProjectHandler(ctx context.Context, input *dto.ReopenProjectInput) (*dto.ReopenProjectOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	projectService := services.NewProjectService()
	if err := projectService.ReopenProject(clientID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.ReopenProjectOutput{
		Body: dto.ReopenProjectResponse{Message: "Project reopened successfully"},
	}, nil
}

// ---------- GetProjectByID ----------

func GetProjectByIDHandler(ctx context.Context, input *dto.GetProjectByIDInput) (*dto.GetProjectByIDOutput, error) {
	projectService := services.NewProjectService()
	res, err := projectService.GetProjectByID(input.ID)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}

	return &dto.GetProjectByIDOutput{Body: *res}, nil
}

// ---------- GetMyProjects ----------

func GetMyProjectsHandler(ctx context.Context, input *dto.GetMyProjectsInput) (*dto.GetMyProjectsOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	projectService := services.NewProjectService()
	res, err := projectService.GetMyProjects(clientID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.GetMyProjectsOutput{Body: *res}, nil
}

// ---------- BrowseProjects ----------

func BrowseProjectsHandler(ctx context.Context, input *dto.BrowseProjectsInput) (*dto.BrowseProjectsOutput, error) {
	projectService := services.NewProjectService()
	res, err := projectService.BrowseProjects()
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.BrowseProjectsOutput{Body: *res}, nil
}

// ---------- SearchProjects ----------

func SearchProjectsHandler(ctx context.Context, input *dto.SearchProjectsInput) (*dto.SearchProjectsOutput, error) {
	req := dto.SearchProjectRequest{
		Keyword:         input.Keyword,
		Category:        input.Category,
		MinBudget:       input.MinBudget,
		MaxBudget:       input.MaxBudget,
		ExperienceLevel: input.ExperienceLevel,
		ProjectType:     input.ProjectType,
		Visibility:      input.Visibility,
		Page:            input.Page,
		Limit:           input.Limit,
	}

	projectService := services.NewProjectService()
	res, err := projectService.SearchProjects(req)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.SearchProjectsOutput{Body: *res}, nil
}
