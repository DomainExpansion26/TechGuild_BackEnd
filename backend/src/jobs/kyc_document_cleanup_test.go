package jobs

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

// setupKYCDocCleanupTestDB gives each test its own in-memory SQLite database
// and points the global postgres.DB at it, matching the pattern used by the
// KYC escalator tests.
func setupKYCDocCleanupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get underlying sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(
		&models.User{},
		&models.VerificationRecord{},
		&models.VerificationDocument{},
	); err != nil {
		t.Fatalf("auto-migrate test schema: %v", err)
	}

	previous := postgres.DB
	postgres.DB = db
	t.Cleanup(func() { postgres.DB = previous })

	return db
}

// seedRecord inserts the user + verification record a document hangs off.
//
// verified_at and rejected_at are left NULL: callers that need a decision
// timestamp use seedDecidedRecord instead.
func seedRecord(t *testing.T, db *gorm.DB, status models.VerificationStatus) models.VerificationRecord {
	t.Helper()

	user := models.User{
		ID:            uuid.New(),
		FirstName:     "Doc",
		LastName:      "Cleanup",
		Email:         fmt.Sprintf("%s@example.test", uuid.NewString()),
		PasswordHash:  "x",
		Status:        models.StatusActive,
		EmailVerified: true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	record := models.VerificationRecord{
		ID:     uuid.New(),
		UserID: user.ID,
		Status: status,
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("create verification record: %v", err)
	}

	return record
}

// ptrTime returns a pointer to t, for the nullable decision columns.
func ptrTime(t time.Time) *time.Time { return &t }

// seedDecidedRecord inserts a terminal-state record whose decision timestamp
// (verified_at for approved, rejected_at for rejected) sits decidedAgo in the
// past.
//
// The DECISION date, not the upload date, anchors the retention clock, so this
// timestamp is what the sweep compares against the cutoff.
func seedDecidedRecord(t *testing.T, db *gorm.DB, status models.VerificationStatus, decidedAgo time.Duration) models.VerificationRecord {
	t.Helper()

	record := seedRecord(t, db, status)
	decidedAt := time.Now().Add(-decidedAgo)

	var column string
	switch status {
	case models.VerificationApproved:
		column = "verified_at"
	case models.VerificationRejected:
		column = "rejected_at"
	default:
		t.Fatalf("seedDecidedRecord: status %q has no decision timestamp", status)
	}

	if err := db.Model(&models.VerificationRecord{}).
		Where("id = ?", record.ID).
		Update(column, decidedAt).Error; err != nil {
		t.Fatalf("set %s: %v", column, err)
	}

	if column == "verified_at" {
		record.VerifiedAt = ptrTime(decidedAt)
	} else {
		record.RejectedAt = ptrTime(decidedAt)
	}

	return record
}

// seedDocument inserts a KYC document with a controlled created_at.
//
// Upload age is deliberately NOT the retention anchor any more: the decision
// date is. created_at is therefore set far in the past (100+ days) by the
// callers to prove that an ancient upload is still preserved when the decision
// is recent or absent.
//
// The URL/public_id are derived from the id the DB actually assigned, because
// VerificationDocument.BeforeCreate stamps its own UUID.
func seedDocument(t *testing.T, db *gorm.DB, recordID uuid.UUID, age time.Duration) models.VerificationDocument {
	t.Helper()

	doc := models.VerificationDocument{
		VerificationRecordID: recordID,
		DocumentType:         "selfie",
		ContentType:          "image/jpeg",
		CreatedAt:            time.Now().Add(-age),
	}
	if err := db.Create(&doc).Error; err != nil {
		t.Fatalf("create verification document: %v", err)
	}

	doc.FileURL = fmt.Sprintf("https://res.cloudinary.com/dkwnlcxbh/image/upload/v1234/verification_documents/%s.jpg", doc.ID)
	doc.PublicID = fmt.Sprintf("verification_documents/%s", doc.ID)
	if err := db.Model(&models.VerificationDocument{}).
		Where("id = ?", doc.ID).
		Updates(map[string]interface{}{
			"file_url":  doc.FileURL,
			"public_id": doc.PublicID,
		}).Error; err != nil {
		t.Fatalf("set document url/public_id: %v", err)
	}

	return doc
}

// reloadDoc reads a document back from the DB (the worker updates via a
// different *gorm.DB handle, so the in-memory struct is stale).
func reloadDoc(t *testing.T, db *gorm.DB, id uuid.UUID) models.VerificationDocument {
	t.Helper()

	var doc models.VerificationDocument
	if err := db.Where("id = ?", id).First(&doc).Error; err != nil {
		t.Fatalf("reload document %s: %v", id, err)
	}
	return doc
}

// stubCloudinaryDelete replaces the Cloudinary delete indirection for the
// duration of a test and records every call it receives.
func stubCloudinaryDelete(t *testing.T, err error) *[]string {
	t.Helper()

	var calls []string
	previous := cloudinaryDeleteFunc
	cloudinaryDeleteFunc = func(publicID string, resourceType string) error {
		calls = append(calls, publicID+"|"+resourceType)
		return err
	}
	t.Cleanup(func() { cloudinaryDeleteFunc = previous })

	return &calls
}

const testRetentionDays = 30

func TestKYCDocCleanup_DeletesOldApprovedDocs(t *testing.T) {
	db := setupKYCDocCleanupTestDB(t)
	calls := stubCloudinaryDelete(t, nil)

	// Approved 31 days ago; the upload itself is far older and irrelevant.
	record := seedDecidedRecord(t, db, models.VerificationApproved, 31*24*time.Hour)
	doc := seedDocument(t, db, record.ID, 100*24*time.Hour)

	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("runKYCDocumentCleanup: %v", err)
	}

	got := reloadDoc(t, db, doc.ID)
	if got.FileURL != "" {
		t.Errorf("expected file_url to be nulled, got %q", got.FileURL)
	}
	if got.PublicID != "" {
		t.Errorf("expected public_id to be nulled, got %q", got.PublicID)
	}
	if got.ContentType != "" {
		t.Errorf("expected content_type to be nulled, got %q", got.ContentType)
	}

	if len(*calls) != 1 {
		t.Fatalf("expected exactly 1 Cloudinary delete call, got %d (%v)", len(*calls), *calls)
	}
	wantPublicID := fmt.Sprintf("verification_documents/%s", doc.ID)
	if (*calls)[0] != wantPublicID+"|image" {
		t.Errorf("Cloudinary called with %q, want %q", (*calls)[0], wantPublicID+"|image")
	}
}

