package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"techguild-backend/src/config"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrWDWalletNotFound    = errors.New("wallet not found")
	ErrWDInsufficient      = errors.New("insufficient wallet balance")
	ErrWDInvalidAmount     = errors.New("withdrawal amount must be positive")
	ErrWDPayoutMethodError = errors.New("payout method not usable")
	ErrWDProviderFailed    = errors.New("payout provider failed")
	ErrWDDuplicate         = errors.New("withdrawal already processed")
	ErrWDNotFound          = errors.New("withdrawal not found")
)

type WalletWithdrawalService struct {
	cfg         *config.Config
	provider    PaymentProvider
	walletRepo  repository.WalletRepository
	wtRepo      repository.WalletTransactionRepository
	payoutRepo  repository.PayoutRepository
	payoutMRepo repository.PayoutMethodRepository
}

func NewWalletWithdrawalService(cfg *config.Config, provider PaymentProvider) *WalletWithdrawalService {
	return &WalletWithdrawalService{
		cfg:         cfg,
		provider:    provider,
		walletRepo:  repository.NewWalletRepository(),
		wtRepo:      repository.NewWalletTransactionRepository(),
		payoutRepo:  repository.NewPayoutRepository(),
		payoutMRepo: repository.NewPayoutMethodRepository(),
	}
}

// ============================================================
// RequestWithdrawal
// ============================================================

