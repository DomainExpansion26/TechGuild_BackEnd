package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/netip"
	"strings"
	"time"

	"techguild-backend/src/config"
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Custom Errors for Verification
var (
	ErrAlreadySubmitted         = errors.New("verification is already submitted and under review")
	ErrDuplicateGovtID          = errors.New("this government ID is already associated with another account")
	ErrDuplicatePAN             = errors.New("this business PAN is already associated with another account")
	ErrInvalidState             = errors.New("invalid state for this action")
	ErrMLQueueFailed            = errors.New("failed to queue verification for AI processing")
	ErrInvalidAdminAction       = errors.New("invalid admin action")
	ErrInvalidDocTypeForAccount = errors.New("this document type is not allowed for your account type")
)

// KYC AI/ML pipeline job queue (Redis Stream). The worker (Phase D) consumes
// this stream, calls the ML vendor, and writes results back to the record.
const (
	kycJobStream       = "kyc:verification:jobs"
	kycJobPayloadField = "payload"

	// kycApprovalBonusPoints is the one-time bonus granted when a verification
	// record reaches approved, on every approve path (manual, bulk, router, cron).
	kycApprovalBonusPoints = 40
)

// kycMLJob is the payload contract between the API (producer) and the ML
// worker (consumer). Documents are Cloudinary URLs — the worker fetches them
// and streams them to the vendor; no PII is stored in Redis beyond the URL.
type kycMLJob struct {
	JobID               string             `json:"job_id"`
	VerificationID      string             `json:"verification_id"`
	UserID              string             `json:"user_id"`
	Type                string             `json:"type"`
	Documents           []kycMLJobDocument `json:"documents"`
	BusinessInfo        kycMLBusinessInfo  `json:"business_info,omitempty"`
	BankDetails         kycMLBankDetails   `json:"bank_details,omitempty"`
	SelectedBusinessDoc string             `json:"selected_business_doc,omitempty"`
}

type kycMLBusinessInfo struct {
	GSTIN string `json:"gstin,omitempty"`
	PAN   string `json:"pan,omitempty"`
	RegNo string `json:"reg_no,omitempty"`
}

type kycMLBankDetails struct {
	IFSC          string `json:"ifsc,omitempty"`
	AccountNumber string `json:"account_number,omitempty"`
}

type kycMLJobDocument struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type VerificationService struct {
	userRepo                repository.UserRepository
	verificationRecordsRepo repository.VerificationRecordsRepository
	redisClient             *redis.Client

	// kycPolicy holds the router policy (verification_router.go). main.go
	// wires it from cfg; nil falls back to config.DefaultKYCPolicy() so the
	// hard defaults (shadow mode ON) can never be bypassed silently.
	kycPolicy *config.KYCPolicy

	// uploadDocumentFn is a seam for tests; production uses Cloudinary.
	uploadDocumentFn func(file multipart.File, filename string) (string, error)

	// Phase 4 rate limiting. All three are wired together by SetRateLimiter;
	// when attemptsRepo/ipResolver are nil the service falls back to the
	// pre-Phase-4 submit path (used by legacy unit tests and internal tooling).
	attemptsRepo repository.AttemptsRepository
	ipResolver   *utils.IPResolver
	ipPepper     []byte
}

// SetRateLimiter wires the KYC submit rate limiter. pepperHex is the
// KYC_IP_HASH_PEPPER value (hex); it is decoded up front so an invalid
// configuration fails at startup rather than on the first submit.
func (s *VerificationService) SetRateLimiter(
	repo repository.AttemptsRepository,
	resolver *utils.IPResolver,
	pepperHex string,
) error {
	pepper, err := hex.DecodeString(pepperHex)
	if err != nil {
		return fmt.Errorf("invalid KYC IP pepper (must be hex): %w", err)
	}
	s.attemptsRepo = repo
	s.ipResolver = resolver
	s.ipPepper = pepper
	return nil
}

// sha256Hex returns the lowercase hex SHA-256 of s. Empty input hashes to the
// SHA-256 of the empty string, which is stable and non-reversible.
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// SetKYCPolicy attaches the env-loaded router policy (main.go, after
// LoadConfig). Safe to call multiple times.
func (s *VerificationService) SetKYCPolicy(p config.KYCPolicy) {
	s.kycPolicy = &p
}

// effectiveKYCPolicy returns the wired policy or the hard defaults. The hard
// defaults keep ShadowMode=true: an un-wired service can never auto-approve.
func (s *VerificationService) effectiveKYCPolicy() config.KYCPolicy {
	if s.kycPolicy != nil {
		return *s.kycPolicy
	}
	return config.DefaultKYCPolicy()
}

// Constructor preserved from original code
func NewVerificationService(redisClient *redis.Client) *VerificationService {
	return &VerificationService{
		userRepo:                repository.NewUserRepository(),
		verificationRecordsRepo: repository.NewVerificationRecordsRepository(),
		redisClient:             redisClient,
		uploadDocumentFn:        utils.UploadVerificationDocument,
	}
}

// isDocTypeAllowedForVerification enforces which document types a verification
// (individual or business) may upload.
//
// Phase 5: individual KYC accepts aadhaar AND pan_card (multi-reference face
// matching) plus the legacy govt_id alias; business rules are unchanged.
func isDocTypeAllowedForVerification(vType models.VerificationType, docType string) bool {
	switch vType {
	case models.VerificationIndividual:
		switch docType {
		case models.DocTypeAadhaar,
			models.DocTypePANCard,
			models.DocTypeGovtID, // legacy alias: single-reference fallback
			models.DocTypeSelfie:
			return true
		}
	case models.VerificationBusiness:
		switch docType {
		case models.DocTypeGSTCert,
			models.DocTypeIncCert,
			models.DocTypePANCard,
			models.DocTypeBizReg,
			models.DocTypeCancelledCheque,
			models.DocTypeSelfie:
			return true
		}
	}
	return false
}

