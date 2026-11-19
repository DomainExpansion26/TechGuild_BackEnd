package dto

import (
	"time"

	"github.com/google/uuid"
)

// ReactionCountResponse groups reactions by emoji
type ReactionCountResponse struct {
	Emoji string      `json:"emoji" example:"👍"`
	Count int         `json:"count" example:"3"`
	Users []uuid.UUID `json:"users"`
	Me    bool        `json:"me" doc:"Whether current user reacted with this emoji"`
}

// ReplySummary provides lightweight context for replied-to messages
type ReplySummary struct {
	ID          uuid.UUID `json:"id"`
	FromUserID  uuid.UUID `json:"from_user_id"`
	SenderName  string    `json:"sender_name"`
	ContentText string    `json:"content_text"`
	MessageType string    `json:"message_type"`
}

// MessageResponse represents a persisted communication record
type MessageResponse struct {
	ID          uuid.UUID               `json:"id"`
	ChannelID   string                  `json:"channel_id"`
	ChannelType string                  `json:"channel_type"`
	FromUserID  uuid.UUID               `json:"from_user_id"`
	SenderName  string                  `json:"sender_name"`
	ToUserID    *uuid.UUID              `json:"to_user_id,omitempty"`
	ReplyToID   *uuid.UUID              `json:"reply_to_id,omitempty"`
	ReplyTo     *ReplySummary           `json:"reply_to,omitempty"`
	ContentRef  *string                 `json:"content_ref,omitempty"`
	ContentText string                  `json:"content_text"`
	MessageType string                  `json:"message_type"`
	Metadata    map[string]interface{}  `json:"metadata,omitempty"`
	IsPinned    bool                    `json:"is_pinned"`
	Deleted     bool                    `json:"deleted"`
	CreatedAt   time.Time               `json:"created_at"`
	EditedAt    *time.Time              `json:"edited_at,omitempty"`
	Reactions   []ReactionCountResponse `json:"reactions,omitempty"`
}

// GetMessagesInput query parameters for cursor-based message history retrieval
type GetMessagesInput struct {
	ID        string `path:"id" doc:"Channel ID" required:"true"`
	Cursor    string `query:"cursor" doc:"Opaque cursor from previous page"`
	Limit     int    `query:"limit" doc:"Number of messages to retrieve (default 20, max 100)" example:"20"`
	Direction string `query:"direction" doc:"Direction of pagination: 'before' or 'after' (default 'before')" example:"before"`
}

type GetMessagesResponse struct {
	ChannelID  string            `json:"channel_id"`
	Messages   []MessageResponse `json:"messages"`
	NextCursor string            `json:"next_cursor,omitempty"`
	HasMore    bool              `json:"has_more"`
}

type GetMessagesOutput struct {
	Body GetMessagesResponse
}

// SendMessageRequest input payload for sending a chat message
type SendMessageRequest struct {
	ContentText string                 `json:"content_text" required:"true" doc:"Message text content" example:"Hello team, here is the updated plan."`
	MessageType string                 `json:"message_type,omitempty" doc:"Message type: text, file_share, system_message" example:"text"`
	ReplyToID   *string                `json:"reply_to_id,omitempty" doc:"Optional ID of the message being replied to" example:"123e4567-e89b-12d3-a456-426614174000"`
	ContentRef  *string                `json:"content_ref,omitempty" doc:"Optional URL or storage reference for attachments"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type SendMessageInput struct {
	ID   string             `path:"id" doc:"Channel ID" required:"true"`
	Body SendMessageRequest `required:"true"`
}

type SendMessageOutput struct {
	Body MessageResponse
}

// EditMessageRequest input payload for editing a message within 5 minutes
type EditMessageRequest struct {
	ContentText string `json:"content_text" required:"true" doc:"Updated message text" example:"Hello team, updated plan with revisions."`
}

type EditMessageInput struct {
	ID    string             `path:"id" doc:"Channel ID" required:"true"`
	MsgID string             `path:"msg_id" doc:"Message ID" required:"true"`
	Body  EditMessageRequest `required:"true"`
}

type EditMessageOutput struct {
	Body MessageResponse
}

// DeleteMessageInput for soft-deleting a message with tombstone
type DeleteMessageInput struct {
	ID    string `path:"id" doc:"Channel ID" required:"true"`
	MsgID string `path:"msg_id" doc:"Message ID" required:"true"`
}

type DeleteMessageResponse struct {
	MessageID string `json:"message_id"`
	Message   string `json:"message" example:"Message deleted successfully"`
	Tombstone bool   `json:"tombstone" example:"true"`
}

type DeleteMessageOutput struct {
	Body DeleteMessageResponse
}

