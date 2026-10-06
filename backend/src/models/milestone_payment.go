package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MilestonePaymentStatus string

const (
	MilestonePaymentPending          MilestonePaymentStatus = "pending"
	MilestonePaymentFunded           MilestonePaymentStatus = "funded"
	MilestonePaymentAwaitingApproval MilestonePaymentStatus = "awaiting_approval"
	MilestonePaymentApproved         MilestonePaymentStatus = "approved"
	MilestonePaymentReleasePending   MilestonePaymentStatus = "release_pending"
	MilestonePaymentReleased         MilestonePaymentStatus = "released"
	MilestonePaymentRefundPending    MilestonePaymentStatus = "refund_pending"
	MilestonePaymentRefunded         MilestonePaymentStatus = "refunded"
	MilestonePaymentDisputed         MilestonePaymentStatus = "disputed"
	MilestonePaymentFailed           MilestonePaymentStatus = "failed"
)

type FeeType string

const (
	FeeTypePercentage FeeType = "percentage"
	FeeTypeFixed      FeeType = "fixed"
)

// MilestonePayment records the gross/fee/freelancer split for one milestone.
//
// Invariant (enforced at DB level + service level):
//
//	GrossAmount = PlatformFee + FreelancerAmount
//
// FeeRateBps is LOCKED at funding time and must NOT be re-read at approval.
type MilestonePayment struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	MilestoneID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	ContractID  uuid.UUID `gorm:"type:uuid;not null;index"`
	ProjectID   uuid.UUID `gorm:"type:uuid;not null;index"`

	PaymentID           uuid.UUID  `gorm:"type:uuid;not null;index"`
	EscrowTransactionID *uuid.UUID `gorm:"type:uuid;index"`

	// All amounts in minor units of Currency
	GrossAmount      int64  `gorm:"not null"`
	PlatformFee      int64  `gorm:"not null;default:0"`
	FreelancerAmount int64  `gorm:"not null;default:0"`
	Currency         string `gorm:"type:varchar(10);not null;default:'INR'"`

	// Fee config LOCKED at funding time
	FeeRateBps int64   `gorm:"not null;default:0"` // basis points, e.g. 250 = 2.50%
	FeeType    FeeType `gorm:"type:varchar(20);not null;default:'percentage'"`

	Status MilestonePaymentStatus `gorm:"type:varchar(30);not null;default:'pending';index"`

	ApprovedAt *time.Time
	ReleasedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (mp *MilestonePayment) BeforeCreate(tx *gorm.DB) error {
	if mp.ID == uuid.Nil {
		mp.ID = uuid.New()
	}
	return nil
}
