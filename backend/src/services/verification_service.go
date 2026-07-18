package services

import (
	"errors"
	"fmt"
	"mime/multipart"
	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
	"techguild-backend/src/utils"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type VerificationService struct {
	userRepo                repository.UserRepository
	verificationRecordsRepo repository.VerificationRecordsRepository
}

func NewVerificationService(redisClient *redis.Client) *VerificationService {
	return &VerificationService{
		userRepo:                repository.NewUserRepository(),
		verificationRecordsRepo: repository.NewVerificationRecordsRepository(),
	}
}
func (s *VerificationService) SubmitIdentityVerification(
	userID string,
	req dto.IdentityVerificationRequest,
	govtIDDocument *multipart.FileHeader,
	selfie *multipart.FileHeader,
) (*dto.IdentityVerificationResponse, error) {

	// -----------------------------
	// Validate User
	// -----------------------------

	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if !user.EmailVerified {
		return nil, errors.New("email is not verified")
	}

	// -----------------------------
	// Existing Verification Check
	// -----------------------------

	existingRecord, _ := s.verificationRecordsRepo.GetVerificationByUser(userID)

	if existingRecord != nil &&
		(existingRecord.Status == models.VerificationPending ||
			existingRecord.Status == models.VerificationReview) {

		return nil, errors.New("verification already submitted")
	}

	// -----------------------------
	// Hash Government ID
	// -----------------------------

	govtHash := utils.HashGovtID(req.GovtIDNumber)

	// -----------------------------
	// Duplicate Government ID Check
	// -----------------------------

	exists, err := s.verificationRecordsRepo.GovtHashExists(govtHash)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, errors.New("this government ID is already associated with another account")
	}

	// -----------------------------
	// Upload Government ID Document
	// -----------------------------

	govtFile, err := govtIDDocument.Open()
	if err != nil {
		return nil, err
	}
	defer govtFile.Close()

	govtFileURL, err := utils.UploadVerificationDocument(
		govtFile,
		fmt.Sprintf("%s_govt_%s", userID, uuid.New().String()),
	)
	if err != nil {
		return nil, err
	}

	// -----------------------------
	// Upload Selfie
	// -----------------------------

	selfieFile, err := selfie.Open()
	if err != nil {
		return nil, err
	}
	defer selfieFile.Close()

	selfieURL, err := utils.UploadVerificationDocument(
		selfieFile,
		fmt.Sprintf("%s_selfie_%s", userID, uuid.New().String()),
	)
	if err != nil {
		return nil, err
	}

	// -----------------------------
	// Create Verification Record
	// -----------------------------

	record := &models.VerificationRecord{
		UserID: uuid.MustParse(userID),
		Type:   models.VerificationIndividual,
		Status: models.VerificationPending,
		Vendor: "hyperverge",
	}

	err = s.verificationRecordsRepo.CreateVerification(record)
	if err != nil {
		return nil, err
	}

	// -----------------------------
	// Save Government ID Document
	// -----------------------------

	err = s.verificationRecordsRepo.CreateDocument(
		&models.VerificationDocument{
			VerificationRecordID: record.ID,
			DocumentType:         "government_id",
			FileURL:              govtFileURL,
		},
	)
	if err != nil {
		return nil, err
	}

	// -----------------------------
	// Save Selfie
	// -----------------------------

	err = s.verificationRecordsRepo.CreateDocument(
		&models.VerificationDocument{
			VerificationRecordID: record.ID,
			DocumentType:         "selfie",
			FileURL:              selfieURL,
		},
	)
	if err != nil {
		return nil, err
	}

	// -----------------------------
	// TODO:
	// Submit documents to HyperVerge
	// Update VendorReferenceID
	// -----------------------------

	return &dto.IdentityVerificationResponse{
		Message:              "Verification submitted successfully",
		VerificationRecordID: record.ID.String(),
	}, nil
}

// =====================================================
// Get Individual Verification Status
// =====================================================

func (s *VerificationService) GetIdentityVerificationStatus(
	userID string,
) (*dto.IdentityStatusResponse, error) {

	record, err := s.verificationRecordsRepo.GetVerificationByUser(userID)
	if err != nil {
		return nil, errors.New("verification record not found")
	}

	return &dto.IdentityStatusResponse{
		Status:          string(record.Status),
		RejectionReason: record.RejectionReason,
	}, nil
}

// =====================================================
// Submit Business Verification
// =====================================================