func TestKYCDocCleanup_PreservesPendingDocs(t *testing.T) {
	db := setupKYCDocCleanupTestDB(t)
	calls := stubCloudinaryDelete(t, nil)

	// No decision yet (verified_at/rejected_at both NULL): even an ancient
	// upload must never be swept.
	record := seedRecord(t, db, models.VerificationPending)
	doc := seedDocument(t, db, record.ID, 100*24*time.Hour)
	originalURL := doc.FileURL

	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("runKYCDocumentCleanup: %v", err)
	}

	got := reloadDoc(t, db, doc.ID)
	if got.FileURL != originalURL {
		t.Errorf("pending doc must not be touched: file_url=%q, want %q", got.FileURL, originalURL)
	}
	if len(*calls) != 0 {
		t.Errorf("pending doc must not reach Cloudinary, got calls %v", *calls)
	}
}

func TestKYCDocCleanup_PreservesUnderReviewDocs(t *testing.T) {
	db := setupKYCDocCleanupTestDB(t)
	calls := stubCloudinaryDelete(t, nil)

	// No decision yet: the admin still needs the image.
	record := seedRecord(t, db, models.VerificationReview)
	doc := seedDocument(t, db, record.ID, 100*24*time.Hour)
	originalURL := doc.FileURL

	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("runKYCDocumentCleanup: %v", err)
	}

	if got := reloadDoc(t, db, doc.ID); got.FileURL != originalURL {
		t.Errorf("under_review doc must not be touched: file_url=%q, want %q", got.FileURL, originalURL)
	}
	if len(*calls) != 0 {
		t.Errorf("under_review doc must not reach Cloudinary, got calls %v", *calls)
	}
}

