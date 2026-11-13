package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"techguild-chat/dto"
	"techguild-chat/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrChannelNotFound   = errors.New("channel not found")
	ErrNotChannelMember  = errors.New("access denied: you are not a member of this channel")
	ErrInvalidRecipient  = errors.New("invalid recipient: cannot create DM with yourself")
	ErrRecipientNotFound = errors.New("recipient user not found")
)

type ChannelService interface {
	ListUserChannels(ctx context.Context, userID uuid.UUID) (*dto.ListChannelsResponse, error)
	CreateDMChannel(ctx context.Context, requesterID uuid.UUID, req dto.CreateDMChannelRequest) (*dto.ChannelDetailResponse, error)
	GetChannelInfo(ctx context.Context, userID uuid.UUID, channelID string) (*dto.ChannelInfoResponse, error)
	ListChannelMembers(ctx context.Context, userID uuid.UUID, channelID string) (*dto.ListChannelMembersResponse, error)
	MuteChannel(ctx context.Context, userID uuid.UUID, channelID string, muted bool) (*dto.MuteChannelResponse, error)

	// Auto-creation helper triggers
	AutoCreateQuestChannel(ctx context.Context, questID uuid.UUID, clientID uuid.UUID, partyID *uuid.UUID, freelancerID *uuid.UUID, questTitle string) (*models.ChatChannel, error)
	AutoCreatePartyChannel(ctx context.Context, partyID uuid.UUID, leaderID uuid.UUID, memberIDs []uuid.UUID, partyName string) (*models.ChatChannel, error)
}

type channelService struct {
	db                *gorm.DB
	centrifugoService CentrifugoService
}

func NewChannelService(db *gorm.DB, centrifugo CentrifugoService) ChannelService {
	return &channelService{
		db:                db,
		centrifugoService: centrifugo,
	}
}

// ListUserChannels lists all channels the user belongs to, grouped by type (quest, party, dm)
func (s *channelService) ListUserChannels(ctx context.Context, userID uuid.UUID) (*dto.ListChannelsResponse, error) {
	// Find all channel IDs where the user is a member
	var memberships []models.ChannelMember
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&memberships).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch memberships: %w", err)
	}

	if len(memberships) == 0 {
		return &dto.ListChannelsResponse{
			Quest: []dto.ChannelSummaryResponse{},
			Party: []dto.ChannelSummaryResponse{},
			DM:    []dto.ChannelSummaryResponse{},
		}, nil
	}

	channelIDMap := make(map[string]bool)
	mutedMap := make(map[string]bool)
	var channelIDs []string
	for _, m := range memberships {
		channelIDs = append(channelIDs, m.ChannelID)
		channelIDMap[m.ChannelID] = true
		mutedMap[m.ChannelID] = m.Muted
	}

	var channels []models.ChatChannel
	if err := s.db.WithContext(ctx).
		Where("id IN ?", channelIDs).
		Find(&channels).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch channels: %w", err)
	}

	// Fetch member counts per channel
	type countResult struct {
		ChannelID string
		Count     int
	}
	var counts []countResult
	s.db.WithContext(ctx).
		Table("channel_members").
		Select("channel_id, count(*) as count").
		Where("channel_id IN ?", channelIDs).
		Group("channel_id").
		Scan(&counts)

	countMap := make(map[string]int)
	for _, c := range counts {
		countMap[c.ChannelID] = c.Count
	}

	// Fetch last message per channel
	type lastMsgResult struct {
		ChannelID   string
		ContentText string
		CreatedAt   time.Time
	}
	var lastMessages []lastMsgResult
	s.db.WithContext(ctx).
		Raw(`
			SELECT DISTINCT ON (channel_id) channel_id, content_text, created_at
			FROM communication_records
			WHERE channel_id IN (?) AND deleted = false
			ORDER BY channel_id, created_at DESC
		`, channelIDs).
		Scan(&lastMessages)

	lastMsgMap := make(map[string]lastMsgResult)
	for _, lm := range lastMessages {
		lastMsgMap[lm.ChannelID] = lm
	}

	res := &dto.ListChannelsResponse{
		Quest: make([]dto.ChannelSummaryResponse, 0),
		Party: make([]dto.ChannelSummaryResponse, 0),
		DM:    make([]dto.ChannelSummaryResponse, 0),
	}

	for _, ch := range channels {
		summary := dto.ChannelSummaryResponse{
			ID:          ch.ID,
			Name:        ch.Name,
			Type:        string(ch.Type),
			QuestID:     ch.QuestID,
			PartyID:     ch.PartyID,
			MemberCount: countMap[ch.ID],
			IsMuted:     mutedMap[ch.ID],
			CreatedAt:   ch.CreatedAt,
			LastActive:  ch.UpdatedAt,
		}

		if lm, ok := lastMsgMap[ch.ID]; ok {
			summary.LastMessage = &lm.ContentText
			summary.LastActive = lm.CreatedAt
		}

		switch ch.Type {
		case models.ChannelTypeQuest:
			res.Quest = append(res.Quest, summary)
		case models.ChannelTypeParty:
			res.Party = append(res.Party, summary)
		case models.ChannelTypeDM:
			res.DM = append(res.DM, summary)
		default:
			res.DM = append(res.DM, summary)
		}
	}

	return res, nil
}

