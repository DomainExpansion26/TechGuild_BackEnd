package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"techguild-backend/src/config"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
)

// ---------------------------------------------------------------------------
// Router test helpers.
//
// The DB is the shared in-memory SQLite instance from setupEscalationTestDB:
// the router's guard SQL and the escalator's claim transaction run for real
// (SQLite's JSON1 functions stand in for PostgreSQL's ->> operators).
// ---------------------------------------------------------------------------

func routerTestPolicy(shadowMode bool, cohortPct int) config.KYCPolicy {
	return config.KYCPolicy{
		ShadowMode:       shadowMode,
		RolloutCohortPct: cohortPct,
		PolicyVersion:    "router-test-v1",
		FaceAutoPass:     0.363,
		FaceReview:       0.300,
		FaceReject:       0.240,
		LivenessPass:     0.90,
		LivenessReject:   0.50,
		OCRPass:          0.80,
		AadhaarFloor:     0.35,
	}
}

// routerEvidenceJSON mirrors the Phase 5 worker's face_match_result shape:
// face_similarity is the multi-reference winner, aadhaar_similarity carries
// the QR-anchor score (nil when no usable Aadhaar photo existed), and
// second_reference_divergence is the Python-side guardrail flag.
func routerEvidenceJSON(
	faceSim float64,
	aadhaarSim *float64,
	divergence bool,
	isLive bool,
	liveScore float64,
) (face, live, ocr datatypes.JSON) {
	faceJSON := map[string]interface{}{
		"face_similarity":             faceSim,
		"winning_reference":           "aadhaar",
		"second_reference_divergence": divergence,
	}
	if aadhaarSim == nil {
		faceJSON["aadhaar_similarity"] = nil
	} else {
		faceJSON["aadhaar_similarity"] = *aadhaarSim
	}
	face, _ = json.Marshal(faceJSON)
	live, _ = json.Marshal(map[string]interface{}{"is_live": isLive, "liveness_score": liveScore})
	ocr, _ = json.Marshal(map[string]interface{}{
		"ocr_confidence":      0.95,
		"text":                "NAME: ROUTER TEST",
		"document_validation": map[string]interface{}{"overall": "verified"},
	})
	return face, live, ocr
}

// ptrRouterFloat is a small helper for routerRecordOpts.aadhaarSim (nil in the
// struct means "use the healthy default", so the value needs explicit boxing).
func ptrRouterFloat(v float64) *float64 { return &v }

// routerRecordOpts builds a record whose shape matches the worker's output.
type routerRecordOpts struct {
	aiPriority      models.VerificationReviewPriority
	faceSim         float64
	isLive          bool
	livenessScore   float64
	mlStatus        models.MLVerificationStatus
	submittedAt     *time.Time
	systemEscalated bool

	// Phase 5 face evidence. aadhaarSim=nil (default) uses a healthy 0.52 so
	// the pre-Phase 5 tests keep passing; use aadhaarNil for an explicit null
	// (worker could not score any Aadhaar photo).
	aadhaarSim *float64
	aadhaarNil bool
	divergence bool
}

func createRouterTestRecord(t *testing.T, db *gorm.DB, opts routerRecordOpts) models.VerificationRecord {
	t.Helper()

	user := models.User{
		FirstName: "Router",
		LastName:  "Test",
		Email:     fmt.Sprintf("router-%s@example.com", uuid.NewString()),
		Status:    models.StatusActive,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create test user: %v", err)
	}

	mlCompletedAt := time.Now().Add(-time.Hour)
	if opts.mlStatus == "" {
		opts.mlStatus = models.MLStatusCompleted
	}
	if opts.mlStatus != models.MLStatusCompleted {
		mlCompletedAt = time.Time{} // zero value: no completion timestamp
	}

	healthyAadhaar := 0.52 // above the 0.35 floor: Phase 5 gate stays green
	if opts.aadhaarSim != nil {
		healthyAadhaar = *opts.aadhaarSim
	}
	faceAadhaar := &healthyAadhaar
	if opts.aadhaarNil {
		faceAadhaar = nil
	}
	face, live, ocr := routerEvidenceJSON(opts.faceSim, faceAadhaar, opts.divergence, opts.isLive, opts.livenessScore)

	rec := models.VerificationRecord{
		UserID:                user.ID,
		Type:                  models.VerificationIndividual,
		Status:                models.VerificationPending,
		ReviewPriority:        opts.aiPriority,
		AIRecommendedPriority: opts.aiPriority,
		MLStatus:              opts.mlStatus,
		MLModelVersion:        "test-model-v1",
		FaceMatchResult:       face,
		LivenessResult:        live,
		OCROcrResult:          ocr,
		RoutingState:          models.RoutingPending,
	}
	if opts.mlStatus == models.MLStatusCompleted {
		rec.MLCompletedAt = &mlCompletedAt
	} else {
		rec.MLCompletedAt = nil
	}
	if opts.submittedAt != nil {
		rec.SubmittedAt = opts.submittedAt
	}
	if opts.systemEscalated {
		escalatedAt := time.Now()
		rec.SystemEscalatedAt = &escalatedAt
		rec.RoutingState = models.RoutingRouted
	}
	if err := db.Omit(clause.Associations).Create(&rec).Error; err != nil {
		t.Fatalf("create verification record: %v", err)
	}
	return rec
}

