package jobs

import (
	"context"
	"log"
	"time"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
	"techguild-backend/src/utils"
)

const (
	// kycDocCleanupInterval is how often the retention sweep runs.
	kycDocCleanupInterval = 24 * time.Hour

	// kycDocCleanupBatch bounds how many documents are fetched per
	// iteration so a large sweep never holds one long transaction.
	kycDocCleanupBatch = 100
)

// cloudinaryDeleteFunc is an indirection over utils.DeleteFromCloudinary so
// tests can assert on (or stub out) the external call. Production always uses
// the real implementation.
var cloudinaryDeleteFunc = utils.DeleteFromCloudinary

// StartKYCDocumentCleanup runs one sweep at boot and then every
// kycDocCleanupInterval until ctx is cancelled.
//
// It deletes KYC document images (Aadhaar, PAN, selfie, ...) belonging to
// records in a terminal state (approved/rejected) that are older than
// retentionDays. This is a DPDP Act obligation: government IDs must not be
// retained indefinitely once a decision has been made.
//
// Records still awaiting a decision (draft/pending/under_review) are never
// touched — admins need those documents for review.
//
// A non-positive retentionDays disables the job entirely, so a
// misconfiguration can never trigger a mass delete.
func StartKYCDocumentCleanup(ctx context.Context, retentionDays int) {
	if retentionDays <= 0 {
		log.Printf("[KYC Doc Cleanup] disabled (retentionDays=%d)", retentionDays)
		return
	}

	log.Printf("[KYC Doc Cleanup] started (retention=%d days, interval=%s)",
		retentionDays, kycDocCleanupInterval)

	// Run once at boot so a long-idle deployment catches up immediately.
	if err := runKYCDocumentCleanup(retentionDays); err != nil {
		log.Printf("[KYC Doc Cleanup] initial run error: %v", err)
	}

	ticker := time.NewTicker(kycDocCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[KYC Doc Cleanup] stopping (context cancelled)")
			return
		case <-ticker.C:
			if err := runKYCDocumentCleanup(retentionDays); err != nil {
				log.Printf("[KYC Doc Cleanup] tick error: %v", err)
			}
		}
	}
}

// runKYCDocumentCleanup performs one full retention sweep, processing the
// matching documents in batches of kycDocCleanupBatch until none remain.
//
// A single document can never abort the sweep: a Cloudinary failure is logged
// and the DB row is still scrubbed (privacy beats tidiness), and a DB failure
// is logged and the loop moves on to the next document.
func runKYCDocumentCleanup(retentionDays int) error {
	if retentionDays <= 0 {
		log.Printf("[KYC Doc Cleanup] refusing to run (retentionDays=%d)", retentionDays)
		return nil
	}

	// Panic recovery, matching the existing cleanup job's defensiveness: a
	// background goroutine must never take the process down.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[KYC Doc Cleanup] PANIC: %v", r)
		}
	}()

	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	totalDeleted := 0
	totalCleaned := 0

	for {
		var docs []models.VerificationDocument

		err := postgres.DB.
			Joins("JOIN verification_records ON verification_records.id = verification_documents.verification_record_id").
			Where(`
				(verification_records.status = ?
				 AND verification_records.verified_at IS NOT NULL
				 AND verification_records.verified_at < ?)
				OR
				(verification_records.status = ?
				 AND verification_records.rejected_at IS NOT NULL
				 AND verification_records.rejected_at < ?)
			`,
				models.VerificationApproved, cutoff,
				models.VerificationRejected, cutoff,
			).
			Where("verification_documents.file_url <> ''").
			Limit(kycDocCleanupBatch).
			Find(&docs).Error
		if err != nil {
			return err
		}

		if len(docs) == 0 {
			break
		}

		for _, doc := range docs {
			// Prefer the stored public_id; fall back to parsing the URL,
			// because older rows frequently have an empty public_id.
			publicID := doc.PublicID
			if publicID == "" {
				publicID = extractCloudinaryPublicID(doc.FileURL)
			}

			var cloudErr error
			if publicID != "" {
				cloudErr = cloudinaryDeleteFunc(publicID, "image")
				if cloudErr != nil {
					log.Printf("[KYC Doc Cleanup] cloud delete failed doc=%s publicID=%s err=%v",
						doc.ID, publicID, cloudErr)
					// Fall through: the DB row is still scrubbed. The image
					// is orphaned rather than retried forever; ops can sweep
					// the Cloudinary account manually.
				}
			}

			// Scrub regardless of Cloudinary's outcome: the privacy
			// obligation is on the stored reference, not the CDN object.
			updateErr := postgres.DB.Model(&models.VerificationDocument{}).
				Where("id = ?", doc.ID).
				Updates(map[string]interface{}{
					"file_url":     "",
					"public_id":    "",
					"content_type": "",
				}).Error
			if updateErr != nil {
				log.Printf("[KYC Doc Cleanup] DB update failed doc=%s err=%v",
					doc.ID, updateErr)
				continue
			}

			if cloudErr == nil {
				totalDeleted++
			}
			totalCleaned++
		}

		log.Printf("[KYC Doc Cleanup] batch: cloud_deleted=%d db_cleaned=%d (continuing)",
			totalDeleted, totalCleaned)

		// Defensive: a partial batch means we have drained the result set.
		if len(docs) < kycDocCleanupBatch {
			break
		}
	}

	if totalCleaned > 0 {
		log.Printf("[KYC Doc Cleanup] complete: cloud_deleted=%d db_cleaned=%d cutoff=%s",
			totalDeleted, totalCleaned, cutoff.Format("2006-01-02"))
	}

	return nil
}
