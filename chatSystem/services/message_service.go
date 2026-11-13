package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"techguild-chat/dto"
	"techguild-chat/models"
	"techguild-chat/utils"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrMessageNotFound      = errors.New("message not found")
	ErrMessageEditWindowOver = errors.New("messages can only be edited within 5 minutes of sending")
	ErrMessageAlreadyDeleted = errors.New("cannot edit a deleted message")
	ErrNotMessageAuthor      = errors.New("only the author can edit this message")
	ErrUnauthorizedDelete    = errors.New("unauthorized: only the author or a channel admin can delete this message")
	ErrEmptyMessage          = errors.New("message text cannot be empty")
)

type MessageService interface {
	SendMessage(ctx context.Context, userID uuid.UUID, channelID string, req dto.SendMessageRequest) (*dto.MessageResponse, error)
	GetMessages(ctx context.Context, userID uuid.UUID, channelID string, cursor string, limit int, direction string) (*dto.GetMessagesResponse, error)
	EditMessage(ctx context.Context, userID uuid.UUID, channelID string, msgID uuid.UUID, req dto.EditMessageRequest) (*dto.MessageResponse, error)
	DeleteMessage(ctx context.Context, userID uuid.UUID, channelID string, msgID uuid.UUID) (*dto.DeleteMessageResponse, error)
}

type messageService struct {
	db         *gorm.DB
	centrifugo CentrifugoService
}

func NewMessageService(db *gorm.DB, centrifugo CentrifugoService) MessageService {
	return &messageService{
		db:         db,
		centrifugo: centrifugo,
	}
}

// SendMessage validates membership, persists message, and broadcasts via Centrifugo
func (s *messageService) SendMessage(ctx context.Context, userID uuid.UUID, channelID string, req dto.SendMessageRequest) (*dto.MessageResponse, error) {
	if strings.TrimSpace(req.ContentText) == "" && req.ContentRef == nil {
		return nil, ErrEmptyMessage
	}

	// 1. Validate user is member of channel
	var member models.ChannelMember
	if err := s.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		First(&member).Error; err != nil {
		return nil, ErrNotChannelMember
	}

	// 2. Fetch channel metadata
	var channel models.ChatChannel
	if err := s.db.WithContext(ctx).Where("id = ?", channelID).First(&channel).Error; err != nil {
		return nil, ErrChannelNotFound
	}

	msgType := models.MessageTypeText
	if req.MessageType != "" {
		msgType = models.MessageType(req.MessageType)
	}

	var replyUUID *uuid.UUID
	if req.ReplyToID != nil && *req.ReplyToID != "" {
		parsed, err := uuid.Parse(*req.ReplyToID)
		if err == nil {
			replyUUID = &parsed
		}
	}

	var metaJSON datatypes.JSON
	if req.Metadata != nil {
		raw, err := json.Marshal(req.Metadata)
		if err == nil {
			metaJSON = raw
		}
	}

	now := time.Now().UTC()
	record := models.CommunicationRecord{
		ID:          uuid.New(),
		QuestID:     channel.QuestID,
		PartyID:     channel.PartyID,
		FromUserID:  userID,
		ChannelID:   channelID,
		ChannelType: channel.Type,
		ReplyToID:   replyUUID,
		ContentRef:  req.ContentRef,
		ContentText: strings.TrimSpace(req.ContentText),
		MessageType: msgType,
		Metadata:    metaJSON,
		IsPinned:    false,
		Deleted:     false,
		CreatedAt:   now,
	}

	// If 1:1 DM channel, populate ToUserID with the other member
	if channel.Type == models.ChannelTypeDM {
		var otherMember models.ChannelMember
		if err := s.db.WithContext(ctx).
			Where("channel_id = ? AND user_id != ?", channelID, userID).
			First(&otherMember).Error; err == nil {
			record.ToUserID = &otherMember.UserID
		}
	}

	// 3. Save to database & touch channel updated_at
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		return tx.Model(&channel).Update("updated_at", now).Error
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	// Fetch sender name
	senderName := s.getUserFullName(ctx, userID)

	resp := s.toMessageResponse(&record, senderName, userID)

	// 4. Publish to Centrifugo for real-time delivery
	event := map[string]interface{}{
		"event":   "message.created",
		"message": resp,
	}
	_ = s.centrifugo.Publish(ctx, channelID, event)
	_ = s.centrifugo.Publish(ctx, fmt.Sprintf("%s:%s", channel.Type, channelID), event)

	return resp, nil
}

