package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"
)

// CreateQuestHandler handles the POST /v1/quests endpoint to draft a new quest.
func CreateQuestHandler(ctx context.Context, input *dto.CreateQuestInput) (*dto.CreateQuestOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	questService := services.NewQuestService()
	res, err := questService.CreateQuestDraft(clientID, input.Body)
	if err != nil {
		if errors.Is(err, services.ErrOnlyClientAllowed) || errors.Is(err, services.ErrClientNotVerified) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if errors.Is(err, services.ErrUserNotFound) || errors.Is(err, services.ErrInvalidUserID) {
			return nil, huma.Error401Unauthorized(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.CreateQuestOutput{
		Status: http.StatusCreated,
		Body:   *res,
	}, nil
}

// PublishQuestHandler handles POST /v1/quests/{id}/publish to transition a draft quest to open.
func PublishQuestHandler(ctx context.Context, input *dto.PublishQuestInput) (*dto.PublishQuestOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	questService := services.NewQuestService()
	res, err := questService.PublishQuest(clientID, input.ID, input.Body)
	if err != nil {
		if errors.Is(err, services.ErrQuestNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrInvalidUserID) || errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error401Unauthorized(err.Error())
		}
		if errors.Is(err, services.ErrClientFundingRequired) || errors.Is(err, services.ErrQuestNotDraft) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.PublishQuestOutput{
		Status: http.StatusOK,
		Body:   *res,
	}, nil
}

// ListQuestsHandler handles GET /v1/quests with asymmetric tier visibility and cursor pagination.
func ListQuestsHandler(ctx context.Context, input *dto.ListQuestsInput) (*dto.ListQuestsOutput, error) {
	userID, _ := utils.GetUserIDFromHumaContext(ctx)

	questService := services.NewQuestService()
	res, err := questService.ListQuests(userID, *input)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.ListQuestsOutput{
		Body: *res,
	}, nil
}

// GetQuestDetailsHandler handles GET /v1/quests/{id} to return full details including milestones and skills.
func GetQuestDetailsHandler(ctx context.Context, input *dto.GetQuestDetailsInput) (*dto.GetQuestDetailsOutput, error) {
	userID, _ := utils.GetUserIDFromHumaContext(ctx)

	questService := services.NewQuestService()
	res, err := questService.GetQuestDetails(userID, input.ID)
	if err != nil {
		if errors.Is(err, services.ErrQuestNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrQuestNotVisible) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.GetQuestDetailsOutput{
		Body: *res,
	}, nil
}

// RejectQuestHandler handles POST /v1/quests/{id}/reject to reject/cancel a quest (client only) with escrow refund.
func RejectQuestHandler(ctx context.Context, input *dto.RejectQuestInput) (*dto.RejectQuestOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	questService := services.NewQuestService()
	res, err := questService.RejectQuest(clientID, input.ID, input.Body)
	if err != nil {
		if errors.Is(err, services.ErrQuestNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrOnlyClientCanCancel) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.RejectQuestOutput{
		Body: *res,
	}, nil
}

// AcceptQuestHandler handles POST /v1/quests/{id}/accept where a Party Lead accepts a quest on behalf of a party.
func AcceptQuestHandler(ctx context.Context, input *dto.AcceptQuestInput) (*dto.AcceptQuestOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	questService := services.NewQuestService()
	res, err := questService.AcceptQuest(userID, input.ID, input.Body)
	if err != nil {
		if errors.Is(err, services.ErrQuestNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrOnlyPartyLeadAllowed) || errors.Is(err, services.ErrPartyNotVerified) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if errors.Is(err, services.ErrPartyCapacityExceeded) ||
			errors.Is(err, services.ErrSkillMatchInsufficient) ||
			errors.Is(err, services.ErrClientConflict) ||
			errors.Is(err, services.ErrEscrowNotFunded) ||
			errors.Is(err, services.ErrQuestNotOpen) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.AcceptQuestOutput{
		Status: http.StatusOK,
		Body:   *res,
	}, nil
}
