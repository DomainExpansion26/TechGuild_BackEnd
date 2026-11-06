package controllers

import (
	"context"

	"techguild-chat/dto"
	"techguild-chat/services"
	"techguild-chat/utils"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type PinController struct {
	pinService services.PinService
}

func NewPinController(pinService services.PinService) *PinController {
	return &PinController{
		pinService: pinService,
	}
}

// PinMessageHandler handles POST /v1/communication/channels/{id}/messages/{msg_id}/pin
func (c *PinController) PinMessageHandler(ctx context.Context, input *dto.PinMessageInput) (*dto.PinMessageOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	msgUUID, err := uuid.Parse(input.MsgID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid message ID format")
	}

	res, err := c.pinService.PinMessage(ctx, userUUID, input.ID, msgUUID)
	if err != nil {
		if err == services.ErrNotChannelMember {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if err == services.ErrCannotPinDeletedMessage {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if err == services.ErrMessageNotFound {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.PinMessageOutput{Body: *res}, nil
}

// UnpinMessageHandler handles DELETE /v1/communication/channels/{id}/messages/{msg_id}/pin
func (c *PinController) UnpinMessageHandler(ctx context.Context, input *dto.UnpinMessageInput) (*dto.UnpinMessageOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	msgUUID, err := uuid.Parse(input.MsgID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid message ID format")
	}

	res, err := c.pinService.UnpinMessage(ctx, userUUID, input.ID, msgUUID)
	if err != nil {
		if err == services.ErrNotChannelMember {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if err == services.ErrMessageNotFound {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.UnpinMessageOutput{Body: *res}, nil
}

// ListPinnedMessagesHandler handles GET /v1/communication/channels/{id}/pinned
func (c *PinController) ListPinnedMessagesHandler(ctx context.Context, input *dto.ListPinnedMessagesInput) (*dto.ListPinnedMessagesOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	res, err := c.pinService.ListPinnedMessages(ctx, userUUID, input.ID)
	if err != nil {
		if err == services.ErrNotChannelMember {
			return nil, huma.Error403Forbidden(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.ListPinnedMessagesOutput{Body: *res}, nil
}

