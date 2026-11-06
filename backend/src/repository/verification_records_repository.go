package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

type VerificationRecordsRepository interface {
	// Verification Records (User Flow)
	CreateVerification(record *models.VerificationRecord) error
	GetActiveVerificationByUser(userID string) (*models.VerificationRecord, error) // NAYA: Drafts ke liye
	GetVerificationByID(id string) (*models.VerificationRecord, error)
	UpdateVerificationRecord(record *models.VerificationRecord) error // NAYA: Step-by-step save karne ke liye
	UpdateStatus(id string, status models.VerificationStatus, reason string) error

	// Deduplication
	CreateGovtHash(hash *models.GovtIDDedup) error
	GovtHashExists(hash string) (bool, error)
	CreateBusinessHash(hash *models.BusinessPANDedup) error
	BusinessHashExists(hash string) (bool, error)

	// Documents (User Flow)
	CreateDocument(document *models.VerificationDocument) error
	GetDocuments(recordID string) ([]models.VerificationDocument, error)
	DeleteDocumentByType(recordID string, docType string) error // NAYA: Re-upload ke time purana delete karne ke liye

	// Admin Flow
	GetPendingVerifications() ([]models.VerificationRecord, error)
	GetVerificationQueue() ([]models.VerificationRecord, error)
	ApproveVerification(id string) error
	RejectVerification(id string, reason string) error

	// KYC AI Pipeline (Phase B: Auto/Manual Review Queues)
	GetAutoEligibleQueue(limit, offset int, minAgeHours float64) ([]models.VerificationRecord, int64, error)
	GetManualReviewQueue(sortBy, sortDir string, limit, offset int) ([]models.VerificationRecord, int64, error)
	GetVerificationDetail(id string) (*models.VerificationRecord, error)
	BulkApprove(verificationIDs []string, adminID uuid.UUID) error
	MarkManualAction(verificationID string, action string, adminID uuid.UUID, reason string) error
	RequestManualReview(verificationID string, adminID uuid.UUID, reason string) error

	// Stale-Pending Escalation (Phase E: Cron)
	FlagOverdueAutoEligible(id uuid.UUID, now time.Time) error
	CreateVerificationAuditLog(entry *models.VerificationAuditLog) error
}

type verificationRecordsRepository struct{}

func NewVerificationRecordsRepository() VerificationRecordsRepository {
	return &verificationRecordsRepository{}
}

// ---------------- Verification Records ----------------

func (r *verificationRecordsRepository) CreateVerification(record *models.VerificationRecord) error {
	return postgres.DB.Create(record).Error
}

