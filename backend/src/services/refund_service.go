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
	ErrRefQuestNotFound     = errors.New("quest not found")
	ErrRefMilestoneNotFound = errors.New("milestone not found")
	ErrRefUnauthorized      = errors.New("only the quest client can request a refund")
	ErrRefInvalidState      = errors.New("refund not allowed in current milestone state")
	ErrRefAlreadyRefunded   = errors.New("milestone already refunded")
	ErrRefPaymentNotFound   = errors.New("milestone payment record not found")
	ErrRefEscrowNotFound    = errors.New("escrow record not found")
	ErrRefProviderFailed    = errors.New("provider refund failed")
	ErrRefNotFound          = errors.New("refund not found")
	ErrRefReasonRequired    = errors.New("refund reason is required and must be at least 5 characters")
)

// ============================================================
// Service
// ============================================================

type RefundService struct {
	cfg       *config.Config
	provider  PaymentProvider
	ledgerSvc *PaymentLedgerService

	questRepo       repository.QuestRepository
	milestoneRepo   repository.QuestMilestoneRepository
	mpRepo          repository.MilestonePaymentRepository
	escrowRepo      repository.EscrowTransactionRepository
	platformFeeRepo repository.PlatformFeeRepository
	refundRepo      repository.RefundRepository
	paymentRepo     repository.PaymentRepository
	activityRepo    repository.TaskManagementRepository
}

func NewRefundService(cfg *config.Config, provider PaymentProvider) *RefundService {
	return &RefundService{
		cfg:             cfg,
		provider:        provider,
		ledgerSvc:       NewPaymentLedgerService(),
		questRepo:       repository.NewQuestRepository(),
		milestoneRepo:   repository.NewQuestMilestoneRepository(),
		mpRepo:          repository.NewMilestonePaymentRepository(),
		escrowRepo:      repository.NewEscrowTransactionRepository(),
		platformFeeRepo: repository.NewPlatformFeeRepository(),
		refundRepo:      repository.NewRefundRepository(),
		paymentRepo:     repository.NewPaymentRepository(),
		activityRepo:    repository.NewTaskManagementRepository(),
	}
}

// ============================================================
// RequestRefund (client only)
//
// Allowed states: work_in_progress, submitted_for_release
// Refund amount: milestone amount only (client fee stays with platform)
// ============================================================