func TestKYCDocCleanup_PreservesDraftDocs(t *testing.T) {
	db := setupKYCDocCleanupTestDB(t)
	calls := stubCloudinaryDelete(t, nil)

	// No decision yet: the applicant may still complete the upload.
	record := seedRecord(t, db, models.VerificationDraft)
	doc := seedDocument(t, db, record.ID, 100*24*time.Hour)
	originalURL := doc.FileURL

	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("runKYCDocumentCleanup: %v", err)
	}

	if got := reloadDoc(t, db, doc.ID); got.FileURL != originalURL {
		t.Errorf("draft doc must not be touched: file_url=%q, want %q", got.FileURL, originalURL)
	}
	if len(*calls) != 0 {
		t.Errorf("draft doc must not reach Cloudinary, got calls %v", *calls)
	}
}

func TestKYCDocCleanup_PreservesRecentApprovedDocs(t *testing.T) {
	db := setupKYCDocCleanupTestDB(t)
	calls := stubCloudinaryDelete(t, nil)

	// Decided 5 days ago but uploaded 100 days ago: the clock runs from the
	// decision, so the image survives for another 25 days.
	record := seedDecidedRecord(t, db, models.VerificationApproved, 5*24*time.Hour)
	doc := seedDocument(t, db, record.ID, 100*24*time.Hour)
	originalURL := doc.FileURL

	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("runKYCDocumentCleanup: %v", err)
	}

	if got := reloadDoc(t, db, doc.ID); got.FileURL != originalURL {
		t.Errorf("approved 5 days ago must survive: file_url=%q, want %q", got.FileURL, originalURL)
	}
	if len(*calls) != 0 {
		t.Errorf("recent doc must not reach Cloudinary, got calls %v", *calls)
	}
}

func TestKYCDocCleanup_DeletesOldRejectedDocs(t *testing.T) {
	db := setupKYCDocCleanupTestDB(t)
	calls := stubCloudinaryDelete(t, nil)

	// Rejected 31 days ago: the post-decision grace period has elapsed.
	record := seedDecidedRecord(t, db, models.VerificationRejected, 31*24*time.Hour)
	doc := seedDocument(t, db, record.ID, 100*24*time.Hour)

	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("runKYCDocumentCleanup: %v", err)
	}

	if got := reloadDoc(t, db, doc.ID); got.FileURL != "" {
		t.Errorf("expected rejected doc file_url to be nulled, got %q", got.FileURL)
	}
	if len(*calls) != 1 {
		t.Errorf("expected 1 Cloudinary delete call, got %d", len(*calls))
	}
}

// The retention threshold is strict on the DECISION date: a decision just
// inside the window survives, one just outside it is swept. (Deliberately not
// pinned to the exact millisecond of the cutoff, which would be flaky.)
//
// Both docs are uploaded 100 days ago, which proves upload age plays no part.
func TestKYCDocCleanup_RetentionThreshold(t *testing.T) {
	db := setupKYCDocCleanupTestDB(t)
	stubCloudinaryDelete(t, nil)

	// Each doc needs its own record, because the threshold now applies to the
	// record's decision timestamp rather than to the document.
	insideRecord := seedDecidedRecord(t, db, models.VerificationApproved, 29*24*time.Hour+23*time.Hour)
	outsideRecord := seedDecidedRecord(t, db, models.VerificationApproved, 30*24*time.Hour+time.Hour)

	justInside := seedDocument(t, db, insideRecord.ID, 100*24*time.Hour)
	justOutside := seedDocument(t, db, outsideRecord.ID, 100*24*time.Hour)

	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("runKYCDocumentCleanup: %v", err)
	}

	if got := reloadDoc(t, db, justInside.ID); got.FileURL == "" {
		t.Error("a decision inside the retention window must not be deleted")
	}
	if got := reloadDoc(t, db, justOutside.ID); got.FileURL != "" {
		t.Error("a decision outside the retention window must be deleted")
	}
}