// GetActiveVerificationByUser fetches the latest record that is NOT approved
func (r *verificationRecordsRepository) GetActiveVerificationByUser(userID string) (*models.VerificationRecord, error) {
	var record models.VerificationRecord
	// Order by created_at DESC to get the latest draft/pending/rejected
	err := postgres.DB.
		Where("user_id = ? AND status IN ?", userID, []string{"draft", "pending", "rejected"}).
		Order("created_at DESC").
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *verificationRecordsRepository) GetVerificationByID(id string) (*models.VerificationRecord, error) {
	var record models.VerificationRecord
	err := postgres.DB.First(&record, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateVerificationRecord saves the entire struct (useful for Wizard steps)
func (r *verificationRecordsRepository) UpdateVerificationRecord(record *models.VerificationRecord) error {
	return postgres.DB.Save(record).Error
}

func (r *verificationRecordsRepository) UpdateStatus(id string, status models.VerificationStatus, reason string) error {
	update := map[string]interface{}{
		"status": status,
	}
	if reason != "" {
		update["rejection_reason"] = reason
	}
	if status == models.VerificationApproved {
		now := time.Now()
		update["verified_at"] = &now
	}
	return postgres.DB.Model(&models.VerificationRecord{}).Where("id = ?", id).Updates(update).Error
}

// ---------------- Deduplication ----------------

func (r *verificationRecordsRepository) CreateGovtHash(hash *models.GovtIDDedup) error {
	return postgres.DB.Create(hash).Error
}

func (r *verificationRecordsRepository) GovtHashExists(hash string) (bool, error) {
	var count int64
	err := postgres.DB.Model(&models.GovtIDDedup{}).Where("govt_id_hash = ?", hash).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *verificationRecordsRepository) CreateBusinessHash(hash *models.BusinessPANDedup) error {
	return postgres.DB.Create(hash).Error
}

func (r *verificationRecordsRepository) BusinessHashExists(hash string) (bool, error) {
	var count int64
	err := postgres.DB.Model(&models.BusinessPANDedup{}).Where("pan_hash = ?", hash).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ---------------- Verification Documents ----------------

func (r *verificationRecordsRepository) CreateDocument(document *models.VerificationDocument) error {
	return postgres.DB.Create(document).Error
}

func (r *verificationRecordsRepository) GetDocuments(recordID string) ([]models.VerificationDocument, error) {
	var documents []models.VerificationDocument
	err := postgres.DB.Where("verification_record_id = ?", recordID).Find(&documents).Error
	return documents, err
}

// DeleteDocumentByType (Useful when user re-uploads a file)
func (r *verificationRecordsRepository) DeleteDocumentByType(recordID string, docType string) error {
	return postgres.DB.
		Where("verification_record_id = ? AND document_type = ?", recordID, docType).
		Delete(&models.VerificationDocument{}).Error
}

// ---------------- Admin Flow ----------------

func (r *verificationRecordsRepository) GetPendingVerifications() ([]models.VerificationRecord, error) {
	var records []models.VerificationRecord
	err := postgres.DB.Where("status = ?", models.VerificationPending).Order("created_at ASC").Find(&records).Error
	return records, err
}

func (r *verificationRecordsRepository) GetVerificationQueue() ([]models.VerificationRecord, error) {
	var records []models.VerificationRecord
	err := postgres.DB.
		Where("status = ?", models.VerificationPending).
		Preload("User").
		Order("submitted_at ASC"). // NAYA: Sort by submission time
		Find(&records).Error
	return records, err
}

func (r *verificationRecordsRepository) ApproveVerification(id string) error {
	return postgres.DB.Model(&models.VerificationRecord{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      models.VerificationApproved,
			"verified_at": time.Now(),
		}).Error
}

func (r *verificationRecordsRepository) RejectVerification(id string, reason string) error {
	return postgres.DB.Model(&models.VerificationRecord{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":           models.VerificationRejected,
			"rejection_reason": reason,
		}).Error
}

// ---------------- KYC AI Pipeline (Auto/Manual Review Queues) ----------------

// GetAutoEligibleQueue returns AI high-confidence records whose ML step has
// completed and whose hold window (minAgeHours after ml_completed_at) has
// elapsed, i.e. candidates for batch (24-48h window) approval. Records pulled
// out by an admin (bulk_review_requested_at set) or already actioned are
// excluded.
func (r *verificationRecordsRepository) GetAutoEligibleQueue(limit, offset int, minAgeHours float64) ([]models.VerificationRecord, int64, error) {
	// Compute the cutoff in Go instead of using Postgres' NOW()/interval syntax.
	// This keeps the query portable (SQLite in tests) and lets the database use
	// the ml_completed_at index instead of evaluating an expression per row.
	cutoff := time.Now().Add(-time.Duration(minAgeHours * float64(time.Hour)))

	// Fresh base per query so Count() doesn't pollute the later Find().
	base := func() *gorm.DB {
		return postgres.DB.Model(&models.VerificationRecord{}).
			Where(
				`review_priority = ? AND status = ? AND ml_status = ?
				 AND bulk_review_requested_at IS NULL
				 AND auto_approved_at IS NULL
				 AND bulk_approved_at IS NULL
				 AND rejected_at IS NULL
				 AND ml_completed_at IS NOT NULL
				 AND ml_completed_at <= ?`,
				models.ReviewPriorityAutoEligible,
				models.VerificationPending,
				models.MLStatusCompleted,
				cutoff,
			)
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []models.VerificationRecord
	err := base().
		Preload("User").
		Order("ml_completed_at ASC"). // oldest completion first = closest to SLA
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	return records, total, err
}

// manualReviewSortColumns maps API sort keys to JSONB score extraction, with a
// fixed allowlist so sorting can't be injected via raw SQL fragments.
var manualReviewSortColumns = map[string]string{
	"liveness_score":  "(liveness_result->>'liveness_score')",
	"face_similarity": "(face_match_result->>'face_similarity')",
}

// buildManualReviewOrder builds a sanitized ORDER BY expression. ASC is the
// weakest-first priority ordering; scores missing from JSONB always sort last
// so fully-scored "weakest" records drive the queue.
func buildManualReviewOrder(sortBy, sortDir string) string {
	dir := "ASC"
	if strings.EqualFold(strings.TrimSpace(sortDir), "desc") {
		dir = "DESC"
	}

	expr, ok := manualReviewSortColumns[strings.ToLower(strings.TrimSpace(sortBy))]
	if !ok {
		return "submitted_at " + dir
	}
	return fmt.Sprintf("%s::numeric %s NULLS LAST", expr, dir)
}

// GetManualReviewQueue returns borderline-confidence records that need a human
// decision. Supports weakest-first sorting by liveness_score or face_similarity.
func (r *verificationRecordsRepository) GetManualReviewQueue(sortBy, sortDir string, limit, offset int) ([]models.VerificationRecord, int64, error) {
	base := func() *gorm.DB {
		return postgres.DB.Model(&models.VerificationRecord{}).
			Where("review_priority = ? AND status = ?", models.ReviewPriorityManual, models.VerificationPending)
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []models.VerificationRecord
	err := base().
		Preload("User").
		Order(buildManualReviewOrder(sortBy, sortDir)).
		Limit(limit).
		Offset(offset).
		Find(&records).Error
	return records, total, err
}

// GetVerificationDetail returns a single record with the User and its uploaded
// documents preloaded for the admin review screen.
func (r *verificationRecordsRepository) GetVerificationDetail(id string) (*models.VerificationRecord, error) {
	var record models.VerificationRecord
	err := postgres.DB.
		Preload("User").
		Preload("Documents").
		First(&record, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// BulkApprove transactionally approves multiple auto-eligible records at once:
// it flips their status/plane, records BulkReviewItem rows (human decision
// trail for the 24-48h window) and per-record audit log entries. All-or-nothing.
func (r *verificationRecordsRepository) BulkApprove(verificationIDs []string, adminID uuid.UUID) error {
	if len(verificationIDs) == 0 {
		return errors.New("no verification IDs provided for bulk approve")
	}

	ids := make([]uuid.UUID, 0, len(verificationIDs))
	for _, raw := range verificationIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			return fmt.Errorf("invalid verification ID %q: %w", raw, err)
		}
		ids = append(ids, id)
	}

	now := time.Now()
	return postgres.DB.Transaction(func(tx *gorm.DB) error {
		var records []models.VerificationRecord
		if err := tx.Preload("User").Where("id IN ?", ids).Find(&records).Error; err != nil {
			return err
		}
		if len(records) == 0 {
			return errors.New("no matching verification records for bulk approve")
		}
		if len(records) != len(ids) {
			return fmt.Errorf("some verification records not found: requested %d, found %d", len(ids), len(records))
		}

		foundIDs := make([]uuid.UUID, 0, len(records))
		bulkItems := make([]models.BulkReviewItem, 0, len(records))
		for _, rec := range records {
			foundIDs = append(foundIDs, rec.ID)
			bulkItems = append(bulkItems, models.BulkReviewItem{
				VerificationID: rec.ID,
				UserEmail:      rec.User.Email,
				Type:           string(rec.Type),
				ReviewedAt:     &now,
				ReviewedBy:     &adminID,
				ReviewedAction: "approve",
			})
		}

		// State update: approved + marked as coming from the bulk flow.
		if err := tx.Model(&models.VerificationRecord{}).
			Where("id IN ?", foundIDs).
			Updates(map[string]interface{}{
				"status":           models.VerificationApproved,
				"verified_at":      now,
				"bulk_approved_at": now,
				"review_priority":  models.ReviewPriorityAutoApproved,
			}).Error; err != nil {
			return err
		}

		// Upsert: rows may already exist from an earlier ADMIN_REVIEW_REQUESTED
		// (verification_id is uniquely indexed).
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "verification_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"reviewed_at", "reviewed_by", "reviewed_action"}),
		}).Create(&bulkItems).Error; err != nil {
			return err
		}

		// Per-record accountability trail.
		for _, rec := range records {
			details, err := json.Marshal(map[string]interface{}{
				"verification_id": rec.ID,
				"admin_id":        adminID,
				"action":          "bulk_approve",
			})
			if err != nil {
				return err
			}
			if err := tx.Create(&models.VerificationAuditLog{
				VerificationID: rec.ID,
				UserID:         rec.UserID,
				ActionType:     "BULK_APPROVE",
				AdminID:        &adminID,
				Details:        datatypes.JSON(details),
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// MarkManualAction records a single approve/reject decision taken by an admin
// from the manual review queue and logs it to the audit trail.
func (r *verificationRecordsRepository) MarkManualAction(verificationID string, action string, adminID uuid.UUID, reason string) error {
	vid, err := uuid.Parse(verificationID)
	if err != nil {
		return fmt.Errorf("invalid verification ID %q: %w", verificationID, err)
	}

	var status models.VerificationStatus
	var auditAction string
	updates := map[string]interface{}{}
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "approve":
		status = models.VerificationApproved
		auditAction = "MANUAL_APPROVE"
		updates["verified_at"] = time.Now()
	case "reject":
		status = models.VerificationRejected
		auditAction = "MANUAL_REJECT"
		now := time.Now()
		updates["rejected_at"] = now
		if reason == "" {
			return errors.New("reason is required for manual reject")
		}
		updates["rejection_reason"] = reason
	default:
		return fmt.Errorf("unsupported action %q (must be approve or reject)", action)
	}
	updates["status"] = status

	auditDetailsBytes, err := json.Marshal(map[string]interface{}{
		"action":   auditAction,
		"admin_id": adminID,
		"reason":   reason,
	})
	if err != nil {
		return err
	}

	return postgres.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.VerificationRecord{}).
			Where("id = ?", vid).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("verification record %s not found", vid)
		}

		userID, err := fetchRecordUserID(tx, vid)
		if err != nil {
			return err
		}

		return tx.Create(&models.VerificationAuditLog{
			VerificationID: vid,
			UserID:         userID,
			ActionType:     auditAction,
			AdminID:        &adminID,
			Details:        datatypes.JSON(auditDetailsBytes),
		}).Error
	})
}

// RequestManualReview pulls an auto-eligible record back to the manual review
// queue, clearing its bulk-candidate flag (so the auto-eligible query no longer
// picks it) and recording WHO asked for the extra scrutiny. A BulkReviewItem is
// created the first time so the 24-48h window keeps a human-decision record.
func (r *verificationRecordsRepository) RequestManualReview(verificationID string, adminID uuid.UUID, reason string) error {
	vid, err := uuid.Parse(verificationID)
	if err != nil {
		return fmt.Errorf("invalid verification ID %q: %w", verificationID, err)
	}

	return postgres.DB.Transaction(func(tx *gorm.DB) error {
		var rec models.VerificationRecord
		if err := tx.Preload("User").First(&rec, "id = ?", vid).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("verification record %s not found", vid)
			}
			return err
		}

		now := time.Now()
		if err := tx.Model(&rec).Updates(map[string]interface{}{
			"review_priority":           models.ReviewPriorityManual,
			"bulk_review_requested_at":  nil,
			"admin_review_requested_by": adminID,
			"admin_review_requested_at": now,
		}).Error; err != nil {
			return err
		}

		// Insert only if not present — verification_id is uniquely indexed and
		// an existing row's reviewed_* state must not be silently overwritten.
		var existingCount int64
		if err := tx.Model(&models.BulkReviewItem{}).
			Where("verification_id = ?", vid).
			Count(&existingCount).Error; err != nil {
			return err
		}
		if existingCount == 0 {
			if err := tx.Create(&models.BulkReviewItem{
				VerificationID: rec.ID,
				UserEmail:      rec.User.Email,
				Type:           string(rec.Type),
			}).Error; err != nil {
				return err
			}
		}

		details, err := json.Marshal(map[string]interface{}{
			"reason":       reason,
			"requested_by": adminID,
		})
		if err != nil {
			return err
		}
		return tx.Create(&models.VerificationAuditLog{
			VerificationID: rec.ID,
			UserID:         rec.UserID,
			ActionType:     "ADMIN_REVIEW_REQUESTED",
			AdminID:        &adminID,
			Details:        datatypes.JSON(details),
		}).Error
	})
}

// fetchRecordUserID resolves the record owner for audit log entries.
func fetchRecordUserID(tx *gorm.DB, id uuid.UUID) (uuid.UUID, error) {
	var rec models.VerificationRecord
	if err := tx.First(&rec, "id = ?", id).Error; err != nil {
		return uuid.Nil, err
	}
	return rec.UserID, nil
}

// ---------------- Stale-Pending Escalation (Cron) ----------------

// FlagOverdueAutoEligible escalates a stale auto-eligible record to the manual
// review queue without changing its pending status.
func (r *verificationRecordsRepository) FlagOverdueAutoEligible(id uuid.UUID, now time.Time) error {
	return postgres.DB.Model(&models.VerificationRecord{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"review_priority":           models.ReviewPriorityManual,
			"admin_review_requested_at": now,
		}).Error
}

// CreateVerificationAuditLog appends an audit-trail entry (used by the cron
// job and the ML worker, which have no admin to attribute actions to).
func (r *verificationRecordsRepository) CreateVerificationAuditLog(entry *models.VerificationAuditLog) error {
	return postgres.DB.Create(entry).Error
}