func (s *RefundService) RequestRefund(
	userIDStr, questIDStr, milestoneIDStr string,
	req dto.RequestRefundBody,
) (*dto.RefundResponse, error) {

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrRefQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrRefMilestoneNotFound
	}
	if req.IdempotencyKey == "" {
		return nil, errors.New("idempotency_key is required")
	}
	if len(req.Reason) < 5 {
		return nil, ErrRefReasonRequired
	}

	// ---- 1. Load + auth ----
	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return nil, ErrRefQuestNotFound
	}
	if quest.ClientID != userUUID {
		return nil, ErrRefUnauthorized
	}

	milestone, err := s.milestoneRepo.FindByID(milestoneUUID)
	if err != nil {
		return nil, ErrRefMilestoneNotFound
	}
	if milestone.QuestID != questUUID {
		return nil, ErrRefMilestoneNotFound
	}

	// ---- 2. Idempotency check ----
	existingRefund, err := s.refundRepo.GetByIdempotencyKey(req.IdempotencyKey)
	if err == nil && existingRefund != nil {
		return s.buildRefundResponseFromExisting(existingRefund, milestone, questUUID)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// ---- 3. Already refunded? ----
	if milestone.Status == models.MilestoneStatusRefunded {
		return s.buildRefundResponseFromMilestone(milestone, questUUID)
	}
	if milestone.Status == models.MilestoneStatusRefundPending {
		return nil, errors.New("refund already in progress")
	}

	// ---- 4. State check ----
	if milestone.Status != models.MilestoneStatusWorkInProgress &&
		milestone.Status != models.MilestoneStatusSubmittedForRelease {
		return nil, ErrRefInvalidState
	}

	// ---- 5. Load payment + escrow ----
	mp, err := s.mpRepo.GetByQuestMilestoneID(milestoneUUID)
	if err != nil {
		return nil, ErrRefPaymentNotFound
	}
	escrow, err := s.escrowRepo.GetByQuestMilestoneID(milestoneUUID)
	if err != nil {
		return nil, ErrRefEscrowNotFound
	}
	if escrow.ProviderEscrowID == nil || *escrow.ProviderEscrowID == "" {
		return nil, errors.New("escrow has no provider reference")
	}

	payment, err := s.paymentRepo.GetByID(mp.PaymentID)
	if err != nil {
		return nil, errors.New("payment record not found")
	}
	if payment.ProviderPaymentID == nil || *payment.ProviderPaymentID == "" {
		return nil, errors.New("payment has no provider reference")
	}

	// Refund amount = milestone amount only
	refundAmount := mp.MilestoneAmountMinor

	// ---- 6. Provider: RefundPayment (OUTSIDE tx) ----
	providerRefund, err := s.provider.RefundPayment(RefundRequest{
		ProviderPaymentID: *payment.ProviderPaymentID,
		Amount:            refundAmount,
		Currency:          mp.Currency,
		Reason:            req.Reason,
		IdempotencyKey:    req.IdempotencyKey,
	})
	if err != nil {
		log.Printf("RefundPayment failed for milestone=%s: %v", milestoneUUID, err)
		return nil, ErrRefProviderFailed
	}

	// ---- 7. DB TX ----
	var refundID uuid.UUID

	err = postgres.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 7a. Create Refund row
		mpID := mp.ID
		providerRefundID := providerRefund.ProviderRefundID
		refund := &models.Refund{
			PaymentID:          mp.PaymentID,
			MilestonePaymentID: &mpID,
			QuestID:            questUUID,
			Provider:           s.provider.Name(),
			ProviderRefundID:   &providerRefundID,
			Amount:             refundAmount,
			Currency:           mp.Currency,
			Status:             models.RefundStatusCompleted,
			Reason:             req.Reason,
			IdempotencyKey:     req.IdempotencyKey,
			CompletedAt:        &now,
		}
		if err := tx.Create(refund).Error; err != nil {
			return fmt.Errorf("create refund: %w", err)
		}
		refundID = refund.ID

		// 7b. Escrow: held → refunded
		if err := tx.Model(&models.EscrowTransaction{}).
			Where("id = ?", escrow.ID).
			Updates(map[string]interface{}{
				"status":      models.EscrowStatusRefunded,
				"refunded_at": now,
			}).Error; err != nil {
			return fmt.Errorf("update escrow: %w", err)
		}

		// 7c. PlatformFee (freelancer side): pending → reversed (cancelled)
		if err := tx.Model(&models.PlatformFee{}).
			Where("milestone_payment_id = ? AND side = ?", mp.ID, models.PlatformFeeSideFreelancer).
			Updates(map[string]interface{}{
				"status": models.PlatformFeeReversed,
			}).Error; err != nil {
			return fmt.Errorf("reverse freelancer fee: %w", err)
		}

		// 7d. MilestonePayment: funded → refunded
		if err := tx.Model(&models.MilestonePayment{}).
			Where("id = ?", mp.ID).
			Updates(map[string]interface{}{
				"status": models.MilestonePaymentRefunded,
			}).Error; err != nil {
			return fmt.Errorf("update milestone_payment: %w", err)
		}

		// 7e. Milestone: → refunded
		if err := tx.Model(&models.QuestMilestone{}).
			Where("id = ?", milestoneUUID).
			Updates(map[string]interface{}{
				"status": models.MilestoneStatusRefunded,
			}).Error; err != nil {
			return fmt.Errorf("update milestone: %w", err)
		}

		// 7f. Ledger entries
		msPtr := milestoneUUID
		paymentPtr := mp.PaymentID
		questPtr := questUUID

		entries := []LedgerEntry{
			{
				PaymentID:        &paymentPtr,
				QuestID:          &questPtr,
				QuestMilestoneID: &msPtr,
				TransactionType:  models.TxTypeEscrowRefund,
				Amount:           -refundAmount,
				Currency:         mp.Currency,
				Provider:         s.provider.Name(),
				Reference:        providerRefund.ProviderRefundID,
				IdempotencyKey:   req.IdempotencyKey + ":ledger:escrow_refund",
			},
			{
				PaymentID:        &paymentPtr,
				QuestID:          &questPtr,
				QuestMilestoneID: &msPtr,
				TransactionType:  models.TxTypeRefund,
				Amount:           -refundAmount,
				Currency:         mp.Currency,
				Provider:         s.provider.Name(),
				Reference:        refund.ID.String(),
				IdempotencyKey:   req.IdempotencyKey + ":ledger:refund",
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
		log.Printf("Refund TX failed for milestone=%s: %v", milestoneUUID, err)
		return nil, fmt.Errorf("failed to process refund: %w", err)
	}

	// ---- 8. Activity log ----
	_ = s.activityRepo.CreateActivity(&models.QuestActivity{
		QuestID:     questUUID,
		MilestoneID: &milestoneUUID,
		ActorID:     userUUID,
		ActionType:  models.ActivityMilestoneRefunded,
		Title:       fmt.Sprintf("Milestone '%s' Refunded", milestone.Title),
		Description: fmt.Sprintf("Client received refund of ₹%.2f. Reason: %s", utils.PaiseToRupees(refundAmount), req.Reason),
	})

	// ---- 9. Response ----
	return &dto.RefundResponse{
		RefundID:               refundID.String(),
		EscrowID:               escrow.ID.String(),
		MilestonePaymentID:     mp.ID.String(),
		QuestID:                questUUID.String(),
		MilestoneID:            milestoneUUID.String(),
		Status:                 string(models.RefundStatusCompleted),
		RefundAmount:           utils.PaiseToRupees(refundAmount),
		RefundAmountMinor:      refundAmount,
		ClientFeeRetained:      utils.PaiseToRupees(mp.ClientFeeMinor),
		ClientFeeRetainedMinor: mp.ClientFeeMinor,
		Currency:               mp.Currency,
		EscrowStatus:           string(models.EscrowStatusRefunded),
		MilestoneStatus:        string(models.MilestoneStatusRefunded),
		Reason:                 req.Reason,
		CompletedAt:            timePtr(time.Now()),
		Message:                "Refund processed. Milestone amount credited back to your original payment method.",
	}, nil
}

// ============================================================
// GetRefund — read existing refund for milestone
// ============================================================

func (s *RefundService) GetRefund(
	userIDStr, questIDStr, milestoneIDStr string,
) (*dto.RefundResponse, error) {

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrRefQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrRefMilestoneNotFound
	}

	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return nil, ErrRefQuestNotFound
	}
	isClient := quest.ClientID == userUUID
	isFreelancer := quest.AssignedFreelancerID != nil && *quest.AssignedFreelancerID == userUUID
	if !isClient && !isFreelancer {
		return nil, ErrRefUnauthorized
	}

	milestone, err := s.milestoneRepo.FindByID(milestoneUUID)
	if err != nil {
		return nil, ErrRefMilestoneNotFound
	}
	if milestone.QuestID != questUUID {
		return nil, ErrRefMilestoneNotFound
	}

	return s.buildRefundResponseFromMilestone(milestone, questUUID)
}

