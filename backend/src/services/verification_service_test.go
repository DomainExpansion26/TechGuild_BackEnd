package services

import (
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm/clause"

	"techguild-backend/src/models"
	"techguild-backend/src/repository"
)

// ---------------------------------------------------------------------------
// Test doubles
//
// Both mocks embed the production interface. Embedding lets us implement only
// the methods a given test exercises; any unexpected call panics loudly instead
// of silently returning a zero value.
// ---------------------------------------------------------------------------

type mockVerificationRecordsRepo struct {
	repository.VerificationRecordsRepository

	getActiveVerificationByUserFn func(userID string) (*models.VerificationRecord, error)
	getDocumentsFn                func(recordID string) ([]models.VerificationDocument, error)
	updateVerificationRecordFn    func(record *models.VerificationRecord) error
	getVerificationByIDFn         func(id string) (*models.VerificationRecord, error)
	bulkApproveFn                 func(ids []string, adminID uuid.UUID) error
	markManualActionFn            func(verificationID, action string, adminID uuid.UUID, reason string) error
	createGovtHashFn              func(hash *models.GovtIDDedup) error
	createBusinessHashFn          func(hash *models.BusinessPANDedup) error
	getAutoEligibleQueueFn        func(limit, offset int, minAgeHours float64) ([]models.VerificationRecord, int64, error)
	flagOverdueAutoEligibleFn     func(id uuid.UUID, now time.Time) error
	createVerificationAuditLogFn  func(entry *models.VerificationAuditLog) error
}

func (m *mockVerificationRecordsRepo) GetActiveVerificationByUser(userID string) (*models.VerificationRecord, error) {
	if m.getActiveVerificationByUserFn == nil {
		panic("unexpected call: GetActiveVerificationByUser")
	}
	return m.getActiveVerificationByUserFn(userID)
}

func (m *mockVerificationRecordsRepo) GetDocuments(recordID string) ([]models.VerificationDocument, error) {
	if m.getDocumentsFn == nil {
		panic("unexpected call: GetDocuments")
	}
	return m.getDocumentsFn(recordID)
}

func (m *mockVerificationRecordsRepo) UpdateVerificationRecord(record *models.VerificationRecord) error {
	if m.updateVerificationRecordFn == nil {
		panic("unexpected call: UpdateVerificationRecord")
	}
	return m.updateVerificationRecordFn(record)
}

func (m *mockVerificationRecordsRepo) GetVerificationByID(id string) (*models.VerificationRecord, error) {
	if m.getVerificationByIDFn == nil {
		panic("unexpected call: GetVerificationByID")
	}
	return m.getVerificationByIDFn(id)
}

func (m *mockVerificationRecordsRepo) BulkApprove(ids []string, adminID uuid.UUID) error {
	if m.bulkApproveFn == nil {
		panic("unexpected call: BulkApprove")
	}
	return m.bulkApproveFn(ids, adminID)
}

func (m *mockVerificationRecordsRepo) MarkManualAction(verificationID, action string, adminID uuid.UUID, reason string) error {
	if m.markManualActionFn == nil {
		panic("unexpected call: MarkManualAction")
	}
	return m.markManualActionFn(verificationID, action, adminID, reason)
}

func (m *mockVerificationRecordsRepo) CreateGovtHash(hash *models.GovtIDDedup) error {
	if m.createGovtHashFn == nil {
		panic("unexpected call: CreateGovtHash")
	}
	return m.createGovtHashFn(hash)
}

func (m *mockVerificationRecordsRepo) CreateBusinessHash(hash *models.BusinessPANDedup) error {
	if m.createBusinessHashFn == nil {
		panic("unexpected call: CreateBusinessHash")
	}
	return m.createBusinessHashFn(hash)
}

func (m *mockVerificationRecordsRepo) GetAutoEligibleQueue(limit, offset int, minAgeHours float64) ([]models.VerificationRecord, int64, error) {
	if m.getAutoEligibleQueueFn == nil {
		panic("unexpected call: GetAutoEligibleQueue")
	}
	return m.getAutoEligibleQueueFn(limit, offset, minAgeHours)
}

func (m *mockVerificationRecordsRepo) FlagOverdueAutoEligible(id uuid.UUID, now time.Time) error {
	if m.flagOverdueAutoEligibleFn == nil {
		panic("unexpected call: FlagOverdueAutoEligible")
	}
	return m.flagOverdueAutoEligibleFn(id, now)
}

func (m *mockVerificationRecordsRepo) CreateVerificationAuditLog(entry *models.VerificationAuditLog) error {
	if m.createVerificationAuditLogFn == nil {
		panic("unexpected call: CreateVerificationAuditLog")
	}
	return m.createVerificationAuditLogFn(entry)
}

type mockUserRepo struct {
	repository.UserRepository

	getUserByIDFn      func(userID string) (*models.User, error)
	addUserPointsFn    func(userID string, points int) error
	updateUserStatusFn func(userID string, status string) error
}

