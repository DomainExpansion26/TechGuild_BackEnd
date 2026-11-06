package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"
)

type VerificationController struct {
	service *services.VerificationService
}

func NewVerificationController(service *services.VerificationService) *VerificationController {
	return &VerificationController{service: service}
}

// Helper to map service errors to Huma HTTP errors
func mapVerificationError(err error) error {
	if err == nil {
		return nil
	}

	// Map specific business logic errors to 409 Conflict
	if errors.Is(err, services.ErrDuplicateGovtID) || errors.Is(err, services.ErrDuplicatePAN) {
		return huma.Error409Conflict(err.Error())
	}

	// Map state errors to 400 Bad Request
	if errors.Is(err, services.ErrAlreadySubmitted) || errors.Is(err, services.ErrInvalidState) {
		return huma.Error400BadRequest(err.Error())
	}

	// Document type must match the account type (individual vs business).
	if errors.Is(err, services.ErrInvalidDocTypeForAccount) {
		return huma.Error400BadRequest("This document type is not allowed for your account type")
	}

	// Default to 400 for business logic validation
	return huma.Error400BadRequest(err.Error())
}

// ==========================================
// 1. User: Info Updates
// ==========================================

func (c *VerificationController) UpdateIndividualInfoHandler(ctx context.Context, input *dto.UpdateIndividualInfoInput) (*dto.UpdateIndividualInfoOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	if err := c.service.UpdateIndividualInfo(userID, input.Body); err != nil {
		return nil, mapVerificationError(err)
	}

	return &dto.UpdateIndividualInfoOutput{
		Body: dto.VerificationMessageResponse{Message: "Individual info saved to draft"},
	}, nil
}

func (c *VerificationController) UpdateBusinessInfoHandler(ctx context.Context, input *dto.UpdateBusinessInfoInput) (*dto.UpdateBusinessInfoOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	if err := c.service.UpdateBusinessInfo(userID, input.Body); err != nil {
		return nil, mapVerificationError(err)
	}

	return &dto.UpdateBusinessInfoOutput{
		Body: dto.VerificationMessageResponse{Message: "Business info saved to draft"},
	}, nil
}

func (c *VerificationController) UpdateBankDetailsHandler(ctx context.Context, input *dto.UpdateBankDetailsInput) (*dto.UpdateBankDetailsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	if err := c.service.UpdateBankDetails(userID, input.Body); err != nil {
		return nil, mapVerificationError(err)
	}

	return &dto.UpdateBankDetailsOutput{
		Body: dto.VerificationMessageResponse{Message: "Bank details securely saved"},
	}, nil
}

// ==========================================
// 2. User: Upload Document (Multipart)
// ==========================================

func (c *VerificationController) UploadDocumentHandler(ctx context.Context, input *dto.UploadDocumentInput) (*dto.UploadDocumentOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	form := input.RawBody.Data()

	// ✅ FIXED: form.File implements multipart.File. Pass it directly with filename.
	res, err := c.service.UploadDocument(userID, form.DocumentType, form.File, form.File.Filename)
	if err != nil {
		return nil, mapVerificationError(err)
	}

	return &dto.UploadDocumentOutput{Body: *res}, nil
}

// ==========================================
// 3. User: Submit & Status
// ==========================================

func (c *VerificationController) SubmitVerificationHandler(ctx context.Context, input *dto.SubmitVerificationInput) (*dto.SubmitVerificationOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	if err := c.service.SubmitVerification(userID); err != nil {
		return nil, mapVerificationError(err)
	}

	return &dto.SubmitVerificationOutput{
		Body: dto.VerificationMessageResponse{Message: "Verification submitted successfully. Pending review."},
	}, nil
}

func (c *VerificationController) GetStatusHandler(ctx context.Context, input *dto.GetVerificationStatusInput) (*dto.GetVerificationStatusOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	status, err := c.service.GetStatus(userID)
	if err != nil {
		return nil, mapVerificationError(err)
	}

	return &dto.GetVerificationStatusOutput{Body: *status}, nil
}

// ==========================================
// 4. Admin Flow
// ==========================================