// A retention of 0 (or negative) must disable the sweep entirely — the
// worst-case misconfiguration must never cause a mass delete.
func TestKYCDocCleanup_DisabledWhenRetentionZero(t *testing.T) {
	for _, retention := range []int{0, -1} {
		t.Run(fmt.Sprintf("retention=%d", retention), func(t *testing.T) {
			db := setupKYCDocCleanupTestDB(t)
			calls := stubCloudinaryDelete(t, nil)

			record := seedDecidedRecord(t, db, models.VerificationApproved, 365*24*time.Hour)
			doc := seedDocument(t, db, record.ID, 365*24*time.Hour)

			// StartKYCDocumentCleanup must return immediately without
			// blocking on a ticker, and without deleting anything.
			done := make(chan struct{})
			go func() {
				defer close(done)
				StartKYCDocumentCleanup(t.Context(), retention)
			}()

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("StartKYCDocumentCleanup did not return early when retention <= 0")
			}

			if got := reloadDoc(t, db, doc.ID); got.FileURL == "" {
				t.Error("disabled job must not null any file_url")
			}
			if len(*calls) != 0 {
				t.Errorf("disabled job must not call Cloudinary, got %v", *calls)
			}

			// runKYCDocumentCleanup must also refuse the same input.
			if err := runKYCDocumentCleanup(retention); err != nil {
				t.Fatalf("runKYCDocumentCleanup(%d): %v", retention, err)
			}
			if got := reloadDoc(t, db, doc.ID); got.FileURL == "" {
				t.Error("runKYCDocumentCleanup must refuse retention <= 0")
			}
		})
	}
}

// Privacy beats tidiness: if Cloudinary rejects the delete, the DB reference
// is still scrubbed, no panic escapes, and the sweep reports no error.
func TestKYCDocCleanup_ContinuesIfCloudinaryFails(t *testing.T) {
	db := setupKYCDocCleanupTestDB(t)
	calls := stubCloudinaryDelete(t, errors.New("cloudinary unavailable"))

	oldRecord := seedDecidedRecord(t, db, models.VerificationApproved, 31*24*time.Hour)
	oldDoc := seedDocument(t, db, oldRecord.ID, 100*24*time.Hour)

	pendingRecord := seedRecord(t, db, models.VerificationPending)
	pendingDoc := seedDocument(t, db, pendingRecord.ID, 100*24*time.Hour)

	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("runKYCDocumentCleanup: %v", err)
	}

	if got := reloadDoc(t, db, oldDoc.ID); got.FileURL != "" {
		t.Errorf("DB URL must be nulled even when Cloudinary fails, got %q", got.FileURL)
	}
	if got := reloadDoc(t, db, pendingDoc.ID); got.FileURL == "" {
		t.Error("a Cloudinary failure must not widen the sweep to pending records")
	}
	if len(*calls) != 1 {
		t.Errorf("expected 1 Cloudinary attempt, got %d", len(*calls))
	}
}

// 150 matching docs must all be processed: the batch bound is 100, so this
// crosses at least two iterations.
func TestKYCDocCleanup_BatchProcessing(t *testing.T) {
	db := setupKYCDocCleanupTestDB(t)
	calls := stubCloudinaryDelete(t, nil)

	record := seedDecidedRecord(t, db, models.VerificationApproved, 31*24*time.Hour)

	const total = 150
	docs := make([]models.VerificationDocument, 0, total)
	for i := 0; i < total; i++ {
		docs = append(docs, seedDocument(t, db, record.ID, 100*24*time.Hour))
	}

	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("runKYCDocumentCleanup: %v", err)
	}

	if len(*calls) != total {
		t.Errorf("expected %d Cloudinary calls, got %d", total, len(*calls))
	}

	var remaining int64
	if err := db.Model(&models.VerificationDocument{}).
		Where("file_url <> ''").
		Count(&remaining).Error; err != nil {
		t.Fatalf("count remaining docs: %v", err)
	}
	if remaining != 0 {
		t.Errorf("expected all %d docs scrubbed, %d still have a file_url", total, remaining)
	}
}

// A doc whose public_id was never persisted must still be deleted from
// Cloudinary by parsing it out of the URL.
func TestKYCDocCleanup_FallsBackToURLForPublicID(t *testing.T) {
	db := setupKYCDocCleanupTestDB(t)
	calls := stubCloudinaryDelete(t, nil)

	record := seedDecidedRecord(t, db, models.VerificationApproved, 31*24*time.Hour)
	doc := seedDocument(t, db, record.ID, 100*24*time.Hour)
	// Simulate the known gap: public_id column empty, URL intact.
	if err := db.Model(&models.VerificationDocument{}).
		Where("id = ?", doc.ID).
		Update("public_id", "").Error; err != nil {
		t.Fatalf("clear public_id: %v", err)
	}

	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("runKYCDocumentCleanup: %v", err)
	}

	wantPublicID := fmt.Sprintf("verification_documents/%s", doc.ID)
	if len(*calls) != 1 || (*calls)[0] != wantPublicID+"|image" {
		t.Errorf("expected URL-derived delete %q, got %v", wantPublicID, *calls)
	}
	if got := reloadDoc(t, db, doc.ID); got.FileURL != "" {
		t.Errorf("expected file_url to be nulled, got %q", got.FileURL)
	}
}