func setupRouterService(db *gorm.DB, shadowMode bool, cohortPct int) *VerificationService {
	svc := &VerificationService{
		verificationRecordsRepo: repository.NewVerificationRecordsRepository(),
	}
	policy := routerTestPolicy(shadowMode, cohortPct)
	svc.SetKYCPolicy(policy)
	return svc
}

func runRouter(t *testing.T, svc *VerificationService, batchSize int) (routed, approved, rejected, manual int) {
	t.Helper()
	routed, approved, rejected, manual, err := svc.RouteMLCompleted(batchSize)
	if err != nil {
		t.Fatalf("RouteMLCompleted: %v", err)
	}
	return routed, approved, rejected, manual
}

func reloadRouterRecord(t *testing.T, db *gorm.DB, id uuid.UUID) models.VerificationRecord {
	t.Helper()
	var got models.VerificationRecord
	if err := db.First(&got, "id = ?", id).Error; err != nil {
		t.Fatalf("reload verification record: %v", err)
	}
	return got
}

func countRouterAudit(t *testing.T, db *gorm.DB, verificationID uuid.UUID, actionType string) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&models.VerificationAuditLog{}).
		Where("verification_id = ? AND action_type = ?", verificationID, actionType).
		Count(&count).Error; err != nil {
		t.Fatalf("count audit log entries: %v", err)
	}
	return count
}

// ---------------------------------------------------------------------------
// Shadow mode (ChatGPT P0 #4)
// ---------------------------------------------------------------------------

func TestRouterShadowModeForcesManual(t *testing.T) {
	db := setupEscalationTestDB(t)

	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:    models.ReviewPriorityAutoEligible,
		faceSim:       0.90,
		isLive:        true,
		livenessScore: 0.96,
	})

	svc := setupRouterService(db, true /* shadow */, 0)
	routed, approved, rejected, manual := runRouter(t, svc, 10)

	if routed != 1 || approved != 0 || rejected != 0 || manual != 1 {
		t.Fatalf("router counted routed=%d approved=%d rejected=%d manual=%d; want 1 manual", routed, approved, rejected, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q (shadow mode)", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.RoutingState != models.RoutingRouted {
		t.Errorf("routing_state = %q, want %q", got.RoutingState, models.RoutingRouted)
	}
	if got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q (manual review, never approved in shadow)", got.Status, models.VerificationPending)
	}
	if got.DecisionSource != routerSourceShadow {
		t.Errorf("decision_source = %q, want %q", got.DecisionSource, routerSourceShadow)
	}
	if !got.ShadowModeApplied {
		t.Error("shadow_mode_applied = false, want true")
	}
	if got.AutoApprovedAt != nil {
		t.Errorf("auto_approved_at = %v, want nil", got.AutoApprovedAt)
	}
}

// ---------------------------------------------------------------------------
// Evidence gates (ChatGPT P0 #2)
// ---------------------------------------------------------------------------