// GetMessages retrieves paginated, cursor-based message history
func (s *messageService) GetMessages(ctx context.Context, userID uuid.UUID, channelID string, cursor string, limit int, direction string) (*dto.GetMessagesResponse, error) {
	// Validate channel membership
	var member models.ChannelMember
	if err := s.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		First(&member).Error; err != nil {
		return nil, ErrNotChannelMember
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	query := s.db.WithContext(ctx).
		Where("channel_id = ?", channelID)

	// Cursor pagination
	if cursor != "" {
		cData, err := utils.DecodeCursor(cursor)
		if err == nil && cData != nil {
			if direction == "after" {
				query = query.Where("(created_at > ?) OR (created_at = ? AND id > ?)", cData.CreatedAt, cData.CreatedAt, cData.ID)
			} else {
				query = query.Where("(created_at < ?) OR (created_at = ? AND id < ?)", cData.CreatedAt, cData.CreatedAt, cData.ID)
			}
		}
	}

	query = query.Order("created_at DESC, id DESC").Limit(limit + 1)

	var records []models.CommunicationRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}

	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}

	var nextCursor string
	if hasMore && len(records) > 0 {
		last := records[len(records)-1]
		nextCursor = utils.EncodeCursor(last.CreatedAt, last.ID)
	}

	// Collect user IDs for sender names
	userIDs := make([]uuid.UUID, 0, len(records))
	msgIDs := make([]uuid.UUID, 0, len(records))
	for _, r := range records {
		userIDs = append(userIDs, r.FromUserID)
		msgIDs = append(msgIDs, r.ID)
	}

	nameMap := s.getUserNames(ctx, userIDs)
	reactionMap := s.getMessageReactions(ctx, msgIDs, userID)

	messages := make([]dto.MessageResponse, len(records))
	for i, r := range records {
		resp := s.toMessageResponse(&r, nameMap[r.FromUserID], userID)
		resp.Reactions = reactionMap[r.ID]

		// Tombstone handling for deleted messages
		if r.Deleted {
			resp.ContentText = "This message was deleted"
			resp.ContentRef = nil
			resp.Metadata = nil
		}

		messages[i] = *resp
	}

	return &dto.GetMessagesResponse{
		ChannelID:  channelID,
		Messages:   messages,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// EditMessage allows author to edit message within 5 minutes of sending
func (s *messageService) EditMessage(ctx context.Context, userID uuid.UUID, channelID string, msgID uuid.UUID, req dto.EditMessageRequest) (*dto.MessageResponse, error) {
	if strings.TrimSpace(req.ContentText) == "" {
		return nil, ErrEmptyMessage
	}

	var record models.CommunicationRecord
	if err := s.db.WithContext(ctx).
		Where("id = ? AND channel_id = ?", msgID, channelID).
		First(&record).Error; err != nil {
		return nil, ErrMessageNotFound
	}

	if record.Deleted {
		return nil, ErrMessageAlreadyDeleted
	}

	if record.FromUserID != userID {
		return nil, ErrNotMessageAuthor
	}

	// 5-minute edit window check
	if time.Since(record.CreatedAt) > 5*time.Minute {
		return nil, ErrMessageEditWindowOver
	}

	// Preserve original text in metadata
	var meta map[string]interface{}
	if len(record.Metadata) > 0 {
		_ = json.Unmarshal(record.Metadata, &meta)
	}
	if meta == nil {
		meta = make(map[string]interface{})
	}

	history, _ := meta["edit_history"].([]interface{})
	history = append(history, map[string]interface{}{
		"text":      record.ContentText,
		"edited_at": time.Now().UTC(),
	})
	meta["original_text"] = record.ContentText
	meta["edit_history"] = history

	newMetaBytes, _ := json.Marshal(meta)
	now := time.Now().UTC()

	record.ContentText = strings.TrimSpace(req.ContentText)
	record.EditedAt = &now
	record.Metadata = newMetaBytes

	if err := s.db.WithContext(ctx).Save(&record).Error; err != nil {
		return nil, fmt.Errorf("failed to update message: %w", err)
	}

	senderName := s.getUserFullName(ctx, userID)
	resp := s.toMessageResponse(&record, senderName, userID)

	// Broadcast edit event to Centrifugo
	event := map[string]interface{}{
		"event":   "message.edited",
		"message": resp,
	}
	_ = s.centrifugo.Publish(ctx, channelID, event)

	return resp, nil
}

// DeleteMessage performs soft delete with tombstone (for audit continuity)
func (s *messageService) DeleteMessage(ctx context.Context, userID uuid.UUID, channelID string, msgID uuid.UUID) (*dto.DeleteMessageResponse, error) {
	var record models.CommunicationRecord
	if err := s.db.WithContext(ctx).
		Where("id = ? AND channel_id = ?", msgID, channelID).
		First(&record).Error; err != nil {
		return nil, ErrMessageNotFound
	}

	// Check if author or channel admin/leader
	if record.FromUserID != userID {
		var member models.ChannelMember
		err := s.db.WithContext(ctx).
			Where("channel_id = ? AND user_id = ?", channelID, userID).
			First(&member).Error
		if err != nil || (member.Role != models.MemberRoleLeader && member.Role != models.MemberRoleAdmin) {
			return nil, ErrUnauthorizedDelete
		}
	}

	// Soft delete with tombstone
	if err := s.db.WithContext(ctx).
		Model(&record).
		Update("deleted", true).Error; err != nil {
		return nil, fmt.Errorf("failed to soft delete message: %w", err)
	}

	// Broadcast delete/tombstone event
	event := map[string]interface{}{
		"event":      "message.deleted",
		"channel_id": channelID,
		"message_id": msgID.String(),
		"tombstone":  true,
	}
	_ = s.centrifugo.Publish(ctx, channelID, event)

	return &dto.DeleteMessageResponse{
		MessageID: msgID.String(),
		Message:   "Message deleted successfully",
		Tombstone: true,
	}, nil
}

// Helpers
func (s *messageService) toMessageResponse(r *models.CommunicationRecord, senderName string, currentUserID uuid.UUID) *dto.MessageResponse {
	var meta map[string]interface{}
	if len(r.Metadata) > 0 {
		_ = json.Unmarshal(r.Metadata, &meta)
	}

	return &dto.MessageResponse{
		ID:          r.ID,
		ChannelID:   r.ChannelID,
		ChannelType: string(r.ChannelType),
		FromUserID:  r.FromUserID,
		SenderName:  senderName,
		ToUserID:    r.ToUserID,
		ReplyToID:   r.ReplyToID,
		ContentRef:  r.ContentRef,
		ContentText: r.ContentText,
		MessageType: string(r.MessageType),
		Metadata:    meta,
		IsPinned:    r.IsPinned,
		Deleted:     r.Deleted,
		CreatedAt:   r.CreatedAt,
		EditedAt:    r.EditedAt,
	}
}

func (s *messageService) getUserFullName(ctx context.Context, userID uuid.UUID) string {
	type userSimple struct {
		FirstName string
		LastName  string
	}
	var u userSimple
	_ = s.db.WithContext(ctx).Table("users").Select("first_name, last_name").Where("id = ?", userID).Scan(&u)
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

func (s *messageService) getUserNames(ctx context.Context, userIDs []uuid.UUID) map[uuid.UUID]string {
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

func (s *messageService) getMessageReactions(ctx context.Context, msgIDs []uuid.UUID, currentUserID uuid.UUID) map[uuid.UUID][]dto.ReactionCountResponse {
	result := make(map[uuid.UUID][]dto.ReactionCountResponse)
	if len(msgIDs) == 0 {
		return result
	}

	var reactions []models.MessageReaction
	_ = s.db.WithContext(ctx).Where("message_id IN ?", msgIDs).Find(&reactions)

	type emojiGroup struct {
		users []uuid.UUID
		me    bool
	}
	msgEmojiMap := make(map[uuid.UUID]map[string]*emojiGroup)

	for _, rx := range reactions {
		if _, ok := msgEmojiMap[rx.MessageID]; !ok {
			msgEmojiMap[rx.MessageID] = make(map[string]*emojiGroup)
		}
		group, exists := msgEmojiMap[rx.MessageID][rx.Emoji]
		if !exists {
			group = &emojiGroup{users: make([]uuid.UUID, 0)}
			msgEmojiMap[rx.MessageID][rx.Emoji] = group
		}
		group.users = append(group.users, rx.UserID)
		if rx.UserID == currentUserID {
			group.me = true
		}
	}

	for msgID, emojiMap := range msgEmojiMap {
		list := make([]dto.ReactionCountResponse, 0, len(emojiMap))
		for emoji, g := range emojiMap {
			list = append(list, dto.ReactionCountResponse{
				Emoji: emoji,
				Count: len(g.users),
				Users: g.users,
				Me:    g.me,
			})
		}
		result[msgID] = list
	}

	return result
}

