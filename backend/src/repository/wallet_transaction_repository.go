package repository

import (
	"errors"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrWalletTransactionDuplicate = errors.New("wallet transaction idempotency key already exists")

type WalletTransactionRepository interface {
	Create(tx *models.WalletTransaction) error
	ExistsByIdempotencyKey(key string) (bool, error)
	ListByUser(userID uuid.UUID, limit, offset int) ([]models.WalletTransaction, int64, error)
	ListByWallet(walletID uuid.UUID) ([]models.WalletTransaction, error)

	WithTransaction(fn func(txRepo WalletTransactionRepository) error) error
}

type walletTransactionRepository struct {
	db *gorm.DB
}

func NewWalletTransactionRepository() WalletTransactionRepository {
	return &walletTransactionRepository{db: postgres.DB}
}

func NewWalletTransactionRepositoryTx(tx *gorm.DB) WalletTransactionRepository {
	return &walletTransactionRepository{db: tx}
}

func (r *walletTransactionRepository) WithTransaction(fn func(txRepo WalletTransactionRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewWalletTransactionRepositoryTx(tx))
	})
}

func (r *walletTransactionRepository) Create(tx *models.WalletTransaction) error {
	return r.db.Create(tx).Error
}

func (r *walletTransactionRepository) ExistsByIdempotencyKey(key string) (bool, error) {
	var count int64
	err := r.db.
		Model(&models.WalletTransaction{}).
		Where("idempotency_key = ?", key).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *walletTransactionRepository) ListByUser(
	userID uuid.UUID,
	limit, offset int,
) ([]models.WalletTransaction, int64, error) {
	var txs []models.WalletTransaction
	var total int64

	query := r.db.Model(&models.WalletTransaction{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&txs).Error
	return txs, total, err
}

func (r *walletTransactionRepository) ListByWallet(walletID uuid.UUID) ([]models.WalletTransaction, error) {
	var txs []models.WalletTransaction
	err := r.db.
		Where("wallet_id = ?", walletID).
		Order("created_at DESC").
		Find(&txs).Error
	return txs, err
}
