package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"techguild-chat/dto"
	"techguild-chat/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrCannotPinDeletedMessage = errors.New("cannot pin a deleted message")
)

type PinService interface {
	PinMessage(ctx context.Context, userID uuid.UUID, channelID string, msgID uuid.UUID) (*dto.PinMessageResponse, error)
	UnpinMessage(ctx context.Context, userID uuid.UUID, channelID string, msgID uuid.UUID) (*dto.PinMessageResponse, error)
	ListPinnedMessages(ctx context.Context, userID uuid.UUID, channelID string) (*dto.ListPinnedMessagesResponse, error)
}

type pinService struct {
	db         *gorm.DB
	centrifugo CentrifugoService
}

func NewPinService(db *gorm.DB, centrifugo CentrifugoService) PinService {
	return &pinService{
		db:         db,
		centrifugo: centrifugo,
	}
}

// PinMessage sets is_pinned = true on communication_records
func (s *pinService) PinMessage(ctx context.Context, userID uuid.UUID, channelID string, msgID uuid.UUID) (*dto.PinMessageResponse, error) {
	// Scoped to channel members only
	var member models.ChannelMember
	if err := s.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		First(&member).Error; err != nil {
		return nil, ErrNotChannelMember
	}

	var message models.CommunicationRecord
	if err := s.db.WithContext(ctx).
		Where("id = ? AND channel_id = ?", msgID, channelID).
		First(&message).Error; err != nil {
		return nil, ErrMessageNotFound
	}

	if message.Deleted {
		return nil, ErrCannotPinDeletedMessage
	}

	if err := s.db.WithContext(ctx).
		Model(&message).
		Update("is_pinned", true).Error; err != nil {
		return nil, fmt.Errorf("failed to pin message: %w", err)
	}

	// Broadcast pin event via Centrifugo
	event := map[string]interface{}{
		"event":      "message.pinned",
		"channel_id": channelID,
		"message_id": msgID.String(),
		"is_pinned":  true,
		"pinned_by":  userID.String(),
	}
	_ = s.centrifugo.Publish(ctx, channelID, event)

	return &dto.PinMessageResponse{
		MessageID: msgID.String(),
		IsPinned:  true,
		Message:   "Message pinned successfully",
	}, nil
}

// UnpinMessage sets is_pinned = false on communication_records
func (s *pinService) UnpinMessage(ctx context.Context, userID uuid.UUID, channelID string, msgID uuid.UUID) (*dto.PinMessageResponse, error) {
	// Scoped to channel members only
	var member models.ChannelMember
	if err := s.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		First(&member).Error; err != nil {
		return nil, ErrNotChannelMember
	}

	var message models.CommunicationRecord
	if err := s.db.WithContext(ctx).
		Where("id = ? AND channel_id = ?", msgID, channelID).
		First(&message).Error; err != nil {
		return nil, ErrMessageNotFound
	}

	if err := s.db.WithContext(ctx).
		Model(&message).
		Update("is_pinned", false).Error; err != nil {
		return nil, fmt.Errorf("failed to unpin message: %w", err)
	}

	// Broadcast unpin event via Centrifugo
	event := map[string]interface{}{
		"event":      "message.unpinned",
		"channel_id": channelID,
		"message_id": msgID.String(),
		"is_pinned":  false,
		"unpinned_by": userID.String(),
	}
	_ = s.centrifugo.Publish(ctx, channelID, event)

	return &dto.PinMessageResponse{
		MessageID: msgID.String(),
		IsPinned:  false,
		Message:   "Message unpinned successfully",
	}, nil
}

// ListPinnedMessages returns fast indexed list of pinned messages
func (s *pinService) ListPinnedMessages(ctx context.Context, userID uuid.UUID, channelID string) (*dto.ListPinnedMessagesResponse, error) {
	// Scoped to channel members only
	var member models.ChannelMember
	if err := s.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		First(&member).Error; err != nil {
		return nil, ErrNotChannelMember
	}

	var records []models.CommunicationRecord
	err := s.db.WithContext(ctx).
		Where("channel_id = ? AND is_pinned = true AND deleted = false", channelID).
		Order("created_at DESC").
		Find(&records).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list pinned messages: %w", err)
	}

	// Fetch sender names
	userIDs := make([]uuid.UUID, 0, len(records))
	for _, r := range records {
		userIDs = append(userIDs, r.FromUserID)
	}
	nameMap := s.getUserNames(ctx, userIDs)

	messages := make([]dto.MessageResponse, len(records))
	for i, r := range records {
		messages[i] = dto.MessageResponse{
			ID:          r.ID,
			ChannelID:   r.ChannelID,
			ChannelType: string(r.ChannelType),
			FromUserID:  r.FromUserID,
			SenderName:  nameMap[r.FromUserID],
			ToUserID:    r.ToUserID,
			ReplyToID:   r.ReplyToID,
			ContentRef:  r.ContentRef,
			ContentText: r.ContentText,
			MessageType: string(r.MessageType),
			IsPinned:    r.IsPinned,
			Deleted:     r.Deleted,
			CreatedAt:   r.CreatedAt,
			EditedAt:    r.EditedAt,
		}
	}

	return &dto.ListPinnedMessagesResponse{
		ChannelID: channelID,
		Total:     len(messages),
		Messages:  messages,
	}, nil
}

func (s *pinService) getUserNames(ctx context.Context, userIDs []uuid.UUID) map[uuid.UUID]string {
	result := make(map[uuid.UUID]string)
	if len(userIDs) == 0 {
		return result
	}

	type userRow struct {
		ID        uuid.UUID
		FirstName string
		LastName  string
	}
	var rows []userRow
	_ = s.db.WithContext(ctx).Table("users").Select("id, first_name, last_name").Where("id IN ?", userIDs).Scan(&rows)
	for _, r := range rows {
		result[r.ID] = strings.TrimSpace(r.FirstName + " " + r.LastName)
	}
	return result
}

