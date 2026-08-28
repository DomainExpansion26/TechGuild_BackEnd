package controllers

import (
	"context"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
)

// ---------- Team CRUD ----------

func CreateTeamHandler(ctx context.Context, input *dto.CreateTeamInput) (*dto.CreateTeamOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	res, err := teamService.CreateTeam(userID, input.Body)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.CreateTeamOutput{Body: *res}, nil
}

func UpdateTeamHandler(ctx context.Context, input *dto.UpdateTeamInput) (*dto.UpdateTeamOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	if err := teamService.UpdateTeam(userID, input.ID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.UpdateTeamOutput{Body: dto.MessageResponse{Message: "Team updated successfully"}}, nil
}

func DeleteTeamHandler(ctx context.Context, input *dto.DeleteTeamInput) (*dto.DeleteTeamOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	if err := teamService.DeleteTeam(userID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.DeleteTeamOutput{Body: dto.MessageResponse{Message: "Team deleted successfully"}}, nil
}

func GetTeamHandler(ctx context.Context, input *dto.GetTeamInput) (*dto.GetTeamOutput, error) {
	teamService := services.NewTeamService()
	res, err := teamService.GetTeam(input.ID)
	if err != nil {
		return nil, huma.Error404NotFound(err.Error())
	}

	return &dto.GetTeamOutput{Body: *res}, nil
}

func GetMyTeamsHandler(ctx context.Context, input *dto.GetMyTeamsInput) (*dto.GetMyTeamsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	res, err := teamService.GetMyTeams(userID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.GetMyTeamsOutput{Body: *res}, nil
}

// ---------- Members ----------

func InviteMemberHandler(ctx context.Context, input *dto.InviteMemberInput) (*dto.InviteMemberOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	if err := teamService.InviteMember(userID, input.TeamID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.InviteMemberOutput{Body: dto.MessageResponse{Message: "Invitation sent successfully"}}, nil
}

func AcceptInvitationHandler(ctx context.Context, input *dto.AcceptInvitationInput) (*dto.AcceptInvitationOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	if err := teamService.AcceptInvitation(userID, input.InvitationID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.AcceptInvitationOutput{Body: dto.MessageResponse{Message: "Invitation accepted successfully"}}, nil
}

func RejectInvitationHandler(ctx context.Context, input *dto.RejectInvitationInput) (*dto.RejectInvitationOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	if err := teamService.RejectInvitation(userID, input.InvitationID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.RejectInvitationOutput{Body: dto.MessageResponse{Message: "Invitation rejected successfully"}}, nil
}

func RemoveMemberHandler(ctx context.Context, input *dto.RemoveMemberInput) (*dto.RemoveMemberOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	if err := teamService.RemoveMember(userID, input.TeamID, input.MemberID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.RemoveMemberOutput{Body: dto.MessageResponse{Message: "Member removed successfully"}}, nil
}

func LeaveTeamHandler(ctx context.Context, input *dto.LeaveTeamInput) (*dto.LeaveTeamOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	if err := teamService.LeaveTeam(userID, input.TeamID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.LeaveTeamOutput{Body: dto.MessageResponse{Message: "Left team successfully"}}, nil
}

// ---------- Portfolio ----------

func CreatePortfolioHandler(ctx context.Context, input *dto.CreatePortfolioInput) (*dto.CreatePortfolioOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	if err := teamService.CreatePortfolio(userID, input.TeamID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.CreatePortfolioOutput{Body: dto.MessageResponse{Message: "Portfolio created successfully"}}, nil
}

func UpdatePortfolioHandler(ctx context.Context, input *dto.UpdatePortfolioInput) (*dto.UpdatePortfolioOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	if err := teamService.UpdatePortfolio(userID, input.PortfolioID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.UpdatePortfolioOutput{Body: dto.MessageResponse{Message: "Portfolio updated successfully"}}, nil
}

func DeletePortfolioHandler(ctx context.Context, input *dto.DeletePortfolioInput) (*dto.DeletePortfolioOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	if err := teamService.DeletePortfolio(userID, input.PortfolioID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.DeletePortfolioOutput{Body: dto.MessageResponse{Message: "Portfolio deleted successfully"}}, nil
}

// ---------- Skills ----------

func AddSkillHandler(ctx context.Context, input *dto.AddSkillInput) (*dto.AddSkillOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	if err := teamService.AddSkill(userID, input.TeamID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.AddSkillOutput{Body: dto.MessageResponse{Message: "Skill added successfully"}}, nil
}

func UpdateSkillHandler(ctx context.Context, input *dto.UpdateSkillInput) (*dto.UpdateSkillOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	if err := teamService.UpdateSkill(userID, input.SkillID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.UpdateSkillOutput{Body: dto.MessageResponse{Message: "Skill updated successfully"}}, nil
}

func DeleteSkillHandler(ctx context.Context, input *dto.DeleteSkillInput) (*dto.DeleteSkillOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	teamService := services.NewTeamService()
	if err := teamService.DeleteSkill(userID, input.SkillID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.DeleteSkillOutput{Body: dto.MessageResponse{Message: "Skill deleted successfully"}}, nil
}
