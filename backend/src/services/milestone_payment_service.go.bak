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

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrMilestoneNotFound        = errors.New("milestone not found")
	ErrMilestoneNotFunded       = errors.New("milestone is not funded")
	ErrMilestoneAlreadyFunded   = errors.New("milestone already funded")
	ErrMilestoneAlreadyReleased = errors.New("milestone already released")
	ErrUnauthorizedMilestone    = errors.New("not authorized for this milestone")
	ErrContractInactive         = errors.New("contract is not active")
	ErrFundingFailed            = errors.New("milestone funding failed")
)

// Default platform fee until config is wired.
// 500 bps = 5.00%
const defaultFeeRateBps int64 = 500

type MilestonePaymentService struct {
	cfg       *config.Config
	provider  PaymentProvider
	feeSvc    *PlatformFeeService
	ledgerSvc *PaymentLedgerService

	paymentRepo repository.PaymentRepository
	mpRepo      repository.MilestonePaymentRepository
	escrowRepo  repository.EscrowTransactionRepository

	milestoneRepo    *repository.MilestoneRepository
	contractRepo     *repository.ContractRepository
	payoutMethodRepo repository.PayoutMethodRepository
}

func NewMilestonePaymentService(
	cfg *config.Config,
	provider PaymentProvider,
) *MilestonePaymentService {
	return &MilestonePaymentService{
		cfg:              cfg,
		provider:         provider,
		feeSvc:           NewPlatformFeeService(),
		ledgerSvc:        NewPaymentLedgerService(),
		paymentRepo:      repository.NewPaymentRepository(),
		mpRepo:           repository.NewMilestonePaymentRepository(),
		escrowRepo:       repository.NewEscrowTransactionRepository(),
		milestoneRepo:    repository.NewMilestoneRepository(),
		contractRepo:     repository.NewContractRepository(),
		payoutMethodRepo: repository.NewPayoutMethodRepository(),
	}
}

