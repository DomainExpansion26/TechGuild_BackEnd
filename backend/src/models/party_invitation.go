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

type PartyInvitation struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Party
	PartyID uuid.UUID `gorm:"type:uuid;not null;index"`
	Party   Party      `gorm:"foreignKey:PartyID;constraint:OnDelete:CASCADE"`

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

func (i *PartyInvitation) BeforeCreate(tx *gorm.DB) error {

	i.ID = uuid.New()

	if i.ExpiresAt.IsZero() {
		i.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
	}

	return nil
}