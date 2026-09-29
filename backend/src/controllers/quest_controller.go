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