func (s *WalletWithdrawalService) RequestWithdrawal(
	userIDStr string,
	req dto.RequestWithdrawalBody,
) (*dto.RequestWithdrawalResponse, error) {

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	if req.IdempotencyKey == "" {
		return nil, errors.New("idempotency_key required")
	}
	if req.Amount <= 0 {
		return nil, ErrWDInvalidAmount
	}
	amountMinor := utils.RupeesToPaise(req.Amount)

	// ---- 1. Idempotency check ----
	existing, err := s.payoutRepo.GetByIdempotencyKey(req.IdempotencyKey)
	if err == nil && existing != nil {
		return s.buildIdempotentResponse(existing)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// ---- 2. Load wallet + check balance ----
	wallet, err := s.walletRepo.GetByUserID(userUUID)
	if err != nil {
		if errors.Is(err, repository.ErrWalletNotFound) {
			return nil, ErrWDWalletNotFound
		}
		return nil, err
	}
	if wallet.AvailableBalanceMinor < amountMinor {
		return nil, ErrWDInsufficient
	}

	// ---- 3. Load payout method ----
	pmUUID, err := uuid.Parse(req.PayoutMethodID)
	if err != nil {
		return nil, ErrWDPayoutMethodError
	}
	pm, err := s.payoutMRepo.GetByIDAndUser(pmUUID, userUUID)
	if err != nil {
		return nil, ErrWDPayoutMethodError
	}
	if pm.Status != models.PayoutMethodStatusVerified {
		return nil, errors.New("payout method must be verified")
	}
	if pm.ProviderMethodID == nil || *pm.ProviderMethodID == "" {
		return nil, errors.New("payout method has no provider reference")
	}

	// ---- 4. Provider: CreatePayout (OUTSIDE tx) ----
	providerPayout, err := s.provider.CreatePayout(PayoutRequest{
		UserID:                userUUID.String(),
		Amount:                amountMinor,
		Currency:              wallet.Currency,
		ProviderFundAccountID: *pm.ProviderMethodID,
		IdempotencyKey:        req.IdempotencyKey,
		Description:           "Wallet withdrawal",
	})
	if err != nil {
		log.Printf("provider CreatePayout failed for user=%s: %v", userUUID, err)
		return nil, ErrWDProviderFailed
	}

	// ---- 5. DB TX ----
	var (
		payoutID   uuid.UUID
		walletTxID uuid.UUID
		newBalance int64
	)

	err = postgres.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 5a. Lock wallet row
		var w models.Wallet
		if err := tx.Raw("SELECT * FROM wallets WHERE id = ? FOR UPDATE", wallet.ID).
			Scan(&w).Error; err != nil {
			return fmt.Errorf("lock wallet: %w", err)
		}
		if w.AvailableBalanceMinor < amountMinor {
			return ErrWDInsufficient
		}

		// 5b. Debit wallet
		newBalance = w.AvailableBalanceMinor - amountMinor
		if err := tx.Model(&models.Wallet{}).
			Where("id = ?", w.ID).
			Updates(map[string]interface{}{
				"available_balance_minor": newBalance,
				"total_withdrawn_minor":   w.TotalWithdrawnMinor + amountMinor,
				"updated_at":              now,
			}).Error; err != nil {
			return fmt.Errorf("debit wallet: %w", err)
		}

		// 5c. Wallet transaction (debit)
		wt := &models.WalletTransaction{
			WalletID:          w.ID,
			UserID:            userUUID,
			Type:              models.WalletTxTypeDebit,
			AmountMinor:       -amountMinor, // negative for debit
			BalanceAfterMinor: newBalance,
			Currency:          w.Currency,
			Reference:         providerPayout.ProviderPayoutID,
			Description:       "Wallet withdrawal",
			IdempotencyKey:    req.IdempotencyKey + ":wallet",
		}
		if err := tx.Create(wt).Error; err != nil {
			return fmt.Errorf("wallet tx: %w", err)
		}
		walletTxID = wt.ID

		// 5d. Payout row
		source := models.PayoutSourceWalletWithdrawal
		payout := &models.Payout{
			UserID:             userUUID,
			Source:             source,
			PayoutMethodID:     pm.ID,
			Provider:           s.provider.Name(),
			ProviderPayoutID:   &providerPayout.ProviderPayoutID,
			Amount:             amountMinor,
			Currency:           wallet.Currency,
			SettlementAmount:   amountMinor,
			SettlementCurrency: wallet.Currency,
			Status:             models.PayoutStatus(providerPayout.Status),
			IdempotencyKey:     req.IdempotencyKey,
			ProcessedAt:        &now,
		}
		if err := tx.Create(payout).Error; err != nil {
			return fmt.Errorf("create payout: %w", err)
		}
		payoutID = payout.ID

		// 5e. Ledger entry (withdrawal) — PaymentID aur QuestID nil (pointers)
		providerRef := providerPayout.ProviderPayoutID
		ledgerRec := &models.PaymentTransaction{
			PaymentID:             nil, // withdrawal me payment nahi
			QuestID:               nil,
			QuestMilestoneID:      nil,
			TransactionType:       models.TxTypeWithdrawal,
			Amount:                -amountMinor,
			Currency:              wallet.Currency,
			Provider:              s.provider.Name(),
			ProviderTransactionID: &providerRef,
			Reference:             payout.ID.String(),
			IdempotencyKey:        req.IdempotencyKey + ":ledger:withdrawal",
			CreatedAt:             now,
		}
		if err := tx.Create(ledgerRec).Error; err != nil {
			return fmt.Errorf("ledger: %w", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("Withdrawal TX failed for user=%s: %v", userUUID, err)
		return nil, fmt.Errorf("failed to process withdrawal: %w", err)
	}

	// ---- 6. Response ----
	return &dto.RequestWithdrawalResponse{
		PayoutID:                    payoutID.String(),
		WalletTxID:                  walletTxID.String(),
		Amount:                      utils.PaiseToRupees(amountMinor),
		AmountMinor:                 amountMinor,
		Currency:                    wallet.Currency,
		Status:                      providerPayout.Status,
		ProviderPayoutID:            providerPayout.ProviderPayoutID,
		WalletAvailableBalance:      utils.PaiseToRupees(newBalance),
		WalletAvailableBalanceMinor: newBalance,
		Message:                     "Withdrawal successful. Funds on the way to your account.",
	}, nil
}

// ============================================================
// ListWithdrawals
// ============================================================

func (s *WalletWithdrawalService) ListWithdrawals(
	userIDStr string,
	limit, offset int,
) (*dto.ListWithdrawalsResponse, error) {

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	payouts, total, err := s.payoutRepo.ListWithdrawalsByUser(userUUID, limit, offset)
	if err != nil {
		return nil, err
	}

	items := make([]dto.WithdrawalItem, 0, len(payouts))
	for _, p := range payouts {
		item := dto.WithdrawalItem{
			ID:             p.ID.String(),
			Amount:         utils.PaiseToRupees(p.Amount),
			AmountMinor:    p.Amount,
			Currency:       p.Currency,
			Status:         string(p.Status),
			PayoutMethodID: p.PayoutMethodID.String(),
			FailureReason:  p.FailureReason,
			ProcessedAt:    p.ProcessedAt,
			CreatedAt:      p.CreatedAt,
		}
		if p.ProviderPayoutID != nil {
			item.ProviderPayoutID = *p.ProviderPayoutID
		}
		items = append(items, item)
	}

	return &dto.ListWithdrawalsResponse{
		Withdrawals: items,
		Total:       total,
		Limit:       limit,
		Offset:      offset,
	}, nil
}

// ============================================================
// GetWithdrawal
// ============================================================

func (s *WalletWithdrawalService) GetWithdrawal(
	userIDStr, payoutIDStr string,
) (*dto.WithdrawalItem, error) {

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	payoutUUID, err := uuid.Parse(payoutIDStr)
	if err != nil {
		return nil, ErrWDNotFound
	}

	p, err := s.payoutRepo.GetByID(payoutUUID)
	if err != nil {
		return nil, ErrWDNotFound
	}
	if p.UserID != userUUID {
		return nil, ErrWDNotFound
	}
	if p.Source != models.PayoutSourceWalletWithdrawal {
		return nil, ErrWDNotFound
	}

	item := &dto.WithdrawalItem{
		ID:             p.ID.String(),
		Amount:         utils.PaiseToRupees(p.Amount),
		AmountMinor:    p.Amount,
		Currency:       p.Currency,
		Status:         string(p.Status),
		PayoutMethodID: p.PayoutMethodID.String(),
		FailureReason:  p.FailureReason,
		ProcessedAt:    p.ProcessedAt,
		CreatedAt:      p.CreatedAt,
	}
	if p.ProviderPayoutID != nil {
		item.ProviderPayoutID = *p.ProviderPayoutID
	}
	return item, nil
}

// ============================================================
// Helper — idempotent response
// ============================================================

func (s *WalletWithdrawalService) buildIdempotentResponse(
	p *models.Payout,
) (*dto.RequestWithdrawalResponse, error) {

	wallet, err := s.walletRepo.GetByUserID(p.UserID)
	if err != nil {
		return nil, err
	}

	var providerID string
	if p.ProviderPayoutID != nil {
		providerID = *p.ProviderPayoutID
	}

	// Find wallet transaction by reference (provider_payout_id)
	var walletTxID uuid.UUID
	var wtx models.WalletTransaction
	if err := postgres.DB.
		Where("reference = ? AND type = ?", providerID, models.WalletTxTypeDebit).
		First(&wtx).Error; err == nil {
		walletTxID = wtx.ID
	}

	return &dto.RequestWithdrawalResponse{
		PayoutID:                    p.ID.String(),
		WalletTxID:                  walletTxID.String(),
		Amount:                      utils.PaiseToRupees(p.Amount),
		AmountMinor:                 p.Amount,
		Currency:                    p.Currency,
		Status:                      string(p.Status),
		ProviderPayoutID:            providerID,
		WalletAvailableBalance:      utils.PaiseToRupees(wallet.AvailableBalanceMinor),
		WalletAvailableBalanceMinor: wallet.AvailableBalanceMinor,
		Message:                     "Withdrawal already processed (idempotent)",
	}, nil
}
