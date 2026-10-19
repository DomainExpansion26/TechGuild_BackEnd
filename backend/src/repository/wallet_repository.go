package repository

import (
	"errors"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrWalletNotFound = errors.New("wallet not found")

type WalletRepository interface {
	// Get or create
	GetByUserID(userID uuid.UUID) (*models.Wallet, error)
	GetByUserIDForUpdate(userID uuid.UUID) (*models.Wallet, error)
	GetOrCreate(userID uuid.UUID) (*models.Wallet, error)
	Create(w *models.Wallet) error
	Save(w *models.Wallet) error

	// Tx-scoped
	WithTransaction(fn func(txRepo WalletRepository) error) error
}

type walletRepository struct {
	db *gorm.DB
}

func NewWalletRepository() WalletRepository {
	return &walletRepository{db: postgres.DB}
}

func NewWalletRepositoryTx(tx *gorm.DB) WalletRepository {
	return &walletRepository{db: tx}
}

func (r *walletRepository) WithTransaction(fn func(txRepo WalletRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewWalletRepositoryTx(tx))
	})
}

func (r *walletRepository) Create(w *models.Wallet) error {
	return r.db.Create(w).Error
}

func (r *walletRepository) GetByUserID(userID uuid.UUID) (*models.Wallet, error) {
	var w models.Wallet
	err := r.db.Where("user_id = ?", userID).First(&w).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}
	return &w, nil
}

// GetByUserIDForUpdate — row-level lock for safe concurrent credits/debits.
func (r *walletRepository) GetByUserIDForUpdate(userID uuid.UUID) (*models.Wallet, error) {
	var w models.Wallet
	err := r.db.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&w).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}
	return &w, nil
}

// GetOrCreate — creates wallet if not exists, returns existing otherwise.
func (r *walletRepository) GetOrCreate(userID uuid.UUID) (*models.Wallet, error) {
	w, err := r.GetByUserID(userID)
	if err == nil {
		return w, nil
	}
	if !errors.Is(err, ErrWalletNotFound) {
		return nil, err
	}

	// Create new
	newWallet := &models.Wallet{
		UserID:                userID,
		AvailableBalanceMinor: 0,
		PendingBalanceMinor:   0,
		TotalEarnedMinor:      0,
		TotalWithdrawnMinor:   0,
		Currency:              "INR",
	}
	if err := r.db.Create(newWallet).Error; err != nil {
		// Race condition: another goroutine created it — retry fetch
		existing, fetchErr := r.GetByUserID(userID)
		if fetchErr == nil {
			return existing, nil
		}
		return nil, err
	}
	return newWallet, nil
}

func (r *walletRepository) Save(w *models.Wallet) error {
	return r.db.Save(w).Error
}
