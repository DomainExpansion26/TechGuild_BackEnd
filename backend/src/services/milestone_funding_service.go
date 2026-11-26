package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	ErrFundQuestNotFound      = errors.New("quest not found")
	ErrFundMilestoneNotFound  = errors.New("milestone not found")
	ErrFundUnauthorized       = errors.New("only the quest client can fund this milestone")
	ErrFundMilestoneNotAgreed = errors.New("milestone must be agreed before funding")
	ErrFundQuestNotInProgress = errors.New("quest is not in progress")
	ErrFundNoPaymentMethod    = errors.New("no active payment method configured")
	ErrFundProviderFailed     = errors.New("payment provider failed")
	ErrFundAlreadyFunded      = errors.New("milestone already funded")
	ErrFundInvalidSignature   = errors.New("razorpay signature verification failed")
	ErrFundNotPending         = errors.New("funding not in pending state")
	ErrFundMissingCredentials = errors.New("provider credentials missing")

	ErrFundProviderVerifyFailed = errors.New("provider could not verify this payment")
	ErrFundPaymentMismatch      = errors.New("provider payment does not match this charge")
	ErrFundOrderMismatch        = errors.New("provider payment belongs to a different order")
	ErrFundPaymentNotFound      = errors.New("payment not found for this milestone")
)

// ============================================================
// Service
// ============================================================

type MilestoneFundingService struct {
	cfg       *config.Config
	provider  PaymentProvider
	feeSvc    *PlatformFeeService
	ledgerSvc *PaymentLedgerService

	paymentRepo     repository.PaymentRepository
	mpRepo          repository.MilestonePaymentRepository
	escrowRepo      repository.EscrowTransactionRepository
	platformFeeRepo repository.PlatformFeeRepository
	clientPMRepo    repository.ClientPaymentMethodRepository
	questRepo       repository.QuestRepository
	milestoneRepo   repository.QuestMilestoneRepository
	activityRepo    repository.TaskManagementRepository
}

func NewMilestoneFundingService(cfg *config.Config, provider PaymentProvider) *MilestoneFundingService {
	return &MilestoneFundingService{
		cfg:             cfg,
		provider:        provider,
		feeSvc:          NewPlatformFeeService(),
		ledgerSvc:       NewPaymentLedgerService(),
		paymentRepo:     repository.NewPaymentRepository(),
		mpRepo:          repository.NewMilestonePaymentRepository(),
		escrowRepo:      repository.NewEscrowTransactionRepository(),
		platformFeeRepo: repository.NewPlatformFeeRepository(),
		clientPMRepo:    repository.NewClientPaymentMethodRepository(),
		questRepo:       repository.NewQuestRepository(),
		milestoneRepo:   repository.NewQuestMilestoneRepository(),
		activityRepo:    repository.NewTaskManagementRepository(),
	}
}

// ============================================================
// FundMilestone — client initiates funding
//
// Mock (sync): payment captured immediately → milestone work_in_progress
// Razorpay (async): order created → milestone funding_pending
//                    Frontend → Checkout → /fund/confirm (or webhook)
// ============================================================

