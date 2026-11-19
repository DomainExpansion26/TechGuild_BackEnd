package dto

import (
	"time"

	"github.com/google/uuid"
)

// ChannelSummaryResponse is the channel summary returned in list views
type ChannelSummaryResponse struct {
	ID          string     `json:"id" example:"quest-123e4567-e89b-12d3-a456-426614174000"`
	Name        string     `json:"name" example:"Website Redesign"`
	Type        string     `json:"type" example:"quest"`
	QuestID     *uuid.UUID `json:"quest_id,omitempty"`
	PartyID     *uuid.UUID `json:"party_id,omitempty"`
	MemberCount int        `json:"member_count" example:"4"`
	IsMuted     bool       `json:"is_muted" example:"false"`
	LastMessage *string    `json:"last_message,omitempty" example:"Looking forward to starting!"`
	LastActive  time.Time  `json:"last_active"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ListChannelsResponse groups channels by type (quest, party, dm)
type ListChannelsResponse struct {
	Quest []ChannelSummaryResponse `json:"quest"`
	Party []ChannelSummaryResponse `json:"party"`
	DM    []ChannelSummaryResponse `json:"dm"`
}

type ListChannelsInput struct{}

type ListChannelsOutput struct {
	Body ListChannelsResponse
}

// CreateDMChannelRequest input for creating 1:1 direct message channel
type CreateDMChannelRequest struct {
	RecipientID string `json:"recipient_id" doc:"User ID of the recipient for 1:1 chat" required:"true" example:"123e4567-e89b-12d3-a456-426614174000"`
}

type CreateDMChannelInput struct {
	Body CreateDMChannelRequest
}

// ChannelDetailResponse is returned upon channel creation or detail fetch
type ChannelDetailResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	QuestID   *uuid.UUID `json:"quest_id,omitempty"`
	PartyID   *uuid.UUID `json:"party_id,omitempty"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CreateChannelOutput struct {
	Body ChannelDetailResponse
}

// ChannelInfoResponse returns extended metadata (budget, duration, milestone count, member count)
type ChannelInfoResponse struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	QuestID         *uuid.UUID `json:"quest_id,omitempty"`
	PartyID         *uuid.UUID `json:"party_id,omitempty"`
	BudgetMinor     *int64     `json:"budget_minor,omitempty" example:"7500000" doc:"Budget in paise for quest channels"`
	BudgetFormatted *string    `json:"budget_formatted,omitempty" example:"₹75,000.00"`
	DurationDays    *int       `json:"duration_days,omitempty" example:"20" doc:"Timeline in days for quest channels"`
	MilestoneCount  *int       `json:"milestone_count,omitempty" example:"3" doc:"Total milestone count for quest channels"`
	MemberCount     int        `json:"member_count" example:"5"`
	IsMuted         bool       `json:"is_muted"`
	CreatedAt       time.Time  `json:"created_at"`
}

type GetChannelInfoInput struct {
	ID string `path:"id" doc:"Channel ID" required:"true"`
}

type GetChannelInfoOutput struct {
	Body ChannelInfoResponse
}

// ChannelMemberResponse represents a member of a channel
type ChannelMemberResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Role      string    `json:"role" example:"leader"`
	Muted     bool      `json:"muted"`
	JoinedAt  time.Time `json:"joined_at"`
}

type ListChannelMembersInput struct {
	ID string `path:"id" doc:"Channel ID" required:"true"`
}

type ListChannelMembersResponse struct {
	ChannelID string                  `json:"channel_id"`
	Total     int                     `json:"total"`
	Members   []ChannelMemberResponse `json:"members"`
}

type ListChannelMembersOutput struct {
	Body ListChannelMembersResponse
}

// MuteChannelRequest body for muting/unmuting notifications
type MuteChannelRequest struct {
	Muted bool `json:"muted" doc:"True to mute channel notifications, false to unmute" example:"true"`
}

type MuteChannelInput struct {
	ID   string             `path:"id" doc:"Channel ID" required:"true"`
	Body MuteChannelRequest `required:"true"`
}

type MuteChannelResponse struct {
	ChannelID string `json:"channel_id"`
	Muted     bool   `json:"muted"`
	Message   string `json:"message"`
}

type MuteChannelOutput struct {
	Body MuteChannelResponse
}