func (s *VerificationService) SubmitBusinessVerification(
	userID string,
	req dto.BusinessVerificationRequest,
	gstCertificate *multipart.FileHeader,
	panCard *multipart.FileHeader,
	authorizedRepresentative *multipart.FileHeader,
) (*dto.BusinessVerificationResponse, error) {

	//----------------------------------------------------
	// Validate User
	//----------------------------------------------------

	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if user.AccountType == nil {
		return nil, errors.New("account type not found")
	}

	if *user.AccountType != models.AccountTypeAgencyAdmin &&
		*user.AccountType != models.AccountTypeClientAdmin {
		return nil, errors.New("only agency/client admins can verify business")
	}

	//----------------------------------------------------
	// Existing Verification
	//----------------------------------------------------

	record, _ := s.verificationRecordsRepo.GetVerificationByUser(userID)

	if record != nil &&
		(record.Status == models.VerificationPending ||
			record.Status == models.VerificationReview) {

		return nil, errors.New("verification already submitted")
	}

	//----------------------------------------------------
	// Hash Business PAN
	//----------------------------------------------------

	panHash := utils.HashBusinessPAN(req.BusinessPAN)

	exists, err := s.verificationRecordsRepo.BusinessHashExists(panHash)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, errors.New("business PAN already exists")
	}

	//----------------------------------------------------
	// TODO
	// Verify GST Returns
	//----------------------------------------------------

	//----------------------------------------------------
	// TODO
	// Verify MCA Representative
	//----------------------------------------------------

	//----------------------------------------------------
	// TODO
	// ₹1 Bank Verification
	//----------------------------------------------------

	//----------------------------------------------------
	// Upload GST Certificate
	//----------------------------------------------------

	gstFile, err := gstCertificate.Open()
	if err != nil {
		return nil, err
	}
	defer gstFile.Close()

	gstURL, err := utils.UploadVerificationDocument(
		gstFile,
		fmt.Sprintf("%s_gst_%s", userID, uuid.New().String()),
	)
	if err != nil {
		return nil, err
	}

	//----------------------------------------------------
	// Upload PAN Card
	//----------------------------------------------------

	panFile, err := panCard.Open()
	if err != nil {
		return nil, err
	}
	defer panFile.Close()

	panURL, err := utils.UploadVerificationDocument(
		panFile,
		fmt.Sprintf("%s_pan_%s", userID, uuid.New().String()),
	)
	if err != nil {
		return nil, err
	}

	//----------------------------------------------------
	// Upload Authorized Representative ID
	//----------------------------------------------------

	repFile, err := authorizedRepresentative.Open()
	if err != nil {
		return nil, err
	}
	defer repFile.Close()

	repURL, err := utils.UploadVerificationDocument(
		repFile,
		fmt.Sprintf("%s_rep_%s", userID, uuid.New().String()),
	)
	if err != nil {
		return nil, err
	}

	//----------------------------------------------------
	// Create Verification Record
	//----------------------------------------------------

	newRecord := &models.VerificationRecord{
		UserID: uuid.MustParse(userID),
		Type:   models.VerificationBusiness,
		Status: models.VerificationReview,
		Vendor: "manual_review",
	}

	err = s.verificationRecordsRepo.CreateVerification(newRecord)
	if err != nil {
		return nil, err
	}

	//----------------------------------------------------
	// Save Documents
	//----------------------------------------------------

	documents := []*models.VerificationDocument{
		{
			VerificationRecordID: newRecord.ID,
			DocumentType:         "gst_certificate",
			FileURL:              gstURL,
		},
		{
			VerificationRecordID: newRecord.ID,
			DocumentType:         "pan_card",
			FileURL:              panURL,
		},
		{
			VerificationRecordID: newRecord.ID,
			DocumentType:         "authorized_representative_id",
			FileURL:              repURL,
		},
	}

	for _, document := range documents {

		err := s.verificationRecordsRepo.CreateDocument(document)
		if err != nil {
			return nil, err
		}
	}

	return &dto.BusinessVerificationResponse{
		Message:              "Business verification submitted successfully",
		VerificationRecordID: newRecord.ID.String(),
	}, nil
}

// =====================================================
// Resubmit Verification
// =====================================================