func TestRouterEvidenceIncompleteRoutesManual(t *testing.T) {
	db := setupEscalationTestDB(t)

	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:    models.ReviewPriorityAutoEligible,
		faceSim:       0.10, // below FaceAutoPass: the AI recommendation must not carry
		isLive:        true,
		livenessScore: 0.96,
	})

	svc := setupRouterService(db, false, 100)
	_, approved, _, manual := runRouter(t, svc, 10)

	if approved != 0 || manual != 1 {
		t.Fatalf("router counted approved=%d manual=%d; want 0 approved, 1 manual", approved, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.DecisionSource != routerSourceSystem {
		t.Errorf("decision_source = %q, want %q", got.DecisionSource, routerSourceSystem)
	}
	if got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationPending)
	}
}

func TestRouterApprovesWithFullEvidence(t *testing.T) {
	db := setupEscalationTestDB(t)

	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:    models.ReviewPriorityAutoEligible,
		faceSim:       0.90, // >= 0.363
		isLive:        true, // != false
		livenessScore: 0.96, // >= 0.90
		// OCR 0.95 >= 0.80 and document_validation = verified (fixture)
	})

	svc := setupRouterService(db, false /* shadow off */, 100 /* everyone in cohort */)
	routed, approved, rejected, manual := runRouter(t, svc, 10)

	if routed != 1 || approved != 1 || rejected != 0 || manual != 0 {
		t.Fatalf("router counted routed=%d approved=%d rejected=%d manual=%d; want 1 approved", routed, approved, rejected, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.Status != models.VerificationApproved {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationApproved)
	}
	if got.ReviewPriority != models.ReviewPriorityAutoEligible {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityAutoEligible)
	}
	if got.VerifiedAt == nil || got.AutoApprovedAt == nil {
		t.Errorf("verified_at=%v auto_approved_at=%v, want both set", got.VerifiedAt, got.AutoApprovedAt)
	}
	if got.RoutingState != models.RoutingRouted {
		t.Errorf("routing_state = %q, want %q", got.RoutingState, models.RoutingRouted)
	}
	if got.DecisionSource != routerSourceAI {
		t.Errorf("decision_source = %q, want %q", got.DecisionSource, routerSourceAI)
	}
	if got.ShadowModeApplied {
		t.Error("shadow_mode_applied = true, want false (cohort rollout)")
	}
	if got.PolicyVersion != "router-test-v1" {
		t.Errorf("policy_version = %q, want %q", got.PolicyVersion, "router-test-v1")
	}
	if countRouterAudit(t, db, rec.ID, routerActionType) != 1 {
		t.Error("one KYC_ROUTED audit row expected for the approval")
	}
}

func TestRouterGuardRejectsWeakEvidence(t *testing.T) {
	db := setupEscalationTestDB(t)
	dialect := db.Dialector.Name()

	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:    models.ReviewPriorityAutoEligible,
		faceSim:       0.10, // weak evidence: the DB-level guard must refuse approval
		isLive:        true,
		livenessScore: 0.96,
	})

	policy := routerTestPolicy(false, 100)

	// Direct SQL: the approval guard is invoked standalone — no Go-side gate —
	// and must touch zero rows because the evidence is below threshold.
	result := db.Exec(routerApproveGuardSQL(dialect),
		true, // shadow_mode_applied arg (any value; the WHERE short-circuits)
		"ai",
		policy.PolicyVersion,
		rec.ID,
		policy.FaceAutoPass,
		policy.AadhaarFloor,
		policy.LivenessPass,
		policy.OCRPass,
	)
	if result.Error != nil {
		t.Fatalf("guard SQL failed: %v", result.Error)
	}
	if result.RowsAffected != 0 {
		t.Fatalf("guard affected %d row(s), want 0 (weak evidence must never approve)", result.RowsAffected)
	}
	if got := reloadRouterRecord(t, db, rec.ID); got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q (direct SQL must not approve)", got.Status, models.VerificationPending)
	}

	// Through the router the weak record falls back to manual review.
	svc := setupRouterService(db, false, 100)
	_, approved, _, manual := runRouter(t, svc, 10)
	if approved != 0 || manual != 1 {
		t.Fatalf("router counted approved=%d manual=%d; want 0 approved, 1 manual", approved, manual)
	}
	got := reloadRouterRecord(t, db, rec.ID)
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityManual)
	}
}

// ---------------------------------------------------------------------------
// Phase 5: Aadhaar floor + divergence guard (defense in depth)
// ---------------------------------------------------------------------------

