package dto

type SearchMessagesInput struct {
	Q         string `query:"q" required:"true" doc:"Search query keyword" example:"architecture"`
	ChannelID string `query:"channel_id" doc:"Optional channel ID to scope search"`
	Limit     int    `query:"limit" doc:"Maximum results to return (default 20, max 100)" example:"20"`
	Offset    int    `query:"offset" doc:"Offset for search results" example:"0"`
}

type MessageSearchResult struct {
	MessageResponse
	ChannelName string `json:"channel_name"`
}

type SearchMessagesResponse struct {
	Query     string                `json:"query"`
	Total     int                   `json:"total"`
	ChannelID *string               `json:"channel_id,omitempty"`
	Messages  []MessageSearchResult `json:"messages"`
}

type SearchMessagesOutput struct {
	Body SearchMessagesResponse
}