// =====================================================
// 1. Helper: Get or Create Draft
// =====================================================
func (s *VerificationService) getOrCreateDraft(userID string) (*models.VerificationRecord, error) {
	record, err := s.verificationRecordsRepo.GetActiveVerificationByUser(userID)
	if err == nil && record != nil {
		return record, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Determine type based on account
	vType := models.VerificationIndividual
	if user.AccountType != nil && (*user.AccountType == models.AccountTypeAgencyAdmin || *user.AccountType == models.AccountTypeClientAdmin) {
		vType = models.VerificationBusiness
	}

	newRecord := &models.VerificationRecord{
		UserID: user.ID,
		Type:   vType,
		Status: models.VerificationDraft,
		Vendor: "hyperverge", // From original code
	}

	if err := s.verificationRecordsRepo.CreateVerification(newRecord); err != nil {
		return nil, err
	}

	return newRecord, nil
}

// =====================================================
// 2. User Flow: Step-by-Step Updates
// =====================================================

func (s *VerificationService) UpdateIndividualInfo(userID string, req dto.UpdateIndividualInfoRequest) error {
	record, err := s.getOrCreateDraft(userID)
	if err != nil {
		return err
	}

	if record.Status != models.VerificationDraft && record.Status != models.VerificationRejected {
		return ErrAlreadySubmitted
	}

	govtHash := utils.HashGovtID(req.IDNumber)

	// Check Dedup (But don't insert into dedup table yet, wait for approval)
	exists, err := s.verificationRecordsRepo.GovtHashExists(govtHash)
	if err != nil {
		return err
	}
	if exists {
		return ErrDuplicateGovtID
	}

	record.GovtIDHash = govtHash
	// We can use VendorReferenceID to temporarily store the doc type
	record.VendorReferenceID = req.DocType

	return s.verificationRecordsRepo.UpdateVerificationRecord(record)
}

func (s *VerificationService) UpdateBusinessInfo(userID string, req dto.UpdateBusinessInfoRequest) error {
	record, err := s.getOrCreateDraft(userID)
	if err != nil {
		return err
	}

	if record.Status != models.VerificationDraft && record.Status != models.VerificationRejected {
		return ErrAlreadySubmitted
	}

	panHash := utils.HashBusinessPAN(req.PANNumber)

	exists, err := s.verificationRecordsRepo.BusinessHashExists(panHash)
	if err != nil {
		return err
	}
	if exists {
		return ErrDuplicatePAN
	}

	record.BusinessPANHash = panHash
	record.BusinessName = req.CompanyName
	record.GSTNumber = req.GSTNumber
	record.RegistrationNumber = req.RegistrationNumber
	record.Website = req.Website
	record.Country = req.Country

	return s.verificationRecordsRepo.UpdateVerificationRecord(record)
}

func (s *VerificationService) UpdateBankDetails(userID string, req dto.UpdateBankDetailsRequest) error {
	record, err := s.getOrCreateDraft(userID)
	if err != nil {
		return err
	}

	if record.Status != models.VerificationDraft && record.Status != models.VerificationRejected {
		return ErrAlreadySubmitted
	}

	encAcc, err := utils.EncryptSecret(req.AccountNumber)
	if err != nil {
		log.Printf("Encryption error: %v", err)
		return errors.New("failed to securely save bank details")
	}

	record.BankName = req.BankName
	record.AccountHolderName = req.AccountHolderName
	record.AccountNumberEnc = encAcc
	record.IFSCCode = req.IFSCCode

	return s.verificationRecordsRepo.UpdateVerificationRecord(record)
}

// =====================================================
// 3. User Flow: Upload Document
// =====================================================

func (s *VerificationService) UploadDocument(userID string, docType string, file multipart.File, filename string) (*dto.UploadDocumentResponse, error) {
	record, err := s.getOrCreateDraft(userID)
	if err != nil {
		return nil, err
	}

	if record.Status != models.VerificationDraft && record.Status != models.VerificationRejected {
		return nil, ErrAlreadySubmitted
	}

	// Enforce the document allow-list from the already-fetched record type
	// BEFORE deleting anything or calling Cloudinary.
	if !isDocTypeAllowedForVerification(record.Type, docType) {
		return nil, ErrInvalidDocTypeForAccount
	}

	// 1. Delete old doc if re-uploading
	existingDocs, _ := s.verificationRecordsRepo.GetDocuments(record.ID.String())
	for _, d := range existingDocs {
		if d.DocumentType == docType {
			_ = utils.DeleteFromCloudinary(d.PublicID, "auto")
			_ = s.verificationRecordsRepo.DeleteDocumentByType(record.ID.String(), docType)
			break
		}
	}

	// 2. Upload to Cloudinary
	finalFilename := fmt.Sprintf("%s_%s_%s", userID, docType, uuid.New().String())

	// ✅ FIXED: Pass `file` directly (no need to call Open() as Huma already opened it)
	upload := s.uploadDocumentFn
	if upload == nil {
		upload = utils.UploadVerificationDocument
	}
	fileURL, err := upload(file, finalFilename)
	if err != nil {
		if strings.Contains(err.Error(), "invalid file type") {
			return nil, err // Return validation error
		}
		return nil, errors.New("failed to upload to secure storage")
	}

	// 3. Save new doc
	doc := &models.VerificationDocument{
		VerificationRecordID: record.ID,
		DocumentType:         docType,
		FileURL:              fileURL,
	}

	if err := s.verificationRecordsRepo.CreateDocument(doc); err != nil {
		return nil, err
	}
	if record.Type == models.VerificationBusiness {
		switch docType {
		case models.DocTypeGSTCert, models.DocTypePANCard, models.DocTypeIncCert, models.DocTypeBizReg:
			record.SelectedBusinessDoc = docType
			if err := s.verificationRecordsRepo.UpdateVerificationRecord(record); err != nil {
				return nil, err
			}
		}
	}

	return &dto.UploadDocumentResponse{
		Message:      "Document uploaded successfully",
		FileURL:      doc.FileURL,
		DocumentType: doc.DocumentType,
	}, nil
}

// =====================================================
// 4. User Flow: Submit
// =====================================================

// SubmitVerification moves the user's draft/rejected record to pending and
// queues the ML job.
//
// Phase 4: the quota reservation and the state transition run in ONE database
// transaction, which closes the concurrent-submit race (two requests can no
// longer both observe 'draft' and both consume a slot). If the Redis publish
// fails after commit, MarkFailed releases the reserved quota.
func (s *VerificationService) SubmitVerification(
	ctx context.Context,
	userID string,
	clientIP netip.Addr,
	userAgent string,
) error {
	if s.attemptsRepo == nil || s.ipResolver == nil {
		return s.submitVerificationLegacy(userID)
	}
	return s.submitVerificationRateLimited(ctx, userID, clientIP, userAgent)
}

// submitVerificationLegacy is the pre-Phase-4 path, retained for callers that
// have not wired the rate limiter (legacy unit tests / internal tooling).
func (s *VerificationService) submitVerificationLegacy(userID string) error {
	record, err := s.getOrCreateDraft(userID)
	if err != nil {
		return err
	}

	if record.Status != models.VerificationDraft && record.Status != models.VerificationRejected {
		return ErrAlreadySubmitted
	}

	docs, err := s.verificationRecordsRepo.GetDocuments(record.ID.String())
	if err != nil {
		return err
	}

	if err := validateSubmitPayload(record, docs); err != nil {
		return err
	}

	applySubmitTransition(record, time.Now())

	if err := s.verificationRecordsRepo.UpdateVerificationRecord(record); err != nil {
		return err
	}

	// Publish the ML job to the Redis Stream; user-facing submit fails if the
	// pipeline can't be reached (the record stays pending+queued and can be
	// re-queued later).
	return s.queueMLJob(record, docs)
}

// submitVerificationRateLimited reserves quota and performs the state
// transition inside a single transaction.
func (s *VerificationService) submitVerificationRateLimited(
	ctx context.Context,
	userID string,
	clientIP netip.Addr,
	userAgent string,
) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidState
	}

	limits := repository.RateLimits{
		UserHourly: 3, UserDaily: 10,
		IPHourly: 10, IPDaily: 30,
	}
	requestID := uuid.NewString()
	ipHash := s.ipResolver.Hash(clientIP, s.ipPepper)
	uaHash := sha256Hex(userAgent)

	var (
		record models.VerificationRecord
		docs   []models.VerificationDocument
	)

	err = postgres.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Reserve quota FIRST, in this same transaction. A limit failure
		// rolls the whole transaction back.
		if err := s.attemptsRepo.ReserveAttempt(ctx, tx, repository.ReserveAttemptRequest{
			UserID:        userUUID,
			Action:        models.AttemptActionSubmit,
			IPHash:        ipHash,
			UserAgentHash: uaHash,
			RequestID:     requestID,
			Limits:        limits,
		}); err != nil {
			return err
		}

		// State transition in the SAME transaction: a concurrent submit that
		// already flipped the record blocks here and rolls back its reserve.
		if err := tx.Where("user_id = ? AND status IN ?", userUUID,
			[]string{string(models.VerificationDraft), string(models.VerificationPending), string(models.VerificationRejected)}).
			Order("created_at DESC").First(&record).Error; err != nil {
			return err
		}
		if record.Status != models.VerificationDraft && record.Status != models.VerificationRejected {
			return ErrAlreadySubmitted
		}

		if err := tx.Where("verification_record_id = ?", record.ID).Find(&docs).Error; err != nil {
			return err
		}
		if err := validateSubmitPayload(&record, docs); err != nil {
			return err
		}

		applySubmitTransition(&record, time.Now())
		return tx.Save(&record).Error
	})
	if err != nil {
		return err
	}

	// AFTER commit: publish the ML job. If Redis is unreachable, compensate by
	// releasing the reserved quota so the user is not charged for a failed
	// submit.
	if pubErr := s.queueMLJob(&record, docs); pubErr != nil {
		_ = s.attemptsRepo.MarkFailed(ctx, requestID)
		return pubErr
	}
	return nil
}

