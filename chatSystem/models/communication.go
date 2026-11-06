package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ChannelType represents the type of communication channel
type ChannelType string

const (
	ChannelTypeDM    ChannelType = "dm"
	ChannelTypeQuest ChannelType = "quest"
	ChannelTypeParty ChannelType = "party"
)

// MessageType represents the type of communication message
type MessageType string

const (
	MessageTypeText          MessageType = "text"
	MessageTypeFileShare     MessageType = "file_share"
	MessageTypeSystemMessage MessageType = "system_message"
)

// ChannelMemberRole represents the role of a user in a channel
type ChannelMemberRole string

const (
	MemberRoleLeader ChannelMemberRole = "leader"
	MemberRoleAdmin  ChannelMemberRole = "admin"
	MemberRoleMember ChannelMemberRole = "member"
	MemberRoleClient ChannelMemberRole = "client"
)

// ChatChannel stores channel metadata for DM, Quest, and Party chats
type ChatChannel struct {
	ID        string      `gorm:"type:varchar(100);primaryKey" json:"id"`
	Type      ChannelType `gorm:"type:varchar(50);not null;index" json:"type"`
	Name      string      `gorm:"type:varchar(255);not null" json:"name"`
	QuestID   *uuid.UUID  `gorm:"type:uuid;index" json:"quest_id,omitempty"`
	PartyID   *uuid.UUID  `gorm:"type:uuid;index" json:"party_id,omitempty"`
	CreatedBy *uuid.UUID  `gorm:"type:uuid;index" json:"created_by,omitempty"`
	CreatedAt time.Time   `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time   `gorm:"not null" json:"updated_at"`

	Members  []ChannelMember       `gorm:"foreignKey:ChannelID;references:ID;constraint:OnDelete:CASCADE" json:"members,omitempty"`
	Messages []CommunicationRecord `gorm:"foreignKey:ChannelID;references:ID;constraint:OnDelete:CASCADE" json:"-"`
}

func (c *ChatChannel) TableName() string {
	return "chat_channels"
}

func (c *ChatChannel) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	return nil
}

// CommunicationRecord represents a message stored in PostgreSQL
type CommunicationRecord struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	QuestID     *uuid.UUID     `gorm:"type:uuid;index" json:"quest_id,omitempty"`
	PartyID     *uuid.UUID     `gorm:"type:uuid;index" json:"party_id,omitempty"`
	FromUserID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"from_user_id"`
	ToUserID    *uuid.UUID     `gorm:"type:uuid;index" json:"to_user_id,omitempty"`
	ChannelID   string         `gorm:"type:varchar(100);not null;index" json:"channel_id"`
	ChannelType ChannelType    `gorm:"type:varchar(50);not null;index" json:"channel_type"`
	ReplyToID   *uuid.UUID     `gorm:"type:uuid;index" json:"reply_to_id,omitempty"`
	ContentRef  *string        `gorm:"type:text" json:"content_ref,omitempty"`
	ContentText string         `gorm:"type:text;not null" json:"content_text"`
	MessageType MessageType    `gorm:"type:varchar(50);not null;default:'text';index" json:"message_type"`
	Metadata    datatypes.JSON `gorm:"type:jsonb" json:"metadata,omitempty"`
	IsPinned    bool           `gorm:"type:boolean;not null;default:false;index" json:"is_pinned"`
	Deleted     bool           `gorm:"type:boolean;not null;default:false;index" json:"deleted"`
	CreatedAt   time.Time      `gorm:"type:timestamptz;not null;index" json:"created_at"`
	EditedAt    *time.Time     `gorm:"type:timestamptz" json:"edited_at,omitempty"`

	// Associations
	Reactions []MessageReaction    `gorm:"foreignKey:MessageID;references:ID;constraint:OnDelete:CASCADE" json:"reactions,omitempty"`
	ReplyTo   *CommunicationRecord `gorm:"foreignKey:ReplyToID;references:ID" json:"reply_to,omitempty"`
}

func (m *CommunicationRecord) TableName() string {
	return "communication_records"
}

func (m *CommunicationRecord) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	return nil
}

// MessageReaction stores reactions to chat messages
type MessageReaction struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	MessageID uuid.UUID `gorm:"type:uuid;not null;index:idx_message_user_emoji,unique;constraint:OnDelete:CASCADE" json:"message_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_message_user_emoji,unique" json:"user_id"`
	Emoji     string    `gorm:"type:varchar(64);not null;index:idx_message_user_emoji,unique" json:"emoji"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null" json:"created_at"`
}

func (r *MessageReaction) TableName() string {
	return "message_reactions"
}

func (r *MessageReaction) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	return nil
}

// ChannelMember stores user membership in a channel
type ChannelMember struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey" json:"id"`
	ChannelID string            `gorm:"type:varchar(100);not null;index:idx_channel_user,unique;index" json:"channel_id"`
	UserID    uuid.UUID         `gorm:"type:uuid;not null;index:idx_channel_user,unique;index" json:"user_id"`
	Role      ChannelMemberRole `gorm:"type:varchar(50);not null;default:'member'" json:"role"`
	Muted     bool              `gorm:"type:boolean;not null;default:false" json:"muted"`
	JoinedAt  time.Time         `gorm:"type:timestamptz;not null" json:"joined_at"`
}

func (cm *ChannelMember) TableName() string {
	return "channel_members"
}

func (cm *ChannelMember) BeforeCreate(tx *gorm.DB) error {
	if cm.ID == uuid.Nil {
		cm.ID = uuid.New()
	}
	if cm.JoinedAt.IsZero() {
		cm.JoinedAt = time.Now().UTC()
	}
	return nil
}