func (c *VerificationController) AdminGetQueueHandler(ctx context.Context, input *dto.GetAdminPendingListInput) (*dto.GetAdminPendingListOutput, error) {
	items, err := c.service.AdminGetQueue()
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	return &dto.GetAdminPendingListOutput{
		Body: dto.AdminVerificationListResponse{Items: items},
	}, nil
}

func (c *VerificationController) AdminReviewHandler(ctx context.Context, input *dto.AdminReviewInput) (*dto.AdminReviewOutput, error) {
	// adminID extraction (agar future mein audit log ke liye chahiye)
	adminID, _ := utils.GetUserIDFromHumaContext(ctx)

	if err := c.service.AdminReview(adminID, input.ID, input.Body); err != nil {
		return nil, mapVerificationError(err)
	}

	return &dto.AdminReviewOutput{
		Body: dto.VerificationMessageResponse{Message: "Verification reviewed successfully"},
	}, nil
}

// ==========================================
// 5. Admin Flow: KYC AI Pipeline (Phase D)
// ==========================================

const (
	adminQueueDefaultLimit = 20
	adminQueueMaxLimit     = 100
)

// mapAdminPipelineError maps the AI-pipeline service errors to Huma errors.
// Business-rule violations -> 400, missing records -> 404, everything else
// (DB/network failures) -> 500 so it surfaces as a server problem, not a
// client payload problem.
func mapAdminPipelineError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, services.ErrInvalidAdminAction) {
		return huma.Error400BadRequest(err.Error())
	}
	if errors.Is(err, services.ErrMLQueueFailed) {
		return huma.Error503ServiceUnavailable(err.Error())
	}
	if strings.Contains(err.Error(), "not found") {
		return huma.Error404NotFound(err.Error())
	}
	return huma.Error500InternalServerError(err.Error())
}

// clampQueueParams normalizes admin pagination inputs.
func clampQueueParams(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = adminQueueDefaultLimit
	}
	if limit > adminQueueMaxLimit {
		limit = adminQueueMaxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// queuePaginationFrom computes the pagination envelope.
func queuePaginationFrom(limit, offset int, total int64) dto.QueuePaginationMeta {
	return dto.QueuePaginationMeta{
		Limit:   limit,
		Offset:  offset,
		Total:   total,
		HasMore: int64(offset+limit) < total,
	}
}

// unmarshalJSONB safely decodes an optional JSONB blob into a generic map.
func unmarshalJSONB(blob []byte) map[string]interface{} {
	if len(blob) == 0 {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(blob, &m); err != nil {
		return nil
	}
	return m
}

// extractJSONFloat pulls the first present numeric key (number or numeric
// string) from a vendor result map. Keys have fallbacks because vendor payload
// shapes vary slightly between runs/models.
func extractJSONFloat(m map[string]interface{}, keys ...string) *float64 {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch f := v.(type) {
		case float64:
			return &f
		case string:
			if parsed, err := strconv.ParseFloat(f, 64); err == nil {
				return &parsed
			}
		}
	}
	return nil
}

// extractJSONString pulls the first present string key.
func extractJSONString(m map[string]interface{}, keys ...string) *string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			return &s
		}
	}
	return nil
}

// parseAIScores converts the JSONB result columns into the DTO. Returns nil if
// none of the blobs contained any usable data.
func parseAIScores(rec *models.VerificationRecord) *dto.VerificationAIScores {
	scores := &dto.VerificationAIScores{}
	populated := false

	if face := unmarshalJSONB(rec.FaceMatchResult); face != nil {
		if v := extractJSONFloat(face, "face_similarity", "similarity", "face_match_score"); v != nil {
			scores.FaceSimilarity = v
			populated = true
		}
	}
	if live := unmarshalJSONB(rec.LivenessResult); live != nil {
		if v := extractJSONFloat(live, "liveness_score", "score"); v != nil {
			scores.LivenessScore = v
			populated = true
		}
	}
	if ocr := unmarshalJSONB(rec.OCROcrResult); ocr != nil {
		if v := extractJSONFloat(ocr, "ocr_confidence", "confidence", "score"); v != nil {
			scores.OCRConfidence = v
			populated = true
		}
		if v := extractJSONString(ocr, "ocr_name", "full_name", "name"); v != nil {
			scores.OCRName = v
			populated = true
		}
		if v := extractJSONString(ocr, "ocr_dob", "date_of_birth", "dob"); v != nil {
			scores.OCRDOB = v
			populated = true
		}
	}

	if !populated {
		return nil
	}
	return scores
}

