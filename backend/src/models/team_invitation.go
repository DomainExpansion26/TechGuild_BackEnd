package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "pending"
	InvitationAccepted InvitationStatus = "accepted"
	InvitationRejected InvitationStatus = "rejected"
	InvitationExpired  InvitationStatus = "expired"
)

type TeamInvitation struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Team
	TeamID uuid.UUID `gorm:"type:uuid;not null;index"`
	Team   Team      `gorm:"foreignKey:TeamID;constraint:OnDelete:CASCADE"`

	// Sender
	InvitedByID uuid.UUID `gorm:"type:uuid;not null;index"`
	InvitedBy   User      `gorm:"foreignKey:InvitedByID;constraint:OnDelete:CASCADE"`

	// Receiver
	InvitedUserID uuid.UUID `gorm:"type:uuid;not null;index"`
	InvitedUser   User      `gorm:"foreignKey:InvitedUserID;constraint:OnDelete:CASCADE"`

	// Invitation Details
	Message string `gorm:"type:text"`

	Status InvitationStatus `gorm:"type:varchar(30);default:'pending'"`

	ExpiresAt time.Time

	RespondedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (i *TeamInvitation) BeforeCreate(tx *gorm.DB) error {

	i.ID = uuid.New()

	if i.ExpiresAt.IsZero() {
		i.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
	}

	return nil
}