func TestRouterRequiresAadhaarFloor(t *testing.T) {
	db := setupEscalationTestDB(t)

	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:    models.ReviewPriorityAutoEligible,
		faceSim:       0.70,                 // PAN wins on the raw score...
		aadhaarSim:    ptrRouterFloat(0.20), // ...but the QR anchor is below floor
		isLive:        true,
		livenessScore: 0.96,
	})

	svc := setupRouterService(db, false, 100)

	// Direct policy assertions (no DB write): the pure functions must reject.
	if svc.evidenceComplete(&rec) {
		t.Error("evidenceComplete = true with aadhaar_similarity below floor, want false")
	}
	if dec := svc.decideRouting(&rec); dec.Effective != models.ReviewPriorityManual || dec.Approve {
		t.Errorf("decideRouting = %+v, want manual review (aadhaar floor)", dec)
	}

	_, approved, _, manual := runRouter(t, svc, 10)

	if approved != 0 || manual != 1 {
		t.Fatalf("router counted approved=%d manual=%d; want 0 approved, 1 manual", approved, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q (aadhaar below floor blocks approval)", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.DecisionSource != routerSourceSystem {
		t.Errorf("decision_source = %q, want %q", got.DecisionSource, routerSourceSystem)
	}
	if got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationPending)
	}
}

func TestRouterAcceptsPanWinnerWithStrongAadhaar(t *testing.T) {
	db := setupEscalationTestDB(t)

	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:    models.ReviewPriorityAutoEligible,
		faceSim:       0.70,                 // PAN wins...
		aadhaarSim:    ptrRouterFloat(0.45), // ...but the Aadhaar corroborates (>= 0.35)
		isLive:        true,
		livenessScore: 0.96,
	})

	svc := setupRouterService(db, false, 100)

	// Direct policy assertions: strong Aadhaar corroboration clears the gate.
	if !svc.evidenceComplete(&rec) {
		t.Error("evidenceComplete = false with strong aadhaar_similarity, want true")
	}
	if dec := svc.decideRouting(&rec); !dec.Approve || dec.Effective != models.ReviewPriorityAutoEligible {
		t.Errorf("decideRouting = %+v, want approve/auto_eligible", dec)
	}

	routed, approved, rejected, manual := runRouter(t, svc, 10)

	if routed != 1 || approved != 1 || rejected != 0 || manual != 0 {
		t.Fatalf("router counted routed=%d approved=%d rejected=%d manual=%d; want 1 approved", routed, approved, rejected, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.Status != models.VerificationApproved {
		t.Errorf("status = %q, want %q (strong Aadhaar + no divergence can approve)", got.Status, models.VerificationApproved)
	}
	if got.ReviewPriority != models.ReviewPriorityAutoEligible {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityAutoEligible)
	}
}

func TestRouterRejectsDivergence(t *testing.T) {
	db := setupEscalationTestDB(t)

	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:    models.ReviewPriorityAutoEligible,
		faceSim:       0.70,                 // winning score is strong...
		aadhaarSim:    ptrRouterFloat(0.20), // ...but diverged hard from the Aadhaar
		divergence:    true,                 // worker flagged second_reference_divergence
		isLive:        true,
		livenessScore: 0.96,
	})

	svc := setupRouterService(db, false, 100)

	// Direct policy assertions: divergence alone must block auto-approval.
	if svc.evidenceComplete(&rec) {
		t.Error("evidenceComplete = true with second_reference_divergence, want false")
	}
	if dec := svc.decideRouting(&rec); dec.Effective != models.ReviewPriorityManual || dec.Approve {
		t.Errorf("decideRouting = %+v, want manual review (divergence)", dec)
	}

	_, approved, _, manual := runRouter(t, svc, 10)

	if approved != 0 || manual != 1 {
		t.Fatalf("router counted approved=%d manual=%d; want 0 approved, 1 manual", approved, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q (divergence caps at manual regardless of max score)", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationPending)
	}
}