// buildDocRefs maps stored documents to url/type refs for admin rendering.
func buildDocRefs(docs []models.VerificationDocument) []dto.VerificationDocRef {
	refs := make([]dto.VerificationDocRef, 0, len(docs))
	for _, d := range docs {
		refs = append(refs, dto.VerificationDocRef{Type: d.DocumentType, URL: d.FileURL})
	}
	return refs
}

// buildVerificationQueueItem assembles a queue row: identity + pipeline state
// + parsed AI scores + document URLs. Documents are passed in by the handler
// (fetched via the service, since queue queries preload only the User).
func buildVerificationQueueItem(rec *models.VerificationRecord, docs []models.VerificationDocument) dto.VerificationQueueItem {
	ageHours := 0.0
	if rec.MLCompletedAt != nil {
		ageHours = time.Since(*rec.MLCompletedAt).Hours()
		if ageHours < 0 {
			ageHours = 0
		}
	}

	return dto.VerificationQueueItem{
		ID:               rec.ID.String(),
		UserID:           rec.UserID.String(),
		UserEmail:        rec.User.Email,
		VerificationType: string(rec.Type),
		SubmittedAt:      rec.SubmittedAt,
		MLCompletedAt:    rec.MLCompletedAt,
		AgeHours:         ageHours,
		ReviewPriority:   string(rec.ReviewPriority),
		AIScores:         parseAIScores(rec),
		DocumentURLs:     buildDocRefs(docs),
		Status:           string(rec.Status),
	}
}

func (c *VerificationController) AdminGetAutoEligibleHandler(ctx context.Context, input *dto.GetAutoEligibleQueueInput) (*dto.GetAutoEligibleQueueOutput, error) {
	// Middleware enforces admin auth; the admin ID is kept here for parity
	// with the rest of the admin handlers.
	if _, err := utils.GetUserIDFromHumaContext(ctx); err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	limit, offset := clampQueueParams(input.Limit, input.Offset)

	records, total, err := c.service.AdminGetAutoEligible(limit, offset, input.MinAgeHours)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	items := make([]dto.VerificationQueueItem, 0, len(records))
	for i := range records {
		docs, _ := c.service.GetRecordDocuments(records[i].ID.String())
		items = append(items, buildVerificationQueueItem(&records[i], docs))
	}

	return &dto.GetAutoEligibleQueueOutput{
		Body: dto.GetAutoEligibleQueueResponse{
			Items:      items,
			Pagination: queuePaginationFrom(limit, offset, total),
		},
	}, nil
}

func (c *VerificationController) AdminGetManualReviewHandler(ctx context.Context, input *dto.GetManualReviewQueueInput) (*dto.GetManualReviewQueueOutput, error) {
	if _, err := utils.GetUserIDFromHumaContext(ctx); err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	limit, offset := clampQueueParams(input.Limit, input.Offset)

	records, total, err := c.service.AdminGetManualReview(input.SortBy, input.SortDir, limit, offset)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	items := make([]dto.VerificationQueueItem, 0, len(records))
	for i := range records {
		docs, _ := c.service.GetRecordDocuments(records[i].ID.String())
		items = append(items, buildVerificationQueueItem(&records[i], docs))
	}

	return &dto.GetManualReviewQueueOutput{
		Body: dto.GetManualReviewQueueResponse{
			Items:      items,
			Pagination: queuePaginationFrom(limit, offset, total),
		},
	}, nil
}

