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

// ============================================================
// Errors
// ============================================================

var (
	ErrRelQuestNotFound         = errors.New("quest not found")
	ErrRelMilestoneNotFound     = errors.New("milestone not found")
	ErrRelUnauthorized          = errors.New("not authorized for this action")
	ErrRelInvalidState          = errors.New("action not allowed in current milestone state")
	ErrRelNotAssignedFreelancer = errors.New("only the assigned freelancer can submit for release")
	ErrRelNotQuestClient        = errors.New("only the quest client can release this milestone")
	ErrRelAlreadyReleased       = errors.New("milestone already released")
	ErrRelProviderFailed        = errors.New("provider escrow release failed")
	ErrRelPaymentNotFound       = errors.New("milestone payment record not found")
)

// ============================================================
// Service
// ============================================================

type MilestoneReleaseService struct {
	cfg       *config.Config
	provider  PaymentProvider
	ledgerSvc *PaymentLedgerService
	walletSvc *WalletService

	questRepo       repository.QuestRepository
	milestoneRepo   repository.QuestMilestoneRepository
	mpRepo          repository.MilestonePaymentRepository
	escrowRepo      repository.EscrowTransactionRepository
	platformFeeRepo repository.PlatformFeeRepository
	activityRepo    repository.TaskManagementRepository
}

func NewMilestoneReleaseService(cfg *config.Config, provider PaymentProvider) *MilestoneReleaseService {
	return &MilestoneReleaseService{
		cfg:             cfg,
		provider:        provider,
		ledgerSvc:       NewPaymentLedgerService(),
		walletSvc:       NewWalletService(cfg),
		questRepo:       repository.NewQuestRepository(),
		milestoneRepo:   repository.NewQuestMilestoneRepository(),
		mpRepo:          repository.NewMilestonePaymentRepository(),
		escrowRepo:      repository.NewEscrowTransactionRepository(),
		platformFeeRepo: repository.NewPlatformFeeRepository(),
		activityRepo:    repository.NewTaskManagementRepository(),
	}
}

// ============================================================
// 1. SubmitForRelease (freelancer)
// work_in_progress → submitted_for_release
// ============================================================

func (s *MilestoneReleaseService) SubmitForRelease(
	userIDStr, questIDStr, milestoneIDStr string,
	req dto.SubmitForReleaseBody,
) (*dto.MilestoneDetailResponse, error) {

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrRelQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrRelMilestoneNotFound
	}

	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return nil, ErrRelQuestNotFound
	}
	if quest.AssignedFreelancerID == nil || *quest.AssignedFreelancerID != userUUID {
		return nil, ErrRelNotAssignedFreelancer
	}

	milestone, err := s.milestoneRepo.FindByID(milestoneUUID)
	if err != nil {
		return nil, ErrRelMilestoneNotFound
	}
	if milestone.QuestID != questUUID {
		return nil, ErrRelMilestoneNotFound
	}
	if milestone.Status != models.MilestoneStatusWorkInProgress {
		return nil, ErrRelInvalidState
	}

	if err := s.milestoneRepo.UpdateStatus(
		milestone.ID,
		models.MilestoneStatusSubmittedForRelease,
	); err != nil {
		return nil, err
	}

	_ = s.activityRepo.CreateActivity(&models.QuestActivity{
		QuestID:     questUUID,
		MilestoneID: &milestone.ID,
		ActorID:     userUUID,
		ActionType:  models.ActivityMilestoneSubmittedForRelease,
		Title:       fmt.Sprintf("Milestone '%s' Submitted for Release", milestone.Title),
		Description: req.Note,
	})

	milestone.Status = models.MilestoneStatusSubmittedForRelease
	resp := toMilestoneDetailResponse(milestone)
	return &resp, nil
}

// ============================================================
// 2. ReleaseMilestone (client)
// submitted_for_release → released
//
// Flow:
//   1. Validate auth + state
//   2. Load MilestonePayment + EscrowTransaction
//   3. Provider: ReleaseEscrow (outside tx)
//   4. DB TX:
//      - escrow: held → released
//      - platform fee (freelancer side): pending → settled
//      - milestone_payment: funded → released
//      - wallet: +freelancer_net
//      - wallet_transaction: credit
//      - ledger: escrow_release, platform_fee_freelancer, wallet_credit
//      - milestone: submitted_for_release → released
//   5. Activity log
// ============================================================

