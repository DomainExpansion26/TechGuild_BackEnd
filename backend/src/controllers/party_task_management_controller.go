package controllers

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"
)

func partyTMDService() services.PartyTaskManagementService {
	return services.NewPartyTaskManagementService()
}

func mapPartyTMDError(err error) error {
	switch {
	case errors.Is(err, services.ErrPartyTMDForbidden):
		return huma.Error403Forbidden(err.Error())
	case errors.Is(err, services.ErrPartyTMDQuestNotFound),
		errors.Is(err, services.ErrPartyTMDMilestoneNotFound),
		errors.Is(err, services.ErrAttachmentNotFound):
		return huma.Error404NotFound(err.Error())
	case errors.Is(err, services.ErrPartyTMDAssignmentState):
		return huma.Error409Conflict(err.Error())
	case errors.Is(err, services.ErrPartyTMDMemberInvalid),
		errors.Is(err, services.ErrPartyTMDInvalidAssignee),
		errors.Is(err, services.ErrPartyTMDInvalidDateRange),
		errors.Is(err, services.ErrPartyTMDInvalidStatus),
		errors.Is(err, services.ErrPartyTMDNoMembers),
		errors.Is(err, services.ErrInvalidUserID):
		return huma.Error400BadRequest(err.Error())
	}
	return huma.Error500InternalServerError(err.Error())
}

// GetPartyTMDDashboardHandler powers the freelancer/party dashboard hero banner.
func GetPartyTMDDashboardHandler(ctx context.Context, input *dto.PartyTMDDashboardInput) (*dto.PartyTMDDashboardOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}
	res, err := partyTMDService().GetDashboard(userID, input.QuestID)
	if err != nil {
		return nil, mapPartyTMDError(err)
	}
	return &dto.PartyTMDDashboardOutput{Body: *res}, nil
}

// GetPartyTMDActiveQuestsHandler lists the caller's in-progress quests.
func GetPartyTMDActiveQuestsHandler(ctx context.Context, _ *dto.PartyTMDActiveQuestsInput) (*dto.PartyTMDActiveQuestsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}
	res, err := partyTMDService().GetActiveQuests(userID)
	if err != nil {
		return nil, mapPartyTMDError(err)
	}
	return &dto.PartyTMDActiveQuestsOutput{Body: *res}, nil
}

// GetPartyTMDMembersHandler powers the members/assignment panel.
func GetPartyTMDMembersHandler(ctx context.Context, input *dto.PartyTMDMembersInput) (*dto.PartyTMDMembersOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}
	res, err := partyTMDService().GetMembersOverview(userID, input.QuestID)
	if err != nil {
		return nil, mapPartyTMDError(err)
	}
	return &dto.PartyTMDMembersOutput{Body: *res}, nil
}

// AssignMilestoneHandler assigns one or more party members to a milestone (lead only).
func AssignMilestoneHandler(ctx context.Context, input *dto.AssignMilestoneInput) (*dto.AssignMilestoneOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}
	res, err := partyTMDService().AssignMilestone(userID, input.MilestoneID, input.Body)
	if err != nil {
		return nil, mapPartyTMDError(err)
	}
	return &dto.AssignMilestoneOutput{Body: *res}, nil
}

// ListMilestoneAssignmentsHandler returns the current assignees of a milestone.
func ListMilestoneAssignmentsHandler(ctx context.Context, input *dto.ListMilestoneAssignmentsInput) (*dto.ListMilestoneAssignmentsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}
	res, err := partyTMDService().ListAssignments(userID, input.MilestoneID)
	if err != nil {
		return nil, mapPartyTMDError(err)
	}
	return &dto.ListMilestoneAssignmentsOutput{Body: *res}, nil
}
