package controllers

import (
	"context"

	"techguild-chat/dto"
	"techguild-chat/services"
	"techguild-chat/utils"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type MessageController struct {
	messageService services.MessageService
}

func NewMessageController(messageService services.MessageService) *MessageController {
	return &MessageController{
		messageService: messageService,
	}
}

// GetMessagesHandler handles GET /v1/communication/channels/{id}/messages
func (c *MessageController) GetMessagesHandler(ctx context.Context, input *dto.GetMessagesInput) (*dto.GetMessagesOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	res, err := c.messageService.GetMessages(ctx, userUUID, input.ID, input.Cursor, input.Limit, input.Direction)
	if err != nil {
		if err == services.ErrNotChannelMember {
			return nil, huma.Error403Forbidden(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.GetMessagesOutput{Body: *res}, nil
}

// SendMessageHandler handles POST /v1/communication/channels/{id}/messages
func (c *MessageController) SendMessageHandler(ctx context.Context, input *dto.SendMessageInput) (*dto.SendMessageOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	res, err := c.messageService.SendMessage(ctx, userUUID, input.ID, input.Body)
	if err != nil {
		if err == services.ErrNotChannelMember {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if err == services.ErrChannelNotFound {
			return nil, huma.Error404NotFound(err.Error())
		}
		if err == services.ErrEmptyMessage {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.SendMessageOutput{Body: *res}, nil
}

// EditMessageHandler handles PATCH /v1/communication/channels/{id}/messages/{msg_id}
func (c *MessageController) EditMessageHandler(ctx context.Context, input *dto.EditMessageInput) (*dto.EditMessageOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	msgUUID, err := uuid.Parse(input.MsgID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid message ID format")
	}

	res, err := c.messageService.EditMessage(ctx, userUUID, input.ID, msgUUID, input.Body)
	if err != nil {
		if err == services.ErrMessageEditWindowOver || err == services.ErrMessageAlreadyDeleted || err == services.ErrEmptyMessage {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if err == services.ErrNotMessageAuthor {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if err == services.ErrMessageNotFound {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.EditMessageOutput{Body: *res}, nil
}

// DeleteMessageHandler handles DELETE /v1/communication/channels/{id}/messages/{msg_id}
func (c *MessageController) DeleteMessageHandler(ctx context.Context, input *dto.DeleteMessageInput) (*dto.DeleteMessageOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	msgUUID, err := uuid.Parse(input.MsgID)
	if err != nil {
		return nil, huma.Error400BadRequest("Invalid message ID format")
	}

	res, err := c.messageService.DeleteMessage(ctx, userUUID, input.ID, msgUUID)
	if err != nil {
		if err == services.ErrUnauthorizedDelete {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if err == services.ErrMessageNotFound {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.DeleteMessageOutput{Body: *res}, nil
}