func (s *MilestoneReleaseService) ReleaseMilestone(
	userIDStr, questIDStr, milestoneIDStr string,
	req dto.ReleaseMilestoneBody,
) (*dto.ReleaseMilestoneResponse, error) {

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrRelQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrRelMilestoneNotFound
	}
	if req.IdempotencyKey == "" {
		return nil, errors.New("idempotency_key is required")
	}

	// ---- 1. Load + Auth ----
	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return nil, ErrRelQuestNotFound
	}
	if quest.ClientID != userUUID {
		return nil, ErrRelNotQuestClient
	}

	milestone, err := s.milestoneRepo.FindByID(milestoneUUID)
	if err != nil {
		return nil, ErrRelMilestoneNotFound
	}
	if milestone.QuestID != questUUID {
		return nil, ErrRelMilestoneNotFound
	}

	// Idempotency: if already released, return existing response
	if milestone.Status == models.MilestoneStatusReleased {
		return s.buildReleaseResponseFromExisting(milestone, questUUID)
	}

	if milestone.Status != models.MilestoneStatusSubmittedForRelease {
		return nil, ErrRelInvalidState
	}

	// ---- 2. Load payment + escrow ----
	mp, err := s.mpRepo.GetByQuestMilestoneID(milestoneUUID)
	if err != nil {
		return nil, ErrRelPaymentNotFound
	}
	escrow, err := s.escrowRepo.GetByQuestMilestoneID(milestoneUUID)
	if err != nil {
		return nil, errors.New("escrow record not found")
	}
	if escrow.ProviderEscrowID == nil || *escrow.ProviderEscrowID == "" {
		return nil, errors.New("escrow has no provider reference")
	}

	// ---- 3. Provider: ReleaseEscrow (OUTSIDE tx) ----
	if _, err := s.provider.ReleaseEscrow(EscrowReleaseRequest{
		ProviderEscrowID: *escrow.ProviderEscrowID,
		Amount:           escrow.Amount,
		Currency:         escrow.Currency,
		IdempotencyKey:   req.IdempotencyKey,
	}); err != nil {
		log.Printf("ReleaseEscrow failed for milestone=%s: %v", milestoneUUID, err)
		return nil, ErrRelProviderFailed
	}

	// ---- 4. DB TX ----
	var walletTxID uuid.UUID
	var walletAfter int64

	err = postgres.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 4a. Escrow: held → released
		if err := tx.Model(&models.EscrowTransaction{}).
			Where("id = ?", escrow.ID).
			Updates(map[string]interface{}{
				"status":      models.EscrowStatusReleased,
				"released_at": now,
			}).Error; err != nil {
			return fmt.Errorf("update escrow: %w", err)
		}

		// 4b. Platform fee (freelancer side): pending → settled
		if err := tx.Model(&models.PlatformFee{}).
			Where("milestone_payment_id = ? AND side = ?", mp.ID, models.PlatformFeeSideFreelancer).
			Updates(map[string]interface{}{
				"status":     models.PlatformFeeSettled,
				"settled_at": now,
			}).Error; err != nil {
			return fmt.Errorf("settle freelancer fee: %w", err)
		}

		// 4c. MilestonePayment: funded → released
		if err := tx.Model(&models.MilestonePayment{}).
			Where("id = ?", mp.ID).
			Updates(map[string]interface{}{
				"status":      models.MilestonePaymentReleased,
				"released_at": now,
			}).Error; err != nil {
			return fmt.Errorf("update milestone_payment: %w", err)
		}

		// 4d. Milestone: submitted_for_release → released
		if err := tx.Model(&models.QuestMilestone{}).
			Where("id = ?", milestoneUUID).
			Updates(map[string]interface{}{
				"status":      models.MilestoneStatusReleased,
				"released_at": now,
			}).Error; err != nil {
			return fmt.Errorf("update milestone: %w", err)
		}

		// 4e. Wallet credit (via tx-scoped repos)
		wallet, err := creditWalletInTx(tx, quest.AssignedFreelancerID, mp.FreelancerNetMinor, mp.Currency, mp.ID.String(), "Milestone release: "+milestone.Title, req.IdempotencyKey+":wallet_credit")
		if err != nil {
			return fmt.Errorf("wallet credit: %w", err)
		}
		walletTxID = wallet.TxID
		walletAfter = wallet.BalanceAfter

		// 4f. Ledger entries (pointers)
		msPtr := milestoneUUID
		paymentPtr := mp.PaymentID
		questPtr := questUUID

		entries := []LedgerEntry{
			{
				PaymentID:        &paymentPtr,
				QuestID:          &questPtr,
				QuestMilestoneID: &msPtr,
				TransactionType:  models.TxTypeEscrowRelease,
				Amount:           -mp.MilestoneAmountMinor,
				Currency:         mp.Currency,
				Provider:         s.provider.Name(),
				Reference:        req.IdempotencyKey,
				IdempotencyKey:   req.IdempotencyKey + ":ledger:escrow_release",
			},
			{
				PaymentID:        &paymentPtr,
				QuestID:          &questPtr,
				QuestMilestoneID: &msPtr,
				TransactionType:  models.TxTypePlatformFeeFreelancer,
				Amount:           mp.FreelancerFeeMinor,
				Currency:         mp.Currency,
				Provider:         s.provider.Name(),
				Reference:        mp.ID.String(),
				IdempotencyKey:   req.IdempotencyKey + ":ledger:platform_fee_freelancer",
			},
			{
				PaymentID:        &paymentPtr,
				QuestID:          &questPtr,
				QuestMilestoneID: &msPtr,
				TransactionType:  models.TxTypeWalletCredit,
				Amount:           mp.FreelancerNetMinor,
				Currency:         mp.Currency,
				Provider:         s.provider.Name(),
				Reference:        walletTxID.String(),
				IdempotencyKey:   req.IdempotencyKey + ":ledger:wallet_credit",
			},
		}
		for _, e := range entries {
			if err := s.ledgerSvc.Append(tx, e); err != nil {
				return fmt.Errorf("append ledger: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Release TX failed for milestone=%s: %v", milestoneUUID, err)
		return nil, fmt.Errorf("failed to release: %w", err)
	}

	// ---- 5. Activity log ----
	_ = s.activityRepo.CreateActivity(&models.QuestActivity{
		QuestID:     questUUID,
		MilestoneID: &milestoneUUID,
		ActorID:     userUUID,
		ActionType:  models.ActivityMilestoneReleased,
		Title:       fmt.Sprintf("Milestone '%s' Released", milestone.Title),
		Description: fmt.Sprintf("Client released ₹%.2f to freelancer wallet", utils.PaiseToRupees(mp.FreelancerNetMinor)),
	})

	// ---- 6. Response ----
	return &dto.ReleaseMilestoneResponse{
		MilestonePaymentID:     mp.ID.String(),
		EscrowID:               escrow.ID.String(),
		PayoutWalletTxID:       walletTxID.String(),
		MilestoneID:            milestoneUUID.String(),
		QuestID:                questUUID.String(),
		Status:                 string(models.MilestonePaymentReleased),
		MilestoneAmount:        utils.PaiseToRupees(mp.MilestoneAmountMinor),
		PlatformFee:            utils.PaiseToRupees(mp.FreelancerFeeMinor),
		FreelancerNet:          utils.PaiseToRupees(mp.FreelancerNetMinor),
		MilestoneAmountMinor:   mp.MilestoneAmountMinor,
		PlatformFeeMinor:       mp.FreelancerFeeMinor,
		FreelancerNetMinor:     mp.FreelancerNetMinor,
		Currency:               mp.Currency,
		WalletAvailableBalance: utils.PaiseToRupees(walletAfter),
		ReleasedAt:             time.Now(),
		Message:                "Milestone released. Funds credited to freelancer wallet.",
	}, nil
}