// CreateDMChannel creates a 1:1 direct message channel (or returns existing one)
func (s *channelService) CreateDMChannel(ctx context.Context, requesterID uuid.UUID, req dto.CreateDMChannelRequest) (*dto.ChannelDetailResponse, error) {
	recipientUUID, err := uuid.Parse(req.RecipientID)
	if err != nil {
		return nil, errors.New("invalid recipient_id format")
	}

	if requesterID == recipientUUID {
		return nil, ErrInvalidRecipient
	}

	// Verify recipient exists
	var count int64
	s.db.WithContext(ctx).Table("users").Where("id = ?", recipientUUID).Count(&count)
	if count == 0 {
		return nil, ErrRecipientNotFound
	}

	// Check if a DM channel already exists between these two users
	var existingChannelID string
	err = s.db.WithContext(ctx).Raw(`
		SELECT cm1.channel_id
		FROM channel_members cm1
		JOIN channel_members cm2 ON cm1.channel_id = cm2.channel_id
		JOIN chat_channels cc ON cc.id = cm1.channel_id
		WHERE cc.type = 'dm'
		  AND cm1.user_id = ?
		  AND cm2.user_id = ?
		LIMIT 1
	`, requesterID, recipientUUID).Scan(&existingChannelID).Error

	if err == nil && existingChannelID != "" {
		var existing models.ChatChannel
		if err := s.db.WithContext(ctx).Where("id = ?", existingChannelID).First(&existing).Error; err == nil {
			return &dto.ChannelDetailResponse{
				ID:        existing.ID,
				Name:      existing.Name,
				Type:      string(existing.Type),
				CreatedBy: existing.CreatedBy,
				CreatedAt: existing.CreatedAt,
				UpdatedAt: existing.UpdatedAt,
			}, nil
		}
	}

	// Fetch recipient name to label the channel
	type userSimple struct {
		FirstName string
		LastName  string
	}
	var u userSimple
	_ = s.db.WithContext(ctx).Table("users").Select("first_name, last_name").Where("id = ?", recipientUUID).Scan(&u)
	channelName := fmt.Sprintf("DM: %s %s", u.FirstName, u.LastName)

	channelID := "dm-" + uuid.New().String()
	now := time.Now().UTC()

	newChannel := models.ChatChannel{
		ID:        channelID,
		Type:      models.ChannelTypeDM,
		Name:      channelName,
		CreatedBy: &requesterID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&newChannel).Error; err != nil {
			return err
		}

		members := []models.ChannelMember{
			{
				ID:        uuid.New(),
				ChannelID: channelID,
				UserID:    requesterID,
				Role:      models.MemberRoleMember,
				Muted:     false,
				JoinedAt:  now,
			},
			{
				ID:        uuid.New(),
				ChannelID: channelID,
				UserID:    recipientUUID,
				Role:      models.MemberRoleMember,
				Muted:     false,
				JoinedAt:  now,
			},
		}

		return tx.Create(&members).Error
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create DM channel: %w", err)
	}

	return &dto.ChannelDetailResponse{
		ID:        newChannel.ID,
		Name:      newChannel.Name,
		Type:      string(newChannel.Type),
		CreatedBy: newChannel.CreatedBy,
		CreatedAt: newChannel.CreatedAt,
		UpdatedAt: newChannel.UpdatedAt,
	}, nil
}