func (s *MilestoneFundingService) FundMilestone(
	userIDStr, questIDStr, milestoneIDStr string,
	req dto.FundMilestoneBody,
) (*dto.FundMilestoneResponse, error) {

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrFundQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrFundMilestoneNotFound
	}
	if req.IdempotencyKey == "" {
		return nil, errors.New("idempotency_key is required")
	}

	// ---- 1. Load quest + milestone ----
	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return nil, ErrFundQuestNotFound
	}
	if quest.ClientID != userUUID {
		return nil, ErrFundUnauthorized
	}
	if quest.Status != models.QuestStatusInProgress {
		return nil, ErrFundQuestNotInProgress
	}

	milestone, err := s.milestoneRepo.FindByID(milestoneUUID)
	if err != nil {
		return nil, ErrFundMilestoneNotFound
	}
	if milestone.QuestID != questUUID {
		return nil, ErrFundMilestoneNotFound
	}

	// ---- 2. Idempotency FIRST ----
	existing, err := s.paymentRepo.GetByIdempotencyKey(req.IdempotencyKey)
	if err == nil && existing != nil {
		return s.buildFundResponseFromExisting(existing)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Only a LIVE attempt blocks funding. A failed row is history:
	// MarkPaymentFailed puts the milestone back to `agreed` precisely so the
	// client can retry, so refusing here would strand the milestone forever.
	// GetByQuestMilestoneID already filters failed attempts out.
	existingMP, err := s.mpRepo.GetByQuestMilestoneID(milestoneUUID)
	if err == nil && existingMP != nil {
		return nil, ErrFundAlreadyFunded
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// ---- 3. State check ----
	if milestone.Status != models.MilestoneStatusAgreed {
		return nil, ErrFundMilestoneNotAgreed
	}

	if _, err := s.resolvePaymentMethod(userUUID, req.PaymentMethodID); err != nil {
		return nil, err
	}

	// ---- 4. Fee calc ----
	clientRateBps := s.cfg.ClientProcessingFeeBps
	freelancerRateBps := s.cfg.FreelancerProcessingFeeBps
	if clientRateBps == 0 {
		clientRateBps = 500
	}
	if freelancerRateBps == 0 {
		freelancerRateBps = 500
	}

	feeSplit, err := s.feeSvc.CalculateDualFee(
		milestone.AmountMinor, clientRateBps, freelancerRateBps,
	)
	if err != nil {
		return nil, fmt.Errorf("fee calculation failed: %w", err)
	}

	currency := "INR"

	// ---- 5. Provider CreatePayment ----
	providerPayment, err := s.provider.CreatePayment(CreatePaymentRequest{
		UserID:           userUUID.String(),
		QuestID:          questUUID.String(),
		QuestMilestoneID: milestoneUUID.String(),
		Amount:           feeSplit.ClientChargedMinor,
		Currency:         currency,
		Description:      fmt.Sprintf("Milestone funding: %s", milestone.Title),
		IdempotencyKey:   req.IdempotencyKey,
	})
	if err != nil {
		log.Printf("CreatePayment failed for milestone=%s: %v", milestoneUUID, err)
		return nil, ErrFundProviderFailed
	}

	// ---- 6. Provider CreateEscrowHold ----
	providerEscrow, err := s.provider.CreateEscrowHold(EscrowHoldRequest{
		ProviderPaymentID: providerPayment.ProviderPaymentID,
		Amount:            feeSplit.MilestoneAmountMinor,
		Currency:          currency,
		IdempotencyKey:    req.IdempotencyKey + ":escrow",
	})
	if err != nil {
		log.Printf("CreateEscrowHold failed for milestone=%s: %v", milestoneUUID, err)
		return nil, ErrFundProviderFailed
	}

	// ---- 7. DB TX ----
	var (
		paymentID uuid.UUID
		mpID      uuid.UUID
		escrowID  uuid.UUID
	)

	// Decide sync vs async
	isAsync := providerPayment.RequiresConfirmation

	err = postgres.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// Re-verify under a row lock: the pre-flight status check above ran outside any
		// transaction, so without this two concurrent funds could both create records.
		lockedMilestone, lockErr := repository.NewQuestMilestoneRepositoryTx(tx).FindByIDForUpdate(milestoneUUID)
		if lockErr != nil {
			return ErrFundMilestoneNotFound
		}
		if lockedMilestone.Status != models.MilestoneStatusAgreed {
			return ErrFundMilestoneNotAgreed
		}
		if existing, exErr := repository.NewMilestonePaymentRepositoryTx(tx).GetByQuestMilestoneID(milestoneUUID); exErr == nil && existing != nil {
			return ErrFundAlreadyFunded
		}

		// Determine initial statuses
		paymentStatus := models.PaymentStatusCaptured
		escrowStatus := models.EscrowStatusHeld
		mpStatus := models.MilestonePaymentFunded
		milestoneStatus := models.MilestoneStatusWorkInProgress
		var capturedAt *time.Time = &now
		var heldAt *time.Time = &now

		if isAsync {
			paymentStatus = models.PaymentStatusPending
			escrowStatus = models.EscrowStatusPending
			mpStatus = models.MilestonePaymentPending
			milestoneStatus = models.MilestoneStatusFundingPending
			capturedAt = nil
			heldAt = nil
		}

		// 7a. Payment
		providerPaymentID := providerPayment.ProviderPaymentID
		payment := &models.Payment{
			UserID:             userUUID,
			QuestID:            questUUID,
			QuestMilestoneID:   milestoneUUID,
			Provider:           s.provider.Name(),
			Amount:             feeSplit.ClientChargedMinor,
			Currency:           currency,
			SettlementAmount:   feeSplit.ClientChargedMinor,
			SettlementCurrency: currency,
			Status:             paymentStatus,
			IdempotencyKey:     req.IdempotencyKey,
			Description:        fmt.Sprintf("Milestone funding: %s", milestone.Title),
			CapturedAt:         capturedAt,
		}
		if isAsync {
			payment.ProviderOrderID = &providerPaymentID
		} else {
			payment.ProviderPaymentID = &providerPaymentID
		}
		if err := tx.Create(payment).Error; err != nil {
			return fmt.Errorf("create payment: %w", err)
		}
		paymentID = payment.ID

		// 7b. Escrow
		escrow := &models.EscrowTransaction{
			PaymentID:          payment.ID,
			QuestID:            questUUID,
			QuestMilestoneID:   milestoneUUID,
			Provider:           s.provider.Name(),
			ProviderEscrowID:   &providerEscrow.ProviderEscrowID,
			Amount:             feeSplit.MilestoneAmountMinor,
			Currency:           currency,
			SettlementAmount:   feeSplit.MilestoneAmountMinor,
			SettlementCurrency: currency,
			Status:             escrowStatus,
			HeldAt:             heldAt,
		}
		if err := tx.Create(escrow).Error; err != nil {
			return fmt.Errorf("create escrow: %w", err)
		}
		escrowID = escrow.ID

		// 7c. MilestonePayment
		mp := &models.MilestonePayment{
			QuestMilestoneID:     milestoneUUID,
			QuestID:              questUUID,
			PaymentID:            payment.ID,
			EscrowTransactionID:  &escrow.ID,
			MilestoneAmountMinor: feeSplit.MilestoneAmountMinor,
			ClientFeeMinor:       feeSplit.ClientFeeMinor,
			ClientFeeRateBps:     clientRateBps,
			FreelancerFeeMinor:   feeSplit.FreelancerFeeMinor,
			FreelancerFeeRateBps: freelancerRateBps,
			ClientChargedMinor:   feeSplit.ClientChargedMinor,
			FreelancerNetMinor:   feeSplit.FreelancerNetMinor,
			PlatformTotalMinor:   feeSplit.PlatformTotalMinor,
			Currency:             currency,
			FeeType:              models.FeeTypePercentage,
			Status:               mpStatus,
		}
		if err := tx.Create(mp).Error; err != nil {
			return fmt.Errorf("create milestone_payment: %w", err)
		}
		mpID = mp.ID

		// 7d. PlatformFee client side (settled if sync, pending if async)
		clientFeeStatus := models.PlatformFeeSettled
		var clientFeeSettledAt *time.Time = &now
		if isAsync {
			clientFeeStatus = models.PlatformFeePending
			clientFeeSettledAt = nil
		}
		clientFeeRow := &models.PlatformFee{
			PaymentID:          payment.ID,
			MilestonePaymentID: mp.ID,
			QuestID:            questUUID,
			Side:               models.PlatformFeeSideClient,
			FeeType:            models.FeeTypePercentage,
			RateBps:            clientRateBps,
			Amount:             feeSplit.ClientFeeMinor,
			Currency:           currency,
			Status:             clientFeeStatus,
			SettledAt:          clientFeeSettledAt,
		}
		if err := tx.Create(clientFeeRow).Error; err != nil {
			return fmt.Errorf("create client platform_fee: %w", err)
		}

		// 7e. PlatformFee freelancer side (always pending until release)
		freelancerFeeRow := &models.PlatformFee{
			PaymentID:          payment.ID,
			MilestonePaymentID: mp.ID,
			QuestID:            questUUID,
			Side:               models.PlatformFeeSideFreelancer,
			FeeType:            models.FeeTypePercentage,
			RateBps:            freelancerRateBps,
			Amount:             feeSplit.FreelancerFeeMinor,
			Currency:           currency,
			Status:             models.PlatformFeePending,
		}
		if err := tx.Create(freelancerFeeRow).Error; err != nil {
			return fmt.Errorf("create freelancer platform_fee: %w", err)
		}

		// 7f. Ledger entries
		msPtr := milestoneUUID
		paymentIDPtr := payment.ID
		questIDPtr := questUUID

		entries := []LedgerEntry{
			{
				PaymentID:        &paymentIDPtr,
				QuestID:          &questIDPtr,
				QuestMilestoneID: &msPtr,
				TransactionType:  models.TxTypePayment,
				Amount:           feeSplit.ClientChargedMinor,
				Currency:         currency,
				Provider:         s.provider.Name(),
				Reference:        providerPayment.ProviderPaymentID,
				IdempotencyKey:   req.IdempotencyKey + ":ledger:payment",
			},
		}

		if !isAsync {
			// Sync path — full ledger now
			entries = append(entries,
				LedgerEntry{
					PaymentID:        &paymentIDPtr,
					QuestID:          &questIDPtr,
					QuestMilestoneID: &msPtr,
					TransactionType:  models.TxTypeCapture,
					Amount:           feeSplit.ClientChargedMinor,
					Currency:         currency,
					Provider:         s.provider.Name(),
					Reference:        providerPayment.ProviderPaymentID,
					IdempotencyKey:   req.IdempotencyKey + ":ledger:capture",
				},
				LedgerEntry{
					PaymentID:        &paymentIDPtr,
					QuestID:          &questIDPtr,
					QuestMilestoneID: &msPtr,
					TransactionType:  models.TxTypePlatformFeeClient,
					Amount:           feeSplit.ClientFeeMinor,
					Currency:         currency,
					Provider:         s.provider.Name(),
					Reference:        mp.ID.String(),
					IdempotencyKey:   req.IdempotencyKey + ":ledger:platform_fee_client",
				},
				LedgerEntry{
					PaymentID:        &paymentIDPtr,
					QuestID:          &questIDPtr,
					QuestMilestoneID: &msPtr,
					TransactionType:  models.TxTypeEscrowHold,
					Amount:           feeSplit.MilestoneAmountMinor,
					Currency:         currency,
					Provider:         s.provider.Name(),
					Reference:        providerEscrow.ProviderEscrowID,
					IdempotencyKey:   req.IdempotencyKey + ":ledger:escrow_hold",
				},
			)
		}

		for _, e := range entries {
			if err := s.ledgerSvc.Append(tx, e); err != nil {
				return fmt.Errorf("append ledger entry: %w", err)
			}
		}

		// 7g. Update milestone status
		updates := map[string]interface{}{
			"status":     milestoneStatus,
			"updated_at": now,
		}
		if !isAsync {
			updates["funded_at"] = now
		}
		if err := tx.Model(&models.QuestMilestone{}).
			Where("id = ?", milestoneUUID).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("update milestone status: %w", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("Fund TX failed for milestone=%s: %v", milestoneUUID, err)
		return nil, fmt.Errorf("failed to persist funding: %w", err)
	}

	// ---- 8. Activity log ----
	if !isAsync {
		_ = s.activityRepo.CreateActivity(&models.QuestActivity{
			QuestID:     questUUID,
			MilestoneID: &milestoneUUID,
			ActorID:     userUUID,
			ActionType:  models.ActivityMilestoneFunded,
			Title:       fmt.Sprintf("Milestone '%s' Funded", milestone.Title),
			Description: fmt.Sprintf("Client funded ₹%.2f into escrow", utils.PaiseToRupees(feeSplit.MilestoneAmountMinor)),
		})
	}

	// ---- 9. Response ----
	respStatus := string(models.MilestonePaymentFunded)
	msg := "Milestone funded successfully. Funds held in escrow."
	if isAsync {
		respStatus = "funding_pending"
		msg = "Order created. Complete payment via Razorpay Checkout."
	}

	resp := &dto.FundMilestoneResponse{
		PaymentID:            paymentID.String(),
		MilestonePaymentID:   mpID.String(),
		EscrowID:             escrowID.String(),
		Status:               respStatus,
		RequiresConfirmation: isAsync,
		MilestoneAmount:      utils.PaiseToRupees(feeSplit.MilestoneAmountMinor),
		ClientFee:            utils.PaiseToRupees(feeSplit.ClientFeeMinor),
		FreelancerFee:        utils.PaiseToRupees(feeSplit.FreelancerFeeMinor),
		ClientCharged:        utils.PaiseToRupees(feeSplit.ClientChargedMinor),
		FreelancerNet:        utils.PaiseToRupees(feeSplit.FreelancerNetMinor),
		PlatformTotal:        utils.PaiseToRupees(feeSplit.PlatformTotalMinor),
		MilestoneAmountMinor: feeSplit.MilestoneAmountMinor,
		ClientFeeMinor:       feeSplit.ClientFeeMinor,
		ClientChargedMinor:   feeSplit.ClientChargedMinor,
		Currency:             currency,
		ClientFeeRateBps:     clientRateBps,
		FreelancerFeeRateBps: freelancerRateBps,
		Message:              msg,
	}

	if isAsync {
		resp.ProviderOrderID = providerPayment.ProviderPaymentID
		resp.ClientKeyID = providerPayment.ClientKeyID
	}

	return resp, nil
}

// ============================================================
// ConfirmFunding — called by frontend after Razorpay Checkout
// Verifies signature, updates state, marks captured.
// ============================================================

func (s *MilestoneFundingService) ConfirmFunding(
	userIDStr, questIDStr, milestoneIDStr string,
	req dto.ConfirmFundingBody,
) (*dto.FundMilestoneResponse, error) {

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrFundQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrFundMilestoneNotFound
	}

	// ---- 1. Auth ----
	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return nil, ErrFundQuestNotFound
	}
	if quest.ClientID != userUUID {
		return nil, ErrFundUnauthorized
	}

	// ---- 2. Verify signature ----
	if s.cfg.RazorpayKeySecret == "" {
		return nil, ErrFundMissingCredentials
	}
	if !verifyRazorpayPaymentSignature(
		req.RazorpayOrderID,
		req.RazorpayPaymentID,
		req.RazorpaySignature,
		s.cfg.RazorpayKeySecret,
	) {
		return nil, ErrFundInvalidSignature
	}

	// ---- 3. Match payment to this milestone ----
	payment, err := s.paymentRepo.GetByProviderOrderID("razorpay", req.RazorpayOrderID)
	if err != nil {
		return nil, errors.New("payment not found for this order")
	}
	if payment.QuestMilestoneID != milestoneUUID {
		return nil, ErrFundMilestoneNotFound
	}

	// ---- 4. Delegate to shared logic ----
	return s.MarkPaymentCaptured(req.RazorpayOrderID, req.RazorpayPaymentID)
}