// Idempotency: an already-scrubbed row (file_url == "") is excluded from the
// sweep, so a second run neither re-calls Cloudinary nor loops forever.
func TestKYCDocCleanup_IdempotentOnSecondRun(t *testing.T) {
	db := setupKYCDocCleanupTestDB(t)
	calls := stubCloudinaryDelete(t, nil)

	record := seedDecidedRecord(t, db, models.VerificationApproved, 31*24*time.Hour)
	doc := seedDocument(t, db, record.ID, 100*24*time.Hour)

	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("expected 1 Cloudinary call after first run, got %d", len(*calls))
	}

	// Second run must be a no-op and must terminate.
	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("second run: %v", err)
	}

	if len(*calls) != 1 {
		t.Errorf("second run must not re-call Cloudinary, got %d calls total", len(*calls))
	}
	if got := reloadDoc(t, db, doc.ID); got.FileURL != "" {
		t.Errorf("expected file_url to stay nulled, got %q", got.FileURL)
	}
}

// ---------------------------------------------------------------------------
// Phase D.1 — RBI retention clock: the DECISION, not the upload, is the anchor.
// ---------------------------------------------------------------------------

// A record that sat in the review queue for 60 days still gets its full 30 days
// of post-decision retention: the clock starts when the decision is recorded.
func TestKYCDocCleanup_DelayedDecision_GetsFull30Days(t *testing.T) {
	db := setupKYCDocCleanupTestDB(t)
	calls := stubCloudinaryDelete(t, nil)

	record := seedDecidedRecord(t, db, models.VerificationApproved, 0) // decided today
	doc := seedDocument(t, db, record.ID, 60*24*time.Hour)             // uploaded 60 days ago
	originalURL := doc.FileURL

	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("runKYCDocumentCleanup: %v", err)
	}

	if got := reloadDoc(t, db, doc.ID); got.FileURL != originalURL {
		t.Errorf("a 60-day-old upload decided today must survive: file_url=%q, want %q",
			got.FileURL, originalURL)
	}
	if len(*calls) != 0 {
		t.Errorf("a doc decided today must not reach Cloudinary, got calls %v", *calls)
	}
}

// 31 days after approval the 30-day grace period has elapsed, so the image is
// swept even though the record was uploaded 90 days ago.
func TestKYCDocCleanup_ApprovalGracePeriodExpires(t *testing.T) {
	db := setupKYCDocCleanupTestDB(t)
	calls := stubCloudinaryDelete(t, nil)

	record := seedDecidedRecord(t, db, models.VerificationApproved, 31*24*time.Hour)
	doc := seedDocument(t, db, record.ID, 90*24*time.Hour)

	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("runKYCDocumentCleanup: %v", err)
	}

	if got := reloadDoc(t, db, doc.ID); got.FileURL != "" {
		t.Errorf("a doc approved 31 days ago must be swept, file_url=%q", got.FileURL)
	}
	if len(*calls) != 1 {
		t.Errorf("expected exactly 1 Cloudinary delete call, got %d (%v)", len(*calls), *calls)
	}
}

// Malformed-data guard: an approved record with a NULL verified_at has no
// retention anchor, so it is skipped rather than silently swept.
func TestKYCDocCleanup_NoVerifiedAtNotDeleted(t *testing.T) {
	db := setupKYCDocCleanupTestDB(t)
	calls := stubCloudinaryDelete(t, nil)

	// seedRecord leaves verified_at and rejected_at NULL.
	record := seedRecord(t, db, models.VerificationApproved)
	doc := seedDocument(t, db, record.ID, 100*24*time.Hour)
	originalURL := doc.FileURL

	if err := runKYCDocumentCleanup(testRetentionDays); err != nil {
		t.Fatalf("runKYCDocumentCleanup: %v", err)
	}

	if got := reloadDoc(t, db, doc.ID); got.FileURL != originalURL {
		t.Errorf("approved record with NULL verified_at must not be swept: file_url=%q, want %q",
			got.FileURL, originalURL)
	}
	if len(*calls) != 0 {
		t.Errorf("a doc with no decision anchor must not reach Cloudinary, got calls %v", *calls)
	}
}
