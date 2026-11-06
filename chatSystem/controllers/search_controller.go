package controllers

import (
	"context"

	"techguild-chat/dto"
	"techguild-chat/services"
	"techguild-chat/utils"

	"github.com/danielgtaylor/huma/v2"
)

type SearchController struct {
	searchService services.SearchService
}

func NewSearchController(searchService services.SearchService) *SearchController {
	return &SearchController{
		searchService: searchService,
	}
}

// SearchMessagesHandler handles GET /v1/communication/search?q=
func (c *SearchController) SearchMessagesHandler(ctx context.Context, input *dto.SearchMessagesInput) (*dto.SearchMessagesOutput, error) {
	userUUID, err := utils.GetUserUUIDFromContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("Unauthorized: user ID missing or invalid")
	}

	res, err := c.searchService.SearchMessages(ctx, userUUID, input.Q, input.ChannelID, input.Limit, input.Offset)
	if err != nil {
		if err == services.ErrEmptySearchQuery {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if err == services.ErrNotChannelMember {
			return nil, huma.Error403Forbidden(err.Error())
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.SearchMessagesOutput{Body: *res}, nil
}