// validateSubmitPayload enforces document/info completeness before a record
// may move to pending.
func validateSubmitPayload(record *models.VerificationRecord, docs []models.VerificationDocument) error {
	switch record.Type {
	case models.VerificationIndividual:
		if record.GovtIDHash == "" {
			return errors.New("government ID info is missing")
		}

		// Ordered doc presence scan (Phase 5): selfie is the face-match
		// anchor; Aadhaar is the required QR-verified reference; PAN and the
		// legacy govt_id alias are accepted per the allow-list.
		present := make(map[string]bool, len(docs))
		for _, d := range docs {
			switch d.DocumentType {
			case models.DocTypeGovtID, "government_id":
				// Legacy alias: kept as its own type so it can NEVER satisfy
				// the Aadhaar requirement silently (only via the fallback).
				present[models.DocTypeGovtID] = true
			default:
				present[d.DocumentType] = true
			}
		}
		if !present[models.DocTypeSelfie] {
			return errors.New("missing required documents (selfie)")
		}

		// Ensure Aadhaar is present (Phase 5 requirement). PAN stays
		// optional — a missing PAN is never a submission error.
		for _, requiredType := range models.RequiredDocTypesFor(record.Type) {
			if present[requiredType] {
				continue
			}
			// Legacy govt_id fallback — accept with a warning: single-
			// reference matching only, the router forces manual review.
			if requiredType == models.DocTypeAadhaar && present[models.DocTypeGovtID] {
				log.Printf("[KYC Submit] legacy govt_id used without aadhaar; " +
					"single-reference matching only, manual review guaranteed")
				continue
			}
			return fmt.Errorf("individual KYC requires an %s document", requiredType)
		}

	case models.VerificationBusiness:
		if record.BusinessName == "" {
			return errors.New("business name is required")
		}
		if record.AccountNumberEnc == "" || record.IFSCCode == "" {
			return errors.New("bank details are incomplete (account number and IFSC code are required)")
		}

		present := make(map[string]bool, len(docs))
		for _, d := range docs {
			present[d.DocumentType] = true
		}

		// Business docs are now an exclusive choice: the merchant selects
		// EXACTLY ONE proof of business existence (prevents ambiguous review
		// when several, possibly conflicting, documents are uploaded).
		// The order is fixed so the error message is deterministic.
		var selectedBusinessDocs []string
		for _, docType := range []string{
			models.DocTypeGSTCert,
			models.DocTypeIncCert,
			models.DocTypePANCard,
			models.DocTypeBizReg,
		} {
			if present[docType] {
				selectedBusinessDocs = append(selectedBusinessDocs, docType)
			}
		}
		if len(selectedBusinessDocs) == 0 {
			return errors.New("missing business document: select exactly one of gst_certificate, incorporation_certificate, pan_card or business_registration")
		}
		if record.SelectedBusinessDoc == "" || !present[record.SelectedBusinessDoc] {
			return errors.New("selected business document is required and must match the uploaded document")
		}
		if len(selectedBusinessDocs) > 1 {
			return fmt.Errorf("select exactly one business document, but %d were provided: %s",
				len(selectedBusinessDocs), strings.Join(selectedBusinessDocs, ", "))
		}

		// The cancelled cheque is still mandatory: it is the evidence backing
		// the bank details validated above, not an alternative business proof.
		if !present[models.DocTypeCancelledCheque] {
			return fmt.Errorf("missing required document: %s", models.DocTypeCancelledCheque)
		}
	}
	return nil
}

