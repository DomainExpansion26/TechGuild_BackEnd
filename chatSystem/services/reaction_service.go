package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"techguild-chat/dto"
	"techguild-chat/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrReactionNotFound    = errors.New("reaction not found")
	ErrNotReactionOwner    = errors.New("unauthorized: you can only remove your own reactions")
	ErrInvalidEmoji        = errors.New("emoji cannot be empty")
	ErrReactionAlreadyDone = errors.New("reaction already added")
)

type ReactionService interface {
	AddReaction(ctx context.Context, userID uuid.UUID, channelID string, msgID uuid.UUID, emoji string) (*dto.ReactionResponse, error)
	RemoveReaction(ctx context.Context, userID uuid.UUID, channelID string, msgID uuid.UUID, reactionID uuid.UUID) (*dto.RemoveReactionResponse, error)
}

type reactionService struct {
	db         *gorm.DB
	centrifugo CentrifugoService
}

func NewReactionService(db *gorm.DB, centrifugo CentrifugoService) ReactionService {
	return &reactionService{
		db:         db,
		centrifugo: centrifugo,
	}
}

// AddReaction validates channel membership, applies unique constraint, and broadcasts event
func (s *reactionService) AddReaction(ctx context.Context, userID uuid.UUID, channelID string, msgID uuid.UUID, emoji string) (*dto.ReactionResponse, error) {
	emoji = strings.TrimSpace(emoji)
	if emoji == "" {
		return nil, ErrInvalidEmoji
	}

	// 1. Scoped to channel members only
	var member models.ChannelMember
	if err := s.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		First(&member).Error; err != nil {
		return nil, ErrNotChannelMember
	}

	// 2. Validate message exists in this channel and is not deleted
	var message models.CommunicationRecord
	if err := s.db.WithContext(ctx).
		Where("id = ? AND channel_id = ? AND deleted = false", msgID, channelID).
		First(&message).Error; err != nil {
		return nil, ErrMessageNotFound
	}

	// 3. Check unique constraint (message_id, user_id, emoji)
	var existing models.MessageReaction
	err := s.db.WithContext(ctx).
		Where("message_id = ? AND user_id = ? AND emoji = ?", msgID, userID, emoji).
		First(&existing).Error

	if err == nil {
		// Idempotent return if already reacted
		return &dto.ReactionResponse{
			ID:        existing.ID,
			MessageID: existing.MessageID,
			UserID:    existing.UserID,
			Emoji:     existing.Emoji,
			CreatedAt: existing.CreatedAt,
		}, nil
	}

	reaction := models.MessageReaction{
		ID:        uuid.New(),
		MessageID: msgID,
		UserID:    userID,
		Emoji:     emoji,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.db.WithContext(ctx).Create(&reaction).Error; err != nil {
		return nil, fmt.Errorf("failed to save reaction: %w", err)
	}

	resp := &dto.ReactionResponse{
		ID:        reaction.ID,
		MessageID: reaction.MessageID,
		UserID:    reaction.UserID,
		Emoji:     reaction.Emoji,
		CreatedAt: reaction.CreatedAt,
	}

	// 4. Broadcast reaction event via Centrifugo
	event := map[string]interface{}{
		"event":      "reaction.added",
		"channel_id": channelID,
		"message_id": msgID.String(),
		"reaction":   resp,
	}
	_ = s.centrifugo.Publish(ctx, channelID, event)

	return resp, nil
}

// RemoveReaction removes reaction if requester is the reaction owner
func (s *reactionService) RemoveReaction(ctx context.Context, userID uuid.UUID, channelID string, msgID uuid.UUID, reactionID uuid.UUID) (*dto.RemoveReactionResponse, error) {
	// Scoped to channel members only
	var member models.ChannelMember
	if err := s.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		First(&member).Error; err != nil {
		return nil, ErrNotChannelMember
	}

	var reaction models.MessageReaction
	if err := s.db.WithContext(ctx).
		Where("id = ? AND message_id = ?", reactionID, msgID).
		First(&reaction).Error; err != nil {
		return nil, ErrReactionNotFound
	}

	if reaction.UserID != userID {
		return nil, ErrNotReactionOwner
	}

	if err := s.db.WithContext(ctx).Delete(&reaction).Error; err != nil {
		return nil, fmt.Errorf("failed to delete reaction: %w", err)
	}

	// Broadcast reaction removal via Centrifugo
	event := map[string]interface{}{
		"event":       "reaction.removed",
		"channel_id":  channelID,
		"message_id":  msgID.String(),
		"reaction_id": reactionID.String(),
		"emoji":       reaction.Emoji,
		"user_id":     userID.String(),
	}
	_ = s.centrifugo.Publish(ctx, channelID, event)

	return &dto.RemoveReactionResponse{
		ReactionID: reactionID.String(),
		Message:    "Reaction removed successfully",
	}, nil
}