func TestRouterAadhaarNilBlocksAuto(t *testing.T) {
	db := setupEscalationTestDB(t)

	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:    models.ReviewPriorityAutoEligible,
		faceSim:       0.70,
		aadhaarNil:    true, // no usable Aadhaar photo: PAN-only cannot auto-approve
		isLive:        true,
		livenessScore: 0.96,
	})

	svc := setupRouterService(db, false, 100)

	// Direct policy assertions: a nil aadhaar_similarity blocks auto-approval.
	if svc.evidenceComplete(&rec) {
		t.Error("evidenceComplete = true with nil aadhaar_similarity, want false")
	}
	if dec := svc.decideRouting(&rec); dec.Effective != models.ReviewPriorityManual || dec.Approve {
		t.Errorf("decideRouting = %+v, want manual review (nil aadhaar)", dec)
	}

	_, approved, _, manual := runRouter(t, svc, 10)

	if approved != 0 || manual != 1 {
		t.Fatalf("router counted approved=%d manual=%d; want 0 approved, 1 manual", approved, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q (nil aadhaar_similarity blocks auto)", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationPending)
	}
}

// ---------------------------------------------------------------------------
// Auto-reject gate
// ---------------------------------------------------------------------------

func TestRouterRejectsLivenessSpoof(t *testing.T) {
	db := setupEscalationTestDB(t)

	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:    models.ReviewPriorityAutoReject,
		faceSim:       0.90,
		isLive:        false,
		livenessScore: 0.10, // clean spoof signal: below the reject threshold
	})

	svc := setupRouterService(db, false, 100)
	routed, approved, rejected, manual := runRouter(t, svc, 10)

	if routed != 1 || approved != 0 || rejected != 1 || manual != 0 {
		t.Fatalf("router counted routed=%d approved=%d rejected=%d manual=%d; want 1 rejected", routed, approved, rejected, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.Status != models.VerificationRejected {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationRejected)
	}
	if got.ReviewPriority != models.ReviewPriorityAutoReject {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityAutoReject)
	}
	if got.RejectedAt == nil {
		t.Error("rejected_at = nil, want set")
	}
	if got.RoutingState != models.RoutingRouted {
		t.Errorf("routing_state = %q, want %q", got.RoutingState, models.RoutingRouted)
	}
	if countRouterAudit(t, db, rec.ID, routerActionType) != 1 {
		t.Error("one KYC_ROUTED audit row expected for the rejection")
	}
}

func TestRouterDoesNotRejectOnFaceAlone(t *testing.T) {
	db := setupEscalationTestDB(t)

	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:    models.ReviewPriorityAutoReject,
		faceSim:       0.05, // face says "different person"...
		isLive:        true, // ...but liveness is clean: no hard signal
		livenessScore: 0.96,
	})

	svc := setupRouterService(db, false, 100)
	_, approved, rejected, manual := runRouter(t, svc, 10)

	if approved != 0 || rejected != 0 || manual != 1 {
		t.Fatalf("router counted approved=%d rejected=%d manual=%d; want 0/0/1", approved, rejected, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q (face alone is never a hard reject)", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationPending)
	}
	if got.RejectedAt != nil {
		t.Errorf("rejected_at = %v, want nil", got.RejectedAt)
	}
}

// ---------------------------------------------------------------------------
// Rollout cohort (deterministic HMAC bucketing, ChatGPT P0 #4)
// ---------------------------------------------------------------------------

func TestRouterCohortDeterministic(t *testing.T) {
	db := setupEscalationTestDB(t)
	_ = db

	userID := uuid.New()

	first := kycCohortBucket(userID)
	for i := 0; i < 5; i++ {
		if got := kycCohortBucket(userID); got != first {
			t.Fatalf("cohort bucket changed across calls: %d vs %d", got, first)
		}
	}

	// shouldEnforceShadow is equally stable for this user.
	svc := setupRouterService(nil, false, 50)
	rec := &models.VerificationRecord{UserID: userID}
	want := svc.shouldEnforceShadow(rec)
	for i := 0; i < 3; i++ {
		if got := svc.shouldEnforceShadow(rec); got != want {
			t.Fatal("shouldEnforceShadow flipped for the same user across calls")
		}
	}
}