// applySubmitTransition stamps the fields that move a record to pending+queued.
func applySubmitTransition(record *models.VerificationRecord, now time.Time) {
	record.Status = models.VerificationPending
	record.SubmittedAt = &now
	record.RejectionReason = "" // Clear previous rejection reason

	// ---- Phase C: queue the record for AI/ML processing ----
	// Priority defaults to manual_review (human fallback) until the AI run
	// upgrades it to auto_eligible or auto_reject based on confidence scores.
	record.MLStatus = models.MLStatusQueued
	record.ReviewPriority = models.ReviewPriorityManual
	record.MLSubmittedAt = &now
	record.MLJobID = "kyc-job-" + uuid.NewString()

	// ---- Phase 3 routing contract (Save writes EVERY field, so
	// RoutingState must be set explicitly here) ----
	//   this submit           -> routing_state = 'pending'   (below)
	//   worker save_results   -> ..., routing_state = 'pending' (Python UPDATE,
	//                            guarded by system_escalated_at IS NULL)
	//   Go router / escalator -> routing_state = 'routed'
	// The default column value is 'routed', so only records that pass through
	// here (new submits, explicit re-queues) are visible to the router.
	record.RoutingState = models.RoutingPending
}

// queueMLJob pushes a verification job onto the Redis Stream consumed by the
// ML worker. The payload carries document (Cloudinary) URLs only — no raw PII.
func (s *VerificationService) queueMLJob(record *models.VerificationRecord, docs []models.VerificationDocument) error {
	accountNumber := ""
	if record.AccountNumberEnc != "" {
		var err error
		accountNumber, err = utils.DecryptSecret(record.AccountNumberEnc)
		if err != nil {
			log.Printf("KYC bank account decrypt failed (verification_id=%s): %v", record.ID, err)
			return ErrMLQueueFailed
		}
	}
	job := kycMLJob{
		JobID: record.MLJobID, VerificationID: record.ID.String(), UserID: record.UserID.String(),
		Type: string(record.Type), Documents: make([]kycMLJobDocument, 0, len(docs)),
		BusinessInfo:        kycMLBusinessInfo{GSTIN: record.GSTNumber, PAN: record.BusinessPANHash, RegNo: record.RegistrationNumber},
		BankDetails:         kycMLBankDetails{IFSC: record.IFSCCode, AccountNumber: accountNumber},
		SelectedBusinessDoc: record.SelectedBusinessDoc,
	}
	for _, d := range docs {
		job.Documents = append(job.Documents, kycMLJobDocument{Type: d.DocumentType, URL: d.FileURL})
	}
	payload, err := json.Marshal(job)
	if err != nil {
		log.Printf("KYC job marshal failed (verification_id=%s): %v", job.VerificationID, err)
		return ErrMLQueueFailed
	}
	if s.redisClient == nil {
		log.Printf("Redis client unavailable; KYC job %s not queued (verification_id=%s)", job.JobID, job.VerificationID)
		return ErrMLQueueFailed
	}
	if err := s.redisClient.XAdd(context.Background(), &redis.XAddArgs{Stream: kycJobStream, Values: map[string]interface{}{kycJobPayloadField: string(payload)}}).Err(); err != nil {
		log.Printf("Failed to publish KYC job %s to stream %s (verification_id=%s): %v", job.JobID, kycJobStream, job.VerificationID, err)
		return ErrMLQueueFailed
	}
	log.Printf("KYC job %s queued (verification_id=%s, user_id=%s)", job.JobID, job.VerificationID, job.UserID)
	return nil
}

