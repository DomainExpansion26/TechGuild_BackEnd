package controllers

import (
	"context"

	"techguild-chat/dto"
	"techguild-chat/services"
	"techguild-chat/utils"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type ReactionController struct {
	reactionService services.ReactionService
}

func NewReactionController(reactionService services.ReactionService) *ReactionController {
	return &ReactionController{
		reactionService: reactionService,
	}
}

// AddReactionHandler handles POST /v1/communication/channels/{id}/messages/{msg_id}/reactions
func (c *ReactionController) AddReactionHandler(ctx context.Context, input *dto.AddReactionInput) (*dto.AddReactionOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	msgUUID, err := uuid.Parse(input.MsgID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid message ID format")
	}

	res, err := c.reactionService.AddReaction(ctx, userUUID, input.ID, msgUUID, input.Body.Emoji)
	if err != nil {
		if err == services.ErrNotChannelMember {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if err == services.ErrMessageNotFound {
			return nil, huma.Error404NotFound(err.Error())
		}
		if err == services.ErrInvalidEmoji {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.AddReactionOutput{Body: *res}, nil
}

// RemoveReactionHandler handles DELETE /v1/communication/channels/{id}/messages/{msg_id}/reactions/{reaction_id}
func (c *ReactionController) RemoveReactionHandler(ctx context.Context, input *dto.RemoveReactionInput) (*dto.RemoveReactionOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	msgUUID, err := uuid.Parse(input.MsgID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid message ID format")
	}

	reactionUUID, err := uuid.Parse(input.ReactionID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid reaction ID format")
	}

	res, err := c.reactionService.RemoveReaction(ctx, userUUID, input.ID, msgUUID, reactionUUID)
	if err != nil {
		if err == services.ErrNotChannelMember || err == services.ErrNotReactionOwner {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if err == services.ErrReactionNotFound {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.RemoveReactionOutput{Body: *res}, nil
}