// ============================================================
// MarkPaymentCaptured — shared by /fund/confirm AND webhook
//
// Idempotent. No signature check here (caller ensures trust).
// ============================================================

func (s *MilestoneFundingService) MarkPaymentCaptured(
	orderID, paymentID string,
) (*dto.FundMilestoneResponse, error) {

	payment, err := s.paymentRepo.GetByProviderOrderID("razorpay", orderID)
	if err != nil {
		return nil, errors.New("payment not found for this order")
	}

	// Idempotency — already captured?
	if payment.Status == models.PaymentStatusCaptured {
		return s.buildFundResponseFromExisting(payment)
	}
	// `failed` is recoverable: Razorpay allows several payment attempts against
	// one order, so an attempt that failed and was then retried successfully on
	// the SAME order must still be able to fund the milestone.
	if payment.Status != models.PaymentStatusPending && payment.Status != models.PaymentStatusFailed {
		return nil, ErrFundNotPending
	}

	// Never trust the caller's word (or a bare webhook payload) that the money
	// actually moved: ask the provider. Covers /fund/confirm, the webhook and
	// the reconciliation job, since all three route through here.
	if err := s.verifyProviderCapture(payment, paymentID); err != nil {
		return nil, err
	}

	milestoneUUID := payment.QuestMilestoneID
	questUUID := payment.QuestID

	// DB TX
	err = postgres.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// Re-read under the row lock: the guards above ran outside the tx, so a
		// concurrent payment.failed could otherwise kill this attempt between the
		// check and the writes.
		locked, lockErr := repository.NewPaymentRepositoryTx(tx).GetByIDForUpdate(payment.ID)
		if lockErr != nil {
			return lockErr
		}
		if locked.Status == models.PaymentStatusCaptured {
			return nil // another delivery already captured it
		}
		if locked.Status != models.PaymentStatusPending && locked.Status != models.PaymentStatusFailed {
			return ErrFundNotPending
		}

		// This attempt's own row: a milestone can hold several attempts once one
		// has failed, so quest_milestone_id would pick the wrong one.
		mp, err := repository.NewMilestonePaymentRepositoryTx(tx).GetByPaymentID(payment.ID)
		if err != nil {
			return err
		}

		// Update Payment
		if err := tx.Model(&models.Payment{}).
			Where("id = ?", payment.ID).
			Updates(map[string]interface{}{
				"status":              models.PaymentStatusCaptured,
				"provider_payment_id": paymentID,
				"captured_at":         now,
			}).Error; err != nil {
			return err
		}

		// Escrow pending (or a recovered failure) → held, for THIS attempt only
		if err := tx.Model(&models.EscrowTransaction{}).
			Where("payment_id = ? AND status IN ?", payment.ID,
				[]models.EscrowStatus{models.EscrowStatusPending, models.EscrowStatusFailed}).
			Updates(map[string]interface{}{
				"status":  models.EscrowStatusHeld,
				"held_at": now,
			}).Error; err != nil {
			return err
		}

		// MilestonePayment pending (or a recovered failure) → funded
		if err := tx.Model(&models.MilestonePayment{}).
			Where("payment_id = ? AND status IN ?", payment.ID,
				[]models.MilestonePaymentStatus{models.MilestonePaymentPending, models.MilestonePaymentFailed}).
			Update("status", models.MilestonePaymentFunded).Error; err != nil {
			return err
		}

		// PlatformFee client side → settled, for THIS attempt's milestone payment.
		// The scope must be keyed by payment_id: quest_milestone_id can now match
		// several rows (failed attempts) and a scalar subquery would error out.
		if err := tx.Model(&models.PlatformFee{}).
			Where("milestone_payment_id = ? AND side = ?", mp.ID, models.PlatformFeeSideClient).
			Updates(map[string]interface{}{
				"status":     models.PlatformFeeSettled,
				"settled_at": now,
			}).Error; err != nil {
			return err
		}

		// Milestone funding_pending → work_in_progress
		if err := tx.Model(&models.QuestMilestone{}).
			Where("id = ?", milestoneUUID).
			Updates(map[string]interface{}{
				"status":    models.MilestoneStatusWorkInProgress,
				"funded_at": now,
			}).Error; err != nil {
			return err
		}

		// Ledger entries — escrow is looked up through THIS attempt's own link
		if mp.EscrowTransactionID == nil {
			return fmt.Errorf("milestone payment %s has no escrow row", mp.ID)
		}
		escrow, err := repository.NewEscrowTransactionRepositoryTx(tx).GetByID(*mp.EscrowTransactionID)
		if err != nil {
			return err
		}

		msPtr := milestoneUUID
		paymentIDPtr := payment.ID
		questIDPtr := questUUID

		var escrowRef string
		if escrow.ProviderEscrowID != nil {
			escrowRef = *escrow.ProviderEscrowID
		}

		entries := []LedgerEntry{
			{
				PaymentID:        &paymentIDPtr,
				QuestID:          &questIDPtr,
				QuestMilestoneID: &msPtr,
				TransactionType:  models.TxTypeCapture,
				Amount:           mp.ClientChargedMinor,
				Currency:         mp.Currency,
				Provider:         "razorpay",
				Reference:        paymentID,
				IdempotencyKey:   payment.IdempotencyKey + ":ledger:capture",
			},
			{
				PaymentID:        &paymentIDPtr,
				QuestID:          &questIDPtr,
				QuestMilestoneID: &msPtr,
				TransactionType:  models.TxTypePlatformFeeClient,
				Amount:           mp.ClientFeeMinor,
				Currency:         mp.Currency,
				Provider:         "razorpay",
				Reference:        mp.ID.String(),
				IdempotencyKey:   payment.IdempotencyKey + ":ledger:platform_fee_client",
			},
			{
				PaymentID:        &paymentIDPtr,
				QuestID:          &questIDPtr,
				QuestMilestoneID: &msPtr,
				TransactionType:  models.TxTypeEscrowHold,
				Amount:           mp.MilestoneAmountMinor,
				Currency:         mp.Currency,
				Provider:         "razorpay",
				Reference:        escrowRef,
				IdempotencyKey:   payment.IdempotencyKey + ":ledger:escrow_hold",
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
		return nil, fmt.Errorf("failed to mark captured: %w", err)
	}

	_ = s.activityRepo.CreateActivity(&models.QuestActivity{
		QuestID:     questUUID,
		MilestoneID: &milestoneUUID,
		ActorID:     payment.UserID,
		ActionType:  models.ActivityMilestoneFunded,
		Title:       "Milestone Funding Confirmed",
		Description: fmt.Sprintf("Payment %s captured", paymentID),
	})

	reloaded, err := s.paymentRepo.GetByID(payment.ID)
	if err != nil {
		return nil, err
	}
	return s.buildFundResponseFromExisting(reloaded)
}

// ============================================================
// MarkPaymentFailed — webhook or explicit
// ============================================================

func (s *MilestoneFundingService) MarkPaymentFailed(orderID, reason string) error {
	payment, err := s.paymentRepo.GetByProviderOrderID("razorpay", orderID)
	if err != nil {
		return nil // orphan — ignore
	}

	if payment.Status == models.PaymentStatusFailed ||
		payment.Status == models.PaymentStatusCaptured {
		return nil // already terminal
	}

	return postgres.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// Re-read under the row lock: the terminal check above ran outside the tx,
		// so a concurrent payment.captured could otherwise land in between.
		locked, lockErr := repository.NewPaymentRepositoryTx(tx).GetByIDForUpdate(payment.ID)
		if lockErr != nil {
			return lockErr
		}
		if locked.Status == models.PaymentStatusFailed ||
			locked.Status == models.PaymentStatusCaptured {
			return nil // another delivery already settled this attempt
		}

		if err := tx.Model(&models.Payment{}).
			Where("id = ?", payment.ID).
			Updates(map[string]interface{}{
				"status":         models.PaymentStatusFailed,
				"failed_at":      now,
				"failure_reason": reason,
			}).Error; err != nil {
			return err
		}

		// Scope downstream rows to THIS attempt. A milestone may now hold several
		// attempts, so matching on quest_milestone_id would knock a newer live
		// attempt's escrow/payment back to failed.
		if err := tx.Model(&models.EscrowTransaction{}).
			Where("payment_id = ? AND status = ?", payment.ID, models.EscrowStatusPending).
			Update("status", models.EscrowStatusFailed).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.MilestonePayment{}).
			Where("payment_id = ? AND status = ?", payment.ID, models.MilestonePaymentPending).
			Update("status", models.MilestonePaymentFailed).Error; err != nil {
			return err
		}

		// Hand the milestone back to `agreed` only when no other live attempt
		// exists — a stale failure must not unfund a newer retry.
		var liveOther int64
		if err := tx.Model(&models.Payment{}).
			Where("quest_milestone_id = ? AND id <> ? AND status IN ?",
				payment.QuestMilestoneID, payment.ID,
				[]models.PaymentStatus{models.PaymentStatusPending, models.PaymentStatusCaptured}).
			Count(&liveOther).Error; err != nil {
			return err
		}
		if liveOther == 0 {
			if err := tx.Model(&models.QuestMilestone{}).
				Where("id = ?", payment.QuestMilestoneID).
				Update("status", models.MilestoneStatusAgreed).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// ============================================================
// GetMilestonePayment — read-only
// ============================================================

func (s *MilestoneFundingService) GetMilestonePayment(
	userIDStr, questIDStr, milestoneIDStr string,
) (*dto.MilestonePaymentDetailResponse, error) {

	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, errors.New("invalid user id")
	}
	questUUID, err := uuid.Parse(questIDStr)
	if err != nil {
		return nil, ErrFundQuestNotFound
	}
	milestoneUUID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, ErrFundMilestoneNotFound
	}

	quest, err := s.questRepo.FindByID(questUUID)
	if err != nil {
		return nil, ErrFundQuestNotFound
	}

	isClient := quest.ClientID == userUUID
	isFreelancer := quest.AssignedFreelancerID != nil && *quest.AssignedFreelancerID == userUUID
	if !isClient && !isFreelancer {
		return nil, ErrFundUnauthorized
	}

	mp, err := s.mpRepo.GetByQuestMilestoneID(milestoneUUID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Fall back to the most recent attempt, failed ones included, so the
		// client can still see why the latest funding attempt failed.
		mp, err = s.mpRepo.GetLatestByQuestMilestoneID(milestoneUUID)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFundPaymentNotFound
		}
		return nil, err
	}

	payment, err := s.paymentRepo.GetByID(mp.PaymentID)
	if err != nil {
		return nil, err
	}

	// Attempt-scoped: a failed attempt's escrow row is status=failed, which
	// GetByQuestMilestoneID filters out.
	if mp.EscrowTransactionID == nil {
		return nil, ErrFundPaymentNotFound
	}
	escrow, err := s.escrowRepo.GetByID(*mp.EscrowTransactionID)
	if err != nil {
		return nil, err
	}

	return &dto.MilestonePaymentDetailResponse{
		PaymentID:            payment.ID.String(),
		MilestonePaymentID:   mp.ID.String(),
		EscrowID:             escrow.ID.String(),
		QuestID:              questUUID.String(),
		QuestMilestoneID:     milestoneUUID.String(),
		MilestoneAmount:      utils.PaiseToRupees(mp.MilestoneAmountMinor),
		ClientFee:            utils.PaiseToRupees(mp.ClientFeeMinor),
		FreelancerFee:        utils.PaiseToRupees(mp.FreelancerFeeMinor),
		ClientCharged:        utils.PaiseToRupees(mp.ClientChargedMinor),
		FreelancerNet:        utils.PaiseToRupees(mp.FreelancerNetMinor),
		PlatformTotal:        utils.PaiseToRupees(mp.PlatformTotalMinor),
		MilestoneAmountMinor: mp.MilestoneAmountMinor,
		ClientFeeMinor:       mp.ClientFeeMinor,
		FreelancerFeeMinor:   mp.FreelancerFeeMinor,
		ClientChargedMinor:   mp.ClientChargedMinor,
		FreelancerNetMinor:   mp.FreelancerNetMinor,
		PlatformTotalMinor:   mp.PlatformTotalMinor,
		Currency:             mp.Currency,
		ClientFeeRateBps:     mp.ClientFeeRateBps,
		FreelancerFeeRateBps: mp.FreelancerFeeRateBps,
		Status:               string(mp.Status),
		PaymentStatus:        string(payment.Status),
		EscrowStatus:         string(escrow.Status),
		CapturedAt:           payment.CapturedAt,
		EscrowHeldAt:         escrow.HeldAt,
		ReleasedAt:           mp.ReleasedAt,
		CreatedAt:            mp.CreatedAt,
		UpdatedAt:            mp.UpdatedAt,
	}, nil
}