// ============================================================
// FundMilestone
// ============================================================
//
// Flow:
//  1. Validate milestone + contract + auth
//  2. Idempotency checks
//  3. Calculate platform fee (rate LOCKED at funding)
//  4. Provider: CreatePayment + CreateEscrowHold (BEFORE tx)
//  5. DB tx: Payment, Escrow, MilestonePayment, PlatformFee, Ledger entries
//  6. Return
//
// Provider calls happen OUTSIDE the DB tx (spec rule).
func (s *MilestonePaymentService) FundMilestone(
	userIDStr, projectIDStr, milestoneIDStr string,
	req dto.FundMilestoneRequest,
) (*dto.FundMilestoneResponse, error) {

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid project id")
	}
	milestoneID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid milestone id")
	}
	if req.IdempotencyKey == "" {
		return nil, fmt.Errorf("idempotency_key is required")
	}

	// ---- 1. Load + auth ----
	milestone, err := s.milestoneRepo.FindByID(milestoneID)
	if err != nil {
		return nil, ErrMilestoneNotFound
	}
	contract, err := s.contractRepo.FindByID(milestone.ContractID)
	if err != nil {
		return nil, errors.New("contract not found for milestone")
	}
	if contract.ProjectID != projectID {
		return nil, ErrMilestoneNotFound
	}
	if contract.ClientID != userID {
		return nil, ErrUnauthorizedMilestone
	}
	if contract.Status != models.ContractActive {
		return nil, ErrContractInactive
	}

	// ---- 2. Idempotency ----
	existing, err := s.paymentRepo.GetByIdempotencyKey(req.IdempotencyKey)
	if err == nil && existing != nil {
		return s.buildFundResponseFromExisting(existing, milestoneID)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	existingMP, err := s.mpRepo.GetByMilestoneID(milestoneID)
	if err == nil && existingMP != nil {
		switch existingMP.Status {
		case models.MilestonePaymentFunded,
			models.MilestonePaymentAwaitingApproval,
			models.MilestonePaymentApproved,
			models.MilestonePaymentReleasePending,
			models.MilestonePaymentReleased:
			return nil, ErrMilestoneAlreadyFunded
		}
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// ---- 3. Fee calculation (LOCKED) ----
	grossAmount := milestone.Amount
	if grossAmount <= 0 {
		return nil, errors.New("milestone amount must be positive")
	}
	feeType := models.FeeTypePercentage
	rateBps := defaultFeeRateBps
	platformFee, freelancerAmount, err := s.feeSvc.CalculateFee(
		grossAmount, feeType, rateBps, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("fee calculation failed: %w", err)
	}

	currency := contract.Currency
	if currency == "" {
		currency = "INR"
	}

	// ---- 4. Provider calls (outside tx) ----
	providerPayment, err := s.provider.CreatePayment(CreatePaymentRequest{
		UserID:         userID.String(),
		ProjectID:      projectID.String(),
		ContractID:     contract.ID.String(),
		MilestoneID:    milestoneID.String(),
		Amount:         grossAmount,
		Currency:       currency,
		Description:    fmt.Sprintf("Milestone funding: %s", milestone.Title),
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		log.Printf("provider CreatePayment failed for milestone=%s: %v", milestoneID, err)
		return nil, ErrFundingFailed
	}

	providerEscrow, err := s.provider.CreateEscrowHold(EscrowHoldRequest{
		ProviderPaymentID: providerPayment.ProviderPaymentID,
		Amount:            grossAmount,
		Currency:          currency,
		IdempotencyKey:    req.IdempotencyKey + ":escrow",
	})
	if err != nil {
		log.Printf("provider CreateEscrowHold failed for milestone=%s: %v", milestoneID, err)
		return nil, ErrFundingFailed
	}

	// ---- 5. DB tx ----
	var (
		paymentID uuid.UUID
		mpID      uuid.UUID
		escrowID  uuid.UUID
	)

	err = postgres.DB.Transaction(func(tx *gorm.DB) error {
		txPayment := repository.NewPaymentRepositoryTx(tx)
		txEscrow := repository.NewEscrowTransactionRepositoryTx(tx)
		txMP := repository.NewMilestonePaymentRepositoryTx(tx)

		now := time.Now()

		// 5a. Payment
		payment := &models.Payment{
			UserID:             userID,
			ProjectID:          projectID,
			ContractID:         contract.ID,
			MilestoneID:        milestoneID,
			Provider:           s.provider.Name(),
			ProviderPaymentID:  &providerPayment.ProviderPaymentID,
			Amount:             grossAmount,
			Currency:           currency,
			SettlementAmount:   grossAmount,
			SettlementCurrency: currency,
			Status:             models.PaymentStatusCaptured,
			IdempotencyKey:     req.IdempotencyKey,
			Description:        fmt.Sprintf("Milestone funding: %s", milestone.Title),
			CapturedAt:         &now,
		}
		if err := txPayment.Create(payment); err != nil {
			return fmt.Errorf("create payment: %w", err)
		}
		paymentID = payment.ID

		// 5b. Escrow
		escrow := &models.EscrowTransaction{
			PaymentID:          payment.ID,
			ProjectID:          projectID,
			ContractID:         contract.ID,
			MilestoneID:        milestoneID,
			Provider:           s.provider.Name(),
			ProviderEscrowID:   &providerEscrow.ProviderEscrowID,
			Amount:             grossAmount,
			Currency:           currency,
			SettlementAmount:   grossAmount,
			SettlementCurrency: currency,
			Status:             models.EscrowStatusHeld,
			HeldAt:             &now,
		}
		if err := txEscrow.Create(escrow); err != nil {
			return fmt.Errorf("create escrow: %w", err)
		}
		escrowID = escrow.ID

		// 5c. MilestonePayment (fee LOCKED)
		mp := &models.MilestonePayment{
			MilestoneID:         milestoneID,
			ContractID:          contract.ID,
			ProjectID:           projectID,
			PaymentID:           payment.ID,
			EscrowTransactionID: &escrow.ID,
			GrossAmount:         grossAmount,
			PlatformFee:         platformFee,
			FreelancerAmount:    freelancerAmount,
			Currency:            currency,
			FeeRateBps:          rateBps,
			FeeType:             feeType,
			Status:              models.MilestonePaymentFunded,
		}
		if err := txMP.Create(mp); err != nil {
			return fmt.Errorf("create milestone payment: %w", err)
		}
		mpID = mp.ID

		// 5d. PlatformFee record (pending — will be settled on release)
		pf := &models.PlatformFee{
			PaymentID:          payment.ID,
			MilestonePaymentID: mp.ID,
			ProjectID:          projectID,
			ContractID:         contract.ID,
			FeeType:            feeType,
			RateBps:            rateBps,
			FixedAmount:        0,
			Amount:             platformFee,
			Currency:           currency,
			Status:             models.PlatformFeePending,
		}
		if err := s.feeSvc.RecordFee(tx, pf); err != nil {
			return fmt.Errorf("record platform fee: %w", err)
		}

		// 5e. Ledger entries
		msPtr := milestoneID
		entries := []LedgerEntry{
			{
				PaymentID:       payment.ID,
				ProjectID:       projectID,
				ContractID:      contract.ID,
				MilestoneID:     &msPtr,
				TransactionType: models.TxTypePayment,
				Amount:          grossAmount,
				Currency:        currency,
				Provider:        s.provider.Name(),
				Reference:       providerPayment.ProviderPaymentID,
				IdempotencyKey:  req.IdempotencyKey + ":ledger:payment",
			},
			{
				PaymentID:       payment.ID,
				ProjectID:       projectID,
				ContractID:      contract.ID,
				MilestoneID:     &msPtr,
				TransactionType: models.TxTypeCapture,
				Amount:          grossAmount,
				Currency:        currency,
				Provider:        s.provider.Name(),
				Reference:       providerPayment.ProviderPaymentID,
				IdempotencyKey:  req.IdempotencyKey + ":ledger:capture",
			},
			{
				PaymentID:       payment.ID,
				ProjectID:       projectID,
				ContractID:      contract.ID,
				MilestoneID:     &msPtr,
				TransactionType: models.TxTypeEscrowHold,
				Amount:          grossAmount,
				Currency:        currency,
				Provider:        s.provider.Name(),
				Reference:       providerEscrow.ProviderEscrowID,
				IdempotencyKey:  req.IdempotencyKey + ":ledger:escrow_hold",
			},
		}
		for _, e := range entries {
			if err := s.ledgerSvc.Append(tx, e); err != nil {
				return fmt.Errorf("append ledger entry: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("milestone fund tx failed: %v", err)
		return nil, fmt.Errorf("failed to persist funding: %w", err)
	}

	// ---- 6. Response ----
	return &dto.FundMilestoneResponse{
		PaymentID:          paymentID.String(),
		MilestonePaymentID: mpID.String(),
		EscrowID:           escrowID.String(),
		Status:             string(models.MilestonePaymentFunded),
		GrossAmount:        grossAmount,
		PlatformFee:        platformFee,
		FreelancerAmount:   freelancerAmount,
		Currency:           currency,
		FeeRateBps:         rateBps,
		FeeType:            string(feeType),
		Message:            "Milestone funded successfully, funds held in escrow",
	}, nil
}

func (s *MilestonePaymentService) buildFundResponseFromExisting(
	payment *models.Payment,
	milestoneID uuid.UUID,
) (*dto.FundMilestoneResponse, error) {
	mp, err := s.mpRepo.GetByMilestoneID(milestoneID)
	if err != nil {
		return nil, err
	}
	escrow, err := s.escrowRepo.GetByMilestoneID(milestoneID)
	if err != nil {
		return nil, err
	}
	return &dto.FundMilestoneResponse{
		PaymentID:          payment.ID.String(),
		MilestonePaymentID: mp.ID.String(),
		EscrowID:           escrow.ID.String(),
		Status:             string(mp.Status),
		GrossAmount:        mp.GrossAmount,
		PlatformFee:        mp.PlatformFee,
		FreelancerAmount:   mp.FreelancerAmount,
		Currency:           mp.Currency,
		FeeRateBps:         mp.FeeRateBps,
		FeeType:            string(mp.FeeType),
		Message:            "Milestone already funded (idempotent)",
	}, nil
}

// ============================================================
// GetMilestonePayment
// ============================================================

func (s *MilestonePaymentService) GetMilestonePayment(
	userIDStr, projectIDStr, milestoneIDStr string,
) (*dto.MilestonePaymentResponse, error) {

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}
	milestoneID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid milestone id")
	}
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid project id")
	}

	milestone, err := s.milestoneRepo.FindByID(milestoneID)
	if err != nil {
		return nil, ErrMilestoneNotFound
	}
	contract, err := s.contractRepo.FindByID(milestone.ContractID)
	if err != nil {
		return nil, errors.New("contract not found")
	}
	if contract.ProjectID != projectID {
		return nil, ErrMilestoneNotFound
	}
	if contract.ClientID != userID && contract.FreelancerID != userID {
		return nil, ErrUnauthorizedMilestone
	}

	mp, err := s.mpRepo.GetByMilestoneID(milestoneID)
	if err != nil {
		return nil, errors.New("milestone payment not found")
	}

	resp := &dto.MilestonePaymentResponse{
		ID:               mp.ID.String(),
		MilestoneID:      mp.MilestoneID.String(),
		ContractID:       mp.ContractID.String(),
		ProjectID:        mp.ProjectID.String(),
		PaymentID:        mp.PaymentID.String(),
		GrossAmount:      mp.GrossAmount,
		PlatformFee:      mp.PlatformFee,
		FreelancerAmount: mp.FreelancerAmount,
		Currency:         mp.Currency,
		FeeRateBps:       mp.FeeRateBps,
		FeeType:          string(mp.FeeType),
		Status:           string(mp.Status),
		CreatedAt:        mp.CreatedAt.Format(time.RFC3339),
	}
	if mp.EscrowTransactionID != nil {
		resp.EscrowTransactionID = mp.EscrowTransactionID.String()
	}
	if mp.ApprovedAt != nil {
		resp.ApprovedAt = mp.ApprovedAt.Format(time.RFC3339)
	}
	if mp.ReleasedAt != nil {
		resp.ReleasedAt = mp.ReleasedAt.Format(time.RFC3339)
	}
	return resp, nil
}

// ============================================================
// ReleaseMilestonePayment
// ============================================================
//
// Multi-phase flow:
//
//	TX1: reserve release intent (MP → release_pending, escrow → release_pending)
//	PROVIDER: ReleaseEscrow
//	TX2: mark released, settle fee, create payout, append ledger
//	PROVIDER: CreatePayout
//	TX3: mark payout paid
//
// Failures leave state recoverable by reconciliation.
func (s *MilestonePaymentService) ReleaseMilestonePayment(
	userIDStr, projectIDStr, milestoneIDStr string,
	req dto.ReleaseMilestonePaymentRequest,
) (*dto.ReleaseMilestonePaymentResponse, error) {

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user id")
	}
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid project id")
	}
	milestoneID, err := uuid.Parse(milestoneIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid milestone id")
	}
	if req.IdempotencyKey == "" {
		return nil, fmt.Errorf("idempotency_key is required")
	}

	// ---- 1. Load + auth ----
	milestone, err := s.milestoneRepo.FindByID(milestoneID)
	if err != nil {
		return nil, ErrMilestoneNotFound
	}
	contract, err := s.contractRepo.FindByID(milestone.ContractID)
	if err != nil {
		return nil, errors.New("contract not found")
	}
	if contract.ProjectID != projectID {
		return nil, ErrMilestoneNotFound
	}
	if contract.ClientID != userID {
		return nil, ErrUnauthorizedMilestone
	}

	// ---- 2. Freelancer payout method pre-check ----
	payoutMethod, err := s.payoutMethodRepo.GetPrimaryByUser(
		contract.FreelancerID,
		models.PayoutMethodPurposePayout,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("freelancer has no primary payout method")
		}
		return nil, err
	}
	if payoutMethod.Status != models.PayoutMethodStatusVerified {
		return nil, errors.New("freelancer payout method not verified")
	}
	if payoutMethod.ProviderMethodID == nil || *payoutMethod.ProviderMethodID == "" {
		return nil, errors.New("freelancer payout method missing provider reference")
	}
	fundAccountID := *payoutMethod.ProviderMethodID

	// ---- 3. TX1: reserve release intent ----
	var (
		mpID             uuid.UUID
		mpPaymentID      uuid.UUID
		mpGrossAmt       int64
		mpFreelancerAmt  int64
		mpCurrency       string
		escrowProviderID string
		escrowAmount     int64
		alreadyReleased  bool
	)
	releaseKey := req.IdempotencyKey

	err = postgres.DB.Transaction(func(tx *gorm.DB) error {
		txMP := repository.NewMilestonePaymentRepositoryTx(tx)
		txEscrow := repository.NewEscrowTransactionRepositoryTx(tx)

		lockedMP, err := txMP.GetByMilestoneIDForUpdate(milestoneID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrMilestoneNotFunded
			}
			return err
		}
		mpID = lockedMP.ID
		mpPaymentID = lockedMP.PaymentID
		mpGrossAmt = lockedMP.GrossAmount
		mpFreelancerAmt = lockedMP.FreelancerAmount
		mpCurrency = lockedMP.Currency

		if lockedMP.Status == models.MilestonePaymentReleased {
			alreadyReleased = true
			return nil
		}
		if lockedMP.Status == models.MilestonePaymentReleasePending {
			return errors.New("release already in progress for this milestone")
		}
		if lockedMP.Status != models.MilestonePaymentFunded &&
			lockedMP.Status != models.MilestonePaymentAwaitingApproval &&
			lockedMP.Status != models.MilestonePaymentApproved {
			return ErrMilestoneNotFunded
		}

		lockedEscrow, err := txEscrow.GetByMilestoneIDForUpdate(milestoneID)
		if err != nil {
			return errors.New("escrow not found for milestone")
		}
		escrowAmount = lockedEscrow.Amount
		if lockedEscrow.ProviderEscrowID != nil {
			escrowProviderID = *lockedEscrow.ProviderEscrowID
		}
		if lockedEscrow.Status == models.EscrowStatusReleased {
			alreadyReleased = true
			return nil
		}

		now := time.Now()
		lockedMP.Status = models.MilestonePaymentReleasePending
		lockedMP.ApprovedAt = &now
		if err := txMP.Save(lockedMP); err != nil {
			return err
		}

		lockedEscrow.Status = models.EscrowStatusReleasePending
		lockedEscrow.ReleaseIdempotencyKey = &releaseKey
		if err := txEscrow.Save(lockedEscrow); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if alreadyReleased {
		return &dto.ReleaseMilestonePaymentResponse{
			MilestonePaymentID: mpID.String(),
			Status:             string(models.MilestonePaymentReleased),
			FreelancerAmount:   mpFreelancerAmt,
			Currency:           mpCurrency,
			Message:            "Milestone already released (idempotent)",
		}, nil
	}

	// ---- 4. PROVIDER: ReleaseEscrow ----
	if escrowProviderID == "" {
		return nil, errors.New("escrow has no provider reference")
	}
	if _, err := s.provider.ReleaseEscrow(EscrowReleaseRequest{
		ProviderEscrowID: escrowProviderID,
		Amount:           escrowAmount,
		Currency:         mpCurrency,
		IdempotencyKey:   releaseKey,
	}); err != nil {
		log.Printf("ReleaseEscrow failed for milestone=%s: %v", milestoneID, err)
		return &dto.ReleaseMilestonePaymentResponse{
			MilestonePaymentID: mpID.String(),
			Status:             string(models.MilestonePaymentReleasePending),
			FreelancerAmount:   mpFreelancerAmt,
			Currency:           mpCurrency,
			Message:            "Release in progress; reconciliation will retry",
		}, nil
	}

	// ---- 5. TX2: mark released + settle fee + create payout + ledger ----
	var payoutID uuid.UUID
	err = postgres.DB.Transaction(func(tx *gorm.DB) error {
		txMP := repository.NewMilestonePaymentRepositoryTx(tx)
		txEscrow := repository.NewEscrowTransactionRepositoryTx(tx)
		txPayout := repository.NewPayoutRepositoryTx(tx)

		lockedMP, err := txMP.GetByMilestoneIDForUpdate(milestoneID)
		if err != nil {
			return err
		}
		now := time.Now()
		lockedMP.Status = models.MilestonePaymentReleased
		lockedMP.ReleasedAt = &now
		if err := txMP.Save(lockedMP); err != nil {
			return err
		}

		lockedEscrow, err := txEscrow.GetByMilestoneIDForUpdate(milestoneID)
		if err != nil {
			return err
		}
		lockedEscrow.Status = models.EscrowStatusReleased
		lockedEscrow.ReleasedAt = &now
		if err := txEscrow.Save(lockedEscrow); err != nil {
			return err
		}

		// Settle platform fee
		if err := s.feeSvc.SettleFee(tx, lockedMP.ID); err != nil {
			return fmt.Errorf("settle platform fee: %w", err)
		}

		// Payout intent
		payout := &models.Payout{
			UserID:             contract.FreelancerID,
			MilestonePaymentID: lockedMP.ID,
			ProjectID:          projectID,
			ContractID:         contract.ID,
			PayoutMethodID:     payoutMethod.ID,
			Provider:           s.provider.Name(),
			Amount:             lockedMP.FreelancerAmount,
			Currency:           lockedMP.Currency,
			SettlementAmount:   lockedMP.FreelancerAmount,
			SettlementCurrency: lockedMP.Currency,
			Status:             models.PayoutStatusPending,
			IdempotencyKey:     releaseKey + ":payout",
		}
		if err := txPayout.Create(payout); err != nil {
			return err
		}
		payoutID = payout.ID

		// Ledger entries
		msPtr := milestoneID
		entries := []LedgerEntry{
			{
				PaymentID:       mpPaymentID,
				ProjectID:       projectID,
				ContractID:      contract.ID,
				MilestoneID:     &msPtr,
				TransactionType: models.TxTypeEscrowRelease,
				Amount:          -mpGrossAmt,
				Currency:        mpCurrency,
				Provider:        s.provider.Name(),
				Reference:       releaseKey,
				IdempotencyKey:  releaseKey + ":ledger:escrow_release",
			},
			{
				PaymentID:       mpPaymentID,
				ProjectID:       projectID,
				ContractID:      contract.ID,
				MilestoneID:     &msPtr,
				TransactionType: models.TxTypePlatformFee,
				Amount:          lockedMP.PlatformFee,
				Currency:        mpCurrency,
				Provider:        s.provider.Name(),
				Reference:       lockedMP.ID.String(),
				IdempotencyKey:  releaseKey + ":ledger:platform_fee",
			},
			{
				PaymentID:       mpPaymentID,
				ProjectID:       projectID,
				ContractID:      contract.ID,
				MilestoneID:     &msPtr,
				TransactionType: models.TxTypePayout,
				Amount:          -mpFreelancerAmt,
				Currency:        mpCurrency,
				Provider:        s.provider.Name(),
				Reference:       payout.ID.String(),
				IdempotencyKey:  releaseKey + ":ledger:payout",
			},
		}
		for _, e := range entries {
			if err := s.ledgerSvc.Append(tx, e); err != nil {
				return fmt.Errorf("append ledger entry: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("post-release persistence failed for milestone=%s: %v", milestoneID, err)
		return &dto.ReleaseMilestonePaymentResponse{
			MilestonePaymentID: mpID.String(),
			Status:             string(models.MilestonePaymentReleasePending),
			FreelancerAmount:   mpFreelancerAmt,
			Currency:           mpCurrency,
			Message:            "Release accepted; will be reconciled",
		}, nil
	}

	// ---- 6. PROVIDER: CreatePayout ----
	payoutResult, err := s.provider.CreatePayout(PayoutRequest{
		UserID:                contract.FreelancerID.String(),
		Amount:                mpFreelancerAmt,
		Currency:              mpCurrency,
		ProviderFundAccountID: fundAccountID,
		IdempotencyKey:        releaseKey + ":payout",
		Description:           fmt.Sprintf("Milestone payout: %s", milestone.Title),
	})
	if err != nil {
		log.Printf("CreatePayout failed for payout=%s: %v", payoutID, err)
		return &dto.ReleaseMilestonePaymentResponse{
			MilestonePaymentID: mpID.String(),
			Status:             string(models.MilestonePaymentReleased),
			FreelancerAmount:   mpFreelancerAmt,
			Currency:           mpCurrency,
			Message:            "Released; payout in progress",
		}, nil
	}

	// ---- 7. TX3: mark payout paid ----
	_ = postgres.DB.Transaction(func(tx *gorm.DB) error {
		txPayout := repository.NewPayoutRepositoryTx(tx)
		p, err := txPayout.GetByID(payoutID)
		if err != nil {
			return err
		}
		now := time.Now()
		p.Status = models.PayoutStatusPaid
		p.ProviderPayoutID = &payoutResult.ProviderPayoutID
		p.ProcessedAt = &now
		return tx.Save(p).Error
	})

	// Non-fatal: mark ProjectMilestone workflow status = paid
	if err := s.milestoneRepo.MarkAsPaid(milestoneID); err != nil {
		log.Printf("failed to mark milestone paid for milestone=%s: %v", milestoneID, err)
	}

	return &dto.ReleaseMilestonePaymentResponse{
		MilestonePaymentID: mpID.String(),
		Status:             string(models.MilestonePaymentReleased),
		FreelancerAmount:   mpFreelancerAmt,
		Currency:           mpCurrency,
		Message:            "Milestone released and payout completed",
	}, nil
}
