package repository

import (
	"errors"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentRepository interface {
	Create(payment *models.Payment) error
	GetByID(id uuid.UUID) (*models.Payment, error)
	GetByProviderPaymentID(provider, providerPaymentID string) (*models.Payment, error)
	GetByIdempotencyKey(key string) (*models.Payment, error)
	UpdateStatus(id uuid.UUID, status models.PaymentStatus) error
	MarkCaptured(id uuid.UUID, providerPaymentID string) error
	MarkFailed(id uuid.UUID, reason string) error
	ListByUser(userID uuid.UUID) ([]models.Payment, error)
	ListByMilestone(milestoneID uuid.UUID) ([]models.Payment, error)

	WithTransaction(fn func(txRepo PaymentRepository) error) error
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository() PaymentRepository {
	return &paymentRepository{db: postgres.DB}
}

func NewPaymentRepositoryTx(tx *gorm.DB) PaymentRepository {
	return &paymentRepository{db: tx}
}

func (r *paymentRepository) WithTransaction(fn func(txRepo PaymentRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewPaymentRepositoryTx(tx))
	})
}

func (r *paymentRepository) Create(payment *models.Payment) error {
	return r.db.Create(payment).Error
}

func (r *paymentRepository) GetByID(id uuid.UUID) (*models.Payment, error) {
	var p models.Payment
	err := r.db.Where("id = ?", id).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *paymentRepository) GetByProviderPaymentID(provider, providerPaymentID string) (*models.Payment, error) {
	var p models.Payment
	err := r.db.
		Where("provider = ? AND provider_payment_id = ?", provider, providerPaymentID).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *paymentRepository) GetByIdempotencyKey(key string) (*models.Payment, error) {
	var p models.Payment
	err := r.db.Where("idempotency_key = ?", key).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *paymentRepository) UpdateStatus(id uuid.UUID, status models.PaymentStatus) error {
	res := r.db.
		Model(&models.Payment{}).
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

func (r *paymentRepository) MarkCaptured(id uuid.UUID, providerPaymentID string) error {
	res := r.db.
		Model(&models.Payment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":              models.PaymentStatusCaptured,
			"provider_payment_id": providerPaymentID,
			"captured_at":         gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *paymentRepository) MarkFailed(id uuid.UUID, reason string) error {
	res := r.db.
		Model(&models.Payment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         models.PaymentStatusFailed,
			"failed_at":      gorm.Expr("NOW()"),
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

func (r *paymentRepository) ListByUser(userID uuid.UUID) ([]models.Payment, error) {
	var payments []models.Payment
	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&payments).Error
	return payments, err
}

func (r *paymentRepository) ListByMilestone(milestoneID uuid.UUID) ([]models.Payment, error) {
	var payments []models.Payment
	err := r.db.
		Where("milestone_id = ?", milestoneID).
		Order("created_at DESC").
		Find(&payments).Error
	return payments, err
}

// Silence unused import if errors package not used elsewhere
var _ = errors.Is