func (m *mockUserRepo) GetUserByID(userID string) (*models.User, error) {
	if m.getUserByIDFn == nil {
		panic("unexpected call: GetUserByID")
	}
	return m.getUserByIDFn(userID)
}

func (m *mockUserRepo) AddUserPoints(userID string, points int) error {
	if m.addUserPointsFn == nil {
		panic("unexpected call: AddUserPoints")
	}
	return m.addUserPointsFn(userID, points)
}

func (m *mockUserRepo) UpdateUserStatus(userID string, status string) error {
	if m.updateUserStatusFn == nil {
		panic("unexpected call: UpdateUserStatus")
	}
	return m.updateUserStatusFn(userID, status)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestVerificationService(
	vr repository.VerificationRecordsRepository,
	ur repository.UserRepository,
	rdb *redis.Client,
) *VerificationService {
	return &VerificationService{
		userRepo:                ur,
		verificationRecordsRepo: vr,
		redisClient:             rdb,
	}
}

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func ptrTime(t time.Time) *time.Time { return &t }

// ---------------------------------------------------------------------------
// UploadDocument
// ---------------------------------------------------------------------------

func TestUploadDocument_RejectsBusinessDocForIndividual(t *testing.T) {
	userID := uuid.New()
	record := &models.VerificationRecord{
		ID:     uuid.New(),
		UserID: userID,
		Type:   models.VerificationIndividual,
		Status: models.VerificationDraft,
	}

	cloudinaryCalled := false
	vr := &mockVerificationRecordsRepo{
		getActiveVerificationByUserFn: func(id string) (*models.VerificationRecord, error) {
			if id != userID.String() {
				t.Fatalf("GetActiveVerificationByUser called with %q, want %q", id, userID)
			}
			return record, nil
		},
	}

	svc := newTestVerificationService(vr, &mockUserRepo{}, nil)
	svc.uploadDocumentFn = func(multipart.File, string) (string, error) {
		cloudinaryCalled = true
		return "https://cdn.example/should-not-happen.jpg", nil
	}

	_, err := svc.UploadDocument(userID.String(), models.DocTypeIncCert, nil, "incorporation.pdf")
	if !errors.Is(err, ErrInvalidDocTypeForAccount) {
		t.Fatalf("UploadDocument error = %v, want ErrInvalidDocTypeForAccount", err)
	}
	if cloudinaryCalled {
		t.Fatal("Cloudinary upload was called for a document type the account cannot use")
	}
}

// ---------------------------------------------------------------------------
// SubmitVerification
// ---------------------------------------------------------------------------

func TestSubmitVerification_PublishesMLJobAndMarksQueued(t *testing.T) {
	rdb := newTestRedis(t)
	userID := uuid.New()

	record := &models.VerificationRecord{
		ID:         uuid.New(),
		UserID:     userID,
		Type:       models.VerificationIndividual,
		Status:     models.VerificationDraft,
		GovtIDHash: "govt-hash",
	}
	docs := []models.VerificationDocument{
		{DocumentType: models.DocTypeSelfie, FileURL: "https://cdn.example/selfie.jpg"},
		{DocumentType: models.DocTypeGovtID, FileURL: "https://cdn.example/govt-id.jpg"},
	}

	var updated *models.VerificationRecord
	vr := &mockVerificationRecordsRepo{
		getActiveVerificationByUserFn: func(id string) (*models.VerificationRecord, error) {
			if id != userID.String() {
				t.Fatalf("GetActiveVerificationByUser called with %q, want %q", id, userID)
			}
			return record, nil
		},
		getDocumentsFn: func(recordID string) ([]models.VerificationDocument, error) {
			if recordID != record.ID.String() {
				t.Fatalf("GetDocuments called with %q, want %q", recordID, record.ID)
			}
			return docs, nil
		},
		updateVerificationRecordFn: func(r *models.VerificationRecord) error {
			updated = r
			return nil
		},
	}

	svc := newTestVerificationService(vr, &mockUserRepo{}, rdb)

	if err := svc.SubmitVerification(userID.String()); err != nil {
		t.Fatalf("SubmitVerification returned an error: %v", err)
	}

	if updated == nil {
		t.Fatal("UpdateVerificationRecord was never called")
	}
	if updated.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q", updated.Status, models.VerificationPending)
	}
	if updated.MLStatus != models.MLStatusQueued {
		t.Errorf("MLStatus = %q, want %q", updated.MLStatus, models.MLStatusQueued)
	}
	if updated.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("ReviewPriority = %q, want %q", updated.ReviewPriority, models.ReviewPriorityManual)
	}
	if updated.SubmittedAt == nil {
		t.Error("SubmittedAt was not set")
	}
	if updated.MLSubmittedAt == nil {
		t.Error("MLSubmittedAt was not set")
	}
	if !strings.HasPrefix(updated.MLJobID, "kyc-job-") {
		t.Errorf("MLJobID = %q, want a kyc-job-* identifier", updated.MLJobID)
	}

	// The job must actually be on the Redis Stream.
	msgs, err := rdb.XRange(context.Background(), kycJobStream, "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange(%s) failed: %v", kycJobStream, err)
	}
	if len(msgs) != 1 {
		t.Fatalf("stream %s contains %d message(s), want 1", kycJobStream, len(msgs))
	}

	raw, ok := msgs[0].Values[kycJobPayloadField].(string)
	if !ok {
		t.Fatalf("stream message is missing the %q field: %#v", kycJobPayloadField, msgs[0].Values)
	}

	var job kycMLJob
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		t.Fatalf("could not decode job payload: %v", err)
	}
	if job.JobID != updated.MLJobID {
		t.Errorf("payload job_id = %q, want %q", job.JobID, updated.MLJobID)
	}
	if job.VerificationID != record.ID.String() {
		t.Errorf("payload verification_id = %q, want %q", job.VerificationID, record.ID)
	}
	if job.UserID != userID.String() {
		t.Errorf("payload user_id = %q, want %q", job.UserID, userID)
	}
	if job.Type != string(models.VerificationIndividual) {
		t.Errorf("payload type = %q, want %q", job.Type, models.VerificationIndividual)
	}
	if len(job.Documents) != len(docs) {
		t.Fatalf("payload contains %d document(s), want %d", len(job.Documents), len(docs))
	}
	for i, doc := range docs {
		if job.Documents[i].Type != doc.DocumentType || job.Documents[i].URL != doc.FileURL {
			t.Errorf("payload document[%d] = %+v, want type=%q url=%q", i, job.Documents[i], doc.DocumentType, doc.FileURL)
		}
	}
}