func (c *VerificationController) AdminGetDetailHandler(ctx context.Context, input *dto.AdminGetDetailInput) (*dto.AdminGetDetailOutput, error) {
	if _, err := utils.GetUserIDFromHumaContext(ctx); err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	rec, err := c.service.AdminGetDetail(input.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, huma.Error404NotFound("verification record not found")
		}
		return nil, mapAdminPipelineError(err)
	}

	docs, _ := c.service.GetRecordDocuments(rec.ID.String())

	body := dto.AdminVerificationDetailResponse{
		ID:                     rec.ID.String(),
		UserID:                 rec.UserID.String(),
		UserEmail:              rec.User.Email,
		VerificationType:       string(rec.Type),
		Status:                 string(rec.Status),
		RejectionReason:        rec.RejectionReason,
		ReviewPriority:         string(rec.ReviewPriority),
		MLStatus:               string(rec.MLStatus),
		MLJobID:                rec.MLJobID,
		MLSubmittedAt:          rec.MLSubmittedAt,
		MLCompletedAt:          rec.MLCompletedAt,
		MLVendor:               rec.MLVendor,
		MLModelVersion:         rec.MLModelVersion,
		MLProcessingTime:       rec.MLProcessingTime,
		SubmittedAt:            rec.SubmittedAt,
		VerifiedAt:             rec.VerifiedAt,
		BulkApprovedAt:         rec.BulkApprovedAt,
		AutoApprovedAt:         rec.AutoApprovedAt,
		RejectedAt:             rec.RejectedAt,
		AIScores:               parseAIScores(rec),
		DocumentURLs:           buildDocRefs(docs),
		BulkReviewRequestedAt:  rec.BulkReviewRequestedAt,
		AdminReviewRequestedAt: rec.AdminReviewRequestedAt,
	}
	if rec.AdminReviewRequestedBy != nil {
		body.AdminReviewRequestedBy = rec.AdminReviewRequestedBy.String()
	}

	return &dto.AdminGetDetailOutput{Body: body}, nil
}

func (c *VerificationController) AdminBulkApproveHandler(ctx context.Context, input *dto.AdminBulkApproveInput) (*dto.AdminBulkApproveOutput, error) {
	adminID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	// 400 for an empty list (body minItems catches it too, but the service
	// boundary stays explicit for direct calls).
	if len(input.Body.VerificationIDs) == 0 {
		return nil, huma.Error400BadRequest("verification_ids must not be empty")
	}

	if err := c.service.AdminBulkApprove(input.Body.VerificationIDs, adminID); err != nil {
		return nil, mapAdminPipelineError(err)
	}

	return &dto.AdminBulkApproveOutput{
		Body: dto.BulkApproveResponse{
			ApprovedCount: len(input.Body.VerificationIDs),
			Message:       "Verifications approved successfully",
		},
	}, nil
}

func (c *VerificationController) AdminManualActionHandler(ctx context.Context, input *dto.AdminManualActionInput) (*dto.AdminManualActionOutput, error) {
	adminID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	action := strings.ToUpper(strings.TrimSpace(input.Body.Action))

	// REJECT requires a reason; APPROVE carries the optional manual note.
	reason := input.Body.ManualNote
	if action == "REJECT" {
		if strings.TrimSpace(input.Body.RejectionReason) == "" {
			return nil, huma.Error400BadRequest("rejection_reason is required when action is REJECT")
		}
		reason = input.Body.RejectionReason
	}

	if err := c.service.AdminManualAction(input.ID, action, adminID, reason); err != nil {
		return nil, mapAdminPipelineError(err)
	}

	return &dto.AdminManualActionOutput{
		Body: dto.VerificationMessageResponse{Message: "Verification " + strings.ToLower(action) + "d successfully"},
	}, nil
}

func (c *VerificationController) AdminEscalateHandler(ctx context.Context, input *dto.AdminEscalateInput) (*dto.AdminEscalateOutput, error) {
	adminID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	if strings.TrimSpace(input.Body.Reason) == "" {
		return nil, huma.Error400BadRequest("reason is required to escalate for manual review")
	}

	if err := c.service.AdminEscalateToManual(input.ID, adminID, input.Body.Reason); err != nil {
		return nil, mapAdminPipelineError(err)
	}

	return &dto.AdminEscalateOutput{
		Body: dto.VerificationMessageResponse{Message: "Verification escalated to manual review"},
	}, nil
}
