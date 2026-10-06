package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContractStatus string

const (
	ContractPending   ContractStatus = "pending"
	ContractActive    ContractStatus = "active"
	ContractCompleted ContractStatus = "completed"
	ContractCancelled ContractStatus = "cancelled"
)

type ProjectContract struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// Relationships
	ProjectID uuid.UUID `gorm:"type:uuid;not null;index"`
	Project   Project   `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE"`

	ApplicationID uuid.UUID          `gorm:"type:uuid;not null;uniqueIndex"`
	Application   ProjectApplication `gorm:"foreignKey:ApplicationID;constraint:OnDelete:CASCADE"`

	ClientID uuid.UUID `gorm:"type:uuid;not null;index"`
	Client   User      `gorm:"foreignKey:ClientID"`

	FreelancerID uuid.UUID `gorm:"type:uuid;not null;index"`
	Freelancer   User      `gorm:"foreignKey:FreelancerID"`

	// Contract Details
	// ContractAmount — int64 minor units of Currency.
	// Example: INR 50000.00 => 5000000 (paise).
	ContractAmount int64  `gorm:"not null;default:0"`
	Currency       string `gorm:"size:10;default:'INR'"`

	StartDate       *time.Time
	ExpectedEndDate *time.Time

	Status ContractStatus `gorm:"type:varchar(30);default:'pending'"`

	SignedByClient     bool `gorm:"default:false"`
	SignedByFreelancer bool `gorm:"default:false"`

	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time

	Milestones []ProjectMilestone `gorm:"foreignKey:ContractID;constraint:OnDelete:CASCADE"`
}

func (c *ProjectContract) BeforeCreate(tx *gorm.DB) error {
	c.ID = uuid.New()
	return nil
}