// GetChannelInfo retrieves channel metadata: budget, duration, milestone count, member count
func (s *channelService) GetChannelInfo(ctx context.Context, userID uuid.UUID, channelID string) (*dto.ChannelInfoResponse, error) {
	// Validate requester is a member of the channel
	var membership models.ChannelMember
	if err := s.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		First(&membership).Error; err != nil {
		return nil, ErrNotChannelMember
	}

	var ch models.ChatChannel
	if err := s.db.WithContext(ctx).Where("id = ?", channelID).First(&ch).Error; err != nil {
		return nil, ErrChannelNotFound
	}

	var memberCount int64
	s.db.WithContext(ctx).Table("channel_members").Where("channel_id = ?", channelID).Count(&memberCount)

	resp := &dto.ChannelInfoResponse{
		ID:          ch.ID,
		Name:        ch.Name,
		Type:        string(ch.Type),
		QuestID:     ch.QuestID,
		PartyID:     ch.PartyID,
		MemberCount: int(memberCount),
		IsMuted:     membership.Muted,
		CreatedAt:   ch.CreatedAt,
	}

	// If this channel is associated with a quest, enrich with budget, duration, and milestones
	if ch.QuestID != nil {
		type questInfo struct {
			BudgetMinor  int64
			TimelineDays int
		}
		var q questInfo
		if err := s.db.WithContext(ctx).
			Table("quests").
			Select("budget_minor, timeline_days").
			Where("id = ?", *ch.QuestID).
			Scan(&q).Error; err == nil {
			resp.BudgetMinor = &q.BudgetMinor
			formatted := fmt.Sprintf("₹%.2f", float64(q.BudgetMinor)/100.0)
			resp.BudgetFormatted = &formatted
			resp.DurationDays = &q.TimelineDays
		}

		var milestoneCount int64
		s.db.WithContext(ctx).
			Table("quest_milestones").
			Where("quest_id = ?", *ch.QuestID).
			Count(&milestoneCount)
		mCount := int(milestoneCount)
		resp.MilestoneCount = &mCount
	}

	return resp, nil
}

// ListChannelMembers lists all channel members with their roles and basic profile
func (s *channelService) ListChannelMembers(ctx context.Context, userID uuid.UUID, channelID string) (*dto.ListChannelMembersResponse, error) {
	// Validate requester is a member of the channel
	var userMember models.ChannelMember
	if err := s.db.WithContext(ctx).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		First(&userMember).Error; err != nil {
		return nil, ErrNotChannelMember
	}

	type memberRow struct {
		UserID    uuid.UUID
		FirstName string
		LastName  string
		Email     string
		Role      string
		Muted     bool
		JoinedAt  time.Time
	}

	var rows []memberRow
	err := s.db.WithContext(ctx).
		Table("channel_members cm").
		Select("cm.user_id, u.first_name, u.last_name, u.email, cm.role, cm.muted, cm.joined_at").
		Joins("LEFT JOIN users u ON u.id = cm.user_id").
		Where("cm.channel_id = ?", channelID).
		Scan(&rows).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch channel members: %w", err)
	}

	members := make([]dto.ChannelMemberResponse, len(rows))
	for i, r := range rows {
		members[i] = dto.ChannelMemberResponse{
			UserID:    r.UserID,
			FirstName: r.FirstName,
			LastName:  r.LastName,
			Email:     r.Email,
			Role:      r.Role,
			Muted:     r.Muted,
			JoinedAt:  r.JoinedAt,
		}
	}

	return &dto.ListChannelMembersResponse{
		ChannelID: channelID,
		Total:     len(members),
		Members:   members,
	}, nil
}

// MuteChannel mutes or unmutes channel notifications for the requesting user
func (s *channelService) MuteChannel(ctx context.Context, userID uuid.UUID, channelID string, muted bool) (*dto.MuteChannelResponse, error) {
	result := s.db.WithContext(ctx).
		Model(&models.ChannelMember{}).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		Update("muted", muted)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to update mute status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, ErrNotChannelMember
	}

	msg := "Channel notifications unmuted"
	if muted {
		msg = "Channel notifications muted"
	}

	return &dto.MuteChannelResponse{
		ChannelID: channelID,
		Muted:     muted,
		Message:   msg,
	}, nil
}

