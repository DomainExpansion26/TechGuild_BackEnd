package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApplicationStatus string

const (
	ApplicationPending   ApplicationStatus = "pending"
	ApplicationAccepted  ApplicationStatus = "accepted"
	ApplicationRejected  ApplicationStatus = "rejected"
	ApplicationWithdrawn ApplicationStatus = "withdrawn"
)

// QuestApplication represents a bid/application by a freelancer OR an agency (party).
type QuestApplication struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Unique: one application per user per quest
	QuestID      uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_quest_freelancer" json:"quest_id"`
	FreelancerID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_quest_freelancer" json:"freelancer_id"`

	// Optional: set if applying as agency. Nil = individual freelancer.
	PartyID *uuid.UUID `gorm:"type:uuid;index" json:"party_id,omitempty"`

	// Relations
	Quest      Quest  `gorm:"foreignKey:QuestID;constraint:OnDelete:CASCADE" json:"-"`
	Freelancer User   `gorm:"foreignKey:FreelancerID;constraint:OnDelete:CASCADE" json:"-"`
	Party      *Party `gorm:"foreignKey:PartyID" json:"-"`

	// Bid details
	CoverLetter       string `gorm:"type:text;not null" json:"cover_letter"`
	ProposedBudget    int64  `gorm:"not null" json:"proposed_budget"` // paise
	Currency          string `gorm:"type:varchar(10);default:'INR'" json:"currency"`
	EstimatedDuration string `gorm:"type:varchar(100)" json:"estimated_duration"`

	Status ApplicationStatus `gorm:"type:varchar(30);default:'pending';index" json:"status"`

	AcceptedAt   *time.Time `json:"accepted_at,omitempty"`
	RejectedAt   *time.Time `json:"rejected_at,omitempty"`
	WithdrawnAt  *time.Time `json:"withdrawn_at,omitempty"`
	RejectReason string     `gorm:"type:text" json:"reject_reason,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (a *QuestApplication) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