// ============================================================
// Helpers
// ============================================================

// creditWalletInTx — reused logic, in same tx as release.
// Returns (wallet_tx_id, balance_after).
func creditWalletInTx(
	tx *gorm.DB,
	userID *uuid.UUID,
	amountMinor int64,
	currency string,
	reference string,
	description string,
	idempotencyKey string,
) (struct {
	TxID         uuid.UUID
	BalanceAfter int64
}, error) {

	var result struct {
		TxID         uuid.UUID
		BalanceAfter int64
	}

	if userID == nil {
		return result, errors.New("no freelancer assigned")
	}

	// Idempotency
	var count int64
	if err := tx.Model(&models.WalletTransaction{}).
		Where("idempotency_key = ?", idempotencyKey).
		Count(&count).Error; err != nil {
		return result, err
	}
	if count > 0 {
		return result, errors.New("wallet transaction already processed")
	}

	// Get or create wallet with lock
	var wallet models.Wallet
	err := tx.Clauses().Where("user_id = ?", *userID).First(&wallet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			wallet = models.Wallet{
				UserID:   *userID,
				Currency: currency,
			}
			if err := tx.Create(&wallet).Error; err != nil {
				return result, fmt.Errorf("create wallet: %w", err)
			}
		} else {
			return result, err
		}
	}

	// Lock existing row
	if err := tx.Raw("SELECT * FROM wallets WHERE id = ? FOR UPDATE", wallet.ID).
		Scan(&wallet).Error; err != nil {
		return result, err
	}

	newBalance := wallet.AvailableBalanceMinor + amountMinor
	updates := map[string]interface{}{
		"available_balance_minor": newBalance,
		"total_earned_minor":      wallet.TotalEarnedMinor + amountMinor,
		"updated_at":              time.Now(),
	}
	if err := tx.Model(&models.Wallet{}).
		Where("id = ?", wallet.ID).
		Updates(updates).Error; err != nil {
		return result, err
	}

	wtx := &models.WalletTransaction{
		WalletID:          wallet.ID,
		UserID:            *userID,
		Type:              models.WalletTxTypeCredit,
		AmountMinor:       amountMinor,
		BalanceAfterMinor: newBalance,
		Currency:          currency,
		Reference:         reference,
		Description:       description,
		IdempotencyKey:    idempotencyKey,
	}
	if err := tx.Create(wtx).Error; err != nil {
		return result, err
	}

	result.TxID = wtx.ID
	result.BalanceAfter = newBalance
	return result, nil
}

