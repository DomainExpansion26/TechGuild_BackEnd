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

type PlatformFeeService struct{}

func NewPlatformFeeService() *PlatformFeeService {
	return &PlatformFeeService{}
}

// CalculateFee returns (platformFee, freelancerAmount, error).
//
// Rules:
//   - Percentage fee: fee = (gross * rateBps) / 10000, half-up rounding.
//   - Fixed fee: fee = fixedAmount (must be <= gross).
//   - freelancerAmount = gross - fee
//
// Overflow-safe for int64.
func (s *PlatformFeeService) CalculateFee(
	grossAmount int64,
	feeType models.FeeType,
	rateBps int64,
	fixedAmount int64,
) (int64, int64, error) {
	if grossAmount <= 0 {
		return 0, 0, ErrFeeCalculationInvalid
	}

	var fee int64
	switch feeType {
	case models.FeeTypePercentage:
		if rateBps < 0 || rateBps > basisPointsDenominator {
			return 0, 0, ErrFeeCalculationInvalid
		}
		calculated, err := percentageHalfUp(grossAmount, rateBps)
		if err != nil {
			return 0, 0, err
		}
		fee = calculated
	case models.FeeTypeFixed:
		if fixedAmount < 0 || fixedAmount > grossAmount {
			return 0, 0, ErrFeeCalculationInvalid
		}
		fee = fixedAmount
	default:
		return 0, 0, ErrFeeCalculationInvalid
	}

	freelancerAmount := grossAmount - fee
	if freelancerAmount < 0 {
		return 0, 0, ErrFeeCalculationInvalid
	}
	return fee, freelancerAmount, nil
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

// RecordFee creates a PlatformFee row (status=pending). Called inside the
// funding TX after MilestonePayment is created.
func (s *PlatformFeeService) RecordFee(tx *gorm.DB, fee *models.PlatformFee) error {
	return tx.Create(fee).Error
}

// SettleFee marks a platform fee as settled. Called inside the release TX.
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
