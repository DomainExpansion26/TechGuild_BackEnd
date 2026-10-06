package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefundRepository interface {
	Create(refund *models.Refund) error
	GetByID(id uuid.UUID) (*models.Refund, error)
	GetByProviderRefundID(provider, providerRefundID string) (*models.Refund, error)
	GetByIdempotencyKey(key string) (*models.Refund, error)
	ListByPayment(paymentID uuid.UUID) ([]models.Refund, error)
	UpdateStatus(id uuid.UUID, status models.RefundStatus) error
	MarkCompleted(id uuid.UUID) error
	MarkFailed(id uuid.UUID, reason string) error

	// SumCompletedByPayment returns total successfully refunded amount for
	// a payment — used to enforce "refund cannot exceed refundable amount".
	SumCompletedByPayment(paymentID uuid.UUID) (int64, error)

	WithTransaction(fn func(txRepo RefundRepository) error) error
}

type refundRepository struct {
	db *gorm.DB
}

func NewRefundRepository() RefundRepository {
	return &refundRepository{db: postgres.DB}
}

func NewRefundRepositoryTx(tx *gorm.DB) RefundRepository {
	return &refundRepository{db: tx}
}

func (r *refundRepository) WithTransaction(fn func(txRepo RefundRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewRefundRepositoryTx(tx))
	})
}

func (r *refundRepository) Create(refund *models.Refund) error {
	return r.db.Create(refund).Error
}

func (r *refundRepository) GetByID(id uuid.UUID) (*models.Refund, error) {
	var ref models.Refund
	err := r.db.Where("id = ?", id).First(&ref).Error
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func (r *refundRepository) GetByProviderRefundID(provider, providerRefundID string) (*models.Refund, error) {
	var ref models.Refund
	err := r.db.
		Where("provider = ? AND provider_refund_id = ?", provider, providerRefundID).
		First(&ref).Error
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func (r *refundRepository) GetByIdempotencyKey(key string) (*models.Refund, error) {
	var ref models.Refund
	err := r.db.Where("idempotency_key = ?", key).First(&ref).Error
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func (r *refundRepository) ListByPayment(paymentID uuid.UUID) ([]models.Refund, error) {
	var refunds []models.Refund
	err := r.db.
		Where("payment_id = ?", paymentID).
		Order("created_at DESC").
		Find(&refunds).Error
	return refunds, err
}

func (r *refundRepository) UpdateStatus(id uuid.UUID, status models.RefundStatus) error {
	res := r.db.
		Model(&models.Refund{}).
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

func (r *refundRepository) MarkCompleted(id uuid.UUID) error {
	res := r.db.
		Model(&models.Refund{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       models.RefundStatusCompleted,
			"completed_at": gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *refundRepository) MarkFailed(id uuid.UUID, reason string) error {
	res := r.db.
		Model(&models.Refund{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         models.RefundStatusFailed,
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

func (r *refundRepository) SumCompletedByPayment(paymentID uuid.UUID) (int64, error) {
	var total *int64
	err := r.db.
		Model(&models.Refund{}).
		Where("payment_id = ? AND status = ?", paymentID, models.RefundStatusCompleted).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, err
	}
	if total == nil {
		return 0, nil
	}
	return *total, nil
}
