package dto

import (
	"time"

	"github.com/google/uuid"
)

type AddReactionRequest struct {
	Emoji string `json:"emoji" required:"true" doc:"The emoji to react with" example:"🚀"`
}

type AddReactionInput struct {
	ID    string             `path:"id" doc:"Channel ID" required:"true"`
	MsgID string             `path:"msg_id" doc:"Message ID" required:"true"`
	Body  AddReactionRequest `required:"true"`
}

type ReactionResponse struct {
	ID        uuid.UUID `json:"id"`
	MessageID uuid.UUID `json:"message_id"`
	UserID    uuid.UUID `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

type AddReactionOutput struct {
	Body ReactionResponse
}

type RemoveReactionInput struct {
	ID         string `path:"id" doc:"Channel ID" required:"true"`
	MsgID      string `path:"msg_id" doc:"Message ID" required:"true"`
	ReactionID string `path:"reaction_id" doc:"Reaction ID" required:"true"`
}

type RemoveReactionResponse struct {
	ReactionID string `json:"reaction_id"`
	Message    string `json:"message" example:"Reaction removed successfully"`
}

type RemoveReactionOutput struct {
	Body RemoveReactionResponse
}