// =====================================================
// 5. Status & Admin Flow
// =====================================================

func (s *VerificationService) GetStatus(userID string) (*dto.VerificationStatusResponse, error) {
	record, err := s.getOrCreateDraft(userID)
	if err != nil {
		return nil, err
	}

	docs, _ := s.verificationRecordsRepo.GetDocuments(record.ID.String())

	return &dto.VerificationStatusResponse{
		Type:              string(record.Type),
		Status:            string(record.Status),
		RejectionReason:   record.RejectionReason,
		SubmittedAt:       record.SubmittedAt,
		HasIndividualInfo: record.GovtIDHash != "",
		HasBusinessInfo:   record.BusinessName != "",
		HasBankDetails:    record.AccountNumberEnc != "",
		UploadedDocsCount: len(docs),
	}, nil
}

func (s *VerificationService) AdminGetQueue() ([]dto.AdminVerificationItem, error) {
	records, err := s.verificationRecordsRepo.GetVerificationQueue()
	if err != nil {
		return nil, err
	}

	var items []dto.AdminVerificationItem
	for _, r := range records {
		item := dto.AdminVerificationItem{
			ID:          r.ID.String(),
			UserID:      r.UserID.String(),
			UserEmail:   r.User.Email,
			Type:        string(r.Type),
			Status:      string(r.Status),
			CompanyName: r.BusinessName,
			SubmittedAt: r.SubmittedAt,
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *VerificationService) AdminReview(adminID, recordID string, req dto.AdminReviewRequest) error {
	record, err := s.verificationRecordsRepo.GetVerificationByID(recordID)
	if err != nil {
		return errors.New("verification record not found")
	}

	if record.Status != models.VerificationPending && record.Status != models.VerificationReview {
		return ErrInvalidState
	}

	switch req.Action {
	case "APPROVE":
		// 1. Update Status
		if err := s.verificationRecordsRepo.ApproveVerification(recordID); err != nil {
			return err
		}

		// 2/3. Dedup registration, KYC bonus points, account activation.
		// Shared with every other approve path so the guard can't be bypassed.
		if err := s.approveVerificationSideEffects(record); err != nil {
			log.Printf("[Verification] approve side effects failed (legacy AdminReview, record %s): %v", record.ID, err)
			return err
		}

	case "REJECT":
		if req.RejectionReason == "" {
			return errors.New("rejection reason is required")
		}
		if err := s.verificationRecordsRepo.RejectVerification(recordID, req.RejectionReason); err != nil {
			return err
		}

	default:
		return errors.New("invalid action")
	}

	return nil
}

// =====================================================
// 6. Admin Flow: KYC AI Pipeline Review Queues (Phase C)
// =====================================================

// AdminGetAutoEligible returns records eligible for the batch (24-48h window)
// approval flow. minAgeHours filters out freshly-ML-completed records so their
// hold window is respected.
func (s *VerificationService) AdminGetAutoEligible(limit, offset int, minAgeHours float64) ([]models.VerificationRecord, int64, error) {
	return s.verificationRecordsRepo.GetAutoEligibleQueue(limit, offset, minAgeHours)
}

// AdminGetManualReview returns borderline records needing human decision.
// sortBy accepts "liveness_score" / "face_similarity" (weakest-first priority)
// or anything else for default submitted-time ordering; sortDir ASC/DESC.
func (s *VerificationService) AdminGetManualReview(sortBy, sortDir string, limit, offset int) ([]models.VerificationRecord, int64, error) {
	return s.verificationRecordsRepo.GetManualReviewQueue(sortBy, sortDir, limit, offset)
}

// AdminGetDetail returns one record (with User and Documents preloaded) for the
// admin review screen.
func (s *VerificationService) AdminGetDetail(id string) (*models.VerificationRecord, error) {
	return s.verificationRecordsRepo.GetVerificationDetail(id)
}

// AdminBulkApprove approves multiple auto-eligible records in one transaction
// and records the human trail (audit logs + bulk review items).
// The approve side effects (dedup hash, +points, activation) run inside that
// same transaction — bulk approve is the primary production approve path, so
// skipping them would defeat the duplicate-Aadhaar/PAN guard entirely.
func (s *VerificationService) AdminBulkApprove(ids []string, adminID string) error {
	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		return fmt.Errorf("%w: invalid admin ID", ErrInvalidAdminAction)
	}
	if len(ids) == 0 {
		return fmt.Errorf("%w: no verification IDs provided", ErrInvalidAdminAction)
	}
	return s.verificationRecordsRepo.BulkApprove(ids, adminUUID, s.approveVerificationSideEffectsTx)
}

// AdminManualAction applies a single approve/reject decision from the manual
// review queue. APPROVE reuses the same side effects as the legacy AdminReview
// flow: dedup registration, +40 points, and account activation.
func (s *VerificationService) AdminManualAction(id string, action string, adminID string, reason string) error {
	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		return fmt.Errorf("%w: invalid admin ID", ErrInvalidAdminAction)
	}

	var normalizedAction string
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "approve":
		normalizedAction = "approve"
	case "reject":
		if reason == "" {
			return fmt.Errorf("%w: rejection reason is required", ErrInvalidAdminAction)
		}
		normalizedAction = "reject"
	default:
		return fmt.Errorf("%w: %q (must be approve or reject)", ErrInvalidAdminAction, action)
	}

	record, err := s.verificationRecordsRepo.GetVerificationByID(id)
	if err != nil {
		return errors.New("verification record not found")
	}

	if err := s.verificationRecordsRepo.MarkManualAction(id, normalizedAction, adminUUID, reason); err != nil {
		return err
	}

	// Approve side effects, shared with the 24h auto-approver cron (Phase E).
	if normalizedAction == "approve" {
		if err := s.approveVerificationSideEffects(record); err != nil {
			// The record is already approved in the DB; surface the partial
			// failure so it isn't lost, it also lands in the audit trail.
			log.Printf("[Verification] approve side effects failed (record %s, admin %s): %v", record.ID, adminID, err)
			return err
		}
	}

	return nil
}

