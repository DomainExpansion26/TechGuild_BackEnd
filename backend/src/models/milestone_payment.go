package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MilestonePaymentStatus string

const (
	MilestonePaymentPending        MilestonePaymentStatus = "pending"
	MilestonePaymentFunded         MilestonePaymentStatus = "funded"
	MilestonePaymentReleasePending MilestonePaymentStatus = "release_pending"
	MilestonePaymentReleased       MilestonePaymentStatus = "released"
	MilestonePaymentRefundPending  MilestonePaymentStatus = "refund_pending"
	MilestonePaymentRefunded       MilestonePaymentStatus = "refunded"
	MilestonePaymentDisputed       MilestonePaymentStatus = "disputed"
	MilestonePaymentFailed         MilestonePaymentStatus = "failed"
)

type FeeType string

const (
	FeeTypePercentage FeeType = "percentage"
	FeeTypeFixed      FeeType = "fixed"
)

// MilestonePayment tracks the complete fee split for one quest milestone.
//
// Dual-sided fee model (Upwork/Fiverr style):
//
//	MilestoneAmount      = ₹10,000 (what freelancer is promised)
//	ClientFee (5%)       = ₹500    (added on top for client)
//	FreelancerFee (5%)   = ₹500    (deducted from freelancer)
//	ClientCharged        = ₹10,500 (client pays this)
//	FreelancerNet        = ₹9,500  (freelancer receives this)
//	PlatformTotal        = ₹1,000  (₹500 + ₹500)
//
// Invariants (enforced at DB level):
//
//	client_charged_minor  = milestone_amount_minor + client_fee_minor
//	freelancer_net_minor  = milestone_amount_minor - freelancer_fee_minor
//	platform_total_minor  = client_fee_minor + freelancer_fee_minor
//	client_charged_minor  = freelancer_net_minor + platform_total_minor
//
// Fee rates are LOCKED at funding time and NEVER re-read.
type MilestonePayment struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	QuestMilestoneID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	QuestID          uuid.UUID `gorm:"type:uuid;not null;index"`

	PaymentID           uuid.UUID  `gorm:"type:uuid;not null;index"`
	EscrowTransactionID *uuid.UUID `gorm:"type:uuid;index"`

	// Milestone amount (what freelancer is promised)
	MilestoneAmountMinor int64 `gorm:"not null"`

	// Client-side fee (added on top)
	ClientFeeMinor   int64 `gorm:"not null;default:0"`
	ClientFeeRateBps int64 `gorm:"not null;default:0"`

	// Freelancer-side fee (deducted)
	FreelancerFeeMinor   int64 `gorm:"not null;default:0"`
	FreelancerFeeRateBps int64 `gorm:"not null;default:0"`

	// Computed totals
	ClientChargedMinor int64 `gorm:"not null;default:0"` // = milestone + client_fee
	FreelancerNetMinor int64 `gorm:"not null;default:0"` // = milestone - freelancer_fee
	PlatformTotalMinor int64 `gorm:"not null;default:0"` // = client_fee + freelancer_fee

	Currency string `gorm:"type:varchar(10);not null;default:'INR'"`

	// Legacy fee fields (kept for compatibility with platform_fee table logic)
	FeeType FeeType `gorm:"type:varchar(20);not null;default:'percentage'"`

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
