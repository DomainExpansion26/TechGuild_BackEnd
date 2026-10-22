package services

import (
	"errors"

	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrFeeCalculationInvalid = errors.New("invalid fee calculation input")
	ErrFeeOverflow           = errors.New("fee calculation overflow")
)

const basisPointsDenominator int64 = 10000

// FeeSplit represents the complete breakdown for one milestone payment.
type FeeSplit struct {
	MilestoneAmountMinor int64

	ClientFeeMinor   int64
	ClientFeeRateBps int64

	FreelancerFeeMinor   int64
	FreelancerFeeRateBps int64

	ClientChargedMinor int64 // milestone + client_fee
	FreelancerNetMinor int64 // milestone - freelancer_fee
	PlatformTotalMinor int64 // client_fee + freelancer_fee
}

type PlatformFeeService struct{}

func NewPlatformFeeService() *PlatformFeeService {
	return &PlatformFeeService{}
}

// CalculateDualFee computes the full dual-sided fee split.
//
// Given milestone ₹10,000, client 5%, freelancer 5%:
//
//	ClientFee       = ₹500
//	ClientCharged   = ₹10,500
//	FreelancerFee   = ₹500
//	FreelancerNet   = ₹9,500
//	PlatformTotal   = ₹1,000
//
// Overflow-safe. Half-up rounding on each fee.
func (s *PlatformFeeService) CalculateDualFee(
	milestoneAmountMinor int64,
	clientFeeRateBps int64,
	freelancerFeeRateBps int64,
) (*FeeSplit, error) {
	if milestoneAmountMinor <= 0 {
		return nil, ErrFeeCalculationInvalid
	}
	if clientFeeRateBps < 0 || clientFeeRateBps > basisPointsDenominator {
		return nil, ErrFeeCalculationInvalid
	}
	if freelancerFeeRateBps < 0 || freelancerFeeRateBps > basisPointsDenominator {
		return nil, ErrFeeCalculationInvalid
	}

	clientFee, err := percentageHalfUp(milestoneAmountMinor, clientFeeRateBps)
	if err != nil {
		return nil, err
	}
	freelancerFee, err := percentageHalfUp(milestoneAmountMinor, freelancerFeeRateBps)
	if err != nil {
		return nil, err
	}

	clientCharged := milestoneAmountMinor + clientFee
	freelancerNet := milestoneAmountMinor - freelancerFee
	platformTotal := clientFee + freelancerFee

	if freelancerNet < 0 {
		return nil, ErrFeeCalculationInvalid
	}

	return &FeeSplit{
		MilestoneAmountMinor: milestoneAmountMinor,
		ClientFeeMinor:       clientFee,
		ClientFeeRateBps:     clientFeeRateBps,
		FreelancerFeeMinor:   freelancerFee,
		FreelancerFeeRateBps: freelancerFeeRateBps,
		ClientChargedMinor:   clientCharged,
		FreelancerNetMinor:   freelancerNet,
		PlatformTotalMinor:   platformTotal,
	}, nil
}

// percentageHalfUp computes round-half-up of (amount * rateBps) / 10000.
func percentageHalfUp(amount, rateBps int64) (int64, error) {
	if amount == 0 || rateBps == 0 {
		return 0, nil
	}
	const maxInt64 = int64(^uint64(0) >> 1)
	if amount > maxInt64/rateBps {
		return 0, ErrFeeOverflow
	}
	product := amount * rateBps
	quotient := product / basisPointsDenominator
	remainder := product % basisPointsDenominator
	if remainder >= basisPointsDenominator/2 {
		quotient++
	}
	return quotient, nil
}

// RecordFee creates a PlatformFee row. Two rows per milestone (client + freelancer).
func (s *PlatformFeeService) RecordFee(tx *gorm.DB, fee *models.PlatformFee) error {
	return tx.Create(fee).Error
}

// SettleFee marks ALL platform fee rows for a milestone_payment as settled.
// (Dual-fee model = 2 rows per milestone_payment.)
func (s *PlatformFeeService) SettleFee(tx *gorm.DB, milestonePaymentID uuid.UUID) error {
	res := tx.
		Model(&models.PlatformFee{}).
		Where("milestone_payment_id = ?", milestonePaymentID).
		Updates(map[string]interface{}{
			"status":     models.PlatformFeeSettled,
			"settled_at": gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
