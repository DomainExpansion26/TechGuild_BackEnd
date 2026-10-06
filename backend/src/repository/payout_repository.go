package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PayoutRepository interface {
	Create(payout *models.Payout) error
	GetByID(id uuid.UUID) (*models.Payout, error)
	GetByProviderPayoutID(provider, providerPayoutID string) (*models.Payout, error)
	GetByIdempotencyKey(key string) (*models.Payout, error)
	GetByMilestonePaymentID(mpID uuid.UUID) (*models.Payout, error)
	UpdateStatus(id uuid.UUID, status models.PayoutStatus) error
	MarkPaid(id uuid.UUID) error
	MarkFailed(id uuid.UUID, reason string) error
	ListByUser(userID uuid.UUID) ([]models.Payout, error)

	WithTransaction(fn func(txRepo PayoutRepository) error) error
}

type payoutRepository struct {
	db *gorm.DB
}

func NewPayoutRepository() PayoutRepository {
	return &payoutRepository{db: postgres.DB}
}

func NewPayoutRepositoryTx(tx *gorm.DB) PayoutRepository {
	return &payoutRepository{db: tx}
}

func (r *payoutRepository) WithTransaction(fn func(txRepo PayoutRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewPayoutRepositoryTx(tx))
	})
}

func (r *payoutRepository) Create(payout *models.Payout) error {
	return r.db.Create(payout).Error
}

func (r *payoutRepository) GetByID(id uuid.UUID) (*models.Payout, error) {
	var p models.Payout
	err := r.db.Where("id = ?", id).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *payoutRepository) GetByProviderPayoutID(provider, providerPayoutID string) (*models.Payout, error) {
	var p models.Payout
	err := r.db.
		Where("provider = ? AND provider_payout_id = ?", provider, providerPayoutID).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *payoutRepository) GetByIdempotencyKey(key string) (*models.Payout, error) {
	var p models.Payout
	err := r.db.Where("idempotency_key = ?", key).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *payoutRepository) GetByMilestonePaymentID(mpID uuid.UUID) (*models.Payout, error) {
	var p models.Payout
	err := r.db.Where("milestone_payment_id = ?", mpID).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *payoutRepository) UpdateStatus(id uuid.UUID, status models.PayoutStatus) error {
	res := r.db.
		Model(&models.Payout{}).
		Where("id = ?", id).
		Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *payoutRepository) MarkPaid(id uuid.UUID) error {
	res := r.db.
		Model(&models.Payout{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       models.PayoutStatusPaid,
			"processed_at": gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *payoutRepository) MarkFailed(id uuid.UUID, reason string) error {
	res := r.db.
		Model(&models.Payout{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         models.PayoutStatusFailed,
			"failure_reason": reason,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *payoutRepository) ListByUser(userID uuid.UUID) ([]models.Payout, error) {
	var payouts []models.Payout
	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&payouts).Error
	return payouts, err
}