// ============================================================
// Helpers
// ============================================================

func (s *RefundService) buildRefundResponseFromExisting(
	refund *models.Refund,
	milestone *models.QuestMilestone,
	questUUID uuid.UUID,
) (*dto.RefundResponse, error) {

	mp, err := s.mpRepo.GetByQuestMilestoneID(milestone.ID)
	if err != nil {
		return nil, err
	}
	escrow, err := s.escrowRepo.GetByQuestMilestoneID(milestone.ID)
	if err != nil {
		return nil, err
	}

	resp := &dto.RefundResponse{
		RefundID:               refund.ID.String(),
		EscrowID:               escrow.ID.String(),
		MilestonePaymentID:     mp.ID.String(),
		QuestID:                questUUID.String(),
		MilestoneID:            milestone.ID.String(),
		Status:                 string(refund.Status),
		RefundAmount:           utils.PaiseToRupees(refund.Amount),
		RefundAmountMinor:      refund.Amount,
		ClientFeeRetained:      utils.PaiseToRupees(mp.ClientFeeMinor),
		ClientFeeRetainedMinor: mp.ClientFeeMinor,
		Currency:               refund.Currency,
		EscrowStatus:           string(escrow.Status),
		MilestoneStatus:        string(milestone.Status),
		Reason:                 refund.Reason,
		CompletedAt:            refund.CompletedAt,
		Message:                "Refund already processed (idempotent)",
	}
	return resp, nil
}

func (s *RefundService) buildRefundResponseFromMilestone(
	milestone *models.QuestMilestone,
	questUUID uuid.UUID,
) (*dto.RefundResponse, error) {

	mp, err := s.mpRepo.GetByQuestMilestoneID(milestone.ID)
	if err != nil {
		return nil, ErrRefPaymentNotFound
	}
	escrow, err := s.escrowRepo.GetByQuestMilestoneID(milestone.ID)
	if err != nil {
		return nil, ErrRefEscrowNotFound
	}

	// Find refund row
	refunds, _ := s.refundRepo.ListByPayment(mp.PaymentID)
	var lastRefund *models.Refund
	for i := range refunds {
		if refunds[i].MilestonePaymentID != nil && *refunds[i].MilestonePaymentID == mp.ID {
			lastRefund = &refunds[i]
			break
		}
	}
	if lastRefund == nil {
		return nil, ErrRefNotFound
	}

	return &dto.RefundResponse{
		RefundID:               lastRefund.ID.String(),
		EscrowID:               escrow.ID.String(),
		MilestonePaymentID:     mp.ID.String(),
		QuestID:                questUUID.String(),
		MilestoneID:            milestone.ID.String(),
		Status:                 string(lastRefund.Status),
		RefundAmount:           utils.PaiseToRupees(lastRefund.Amount),
		RefundAmountMinor:      lastRefund.Amount,
		ClientFeeRetained:      utils.PaiseToRupees(mp.ClientFeeMinor),
		ClientFeeRetainedMinor: mp.ClientFeeMinor,
		Currency:               lastRefund.Currency,
		EscrowStatus:           string(escrow.Status),
		MilestoneStatus:        string(milestone.Status),
		Reason:                 lastRefund.Reason,
		CompletedAt:            lastRefund.CompletedAt,
		Message:                "Refund found",
	}, nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}