func TestSubmitVerification_ValidationFailureDoesNotPublish(t *testing.T) {
	rdb := newTestRedis(t)
	userID := uuid.New()

	record := &models.VerificationRecord{
		ID:         uuid.New(),
		UserID:     userID,
		Type:       models.VerificationIndividual,
		Status:     models.VerificationDraft,
		GovtIDHash: "govt-hash",
	}
	vr := &mockVerificationRecordsRepo{
		getActiveVerificationByUserFn: func(string) (*models.VerificationRecord, error) {
			return record, nil
		},
		getDocumentsFn: func(string) ([]models.VerificationDocument, error) {
			// Selfie present, government ID missing.
			return []models.VerificationDocument{{DocumentType: models.DocTypeSelfie}}, nil
		},
	}

	svc := newTestVerificationService(vr, &mockUserRepo{}, rdb)

	if err := svc.SubmitVerification(userID.String()); err == nil {
		t.Fatal("SubmitVerification succeeded with missing required documents")
	}

	count, err := rdb.XLen(context.Background(), kycJobStream).Result()
	if err != nil {
		t.Fatalf("XLen failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("stream contains %d message(s) after a validation failure, want 0", count)
	}
}

func TestSubmitVerification_FailsWhenRedisUnavailable(t *testing.T) {
	userID := uuid.New()
	record := &models.VerificationRecord{
		ID:         uuid.New(),
		UserID:     userID,
		Type:       models.VerificationIndividual,
		Status:     models.VerificationDraft,
		GovtIDHash: "govt-hash",
	}
	docs := []models.VerificationDocument{
		{DocumentType: models.DocTypeSelfie, FileURL: "https://cdn.example/selfie.jpg"},
		{DocumentType: models.DocTypeGovtID, FileURL: "https://cdn.example/govt-id.jpg"},
	}
	vr := &mockVerificationRecordsRepo{
		getActiveVerificationByUserFn: func(string) (*models.VerificationRecord, error) {
			return record, nil
		},
		getDocumentsFn: func(string) ([]models.VerificationDocument, error) {
			return docs, nil
		},
		updateVerificationRecordFn: func(*models.VerificationRecord) error { return nil },
	}

	svc := newTestVerificationService(vr, &mockUserRepo{}, nil)

	err := svc.SubmitVerification(userID.String())
	if !errors.Is(err, ErrMLQueueFailed) {
		t.Fatalf("SubmitVerification error = %v, want ErrMLQueueFailed", err)
	}
	// The record must still be persisted as pending+queued so it can be requeued.
	if record.MLStatus != models.MLStatusQueued {
		t.Errorf("MLStatus = %q, want %q", record.MLStatus, models.MLStatusQueued)
	}
}

// ---------------------------------------------------------------------------
// AdminBulkApprove
// ---------------------------------------------------------------------------

func TestAdminBulkApprove(t *testing.T) {
	adminID := uuid.New()

	t.Run("rejects empty list", func(t *testing.T) {
		svc := newTestVerificationService(&mockVerificationRecordsRepo{}, &mockUserRepo{}, nil)

		err := svc.AdminBulkApprove(nil, adminID.String())
		if !errors.Is(err, ErrInvalidAdminAction) {
			t.Fatalf("error = %v, want ErrInvalidAdminAction", err)
		}
	})

	t.Run("rejects invalid admin UUID", func(t *testing.T) {
		svc := newTestVerificationService(&mockVerificationRecordsRepo{}, &mockUserRepo{}, nil)

		err := svc.AdminBulkApprove([]string{uuid.NewString()}, "not-a-uuid")
		if !errors.Is(err, ErrInvalidAdminAction) {
			t.Fatalf("error = %v, want ErrInvalidAdminAction", err)
		}
	})

	t.Run("parses IDs and delegates to repository", func(t *testing.T) {
		ids := []string{uuid.NewString(), uuid.NewString()}

		var gotIDs []string
		var gotAdmin uuid.UUID
		vr := &mockVerificationRecordsRepo{
			bulkApproveFn: func(got []string, admin uuid.UUID) error {
				gotIDs = append([]string(nil), got...)
				gotAdmin = admin
				return nil
			},
		}
		svc := newTestVerificationService(vr, &mockUserRepo{}, nil)

		if err := svc.AdminBulkApprove(ids, adminID.String()); err != nil {
			t.Fatalf("AdminBulkApprove returned an error: %v", err)
		}
		if len(gotIDs) != len(ids) {
			t.Fatalf("repository received %d ID(s), want %d", len(gotIDs), len(ids))
		}
		for i := range ids {
			if gotIDs[i] != ids[i] {
				t.Errorf("repository ID[%d] = %q, want %q", i, gotIDs[i], ids[i])
			}
		}
		if gotAdmin != adminID {
			t.Errorf("repository admin ID = %s, want %s", gotAdmin, adminID)
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repoErr := errors.New("bulk approve failed")
		vr := &mockVerificationRecordsRepo{
			bulkApproveFn: func([]string, uuid.UUID) error { return repoErr },
		}
		svc := newTestVerificationService(vr, &mockUserRepo{}, nil)

		err := svc.AdminBulkApprove([]string{uuid.NewString()}, adminID.String())
		if !errors.Is(err, repoErr) {
			t.Fatalf("error = %v, want %v", err, repoErr)
		}
	})
}

// ---------------------------------------------------------------------------
// AdminManualAction
// ---------------------------------------------------------------------------

func TestAdminManualAction_ApproveTriggersSideEffects(t *testing.T) {
	adminID := uuid.New()
	userID := uuid.New()
	record := &models.VerificationRecord{
		ID:         uuid.New(),
		UserID:     userID,
		Type:       models.VerificationIndividual,
		Status:     models.VerificationPending,
		GovtIDHash: "govt-hash",
	}

	var (
		markedID      string
		markedAction  string
		markedAdmin   uuid.UUID
		dedupHash     *models.GovtIDDedup
		pointsUser    string
		pointsAwarded int
		statusUser    string
		statusSet     string
	)

	vr := &mockVerificationRecordsRepo{
		getVerificationByIDFn: func(id string) (*models.VerificationRecord, error) {
			return record, nil
		},
		markManualActionFn: func(verificationID, action string, admin uuid.UUID, reason string) error {
			markedID, markedAction, markedAdmin = verificationID, action, admin
			return nil
		},
		createGovtHashFn: func(hash *models.GovtIDDedup) error {
			dedupHash = hash
			return nil
		},
	}
	ur := &mockUserRepo{
		addUserPointsFn: func(id string, points int) error {
			pointsUser, pointsAwarded = id, points
			return nil
		},
		updateUserStatusFn: func(id, status string) error {
			statusUser, statusSet = id, status
			return nil
		},
	}

	svc := newTestVerificationService(vr, ur, nil)

	if err := svc.AdminManualAction(record.ID.String(), "APPROVE", adminID.String(), ""); err != nil {
		t.Fatalf("AdminManualAction returned an error: %v", err)
	}

	if markedID != record.ID.String() || markedAction != "approve" || markedAdmin != adminID {
		t.Errorf("MarkManualAction called with (%q, %q, %s), want (%q, %q, %s)",
			markedID, markedAction, markedAdmin, record.ID, "approve", adminID)
	}
	if dedupHash == nil {
		t.Fatal("government ID hash was not registered for deduplication")
	}
	if dedupHash.GovtIDHash != record.GovtIDHash || dedupHash.UserID != userID {
		t.Errorf("dedup entry = %+v, want hash=%q user=%s", dedupHash, record.GovtIDHash, userID)
	}
	if pointsUser != userID.String() || pointsAwarded != 40 {
		t.Errorf("AddUserPoints called with (%q, %d), want (%q, 40)", pointsUser, pointsAwarded, userID)
	}
	if statusUser != userID.String() || statusSet != string(models.StatusActive) {
		t.Errorf("UpdateUserStatus called with (%q, %q), want (%q, %q)",
			statusUser, statusSet, userID, models.StatusActive)
	}
}

func TestAdminManualAction_ApproveBusinessRegistersPAN(t *testing.T) {
	adminID := uuid.New()
	userID := uuid.New()
	record := &models.VerificationRecord{
		ID:              uuid.New(),
		UserID:          userID,
		Type:            models.VerificationBusiness,
		Status:          models.VerificationPending,
		BusinessPANHash: "pan-hash",
	}

	var dedupPAN *models.BusinessPANDedup
	vr := &mockVerificationRecordsRepo{
		getVerificationByIDFn: func(string) (*models.VerificationRecord, error) { return record, nil },
		markManualActionFn:    func(string, string, uuid.UUID, string) error { return nil },
		createBusinessHashFn: func(hash *models.BusinessPANDedup) error {
			dedupPAN = hash
			return nil
		},
	}
	ur := &mockUserRepo{
		addUserPointsFn:    func(string, int) error { return nil },
		updateUserStatusFn: func(string, string) error { return nil },
	}

	svc := newTestVerificationService(vr, ur, nil)
	if err := svc.AdminManualAction(record.ID.String(), "approve", adminID.String(), ""); err != nil {
		t.Fatalf("AdminManualAction returned an error: %v", err)
	}
	if dedupPAN == nil {
		t.Fatal("business PAN hash was not registered for deduplication")
	}
	if dedupPAN.PANHash != record.BusinessPANHash || dedupPAN.UserID != userID {
		t.Errorf("dedup entry = %+v, want hash=%q user=%s", dedupPAN, record.BusinessPANHash, userID)
	}
}

func TestAdminManualAction_Reject(t *testing.T) {
	adminID := uuid.New()
	record := &models.VerificationRecord{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Type:   models.VerificationIndividual,
		Status: models.VerificationPending,
	}

	sideEffectsCalled := false
	ur := &mockUserRepo{
		addUserPointsFn:    func(string, int) error { sideEffectsCalled = true; return nil },
		updateUserStatusFn: func(string, string) error { sideEffectsCalled = true; return nil },
	}

	t.Run("requires a reason", func(t *testing.T) {
		vr := &mockVerificationRecordsRepo{
			getVerificationByIDFn: func(string) (*models.VerificationRecord, error) { return record, nil },
		}
		svc := newTestVerificationService(vr, ur, nil)

		err := svc.AdminManualAction(record.ID.String(), "reject", adminID.String(), "")
		if !errors.Is(err, ErrInvalidAdminAction) {
			t.Fatalf("error = %v, want ErrInvalidAdminAction", err)
		}
	})

	t.Run("records rejection without approval side effects", func(t *testing.T) {
		var gotAction, gotReason string
		vr := &mockVerificationRecordsRepo{
			getVerificationByIDFn: func(string) (*models.VerificationRecord, error) { return record, nil },
			markManualActionFn: func(_ string, action string, _ uuid.UUID, reason string) error {
				gotAction, gotReason = action, reason
				return nil
			},
		}
		svc := newTestVerificationService(vr, ur, nil)

		if err := svc.AdminManualAction(record.ID.String(), "REJECT", adminID.String(), "blurry document"); err != nil {
			t.Fatalf("AdminManualAction returned an error: %v", err)
		}
		if gotAction != "reject" || gotReason != "blurry document" {
			t.Errorf("MarkManualAction called with action=%q reason=%q", gotAction, gotReason)
		}
		if sideEffectsCalled {
			t.Error("approval side effects ran for a rejection")
		}
	})
}

func TestAdminManualAction_RejectsInvalidInput(t *testing.T) {
	record := &models.VerificationRecord{ID: uuid.New(), UserID: uuid.New()}

	t.Run("invalid admin UUID", func(t *testing.T) {
		svc := newTestVerificationService(&mockVerificationRecordsRepo{}, &mockUserRepo{}, nil)
		err := svc.AdminManualAction(record.ID.String(), "approve", "nope", "")
		if !errors.Is(err, ErrInvalidAdminAction) {
			t.Fatalf("error = %v, want ErrInvalidAdminAction", err)
		}
	})

	t.Run("unsupported action", func(t *testing.T) {
		svc := newTestVerificationService(&mockVerificationRecordsRepo{}, &mockUserRepo{}, nil)
		err := svc.AdminManualAction(record.ID.String(), "escalate", uuid.NewString(), "")
		if !errors.Is(err, ErrInvalidAdminAction) {
			t.Fatalf("error = %v, want ErrInvalidAdminAction", err)
		}
	})
}

func TestAdminManualAction_PropagatesMarkErrorAndSkipsSideEffects(t *testing.T) {
	adminID := uuid.New()
	record := &models.VerificationRecord{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		Type:       models.VerificationIndividual,
		GovtIDHash: "govt-hash",
	}
	repoErr := errors.New("write failed")

	sideEffectsCalled := false
	vr := &mockVerificationRecordsRepo{
		getVerificationByIDFn: func(string) (*models.VerificationRecord, error) { return record, nil },
		markManualActionFn:    func(string, string, uuid.UUID, string) error { return repoErr },
	}
	ur := &mockUserRepo{
		addUserPointsFn:    func(string, int) error { sideEffectsCalled = true; return nil },
		updateUserStatusFn: func(string, string) error { sideEffectsCalled = true; return nil },
	}

	svc := newTestVerificationService(vr, ur, nil)
	err := svc.AdminManualAction(record.ID.String(), "approve", adminID.String(), "")
	if !errors.Is(err, repoErr) {
		t.Fatalf("error = %v, want %v", err, repoErr)
	}
	if sideEffectsCalled {
		t.Error("approval side effects ran after MarkManualAction failed")
	}
}

// ---------------------------------------------------------------------------
// ProcessStalePendingEscalations (24h stale / 48h hard-stop, manual only)
//
// The escalation write goes through postgres.DB (markEscalated, Phase 3): it
// is a single guarded UPDATE that stamps system_escalated_at and
// routing_state='routed' as a terminal marker. These tests therefore run
// against the shared in-memory SQLite database (setupEscalationTestDB) so the
// write path is exercised for real, while the queue/audit seams stay mocked.
// ---------------------------------------------------------------------------

func TestProcessStalePendingEscalations_EscalatesAfter24Hours(t *testing.T) {
	db := setupEscalationTestDB(t)

	userID := uuid.New()
	mlCompletedAt := time.Now().Add(-25 * time.Hour)
	rec := models.VerificationRecord{
		ID:             uuid.New(),
		UserID:         userID,
		Type:           models.VerificationIndividual,
		Status:         models.VerificationPending,
		ReviewPriority: models.ReviewPriorityAutoEligible,
		MLStatus:       models.MLStatusCompleted,
		MLCompletedAt:  &mlCompletedAt,
		RoutingState:   models.RoutingPending,
	}
	if err := db.Omit(clause.Associations).Create(&rec).Error; err != nil {
		t.Fatalf("create verification record: %v", err)
	}

	var (
		minAges    []float64
		auditEntry *models.VerificationAuditLog
	)

	vr := &mockVerificationRecordsRepo{
		getAutoEligibleQueueFn: func(limit, offset int, minAgeHours float64) ([]models.VerificationRecord, int64, error) {
			minAges = append(minAges, minAgeHours)
			if offset != 0 {
				t.Errorf("GetAutoEligibleQueue offset = %d, want 0", offset)
			}
			if minAgeHours == kycStalePendingWindowHours {
				return []models.VerificationRecord{rec}, 1, nil
			}
			return nil, 0, nil
		},
		createVerificationAuditLogFn: func(entry *models.VerificationAuditLog) error {
			auditEntry = entry
			return nil
		},
	}

	svc := newTestVerificationService(vr, &mockUserRepo{}, nil)
	svc.ProcessStalePendingEscalations()

	if len(minAges) != 2 || minAges[0] != kycAutoEscalateWindowHours || minAges[1] != kycStalePendingWindowHours {
		t.Errorf("escalation queue min ages = %v, want [%v %v]",
			minAges, kycAutoEscalateWindowHours, kycStalePendingWindowHours)
	}
	if auditEntry == nil {
		t.Fatal("no audit log entry was written")
	}
	if auditEntry.ActionType != "AI_ESCALATED_STALE" {
		t.Errorf("audit action = %q, want AI_ESCALATED_STALE", auditEntry.ActionType)
	}
	if auditEntry.VerificationID != rec.ID || auditEntry.UserID != userID {
		t.Errorf("audit entry references (%s, %s), want (%s, %s)",
			auditEntry.VerificationID, auditEntry.UserID, rec.ID, userID)
	}

	var got models.VerificationRecord
	if err := db.First(&got, "id = ?", rec.ID).Error; err != nil {
		t.Fatalf("reload verification record: %v", err)
	}
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.SystemEscalatedAt == nil {
		t.Error("system_escalated_at = nil, want set (terminal escalation marker)")
	}
	if got.RoutingState != models.RoutingRouted {
		t.Errorf("routing_state = %q, want %q (routed records are invisible to the Go router)", got.RoutingState, models.RoutingRouted)
	}
	if got.AdminReviewRequestedAt == nil {
		t.Error("admin_review_requested_at = nil, want set")
	}
}

func TestProcessStalePendingEscalations_Preserves48HourEscalation(t *testing.T) {
	db := setupEscalationTestDB(t)

	userID := uuid.New()
	mlCompletedAt := time.Now().Add(-49 * time.Hour)
	rec := models.VerificationRecord{
		ID:             uuid.New(),
		UserID:         userID,
		Type:           models.VerificationIndividual,
		Status:         models.VerificationPending,
		ReviewPriority: models.ReviewPriorityAutoEligible,
		MLStatus:       models.MLStatusCompleted,
		MLCompletedAt:  &mlCompletedAt,
	}
	if err := db.Omit(clause.Associations).Create(&rec).Error; err != nil {
		t.Fatalf("create verification record: %v", err)
	}

	var (
		minAges    []float64
		auditEntry *models.VerificationAuditLog
	)

	vr := &mockVerificationRecordsRepo{
		getAutoEligibleQueueFn: func(limit, offset int, minAgeHours float64) ([]models.VerificationRecord, int64, error) {
			minAges = append(minAges, minAgeHours)
			if minAgeHours == kycAutoEscalateWindowHours {
				return []models.VerificationRecord{rec}, 1, nil
			}
			return nil, 0, nil
		},
		createVerificationAuditLogFn: func(entry *models.VerificationAuditLog) error {
			auditEntry = entry
			return nil
		},
	}

	svc := newTestVerificationService(vr, &mockUserRepo{}, nil)
	svc.ProcessStalePendingEscalations()

	if len(minAges) == 0 || minAges[0] != kycAutoEscalateWindowHours {
		t.Errorf("first escalation queue min age = %v, want %v", minAges, kycAutoEscalateWindowHours)
	}
	if auditEntry == nil {
		t.Fatal("no audit log entry was written")
	}
	if auditEntry.ActionType != "AI_ESCALATED_48H" {
		t.Errorf("audit action = %q, want AI_ESCALATED_48H", auditEntry.ActionType)
	}

	var got models.VerificationRecord
	if err := db.First(&got, "id = ?", rec.ID).Error; err != nil {
		t.Fatalf("reload verification record: %v", err)
	}
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.AutoApprovedAt != nil {
		t.Errorf("auto_approved_at = %v, want nil (escalation never approves)", got.AutoApprovedAt)
	}
	if got.SystemEscalatedAt == nil {
		t.Error("system_escalated_at = nil, want set (terminal escalation marker)")
	}
}

func TestProcessStalePendingEscalations_StopsWhenFullBatchFails(t *testing.T) {
	// A full batch whose rows cannot be escalated (here: the record left the
	// escalatable state before the write, e.g. an admin approved it between
	// the queue SELECT and the UPDATE) must terminate the pass, not hot-loop.
	db := setupEscalationTestDB(t)

	records := make([]models.VerificationRecord, kycAutoBatchSize)
	for i := range records {
		records[i] = models.VerificationRecord{
			ID:             uuid.New(),
			UserID:         uuid.New(),
			Type:           models.VerificationIndividual,
			Status:         models.VerificationApproved, // already decided: guard rejects
			ReviewPriority: models.ReviewPriorityAutoEligible,
			MLStatus:       models.MLStatusCompleted,
		}
	}
	if err := db.Omit(clause.Associations).Create(&records).Error; err != nil {
		t.Fatalf("create verification records: %v", err)
	}

	queueCalls := 0
	auditCalls := 0
	vr := &mockVerificationRecordsRepo{
		getAutoEligibleQueueFn: func(int, int, float64) ([]models.VerificationRecord, int64, error) {
			queueCalls++
			return records, int64(len(records)), nil
		},
		createVerificationAuditLogFn: func(*models.VerificationAuditLog) error {
			auditCalls++
			return nil
		},
	}

	svc := newTestVerificationService(vr, &mockUserRepo{}, nil)
	svc.ProcessStalePendingEscalations() // must terminate, not hot-loop on un-escalatable rows

	if queueCalls != 2 {
		t.Errorf("GetAutoEligibleQueue called %d time(s), want 2 (one stale pass per auto-eligible window)", queueCalls)
	}
	if auditCalls != 0 {
		t.Errorf("CreateVerificationAuditLog called %d time(s), want 0 (nothing was escalated)", auditCalls)
	}
}

func TestProcessStalePendingEscalations_NoopWhenQueuesEmpty(t *testing.T) {
	// The stuck-in-ML pass reads postgres.DB directly, so this noop test runs
	// against a real (empty) SQLite database too.
	db := setupEscalationTestDB(t)
	_ = db

	queueCalls := 0
	vr := &mockVerificationRecordsRepo{
		getAutoEligibleQueueFn: func(int, int, float64) ([]models.VerificationRecord, int64, error) {
			queueCalls++
			return nil, 0, nil
		},
	}

	svc := newTestVerificationService(vr, &mockUserRepo{}, nil)
	svc.ProcessStalePendingEscalations() // must not panic or call any mutation method

	if queueCalls != 2 {
		t.Errorf("GetAutoEligibleQueue called %d time(s), want 2", queueCalls)
	}
}

// ---------------------------------------------------------------------------
// Phase 5 individual doc-type allow-list + submit validation
// ---------------------------------------------------------------------------

func TestSubmitRequiresAadhaar(t *testing.T) {
	userID := uuid.New()
	record := &models.VerificationRecord{
		ID:         uuid.New(),
		UserID:     userID,
		Type:       models.VerificationIndividual,
		Status:     models.VerificationDraft,
		GovtIDHash: "govt-hash",
	}
	docs := []models.VerificationDocument{
		{DocumentType: models.DocTypeSelfie, FileURL: "https://cdn.example/selfie.jpg"},
		{DocumentType: models.DocTypePANCard, FileURL: "https://cdn.example/pan.jpg"},
	}
	vr := &mockVerificationRecordsRepo{
		getActiveVerificationByUserFn: func(string) (*models.VerificationRecord, error) {
			return record, nil
		},
		getDocumentsFn: func(string) ([]models.VerificationDocument, error) {
			return docs, nil
		},
	}
	svc := newTestVerificationService(vr, &mockUserRepo{}, nil)

	err := svc.SubmitVerification(userID.String())
	if err == nil {
		t.Fatal("SubmitVerification succeeded with only pan_card (aadhaar required)")
	}
	if !strings.Contains(err.Error(), "aadhaar") {
		t.Errorf("error = %v, want it to mention the required aadhaar document", err)
	}
}

func TestSubmitAcceptsAadhaarPlusPan(t *testing.T) {
	rdb := newTestRedis(t)
	userID := uuid.New()
	record := &models.VerificationRecord{
		ID:         uuid.New(),
		UserID:     userID,
		Type:       models.VerificationIndividual,
		Status:     models.VerificationDraft,
		GovtIDHash: "govt-hash",
	}
	docs := []models.VerificationDocument{
		{DocumentType: models.DocTypeSelfie, FileURL: "https://cdn.example/selfie.jpg"},
		{DocumentType: models.DocTypeAadhaar, FileURL: "https://cdn.example/aadhaar.jpg"},
		{DocumentType: models.DocTypePANCard, FileURL: "https://cdn.example/pan.jpg"},
	}
	updated := false
	vr := &mockVerificationRecordsRepo{
		getActiveVerificationByUserFn: func(string) (*models.VerificationRecord, error) {
			return record, nil
		},
		getDocumentsFn: func(string) ([]models.VerificationDocument, error) {
			return docs, nil
		},
		updateVerificationRecordFn: func(*models.VerificationRecord) error {
			updated = true
			return nil
		},
	}
	svc := newTestVerificationService(vr, &mockUserRepo{}, rdb)

	if err := svc.SubmitVerification(userID.String()); err != nil {
		t.Fatalf("SubmitVerification with aadhaar+pan returned an error: %v", err)
	}
	if !updated {
		t.Fatal("UpdateVerificationRecord was never called for a valid aadhaar+pan submission")
	}
}

func TestSubmitLegacyGovtIDLogsWarning(t *testing.T) {
	rdb := newTestRedis(t)
	userID := uuid.New()
	record := &models.VerificationRecord{
		ID:         uuid.New(),
		UserID:     userID,
		Type:       models.VerificationIndividual,
		Status:     models.VerificationDraft,
		GovtIDHash: "govt-hash",
	}
	docs := []models.VerificationDocument{
		{DocumentType: models.DocTypeSelfie, FileURL: "https://cdn.example/selfie.jpg"},
		{DocumentType: models.DocTypeGovtID, FileURL: "https://cdn.example/govt-id.jpg"},
	}
	updated := false
	vr := &mockVerificationRecordsRepo{
		getActiveVerificationByUserFn: func(string) (*models.VerificationRecord, error) {
			return record, nil
		},
		getDocumentsFn: func(string) ([]models.VerificationDocument, error) {
			return docs, nil
		},
		updateVerificationRecordFn: func(*models.VerificationRecord) error {
			updated = true
			return nil
		},
	}
	svc := newTestVerificationService(vr, &mockUserRepo{}, rdb)

	if err := svc.SubmitVerification(userID.String()); err != nil {
		t.Fatalf("SubmitVerification with legacy govt_id returned an error: %v", err)
	}
	if !updated {
		t.Fatal("UpdateVerificationRecord was never called for a legacy govt_id submission")
	}
}

func TestDocTypeAadhaarAllowedForIndividual(t *testing.T) {
	if !isDocTypeAllowedForVerification(models.VerificationIndividual, models.DocTypeAadhaar) {
		t.Error("isDocTypeAllowedForVerification(Individual, Aadhaar) = false, want true")
	}
	if !isDocTypeAllowedForVerification(models.VerificationIndividual, models.DocTypePANCard) {
		t.Error("isDocTypeAllowedForVerification(Individual, PANCard) = false, want true")
	}
	if isDocTypeAllowedForVerification(models.VerificationBusiness, models.DocTypeAadhaar) {
		t.Error("isDocTypeAllowedForVerification(Business, Aadhaar) = true, want false")
	}
}
