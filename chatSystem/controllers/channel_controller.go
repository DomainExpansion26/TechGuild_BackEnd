package controllers

import (
	"context"

	"techguild-chat/dto"
	"techguild-chat/services"
	"techguild-chat/utils"

	"github.com/danielgtaylor/huma/v2"
)

type ChannelController struct {
	channelService services.ChannelService
}

func NewChannelController(channelService services.ChannelService) *ChannelController {
	return &ChannelController{
		channelService: channelService,
	}
}

// ListChannelsHandler handles GET /v1/communication/channels
func (c *ChannelController) ListChannelsHandler(ctx context.Context, input *dto.ListChannelsInput) (*dto.ListChannelsOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	res, err := c.channelService.ListUserChannels(ctx, userUUID)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.ListChannelsOutput{Body: *res}, nil
}

// CreateDMChannelHandler handles POST /v1/communication/channels
func (c *ChannelController) CreateDMChannelHandler(ctx context.Context, input *dto.CreateDMChannelInput) (*dto.CreateChannelOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	res, err := c.channelService.CreateDMChannel(ctx, userUUID, input.Body)
	if err != nil {
		if err == services.ErrInvalidRecipient || err == services.ErrRecipientNotFound {
			return nil, huma.Error400BadRequest(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.CreateChannelOutput{Body: *res}, nil
}

// GetChannelInfoHandler handles GET /v1/communication/channels/{id}
func (c *ChannelController) GetChannelInfoHandler(ctx context.Context, input *dto.GetChannelInfoInput) (*dto.GetChannelInfoOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	res, err := c.channelService.GetChannelInfo(ctx, userUUID, input.ID)
	if err != nil {
		if err == services.ErrNotChannelMember {
			return nil, huma.Error403Forbidden(err.Error())
		}
		if err == services.ErrChannelNotFound {
			return nil, huma.Error404NotFound(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.GetChannelInfoOutput{Body: *res}, nil
}

// ListChannelMembersHandler handles GET /v1/communication/channels/{id}/members
func (c *ChannelController) ListChannelMembersHandler(ctx context.Context, input *dto.ListChannelMembersInput) (*dto.ListChannelMembersOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	res, err := c.channelService.ListChannelMembers(ctx, userUUID, input.ID)
	if err != nil {
		if err == services.ErrNotChannelMember {
			return nil, huma.Error403Forbidden(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.ListChannelMembersOutput{Body: *res}, nil
}

// MuteChannelHandler handles PATCH /v1/communication/channels/{id}/mute
func (c *ChannelController) MuteChannelHandler(ctx context.Context, input *dto.MuteChannelInput) (*dto.MuteChannelOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	res, err := c.channelService.MuteChannel(ctx, userUUID, input.ID, input.Body.Muted)
	if err != nil {
		if err == services.ErrNotChannelMember {
			return nil, huma.Error403Forbidden(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.MuteChannelOutput{Body: *res}, nil
}

