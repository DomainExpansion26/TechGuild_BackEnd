package services

import (
	"errors"
	"fmt"
	"time"

	"techguild-backend/src/config"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrWalletNotFoundErr       = errors.New("wallet not found")
	ErrWalletInsufficientFunds = errors.New("insufficient wallet balance")
	ErrWalletInvalidAmount     = errors.New("wallet amount must be positive")
	ErrWalletDuplicate         = errors.New("wallet transaction already processed")
)

type WalletService struct {
	cfg        *config.Config
	walletRepo repository.WalletRepository
	txRepo     repository.WalletTransactionRepository
}

func NewWalletService(cfg *config.Config) *WalletService {
	return &WalletService{
		cfg:        cfg,
		walletRepo: repository.NewWalletRepository(),
		txRepo:     repository.NewWalletTransactionRepository(),
	}
}

// ============================================================
// Credit — adds money to available balance
// Used at milestone release (freelancer gets paid)
// ============================================================
//
// amountMinor must be positive.
// idempotencyKey — prevents double-credit.
func (s *WalletService) Credit(
	userID uuid.UUID,
	amountMinor int64,
	reference string,
	description string,
	idempotencyKey string,
) (*models.WalletTransaction, error) {

	if amountMinor <= 0 {
		return nil, ErrWalletInvalidAmount
	}
	if idempotencyKey == "" {
		return nil, errors.New("idempotency_key required")
	}

	var result *models.WalletTransaction

	err := postgres.DB.Transaction(func(tx *gorm.DB) error {
		txWalletRepo := repository.NewWalletRepositoryTx(tx)
		txTxRepo := repository.NewWalletTransactionRepositoryTx(tx)

		// Idempotency check
		exists, err := txTxRepo.ExistsByIdempotencyKey(idempotencyKey)
		if err != nil {
			return err
		}
		if exists {
			return ErrWalletDuplicate
		}

		// Get or create wallet (with row lock if exists)
		wallet, err := txWalletRepo.GetByUserIDForUpdate(userID)
		if err != nil {
			if errors.Is(err, repository.ErrWalletNotFound) {
				// Create it
				wallet = &models.Wallet{
					UserID:   userID,
					Currency: "INR",
				}
				if err := txWalletRepo.Create(wallet); err != nil {
					return fmt.Errorf("create wallet: %w", err)
				}
			} else {
				return err
			}
		}

		// Update balance
		newAvailable := wallet.AvailableBalanceMinor + amountMinor
		wallet.AvailableBalanceMinor = newAvailable
		wallet.TotalEarnedMinor = wallet.TotalEarnedMinor + amountMinor

		if err := txWalletRepo.Save(wallet); err != nil {
			return fmt.Errorf("update wallet: %w", err)
		}

		// Create wallet transaction
		wtx := &models.WalletTransaction{
			WalletID:          wallet.ID,
			UserID:            userID,
			Type:              models.WalletTxTypeCredit,
			AmountMinor:       amountMinor,
			BalanceAfterMinor: newAvailable,
			Currency:          wallet.Currency,
			Reference:         reference,
			Description:       description,
			IdempotencyKey:    idempotencyKey,
		}
		if err := txTxRepo.Create(wtx); err != nil {
			return fmt.Errorf("create wallet tx: %w", err)
		}
		result = wtx
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// ============================================================
// GetWallet — fetch wallet (create if not exists)
// ============================================================

func (s *WalletService) GetWallet(userID uuid.UUID) (*models.Wallet, error) {
	wallet, err := s.walletRepo.GetOrCreate(userID)
	if err != nil {
		return nil, err
	}
	return wallet, nil
}

// ============================================================
// ListTransactions — paginated
// ============================================================

func (s *WalletService) ListTransactions(
	userID uuid.UUID,
	limit, offset int,
) ([]models.WalletTransaction, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.txRepo.ListByUser(userID, limit, offset)
}

// ============================================================
// Helper — format balance for response
// ============================================================

func (s *WalletService) WalletToRupees(w *models.Wallet) map[string]interface{} {
	return map[string]interface{}{
		"available_balance": utils.PaiseToRupees(w.AvailableBalanceMinor),
		"pending_balance":   utils.PaiseToRupees(w.PendingBalanceMinor),
		"total_earned":      utils.PaiseToRupees(w.TotalEarnedMinor),
		"total_withdrawn":   utils.PaiseToRupees(w.TotalWithdrawnMinor),
		"currency":          w.Currency,
	}
}

// Silence unused import
var _ = time.Now
