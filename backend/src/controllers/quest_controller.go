package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/config"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"
)

// QuestController holds the quest handlers that need provider/config state —
// today only cancel-with-refund.
type QuestController struct {
	svc services.QuestService
}

func NewQuestController(cfg *config.Config, provider services.PaymentProvider) *QuestController {
	return &QuestController{
		svc: services.NewQuestServiceWithRefund(services.NewRefundService(cfg, provider)),
	}
}

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

// RejectQuestHandler handles POST /v1/quests/{id}/reject to cancel a quest
// (client only), refunding every milestone that still holds escrow.
func (c *QuestController) RejectQuestHandler(ctx context.Context, input *dto.RejectQuestInput) (*dto.RejectQuestOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := c.svc.RejectQuest(clientID, input.ID, input.Body)
	if err != nil {
		if errors.Is(err, services.ErrQuestNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrOnlyClientCanCancel) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if errors.Is(err, services.ErrQuestLegacyEscrow) || errors.Is(err, services.ErrQuestAlreadyFinished) {
			return nil, huma.Error409Conflict(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.RejectQuestOutput{
		Body: *res,
	}, nil
}

// UpdateQuestHandler handles PATCH /v1/quests/{id} (draft-only partial edit).
func (c *QuestController) UpdateQuestHandler(ctx context.Context, input *dto.UpdateQuestInput) (*dto.UpdateQuestOutput, error) {
	clientID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := c.svc.UpdateQuest(clientID, input.ID, input.Body)
	if err != nil {
		if errors.Is(err, services.ErrQuestNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrQuestNotOwner) {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if errors.Is(err, services.ErrQuestNotEditable) {
			return nil, huma.Error409Conflict(err.Error())
		}
		if errors.Is(err, services.ErrInvalidUserID) || errors.Is(err, services.ErrUserNotFound) {
			return nil, huma.Error401Unauthorized(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.UpdateQuestOutput{Body: *res}, nil
}

// SaveQuestHandler handles POST /v1/quests/{id}/save (bookmark toggle).
func (c *QuestController) SaveQuestHandler(ctx context.Context, input *dto.SaveQuestInput) (*dto.SaveQuestOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := c.svc.SaveQuest(userID, input.ID)
	if err != nil {
		if errors.Is(err, services.ErrQuestNotFound) {
			return nil, huma.Error404NotFound(err.Error())
		}
		if errors.Is(err, services.ErrInvalidUserID) {
			return nil, huma.Error401Unauthorized(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.SaveQuestOutput{Body: *res}, nil
}

// ListSavedQuestsHandler handles GET /v1/quests/saved.
func (c *QuestController) ListSavedQuestsHandler(ctx context.Context, input *dto.ListSavedQuestsInput) (*dto.ListSavedQuestsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := c.svc.ListSavedQuests(userID)
	if err != nil {
		if errors.Is(err, services.ErrInvalidUserID) {
			return nil, huma.Error401Unauthorized(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.ListSavedQuestsOutput{Body: *res}, nil
}

// BoardSummaryHandler handles GET /v1/quests/board/summary.
func (c *QuestController) BoardSummaryHandler(ctx context.Context, input *dto.QuestBoardSummaryInput) (*dto.QuestBoardSummaryOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	res, err := c.svc.GetBoardSummary(userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) || errors.Is(err, services.ErrInvalidUserID) {
			return nil, huma.Error401Unauthorized(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.QuestBoardSummaryOutput{Body: *res}, nil
}