func (s *VerificationService) ResubmitVerification(
	userID string,
	recordID string,
	req dto.ResubmitVerificationRequest,
	document *multipart.FileHeader,
) (*dto.ResubmitVerificationResponse, error) {

	//----------------------------------------------------
	// Fetch Previous Verification
	//----------------------------------------------------

	oldRecord, err := s.verificationRecordsRepo.GetVerificationByID(recordID)
	if err != nil {
		return nil, errors.New("verification record not found")
	}

	//----------------------------------------------------
	// Check Ownership
	//----------------------------------------------------

	if oldRecord.UserID.String() != userID {
		return nil, errors.New("you are not authorized to resubmit this verification")
	}

	//----------------------------------------------------
	// Only Rejected Records Can Be Resubmitted
	//----------------------------------------------------

	if oldRecord.Status != models.VerificationRejected {
		return nil, errors.New("only rejected verifications can be resubmitted")
	}

	//----------------------------------------------------
	// Upload New Document
	//----------------------------------------------------

	file, err := document.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	FileURL, err := utils.UploadVerificationDocument(
		file,
		fmt.Sprintf("%s_resubmit_%s", userID, uuid.New().String()),
	)
	if err != nil {
		return nil, err
	}

	//----------------------------------------------------
	// Create New Verification Record
	//----------------------------------------------------

	newRecord := &models.VerificationRecord{
		UserID:           oldRecord.UserID,
		Type:             oldRecord.Type,
		Status:           models.VerificationPending,
		Vendor:           oldRecord.Vendor,
		PreviousRecordID: &oldRecord.ID,
	}

	err = s.verificationRecordsRepo.CreateVerification(newRecord)
	if err != nil {
		return nil, err
	}

	//----------------------------------------------------
	// Save Uploaded Document
	//----------------------------------------------------

	err = s.verificationRecordsRepo.CreateDocument(
		&models.VerificationDocument{
			VerificationRecordID: newRecord.ID,
			DocumentType:         req.DocumentType,
			FileURL:              FileURL,
		},
	)
	if err != nil {
		return nil, err
	}

	//----------------------------------------------------
	// TODO:
	// Re-submit to HyperVerge if Individual Verification
	//----------------------------------------------------

	return &dto.ResubmitVerificationResponse{
		Message:              "Verification resubmitted successfully",
		VerificationRecordID: newRecord.ID.String(),
	}, nil
}

// =====================================================
// Admin Verification Queue
// =====================================================

func (s *VerificationService) GetVerificationQueue() ([]dto.VerificationQueueItem, error) {

	records, err := s.verificationRecordsRepo.GetPendingVerifications()
	if err != nil {
		return nil, err
	}

	queue := make([]dto.VerificationQueueItem, 0)

	for _, record := range records {

		queue = append(queue, dto.VerificationQueueItem{
			ID:        record.ID.String(),
			UserID:    record.UserID.String(),
			Type:      string(record.Type),
			Status:    string(record.Status),
			CreatedAt: record.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return queue, nil
}

// =====================================================
// Approve Verification
// =====================================================

func (s *VerificationService) ApproveVerification(recordID string) error {

	record, err := s.verificationRecordsRepo.GetVerificationByID(recordID)
	if err != nil {
		return errors.New("verification record not found")
	}

	if record.Status == models.VerificationApproved {
		return nil
	}

	err = s.verificationRecordsRepo.UpdateStatus(
		recordID,
		models.VerificationApproved,
		"",
	)
	if err != nil {
		return err
	}

	user, err := s.userRepo.GetUserByID(record.UserID.String())
	if err != nil {
		return err
	}

	user.Status = models.StatusActive

	err = s.userRepo.UpdateUser(user)
	if err != nil {
		return err
	}

	if record.Type == models.VerificationIndividual &&
		record.GovtIDHash != "" {

		err = s.verificationRecordsRepo.CreateGovtHash(
			&models.GovtIDDedup{
				GovtIDHash: record.GovtIDHash,
				UserID:     record.UserID,
			},
		)
		if err != nil {
			return err
		}
	}

	if record.Type == models.VerificationBusiness &&
		record.BusinessPANHash != "" {

		err = s.verificationRecordsRepo.CreateBusinessHash(
			&models.BusinessPANDedup{
				PANHash: record.BusinessPANHash,
				UserID:  record.UserID,
			},
		)
		if err != nil {
			return err
		}
	}

	// TODO: Send notification
	// TODO: Create audit log

	return nil
}

// =====================================================
// Reject Verification
// =====================================================

func (s *VerificationService) RejectVerification(
	recordID string,
	reason string,
) error {

	_, err := s.verificationRecordsRepo.GetVerificationByID(recordID)
	if err != nil {
		return errors.New("verification record not found")
	}

	err = s.verificationRecordsRepo.UpdateStatus(
		recordID,
		models.VerificationRejected,
		reason,
	)
	if err != nil {
		return err
	}

	// TODO: Send notification
	// TODO: Create audit log

	return nil
}