// buildReleaseResponseFromExisting — idempotent release response.
func (s *MilestoneReleaseService) buildReleaseResponseFromExisting(
	milestone *models.QuestMilestone,
	questUUID uuid.UUID,
) (*dto.ReleaseMilestoneResponse, error) {
	mp, err := s.mpRepo.GetByQuestMilestoneID(milestone.ID)
	if err != nil {
		return nil, err
	}
	escrow, err := s.escrowRepo.GetByQuestMilestoneID(milestone.ID)
	if err != nil {
		return nil, err
	}

	// Fetch wallet + wallet transaction for accurate values
	var walletBalance int64
	var walletTxID uuid.UUID

	walletRepo := repository.NewWalletRepository()
	if quest, err := s.questRepo.FindByID(questUUID); err == nil && quest.AssignedFreelancerID != nil {
		if w, err := walletRepo.GetByUserID(*quest.AssignedFreelancerID); err == nil {
			walletBalance = w.AvailableBalanceMinor
		}

		// Find wallet transaction by reference (milestone_payment_id)
		var wtx models.WalletTransaction
		if err := postgres.DB.
			Where("reference = ? AND type = ?", mp.ID.String(), models.WalletTxTypeCredit).
			First(&wtx).Error; err == nil {
			walletTxID = wtx.ID
		}
	}

	releasedAt := time.Now()
	if mp.ReleasedAt != nil {
		releasedAt = *mp.ReleasedAt
	}

	return &dto.ReleaseMilestoneResponse{
		MilestonePaymentID:     mp.ID.String(),
		EscrowID:               escrow.ID.String(),
		PayoutWalletTxID:       walletTxID.String(),
		MilestoneID:            milestone.ID.String(),
		QuestID:                questUUID.String(),
		Status:                 string(mp.Status),
		MilestoneAmount:        utils.PaiseToRupees(mp.MilestoneAmountMinor),
		PlatformFee:            utils.PaiseToRupees(mp.FreelancerFeeMinor),
		FreelancerNet:          utils.PaiseToRupees(mp.FreelancerNetMinor),
		MilestoneAmountMinor:   mp.MilestoneAmountMinor,
		PlatformFeeMinor:       mp.FreelancerFeeMinor,
		FreelancerNetMinor:     mp.FreelancerNetMinor,
		Currency:               mp.Currency,
		WalletAvailableBalance: utils.PaiseToRupees(walletBalance),
		ReleasedAt:             releasedAt,
		Message:                "Milestone already released (idempotent)",
	}, nil
}

// toMilestoneDetailResponse — simple helper for milestone response without revisions.
func toMilestoneDetailResponse(m *models.QuestMilestone) dto.MilestoneDetailResponse {
	resp := dto.MilestoneDetailResponse{
		ID:             m.ID.String(),
		QuestID:        m.QuestID.String(),
		CreatedByID:    m.CreatedByID.String(),
		Title:          m.Title,
		Description:    m.Description,
		AmountMinor:    m.AmountMinor,
		Amount:         utils.PaiseToRupees(m.AmountMinor),
		Currency:       "INR",
		DueDays:        m.DueDays,
		OrderIndex:     m.OrderIndex,
		Status:         string(m.Status),
		RevisionCount:  m.RevisionCount,
		ClientFeedback: m.ClientFeedback,
		StartDate:      m.StartDate,
		DueDate:        m.DueDate,
		AgreedAt:       m.AgreedAt,
		FundedAt:       m.FundedAt,
		ReleasedAt:     m.ReleasedAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}

	tasks := make([]dto.MilestoneTaskSummary, 0, len(m.Tasks))
	for _, t := range m.Tasks {
		tasks = append(tasks, dto.MilestoneTaskSummary{
			ID:          t.ID.String(),
			Title:       t.Title,
			Description: t.Description,
			Status:      string(t.Status),
			StartDate:   t.StartDate,
			EndDate:     t.EndDate,
		})
	}
	resp.Tasks = tasks
	return resp
}