// approveVerificationSideEffects contains everything that must happen when a
// verification record becomes approved, regardless of WHO approved it (admin
// manual action, bulk approve, router auto-approve, or the 24h auto-approver
// cron).
// Each step runs even if an earlier one failed; the first error is returned so
// callers can log/surface it without blocking the remaining steps.
//
// Used by the non-transactional paths (AdminManualAction). Paths that already
// hold an open transaction MUST use approveVerificationSideEffectsTx instead so
// the side effects commit or roll back atomically with the status flip.
func (s *VerificationService) approveVerificationSideEffects(record *models.VerificationRecord) error {
	var firstErr error
	noteErr := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// No global DB handle (unit tests wire repositories directly): fall back to
	// the repository seam so the same three side effects still run.
	if postgres.DB == nil {
		if record.Type == models.VerificationIndividual && record.GovtIDHash != "" {
			noteErr(s.verificationRecordsRepo.CreateGovtHash(&models.GovtIDDedup{
				GovtIDHash: record.GovtIDHash,
				UserID:     record.UserID,
			}))
		}
		if record.Type == models.VerificationBusiness && record.BusinessPANHash != "" {
			noteErr(s.verificationRecordsRepo.CreateBusinessHash(&models.BusinessPANDedup{
				PANHash: record.BusinessPANHash,
				UserID:  record.UserID,
			}))
		}
		noteErr(s.userRepo.AddUserPoints(record.UserID.String(), kycApprovalBonusPoints))
		noteErr(s.userRepo.UpdateUserStatus(record.UserID.String(), string(models.StatusActive)))
		return firstErr
	}

	noteErr(s.approveVerificationSideEffectsTx(postgres.DB, record))

	return firstErr
}

// approveVerificationSideEffectsTx registers the dedup hashes, awards the KYC
// bonus points and activates the account, all on the supplied transaction
// handle. Every pipeline that flips a record to approved must call this so the
// duplicate-Aadhaar/PAN guard and the reward/activation contract cannot be
// bypassed by choosing a different approve entry point.
//
// It returns the first error encountered. Callers inside a transaction should
// propagate it and abort, so a failed dedup insert never leaves a record
// approved without its uniqueness claim (which would silently allow the same
// government ID on a second account).
func (s *VerificationService) approveVerificationSideEffectsTx(tx *gorm.DB, record *models.VerificationRecord) error {
	if tx == nil {
		tx = postgres.DB
	}
	if record == nil || record.UserID == uuid.Nil {
		return errors.New("approve side effects: record has no user")
	}

	// 1. Register dedup hashes. OnConflict DoNothing keeps the call idempotent:
	// a retried approve (cron replay, admin double-click, router re-claim)
	// must not fail on a hash row that is already there.
	switch record.Type {
	case models.VerificationIndividual:
		if record.GovtIDHash != "" {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
				Create(&models.GovtIDDedup{
					GovtIDHash: record.GovtIDHash,
					UserID:     record.UserID,
				}).Error; err != nil {
				return fmt.Errorf("register govt-id dedup hash: %w", err)
			}
		}
	case models.VerificationBusiness:
		if record.BusinessPANHash != "" {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
				Create(&models.BusinessPANDedup{
					PANHash: record.BusinessPANHash,
					UserID:  record.UserID,
				}).Error; err != nil {
				return fmt.Errorf("register business PAN dedup hash: %w", err)
			}
		}
	}

	// 2. Award KYC bonus points (points_only: no rank side effects here).
	if err := tx.Model(&models.User{}).
		Where("id = ?", record.UserID).
		UpdateColumn("points", gorm.Expr("points + ?", kycApprovalBonusPoints)).Error; err != nil {
		return fmt.Errorf("award kyc bonus points: %w", err)
	}

	// 3. Activate the account. Scoped to non-terminal-competition states is
	// overkill here; activation is the intended post-approval state.
	if err := tx.Model(&models.User{}).
		Where("id = ?", record.UserID).
		Update("status", string(models.StatusActive)).Error; err != nil {
		return fmt.Errorf("activate account: %w", err)
	}

	return nil
}