// ============================================================
// Helpers
// ============================================================

// verifyProviderCapture asks the provider whether this payment is genuinely
// captured, genuinely ours, and genuinely for the recorded amount.
//
// ponytail: non-razorpay providers skip this entirely — the mock is synchronous
// and exposes no provider-side payment state to verify against. Swap in a mock
// payment store if the mock ever stops being synchronous.
func (s *MilestoneFundingService) verifyProviderCapture(payment *models.Payment, paymentID string) error {
	if s.provider == nil || s.provider.Name() != "razorpay" {
		return nil
	}

	res, err := s.provider.GetPayment(paymentID)
	if errors.Is(err, ErrPaymentNotCaptured) {
		// Authorized but not captured yet — capture it ourselves.
		res, err = s.provider.CapturePayment(paymentID, payment.Amount)
	}
	if err != nil {
		log.Printf("provider verify failed for payment=%s: %v", paymentID, err)
		return ErrFundProviderVerifyFailed
	}
	if res == nil || res.Status != "captured" {
		log.Printf("provider payment %s is not captured", paymentID)
		return ErrFundProviderVerifyFailed
	}

	// The strong check: a client can hand us any valid payment id, so make sure
	// this payment actually belongs to the order WE created.
	if res.OrderID != "" && payment.ProviderOrderID != nil && res.OrderID != *payment.ProviderOrderID {
		log.Printf("provider payment %s belongs to order %s, expected %s", paymentID, res.OrderID, *payment.ProviderOrderID)
		return ErrFundOrderMismatch
	}
	if res.Amount > 0 && res.Amount != payment.Amount {
		log.Printf("provider payment %s amount=%d, expected %d", paymentID, res.Amount, payment.Amount)
		return ErrFundPaymentMismatch
	}
	if res.Currency != "" && res.Currency != payment.Currency {
		log.Printf("provider payment %s currency=%s, expected %s", paymentID, res.Currency, payment.Currency)
		return ErrFundPaymentMismatch
	}
	return nil
}