// AutoCreateQuestChannel is invoked when a quest is accepted (Task P2-3.4 / Acceptance Criteria)
func (s *channelService) AutoCreateQuestChannel(ctx context.Context, questID uuid.UUID, clientID uuid.UUID, partyID *uuid.UUID, freelancerID *uuid.UUID, questTitle string) (*models.ChatChannel, error) {
	channelID := fmt.Sprintf("quest-%s", questID.String())
	now := time.Now().UTC()

	var existing models.ChatChannel
	if err := s.db.WithContext(ctx).Where("id = ?", channelID).First(&existing).Error; err == nil {
		return &existing, nil
	}

	channel := models.ChatChannel{
		ID:        channelID,
		Type:      models.ChannelTypeQuest,
		Name:      fmt.Sprintf("Quest: %s", questTitle),
		QuestID:   &questID,
		PartyID:   partyID,
		CreatedBy: &clientID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&channel).Error; err != nil {
			return err
		}

		members := []models.ChannelMember{
			{
				ID:        uuid.New(),
				ChannelID: channelID,
				UserID:    clientID,
				Role:      models.MemberRoleClient,
				Muted:     false,
				JoinedAt:  now,
			},
		}

		// If accepted with a party, add all active party members
		if partyID != nil {
			type partyMemberRow struct {
				UserID uuid.UUID
				Role   string
			}
			var partyMembers []partyMemberRow
			tx.Table("party_members").
				Select("user_id, role").
				Where("party_id = ? AND status = 'active'", *partyID).
				Scan(&partyMembers)

			for _, pm := range partyMembers {
				role := models.MemberRoleMember
				if pm.Role == "leader" {
					role = models.MemberRoleLeader
				}
				members = append(members, models.ChannelMember{
					ID:        uuid.New(),
					ChannelID: channelID,
					UserID:    pm.UserID,
					Role:      role,
					Muted:     false,
					JoinedAt:  now,
				})
			}
		} else if freelancerID != nil {
			// Single freelancer accepted
			members = append(members, models.ChannelMember{
				ID:        uuid.New(),
				ChannelID: channelID,
				UserID:    *freelancerID,
				Role:      models.MemberRoleMember,
				Muted:     false,
				JoinedAt:  now,
			})
		}

		if err := tx.Create(&members).Error; err != nil {
			return err
		}

		// Create initial system message
		sysMsg := models.CommunicationRecord{
			ID:          uuid.New(),
			QuestID:     &questID,
			PartyID:     partyID,
			FromUserID:  clientID,
			ChannelID:   channelID,
			ChannelType: models.ChannelTypeQuest,
			ContentText: fmt.Sprintf("Quest channel initialized for '%s'. Welcome!", questTitle),
			MessageType: models.MessageTypeSystemMessage,
			CreatedAt:   now,
		}

		return tx.Create(&sysMsg).Error
	})

	if err != nil {
		log.Printf("[ChannelService] Error auto-creating quest channel: %v", err)
		return nil, err
	}

	return &channel, nil
}

// AutoCreatePartyChannel is invoked when a party reaches required formation threshold
func (s *channelService) AutoCreatePartyChannel(ctx context.Context, partyID uuid.UUID, leaderID uuid.UUID, memberIDs []uuid.UUID, partyName string) (*models.ChatChannel, error) {
	channelID := fmt.Sprintf("party-%s", partyID.String())
	now := time.Now().UTC()

	var existing models.ChatChannel
	if err := s.db.WithContext(ctx).Where("id = ?", channelID).First(&existing).Error; err == nil {
		return &existing, nil
	}

	channel := models.ChatChannel{
		ID:        channelID,
		Type:      models.ChannelTypeParty,
		Name:      fmt.Sprintf("Party: %s", partyName),
		PartyID:   &partyID,
		CreatedBy: &leaderID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&channel).Error; err != nil {
			return err
		}

		members := []models.ChannelMember{
			{
				ID:        uuid.New(),
				ChannelID: channelID,
				UserID:    leaderID,
				Role:      models.MemberRoleLeader,
				Muted:     false,
				JoinedAt:  now,
			},
		}

		for _, uid := range memberIDs {
			if uid != leaderID {
				members = append(members, models.ChannelMember{
					ID:        uuid.New(),
					ChannelID: channelID,
					UserID:    uid,
					Role:      models.MemberRoleMember,
					Muted:     false,
					JoinedAt:  now,
				})
			}
		}

		return tx.Create(&members).Error
	})

	if err != nil {
		log.Printf("[ChannelService] Error auto-creating party channel: %v", err)
		return nil, err
	}

	return &channel, nil
}