// AdminEscalateToManual pulls an auto-eligible record back to the manual review
// queue (an admin wants eyes on it before the 24-48h bulk window fires).
func (s *VerificationService) AdminEscalateToManual(id string, adminID string, reason string) error {
	adminUUID, err := uuid.Parse(adminID)
	if err != nil {
		return fmt.Errorf("%w: invalid admin ID", ErrInvalidAdminAction)
	}
	return s.verificationRecordsRepo.RequestManualReview(id, adminUUID, reason)
}

// GetRecordDocuments exposes the uploaded documents for admin queue/detail
// mapping (the queue queries preload only the User, not documents).
func (s *VerificationService) GetRecordDocuments(recordID string) ([]models.VerificationDocument, error) {
	return s.verificationRecordsRepo.GetDocuments(recordID)
}

// =====================================================
// 7. Stale-Pending Escalation Cron (Phase E)
// =====================================================

const (
	// kycStalePendingWindowHours: once the former 24h auto-approval window
	// expires, stale auto_eligible records must go to a human, never be
	// approved automatically.
	kycStalePendingWindowHours = 24.0
	// kycAutoEscalateWindowHours: legacy hard-stop for records stuck past 48h.
	kycAutoEscalateWindowHours = 48.0
	// Process records in bounded batches so one run can't hog the DB.
	kycAutoBatchSize = 200
	// Records stuck in queued/processing/idle are claimed per-run in bounded
	// batches; anything left over is picked up by the next cron tick.
	kycStaleMLBatchSize = 200
)

// ProcessStalePendingEscalations moves stuck records into the manual review
// queue. It never approves a record:
//
//   - 48h hard-stop: auto_eligible + ml_completed records older than 48h.
//   - 24h stale pass: auto_eligible + ml_completed records past the former
//     auto-approval window.
//   - 24h stuck-in-ML pass (Phase 3): queued/processing/idle records the
//     auto_eligible query never sees (Gap 1) — flipped to ml_failed +
//     manual_review so they cannot linger forever.
//
// Every escalated record gets system_escalated_at = NOW() and
// routing_state='routed': a terminal marker. The worker's save_results UPDATE
// refuses (WHERE system_escalated_at IS NULL) to overwrite an escalation, so
// a late/duplicate completion can never resurrect a stuck record, and the
// router's batch query (routing_state='pending') no longer sees it.
//
// Called every 15 minutes by jobs.StartStalePendingEscalator.
func (s *VerificationService) ProcessStalePendingEscalations() {
	started := time.Now()

	// Keep the existing 48h hard-stop first so long-overdue records retain
	// their dedicated audit trail.
	overdue := s.processEscalateBatch()
	stale := s.processStalePendingBatch()
	stuck := s.processStaleMLBatch()

	if overdue > 0 || stale > 0 || stuck > 0 {
		log.Printf("[KYC Escalator] run complete in %s: %d overdue(48h), %d stale(24h) auto_eligible and %d stuck-in-ML record(s) moved to manual review",
			time.Since(started).Round(time.Millisecond), overdue, stale, stuck)
	}
}

// processStalePendingBatch drains auto_eligible records whose former 24h
// auto-approval window has expired. They are escalated, not approved.
func (s *VerificationService) processStalePendingBatch() int {
	return s.escalateBatch(kycStalePendingWindowHours, "AI_ESCALATED_STALE")
}

// processEscalateBatch drains auto_eligible records older than 48h that were
// never decided. This preserves the original hard-stop escalation behavior.
func (s *VerificationService) processEscalateBatch() int {
	return s.escalateBatch(kycAutoEscalateWindowHours, "AI_ESCALATED_48H")
}

// escalateBatch moves a bounded batch of stale auto_eligible records to the
// manual review queue. The update guards on id + status + current priority:
// if the router (or an admin) acted on the record between the queue SELECT
// and this write, RowsAffected is 0 and no audit entry is written.
func (s *VerificationService) escalateBatch(minAgeHours float64, auditAction string) int {
	escalated := 0
	for {
		records, _, err := s.verificationRecordsRepo.GetAutoEligibleQueue(kycAutoBatchSize, 0, minAgeHours)
		if err != nil {
			log.Printf("[KYC Escalator] auto-eligible queue query failed (min age %.0fh): %v (retrying next run)", minAgeHours, err)
			return escalated
		}
		if len(records) == 0 {
			return escalated
		}

		escalatedInBatch := 0
		for _, rec := range records {
			if !s.markEscalated(rec, time.Now()) {
				log.Printf("[KYC Escalator] failed to escalate %s", rec.ID)
				continue
			}
			escalatedInBatch++

			detailsBytes, err := json.Marshal(map[string]interface{}{
				"verification_id": rec.ID,
				"reason":          fmt.Sprintf("%.0fh hold window expired without decision (system escalated)", minAgeHours),
			})
			if err != nil {
				log.Printf("[KYC Escalator] audit details marshal failed for %s: %v", rec.ID, err)
				escalated++
				continue
			}

			if err := s.verificationRecordsRepo.CreateVerificationAuditLog(&models.VerificationAuditLog{
				VerificationID: rec.ID,
				UserID:         rec.UserID,
				ActionType:     auditAction,
				Details:        datatypes.JSON(detailsBytes),
			}); err != nil {
				log.Printf("[KYC Escalator] audit log write failed for %s: %v", rec.ID, err)
			}
			escalated++
		}

		if len(records) < kycAutoBatchSize {
			return escalated
		}
		// Records whose flag update failed still match the query, so a full
		// batch with zero successes would loop forever. Stop and let the next
		// cron run retry.
		if escalatedInBatch == 0 {
			log.Printf("[KYC Escalator] full batch produced no escalations (min age %.0fh); stopping to avoid a hot loop", minAgeHours)
			return escalated
		}
	}
}

