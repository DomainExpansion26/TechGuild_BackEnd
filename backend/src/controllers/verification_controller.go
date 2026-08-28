package controllers

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"regexp"
	"strings"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
)

func validateUploadedFile(file *multipart.FileHeader) error {
	// Size limit: 5MB (5 * 1024 * 1024 bytes)
	const MaxFileSize = 5 * 1024 * 1024
	if file.Size > MaxFileSize {
		return fmt.Errorf("file %s exceeds the maximum limit of 5MB", file.Filename)
	}

	// Extension check
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" && ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return fmt.Errorf("file %s has an unsupported format. Allowed formats are: .pdf, .png, .jpeg", file.Filename)
	}

	return nil
}

func getFormFile(form multipart.Form, name string) *multipart.FileHeader {
	files := form.File[name]
	if len(files) == 0 {
		return nil
	}
	return files[0]
}

func getFormValue(form multipart.Form, name string) string {
	vals := form.Value[name]
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

// =========================
// Submit Individual Verification (file-upload, Huma)
// =========================

func SubmitIdentityVerificationHandler(ctx context.Context, input *dto.SubmitIdentityVerificationInput) (*dto.SubmitIdentityVerificationOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	govtIDDocument := getFormFile(input.RawBody, "govt_id_document")
	if govtIDDocument == nil {
		return nil, huma.Error400BadRequest("government ID document is required")
	}
	if err := validateUploadedFile(govtIDDocument); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	selfie := getFormFile(input.RawBody, "selfie")
	if selfie == nil {
		return nil, huma.Error400BadRequest("selfie is required")
	}
	if err := validateUploadedFile(selfie); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	req := dto.IdentityVerificationRequest{
		GovtIDType:   getFormValue(input.RawBody, "govt_id_type"),
		GovtIDNumber: getFormValue(input.RawBody, "govt_id_number"),
	}

	verificationService := services.NewVerificationService(postgres.RedisDB)
	res, err := verificationService.SubmitIdentityVerification(userID, req, govtIDDocument, selfie)
	if err != nil {
		if err.Error() == "this government ID is already associated with another account" {
			return nil, huma.Error409Conflict(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.SubmitIdentityVerificationOutput{Body: *res}, nil
}

// =========================
// Get Identity Verification Status
// =========================

func GetIdentityVerificationStatusHandler(ctx context.Context, input *dto.GetIdentityVerificationStatusInput) (*dto.GetIdentityVerificationStatusOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	verificationService := services.NewVerificationService(postgres.RedisDB)
	res, err := verificationService.GetIdentityVerificationStatus(userID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.GetIdentityVerificationStatusOutput{Body: *res}, nil
}

// =========================
// Get Verification Status (generic)
// =========================

func GetVerificationStatusHandler(ctx context.Context, input *dto.GetVerificationStatusInput) (*dto.GetVerificationStatusOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	verificationService := services.NewVerificationService(postgres.RedisDB)
	res, err := verificationService.GetVerificationStatus(userID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.GetVerificationStatusOutput{Body: *res}, nil
}

// =========================
// Submit Business Verification (file-upload, Huma)
// =========================

func SubmitBusinessVerificationHandler(ctx context.Context, input *dto.SubmitBusinessVerificationInput) (*dto.SubmitBusinessVerificationOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	req := dto.BusinessVerificationRequest{
		BusinessName:       getFormValue(input.RawBody, "business_name"),
		BusinessPAN:        getFormValue(input.RawBody, "business_pan"),
		GSTNumber:          getFormValue(input.RawBody, "gst_number"),
		RegistrationNumber: getFormValue(input.RawBody, "registration_number"),
		Website:            getFormValue(input.RawBody, "website"),
		Country:            getFormValue(input.RawBody, "country"),
		BankName:           getFormValue(input.RawBody, "bank_name"),
		AccountHolderName:  getFormValue(input.RawBody, "account_holder_name"),
		BankAccountNumber:  getFormValue(input.RawBody, "bank_account_number"),
		BankIFSC:           getFormValue(input.RawBody, "bank_ifsc"),
	}

	panRegex := regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]{1}$`)
	if !panRegex.MatchString(strings.ToUpper(req.BusinessPAN)) {
		return nil, huma.Error400BadRequest("invalid business PAN format")
	}

	gstRegex := regexp.MustCompile(`^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z]{1}[1-9A-Z]{1}Z[0-9A-Z]{1}$`)
	if !gstRegex.MatchString(strings.ToUpper(req.GSTNumber)) {
		return nil, huma.Error400BadRequest("invalid GST number format")
	}

	gstCertificate := getFormFile(input.RawBody, "gst_certificate")
	if gstCertificate == nil {
		return nil, huma.Error400BadRequest("gst_certificate is required")
	}
	if err := validateUploadedFile(gstCertificate); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	incorporationCertificate := getFormFile(input.RawBody, "incorporation_certificate")
	if incorporationCertificate == nil {
		return nil, huma.Error400BadRequest("incorporation_certificate is required")
	}
	if err := validateUploadedFile(incorporationCertificate); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	panCard := getFormFile(input.RawBody, "pan_card")
	if panCard == nil {
		return nil, huma.Error400BadRequest("pan_card is required")
	}
	if err := validateUploadedFile(panCard); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	businessRegistration := getFormFile(input.RawBody, "business_registration")
	if businessRegistration == nil {
		return nil, huma.Error400BadRequest("business_registration is required")
	}
	if err := validateUploadedFile(businessRegistration); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	cancelledCheque := getFormFile(input.RawBody, "cancelled_cheque")
	if cancelledCheque == nil {
		return nil, huma.Error400BadRequest("cancelled_cheque is required")
	}
	if err := validateUploadedFile(cancelledCheque); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	verificationService := services.NewVerificationService(postgres.RedisDB)
	res, err := verificationService.SubmitBusinessVerification(userID, req, gstCertificate, incorporationCertificate, panCard, businessRegistration, cancelledCheque)
	if err != nil {
		if err.Error() == "business PAN already exists" {
			return nil, huma.Error409Conflict(err.Error())
		}
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.SubmitBusinessVerificationOutput{Body: *res}, nil
}

// =========================
// Resubmit Verification (file-upload, Huma)
// =========================

func ResubmitVerificationHandler(ctx context.Context, input *dto.ResubmitVerificationInput) (*dto.ResubmitVerificationOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	govtIDDocument := getFormFile(input.RawBody, "govt_id_document")
	if govtIDDocument == nil {
		return nil, huma.Error400BadRequest("government ID document is required")
	}
	if err := validateUploadedFile(govtIDDocument); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	req := dto.ResubmitVerificationRequest{
		DocumentType: getFormValue(input.RawBody, "document_type"),
	}

	verificationService := services.NewVerificationService(postgres.RedisDB)
	res, err := verificationService.ResubmitVerification(userID, input.RecordID, req, govtIDDocument)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.ResubmitVerificationOutput{Body: *res}, nil
}

// =========================
// Admin Queue / Approve / Reject (Huma)
// =========================

func GetVerificationQueueHandler(ctx context.Context, input *dto.GetVerificationQueueInput) (*dto.GetVerificationQueueOutput, error) {
	verificationService := services.NewVerificationService(postgres.RedisDB)

	queue, err := verificationService.GetVerificationQueue()
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.GetVerificationQueueOutput{Body: dto.VerificationQueueResponse{Queue: queue}}, nil
}

func ApproveVerificationHandler(ctx context.Context, input *dto.ApproveVerificationInput) (*dto.ApproveVerificationOutput, error) {
	verificationService := services.NewVerificationService(postgres.RedisDB)

	if err := verificationService.ApproveVerification(input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.ApproveVerificationOutput{Body: dto.AdminApproveResponse{Message: "Verification approved successfully"}}, nil
}

func RejectVerificationHandler(ctx context.Context, input *dto.RejectVerificationInput) (*dto.RejectVerificationOutput, error) {
	verificationService := services.NewVerificationService(postgres.RedisDB)

	if err := verificationService.RejectVerification(input.ID, input.Body.Reason); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.RejectVerificationOutput{Body: dto.AdminRejectResponse{Message: "Verification rejected successfully"}}, nil
}