func TestRouterCohortZeroForcesShadow(t *testing.T) {
	db := setupEscalationTestDB(t)

	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:    models.ReviewPriorityAutoEligible,
		faceSim:       0.90,
		isLive:        true,
		livenessScore: 0.96,
	})

	// shadow off + cohort 0 => nobody is released: shadow is enforced anyway.
	svc := setupRouterService(db, false /* shadow off */, 0)
	if !svc.shouldEnforceShadow(&models.VerificationRecord{UserID: rec.UserID}) {
		t.Fatal("cohort_pct=0 must enforce shadow even with shadow_mode=false")
	}

	_, approved, _, manual := runRouter(t, svc, 10)
	if approved != 0 || manual != 1 {
		t.Fatalf("router counted approved=%d manual=%d; want 0 approved, 1 manual", approved, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.DecisionSource != routerSourceShadow || !got.ShadowModeApplied {
		t.Errorf("decision_source=%q shadow_mode_applied=%v, want shadow/true", got.DecisionSource, got.ShadowModeApplied)
	}
}

// ---------------------------------------------------------------------------
// Concurrency: two router runs must never double-route (ChatGPT P0 #7)
// ---------------------------------------------------------------------------

func TestRouterConcurrentRunsDoNotDoubleRoute(t *testing.T) {
	db := setupEscalationTestDB(t)

	const recordCount = 3
	ids := make([]uuid.UUID, recordCount)
	for i := range ids {
		rec := createRouterTestRecord(t, db, routerRecordOpts{
			aiPriority:    models.ReviewPriorityAutoEligible,
			faceSim:       0.90,
			isLive:        true,
			livenessScore: 0.96,
		})
		ids[i] = rec.ID
	}

	svc := setupRouterService(db, false, 100)

	var wg sync.WaitGroup
	const runs = 5
	totalApproved := 0
	var mu sync.Mutex
	for i := 0; i < runs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, approved, _, _, err := svc.RouteMLCompleted(10)
			if err != nil {
				t.Errorf("concurrent RouteMLCompleted: %v", err)
				return
			}
			mu.Lock()
			totalApproved += approved
			mu.Unlock()
		}()
	}
	wg.Wait()

	if totalApproved != recordCount {
		t.Errorf("total approvals across %d concurrent runs = %d, want %d (no duplicate routing)", runs, totalApproved, recordCount)
	}

	for _, id := range ids {
		got := reloadRouterRecord(t, db, id)
		if got.Status != models.VerificationApproved {
			t.Errorf("record %s status = %q, want %q", id, got.Status, models.VerificationApproved)
		}
		if got.RoutingState != models.RoutingRouted {
			t.Errorf("record %s routing_state = %q, want %q", id, got.RoutingState, models.RoutingRouted)
		}
		if n := countRouterAudit(t, db, id, routerActionType); n != 1 {
			t.Errorf("record %s has %d KYC_ROUTED audit rows, want exactly 1", id, n)
		}
	}
}

// ---------------------------------------------------------------------------
// Stale escalator: stuck-in-ML pass + idempotency (ChatGPT P0 #1 / Phase 1.5 Gap 1)
// ---------------------------------------------------------------------------

func TestStaleEscalatorCatchesQueuedRecords(t *testing.T) {
	db := setupEscalationTestDB(t)

	submittedAt := time.Now().Add(-25 * time.Hour)
	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:  models.ReviewPriorityManual,
		mlStatus:    models.MLStatusQueued,
		submittedAt: &submittedAt,
	})

	svc := &VerificationService{
		verificationRecordsRepo: repository.NewVerificationRecordsRepository(),
	}
	svc.ProcessStalePendingEscalations()

	got := reloadRouterRecord(t, db, rec.ID)
	if got.MLStatus != models.MLStatusFailed {
		t.Errorf("ml_status = %q, want %q", got.MLStatus, models.MLStatusFailed)
	}
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.SystemEscalatedAt == nil {
		t.Error("system_escalated_at = nil, want set")
	}
	if got.RoutingState != models.RoutingRouted {
		t.Errorf("routing_state = %q, want %q", got.RoutingState, models.RoutingRouted)
	}
	if got.DecisionSource != "system_stale" {
		t.Errorf("decision_source = %q, want %q", got.DecisionSource, "system_stale")
	}
	if countRouterAudit(t, db, rec.ID, "SYSTEM_ESCALATED_STALE_24H") != 1 {
		t.Error("one SYSTEM_ESCALATED_STALE_24H audit row expected")
	}
}