// markEscalated stamps the escalation marker in one guarded UPDATE. This
// replaces FlagOverdueAutoEligible so the escalation also becomes terminal for
// the router (routing_state='routed') and for late worker completions
// (system_escalated_at NOT NULL makes the worker's save_results a no-op).
func (s *VerificationService) markEscalated(rec models.VerificationRecord, now time.Time) bool {
	result := postgres.DB.Model(&models.VerificationRecord{}).
		Where("id = ?", rec.ID).
		Where("status = ?", models.VerificationPending).
		Where("review_priority = ?", rec.ReviewPriority).
		Updates(map[string]interface{}{
			"review_priority":           models.ReviewPriorityManual,
			"admin_review_requested_at": now,
			"system_escalated_at":       now,
			"routing_state":             models.RoutingRouted,
		})
	return result.Error == nil && result.RowsAffected > 0
}

// processStaleMLBatch closes Gap 1: records stuck in queued/processing/idle
// that the auto_eligible query never sees (they have no completed ML scores).
// After 24h they are failed into manual review — the Go router does not own
// them, but leaving them in limbo was worse.
//
// The claim runs in ONE transaction (row locks held from SELECT to UPDATE so
// a concurrent escalator can never claim the same rows). FOR UPDATE SKIP
// LOCKED is PostgreSQL-only; the SQLite test database serialises on a single
// connection and gets the same logic without the locking clause.
func (s *VerificationService) processStaleMLBatch() int {
	// Compute the cutoff outside the transaction so every row in one claim
	// batch shares the same age window.
	cutoff := time.Now().Add(-time.Duration(kycStalePendingWindowHours * float64(time.Hour)))
	escalatedAt := time.Now()
	batchSize := kycStaleMLBatchSize

	var escalated int
	err := postgres.DB.Transaction(func(tx *gorm.DB) error {
		selectSQL := `
			SELECT id, user_id, ml_status
			  FROM verification_records
			 WHERE status = 'pending'
			   AND ml_status IN ('queued', 'processing', 'idle')
			   AND COALESCE(submitted_at, created_at) <= ?
			   AND system_escalated_at IS NULL
			 ORDER BY COALESCE(submitted_at, created_at)
			 LIMIT ?`
		if tx.Dialector.Name() == "postgres" {
			selectSQL += " FOR UPDATE SKIP LOCKED"
		}

		rows, err := tx.Raw(selectSQL, cutoff, batchSize).Rows()
		if err != nil {
			return err
		}
		defer rows.Close()

		type staleRow struct {
			id     uuid.UUID
			userID uuid.UUID
			prior  string
		}
		var claimed []staleRow
		for rows.Next() {
			var id, userID uuid.UUID
			var prior string
			if err := rows.Scan(&id, &userID, &prior); err != nil {
				return err
			}
			claimed = append(claimed, staleRow{id: id, userID: userID, prior: prior})
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if len(claimed) == 0 {
			return nil
		}

		ids := make([]uuid.UUID, 0, len(claimed))
		for _, row := range claimed {
			ids = append(ids, row.id)
		}

		// Recheck inside the UPDATE as belt-and-braces (a row marked escalated
		// after our SELECT but before this UPDATE cannot slip through). GORM
		// sets updated_at automatically for Updates on this model.
		result := tx.Model(&models.VerificationRecord{}).
			Where("id IN ?", ids).
			Where("status = ?", models.VerificationPending).
			Where("ml_status IN ?", []models.MLVerificationStatus{
				models.MLStatusQueued, models.MLStatusProcessing, models.MLStatusIdle,
			}).
			Where("system_escalated_at IS NULL").
			Updates(map[string]interface{}{
				"ml_status":           models.MLStatusFailed,
				"review_priority":     models.ReviewPriorityManual,
				"routing_state":       models.RoutingRouted,
				"system_escalated_at": escalatedAt,
				"decision_source":     "system_stale",
			})
		if result.Error != nil {
			return result.Error
		}

		// One audit entry per escalated row (prior state preserved for ops).
		for _, row := range claimed {
			details, err := json.Marshal(map[string]interface{}{
				"verification_id": row.id,
				"reason":          "stuck_in_ml",
				"prior_ml_status": row.prior,
			})
			if err != nil {
				log.Printf("[KYC Escalator] stale-ML audit marshal failed for %s: %v", row.id, err)
				continue
			}
			if err := tx.Create(&models.VerificationAuditLog{
				VerificationID: row.id,
				UserID:         row.userID,
				ActionType:     "SYSTEM_ESCALATED_STALE_24H",
				Details:        datatypes.JSON(details),
			}).Error; err != nil {
				log.Printf("[KYC Escalator] stale-ML audit write failed for %s: %v", row.id, err)
			}
		}
		escalated = int(result.RowsAffected)
		return nil
	})
	if err != nil {
		log.Printf("[KYC Escalator] stuck-in-ML escalation failed: %v (retrying next run)", err)
		return 0
	}
	return escalated
}
