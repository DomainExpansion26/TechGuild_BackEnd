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
	ErrEmptySearchQuery = errors.New("search query 'q' cannot be empty")
)

type SearchService interface {
	SearchMessages(ctx context.Context, userID uuid.UUID, query string, channelID string, limit, offset int) (*dto.SearchMessagesResponse, error)
}

type searchService struct {
	db *gorm.DB
}

func NewSearchService(db *gorm.DB) SearchService {
	return &searchService{db: db}
}

// SearchMessages performs keyword search strictly scoped to channels the requester is a member of
func (s *searchService) SearchMessages(ctx context.Context, userID uuid.UUID, query string, channelID string, limit, offset int) (*dto.SearchMessagesResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrEmptySearchQuery
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// 1. Fetch channel IDs the user is a member of
	var allowedChannelIDs []string
	if err := s.db.WithContext(ctx).
		Table("channel_members").
		Where("user_id = ?", userID).
		Pluck("channel_id", &allowedChannelIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch user memberships: %w", err)
	}

	if len(allowedChannelIDs) == 0 {
		return &dto.SearchMessagesResponse{
			Query:    query,
			Total:    0,
			Messages: []dto.MessageSearchResult{},
		}, nil
	}

	// If channelID is specified, check if user is a member of that specific channel
	if channelID != "" {
		isMember := false
		for _, cid := range allowedChannelIDs {
			if cid == channelID {
				isMember = true
				break
			}
		}
		if !isMember {
			return nil, ErrNotChannelMember
		}
		allowedChannelIDs = []string{channelID}
	}

	// 2. Query messages matching keyword (case-insensitive ILIKE)
	dbQuery := s.db.WithContext(ctx).
		Table("communication_records cr").
		Joins("JOIN chat_channels cc ON cc.id = cr.channel_id").
		Where("cr.channel_id IN ? AND cr.deleted = false", allowedChannelIDs).
		Where("cr.content_text ILIKE ?", "%"+query+"%")

	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count search results: %w", err)
	}

	type searchRow struct {
		models.CommunicationRecord
		ChannelName string
	}

	var rows []searchRow
	err := dbQuery.
		Select("cr.*, cc.name as channel_name").
		Order("cr.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch search results: %w", err)
	}

	// Fetch sender names
	userIDs := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		userIDs = append(userIDs, r.FromUserID)
	}
	nameMap := s.getUserNames(ctx, userIDs)

	results := make([]dto.MessageSearchResult, len(rows))
	for i, r := range rows {
		results[i] = dto.MessageSearchResult{
			ChannelName: r.ChannelName,
			MessageResponse: dto.MessageResponse{
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
			},
		}
	}

	var cidPtr *string
	if channelID != "" {
		cidPtr = &channelID
	}

	return &dto.SearchMessagesResponse{
		Query:     query,
		Total:     int(total),
		ChannelID: cidPtr,
		Messages:  results,
	}, nil
}

func (s *searchService) getUserNames(ctx context.Context, userIDs []uuid.UUID) map[uuid.UUID]string {
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