func TestStaleEscalatorIdempotent(t *testing.T) {
	db := setupEscalationTestDB(t)

	submittedAt := time.Now().Add(-25 * time.Hour)
	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:  models.ReviewPriorityManual,
		mlStatus:    models.MLStatusQueued,
		submittedAt: &submittedAt,
	})

	svc := &VerificationService{
		verificationRecordsRepo: repository.NewVerificationRecordsRepository(),
	}
	svc.ProcessStalePendingEscalations()

	first := reloadRouterRecord(t, db, rec.ID)
	if first.SystemEscalatedAt == nil {
		t.Fatal("first run did not escalate")
	}
	firstEscalatedAt := *first.SystemEscalatedAt

	svc.ProcessStalePendingEscalations()

	second := reloadRouterRecord(t, db, rec.ID)
	if second.SystemEscalatedAt == nil {
		t.Fatal("second run cleared system_escalated_at")
	}
	if !second.SystemEscalatedAt.Equal(firstEscalatedAt) {
		t.Errorf("system_escalated_at changed on the second run: %v -> %v", firstEscalatedAt, *second.SystemEscalatedAt)
	}
	if countRouterAudit(t, db, rec.ID, "SYSTEM_ESCALATED_STALE_24H") != 1 {
		t.Error("audit rows changed on the second run (not idempotent)")
	}
}

// ---------------------------------------------------------------------------
// Worker guard: a late completion cannot overwrite an escalated record
// (the stale-escalation race, ChatGPT P0 #1)
// ---------------------------------------------------------------------------

func TestWorkerCannotOverwriteEscalated(t *testing.T) {
	db := setupEscalationTestDB(t)

	submittedAt := time.Now().Add(-30 * time.Hour)
	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:      models.ReviewPriorityManual,
		mlStatus:        models.MLStatusQueued,
		submittedAt:     &submittedAt,
		systemEscalated: true, // escalator already claimed this record
	})
	before := reloadRouterRecord(t, db, rec.ID)

	// SQL mirrors the WHERE clause of kyc-ml-service/worker.py save_results.
	// If these guards are dropped on the Python side, a late completion
	// overwrites an escalation — the exact race the P0 review flagged.
	savedAt := time.Now()
	result := db.Exec(`
		UPDATE verification_records
		   SET ml_status = 'ml_completed',
		       ml_completed_at = ?,
		       ai_recommended_priority = 'auto_eligible',
		       review_priority = 'auto_eligible',
		       routing_state = 'pending',
		       updated_at = ?
		 WHERE id = ?
		   AND ml_status <> 'ml_completed'
		   AND system_escalated_at IS NULL
		   AND routing_state = 'pending'
	`, savedAt, savedAt, rec.ID)
	if result.Error != nil {
		t.Fatalf("worker-style update failed: %v", result.Error)
	}
	if result.RowsAffected != 0 {
		t.Fatalf("worker-style update affected %d row(s), want 0 (escalated record is terminal)", result.RowsAffected)
	}

	after := reloadRouterRecord(t, db, rec.ID)
	if after.MLStatus != models.MLStatusQueued {
		t.Errorf("ml_status changed to %q, want %q (unchanged)", after.MLStatus, models.MLStatusQueued)
	}
	if after.ReviewPriority != before.ReviewPriority {
		t.Errorf("review_priority changed: %q -> %q", before.ReviewPriority, after.ReviewPriority)
	}
	if after.SystemEscalatedAt == nil || !after.SystemEscalatedAt.Equal(*before.SystemEscalatedAt) {
		t.Error("system_escalated_at changed")
	}

	// Same race via mark_processing: also refuses (guard ml_status IN queued/processing
	// AND system_escalated_at IS NULL).
	moved := db.Exec(`
		UPDATE verification_records
		   SET ml_status = 'processing', updated_at = ?
		 WHERE id = ?
		   AND ml_status IN ('queued', 'processing')
		   AND system_escalated_at IS NULL
	`, savedAt, rec.ID)
	if moved.Error != nil {
		t.Fatalf("mark_processing-style update failed: %v", moved.Error)
	}
	if moved.RowsAffected != 0 {
		t.Errorf("mark_processing-style update affected %d row(s), want 0", moved.RowsAffected)
	}
}
