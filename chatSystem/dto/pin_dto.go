package dto

type PinMessageInput struct {
	ID    string `path:"id" doc:"Channel ID" required:"true"`
	MsgID string `path:"msg_id" doc:"Message ID" required:"true"`
}

type PinMessageResponse struct {
	MessageID string `json:"message_id"`
	IsPinned  bool   `json:"is_pinned"`
	Message   string `json:"message" example:"Message pinned successfully"`
}

type PinMessageOutput struct {
	Body PinMessageResponse
}

type UnpinMessageInput struct {
	ID    string `path:"id" doc:"Channel ID" required:"true"`
	MsgID string `path:"msg_id" doc:"Message ID" required:"true"`
}

type UnpinMessageOutput struct {
	Body PinMessageResponse
}

type ListPinnedMessagesInput struct {
	ID string `path:"id" doc:"Channel ID" required:"true"`
}

type ListPinnedMessagesResponse struct {
	ChannelID string            `json:"channel_id"`
	Total     int               `json:"total"`
	Messages  []MessageResponse `json:"messages"`
}

type ListPinnedMessagesOutput struct {
	Body ListPinnedMessagesResponse
}

