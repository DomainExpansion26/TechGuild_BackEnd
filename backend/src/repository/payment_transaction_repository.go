package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentTransactionRepository interface {
	// Append-only: sirf insert hai, koi update/delete nahi.
	Create(tx *models.PaymentTransaction) error
	ExistsByIdempotencyKey(key string) (bool, error)
	ListByPayment(paymentID uuid.UUID) ([]models.PaymentTransaction, error)
	ListByMilestone(milestoneID uuid.UUID) ([]models.PaymentTransaction, error)

	WithTransaction(fn func(txRepo PaymentTransactionRepository) error) error
}

type paymentTransactionRepository struct {
	db *gorm.DB
}

func NewPaymentTransactionRepository() PaymentTransactionRepository {
	return &paymentTransactionRepository{db: postgres.DB}
}

func NewPaymentTransactionRepositoryTx(tx *gorm.DB) PaymentTransactionRepository {
	return &paymentTransactionRepository{db: tx}
}

func (r *paymentTransactionRepository) WithTransaction(fn func(txRepo PaymentTransactionRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewPaymentTransactionRepositoryTx(tx))
	})
}

func (r *paymentTransactionRepository) Create(tx *models.PaymentTransaction) error {
	return r.db.Create(tx).Error
}

func (r *paymentTransactionRepository) ExistsByIdempotencyKey(key string) (bool, error) {
	var count int64
	err := r.db.
		Model(&models.PaymentTransaction{}).
		Where("idempotency_key = ?", key).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *paymentTransactionRepository) ListByPayment(paymentID uuid.UUID) ([]models.PaymentTransaction, error) {
	var txs []models.PaymentTransaction
	err := r.db.
		Where("payment_id = ?", paymentID).
		Order("created_at ASC").
		Find(&txs).Error
	return txs, err
}

func (r *paymentTransactionRepository) ListByMilestone(milestoneID uuid.UUID) ([]models.PaymentTransaction, error) {
	var txs []models.PaymentTransaction
	err := r.db.
		Where("milestone_id = ?", milestoneID).
		Order("created_at ASC").
		Find(&txs).Error
	return txs, err
}