func (s *MilestoneFundingService) resolvePaymentMethod(
	userID uuid.UUID,
	providedID string,
) (*models.ClientPaymentMethod, error) {

	if providedID != "" {
		pmUUID, err := uuid.Parse(providedID)
		if err != nil {
			return nil, errors.New("invalid payment_method_id")
		}
		pm, err := s.clientPMRepo.GetByIDAndUser(pmUUID, userID)
		if err != nil {
			return nil, errors.New("payment method not found")
		}
		if pm.Status != models.ClientPaymentMethodStatusActive {
			return nil, errors.New("payment method is not active")
		}
		return pm, nil
	}

	pms, err := s.clientPMRepo.ListByUser(userID)
	if err != nil {
		return nil, err
	}
	for _, pm := range pms {
		if pm.IsPrimary && pm.Status == models.ClientPaymentMethodStatusActive {
			return &pm, nil
		}
	}
	for _, pm := range pms {
		if pm.Status == models.ClientPaymentMethodStatusActive {
			return &pm, nil
		}
	}
	return nil, ErrFundNoPaymentMethod
}

func (s *MilestoneFundingService) buildFundResponseFromExisting(
	payment *models.Payment,
) (*dto.FundMilestoneResponse, error) {

	// Attempt-scoped: this builds the response for ONE payment, and a milestone
	// may hold several attempts — GetByQuestMilestoneID would return the live one.
	mp, err := s.mpRepo.GetByPaymentID(payment.ID)
	if err != nil {
		return nil, err
	}
	if mp.EscrowTransactionID == nil {
		return nil, fmt.Errorf("milestone payment %s has no escrow row", mp.ID)
	}
	escrow, err := s.escrowRepo.GetByID(*mp.EscrowTransactionID)
	if err != nil {
		return nil, err
	}

	respStatus := string(mp.Status)
	msg := "Milestone already funded (idempotent)"
	if payment.Status == models.PaymentStatusPending {
		respStatus = "funding_pending"
		msg = "Funding pending — complete checkout or call /fund/confirm"
	}

	resp := &dto.FundMilestoneResponse{
		PaymentID:            payment.ID.String(),
		MilestonePaymentID:   mp.ID.String(),
		EscrowID:             escrow.ID.String(),
		Status:               respStatus,
		MilestoneAmount:      utils.PaiseToRupees(mp.MilestoneAmountMinor),
		ClientFee:            utils.PaiseToRupees(mp.ClientFeeMinor),
		FreelancerFee:        utils.PaiseToRupees(mp.FreelancerFeeMinor),
		ClientCharged:        utils.PaiseToRupees(mp.ClientChargedMinor),
		FreelancerNet:        utils.PaiseToRupees(mp.FreelancerNetMinor),
		PlatformTotal:        utils.PaiseToRupees(mp.PlatformTotalMinor),
		MilestoneAmountMinor: mp.MilestoneAmountMinor,
		ClientFeeMinor:       mp.ClientFeeMinor,
		ClientChargedMinor:   mp.ClientChargedMinor,
		Currency:             mp.Currency,
		ClientFeeRateBps:     mp.ClientFeeRateBps,
		FreelancerFeeRateBps: mp.FreelancerFeeRateBps,
		Message:              msg,
	}

	if payment.ProviderOrderID != nil {
		resp.ProviderOrderID = *payment.ProviderOrderID
		resp.ClientKeyID = s.cfg.RazorpayKeyID
		resp.RequiresConfirmation = payment.Status == models.PaymentStatusPending
	}

	return resp, nil
}

// verifyRazorpayPaymentSignature validates Razorpay checkout callback.
//
// Per Razorpay docs:
//
//	data = razorpay_order_id + "|" + razorpay_payment_id
//	expected = HMAC_SHA256(data, key_secret)
func verifyRazorpayPaymentSignature(orderID, paymentID, signature, keySecret string) bool {
	if orderID == "" || paymentID == "" || signature == "" || keySecret == "" {
		return false
	}
	data := orderID + "|" + paymentID
	h := hmac.New(sha256.New, []byte(keySecret))
	h.Write([]byte(data))
	expected := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
