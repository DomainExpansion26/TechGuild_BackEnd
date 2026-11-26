package services

import (
	"encoding/json"
	"fmt"
	"strings"
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

		// Phase B: the document-in-selfie gate replaced the aadhaar floor.
		// Same defaults as env.go so the tests exercise production values.
		DocInSelfieApproveThreshold: 75,
		DocInSelfieManualThreshold:  25,
		DocInSelfieRejectOnMismatch: true,
		RequireQRVerified:           true,
	}
}

// routerEvidenceJSONWithDocInSelfie is the Phase B-aware form: decision/score
// describe the worker's document-in-selfie verdict so tests can drive the gate.
func routerEvidenceJSONWithDocInSelfie(
	faceSim float64,
	aadhaarSim *float64,
	divergence bool,
	docInSelfieDecision string,
	docInSelfieScore *int,
	docInSelfieMatch bool,
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
	if docInSelfieDecision != "" || docInSelfieScore != nil {
		faceJSON["document_in_selfie_decision"] = docInSelfieDecision
		faceJSON["document_in_selfie_match"] = docInSelfieMatch
		if docInSelfieScore == nil {
			faceJSON["document_in_selfie_score"] = nil
		} else {
			faceJSON["document_in_selfie_score"] = *docInSelfieScore
		}
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

// routerApproveGuardArgCount is the ONE place the approval guard's placeholder
// count is written down. routerApproveGuardSQL must have exactly this many ?
// markers per dialect, every call site must pass exactly this many arguments,
// and assertRouterApproveGuardArgs enforces both at runtime — so the statement
// and its bindings can never silently drift apart again (P0-1: a stale extra
// argument shifted every threshold binding by one position).
const routerApproveGuardArgCount = 8

// assertRouterApproveGuardArgs fails the test when a call site's argument list
// does not match routerApproveGuardArgCount.
func assertRouterApproveGuardArgs(t *testing.T, args []interface{}) {
	t.Helper()
	if len(args) != routerApproveGuardArgCount {
		t.Fatalf("approve-guard arg list has %d entries, want %d (the statement has exactly %d ? placeholders)",
			len(args), routerApproveGuardArgCount, routerApproveGuardArgCount)
	}
}

// countSQLPlaceholders counts the ? bind markers in a statement that are REAL
// placeholders — i.e. outside string literals and SQL comments. Counting raw
// '?' bytes would also count one inside a quoted literal such as
// status = 'ready?' and report a false mismatch.
func countSQLPlaceholders(sql string) int {
	count := 0
	inSingle, inDouble, inLineComment, inBlockComment := false, false, false, false
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		switch {
		case inLineComment:
			if ch == '\n' {
				inLineComment = false
			}
		case inBlockComment:
			if ch == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				inBlockComment = false
				i++
			}
		case inSingle:
			if ch == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					i++ // doubled quote: '' stays inside the literal
				} else {
					inSingle = false
				}
			}
		case inDouble:
			if ch == '"' {
				inDouble = false
			}
		default:
			switch ch {
			case '\'':
				inSingle = true
			case '"':
				inDouble = true
			case '?':
				count++
			case '-':
				if i+1 < len(sql) && sql[i+1] == '-' {
					inLineComment = true
					i++
				}
			case '/':
				if i+1 < len(sql) && sql[i+1] == '*' {
					inBlockComment = true
					i++
				}
			}
		}
	}
	return count
}

// nthRealPlaceholder returns the byte offset of the n-th (0-based) ? marker that
// is a real bind placeholder (outside string literals and comments), or -1.
// It is the positional partner of countSQLPlaceholders: the placeholder
// regression test marks up the statement with it so the ORDER of the bindings
// can be read back from the failing test output, not only the count.
func nthRealPlaceholder(sql string, n int) int {
	seen := 0
	inSingle, inDouble, inLineComment, inBlockComment := false, false, false, false
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		switch {
		case inLineComment:
			if ch == '\n' {
				inLineComment = false
			}
		case inBlockComment:
			if ch == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				inBlockComment = false
				i++
			}
		case inSingle:
			if ch == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					i++
				} else {
					inSingle = false
				}
			}
		case inDouble:
			if ch == '"' {
				inDouble = false
			}
		default:
			switch ch {
			case '\'':
				inSingle = true
			case '"':
				inDouble = true
			case '?':
				if seen == n {
					return i
				}
				seen++
			case '-':
				if i+1 < len(sql) && sql[i+1] == '-' {
					inLineComment = true
					i++
				}
			case '/':
				if i+1 < len(sql) && sql[i+1] == '*' {
					inBlockComment = true
					i++
				}
			}
		}
	}
	return -1
}

// ptrRouterFloat is a small helper for routerRecordOpts.aadhaarSim (nil in the
// struct means "use the healthy default", so the value needs explicit boxing).
func ptrRouterFloat(v float64) *float64 { return &v }

// ptrRouterInt boxes a document_in_selfie_score so 0 is distinguishable from
// "worker never scored it".
func ptrRouterInt(v int) *int { return &v }

// Phase B verdict constants (mirror the worker's strings, not shared code).
const (
	docInSelfieApprove = "approve"
	docInSelfieManual  = "manual"
	docInSelfieReject  = "reject"
)

// routerRecordOpts builds a record whose shape matches the worker's output.
type routerRecordOpts struct {
	// verificationType is the record's KYC type. The ZERO VALUE ("" ) is
	// individual so every pre-Phase-E test keeps its exact setup; set
	// models.VerificationBusiness for a business record.
	verificationType models.VerificationType

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
	//
	// Phase B: these values are still WRITTEN to the fixture (the struct keeps
	// the fields) but they are no longer what the router gates on — only the
	// divergence flag still blocks approval.
	aadhaarSim *float64
	aadhaarNil bool
	divergence bool

	// Phase B document-in-selfie verdict. The zero value is "not set", which
	// createRouterTestRecord turns into the healthy verdict (decision
	// "approve", score 100, match true) so existing tests stay green. Set
	// docInSelfieDecision for manual/reject cases, and docInSelfieScore when
	// the score itself is under test. docInSelfieScoreNil writes an explicit
	// null score (worker could not score the match).
	docInSelfieDecision string
	docInSelfieScore    *int
	docInSelfieScoreNil bool
	docInSelfieMatch    bool
	docInSelfieMatchSet bool

	// docValidationOverall overrides ocr_ocr_result.document_validation.overall
	// (default "verified") so the Phase B QR-requirement tests can drive
	// THRESHOLD_REQUIRE_QR. Any non-"verified" value models a failed/pending
	// UIDAI QR check.
	docValidationOverall string

	// govtIDHash, when set, is stored on the record so approve side effects can
	// register a dedup claim (individual KYC).
	govtIDHash string

	// ---- Phase E business KYC ----
	//
	// businessDocsOmitAll documents the intent of a caller that writes a
	// face_match_result with NO business verdict at all (faceMatchOverride set to
	// the keyless shape): the Go gates and the SQL guard must both read that as
	// NOT approved, never fail open.
	//
	// faceMatchOverride replaces the whole face_match_result column when set, so
	// a test can write a business record with NO document_in_selfie / liveness
	// fields (exactly what the worker writes for business). livenessOverride
	// does the same for liveness_result.
	faceMatchOverride   datatypes.JSON
	livenessOverride    datatypes.JSON
	businessDocsOmitAll bool
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

	// Phase B defaults: a healthy worker verdict unless the test overrides it.
	docDecision := opts.docInSelfieDecision
	if docDecision == "" {
		docDecision = docInSelfieApprove
	}
	docScore := opts.docInSelfieScore
	if docScore == nil && !opts.docInSelfieScoreNil {
		docScore = ptrRouterInt(100)
	}
	docMatch := opts.docInSelfieMatch
	if !opts.docInSelfieMatchSet {
		docMatch = true
	}

	face, live, _ := routerEvidenceJSONWithDocInSelfie(
		opts.faceSim, faceAadhaar, opts.divergence,
		docDecision, docScore, docMatch,
		opts.isLive, opts.livenessScore,
	)
	if opts.faceMatchOverride != nil {
		face = opts.faceMatchOverride
	}
	if opts.livenessOverride != nil {
		live = opts.livenessOverride
	}
	docOverall := opts.docValidationOverall
	if docOverall == "" {
		docOverall = "verified"
	}
	ocr, _ := json.Marshal(map[string]interface{}{
		"ocr_confidence":      0.95,
		"text":                "NAME: ROUTER TEST",
		"document_validation": map[string]interface{}{"overall": docOverall},
	})

	// Phase E: the zero value of verificationType is individual, so every test
	// written before business KYC existed keeps its exact record shape.
	recType := opts.verificationType
	if recType == "" {
		recType = models.VerificationIndividual
	}

	rec := models.VerificationRecord{
		UserID:                user.ID,
		Type:                  recType,
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
	if opts.govtIDHash != "" {
		rec.GovtIDHash = opts.govtIDHash
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

// businessFaceMatchJSON builds the face_match_result a business record carries:
// NO document_in_selfie_*, NO second_reference_divergence, and NO face_similarity
// (a business KYC has no selfie at all) — only the business document verdict in
// the nested and/or flat shape the worker writes. Empty/dash values omit the key
// entirely, which is the "worker never wrote that key" case the gates must treat
// as NOT approved.
func businessFaceMatchJSON(nestedDecision string, nestedScore *int, flatDecision string, flatScore *int) datatypes.JSON {
	faceJSON := map[string]interface{}{
		"business_docs_match": true,
	}
	if nestedDecision != "" || nestedScore != nil {
		nested := map[string]interface{}{}
		if nestedDecision != "" {
			nested["decision"] = nestedDecision
		}
		if nestedScore != nil {
			nested["score"] = *nestedScore
		}
		faceJSON["business_docs"] = nested
	}
	if flatDecision != "" || flatScore != nil {
		faceJSON["business_docs_decision"] = flatDecision
		if flatScore != nil {
			faceJSON["business_docs_score"] = *flatScore
		}
	}
	out, _ := json.Marshal(faceJSON)
	return out
}

// healthyBusinessOpts is the "everything is green" business baseline: the ML
// model version is set by createRouterTestRecord, the OCR confidence is 0.95 and
// document_validation.overall is "verified", and the worker wrote BOTH shapes of
// the document verdict as an approval.
func healthyBusinessOpts() routerRecordOpts {
	return routerRecordOpts{
		verificationType:  models.VerificationBusiness,
		aiPriority:        models.ReviewPriorityAutoEligible,
		faceMatchOverride: businessFaceMatchJSON("approve", ptrRouterInt(100), "approve", ptrRouterInt(100)),
		livenessOverride:  datatypes.JSON([]byte("{}")), // business has no liveness_result at all
	}
}

func setupRouterService(_ *gorm.DB, shadowMode bool, cohortPct int) *VerificationService {
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

// TestRouterEvidenceIncompleteRoutesManual drives the ACTUAL Phase B evidence
// gate: the worker never scored the document-in-selfie match
// (document_in_selfie_score is absent). The AI recommendation must not carry.
//
// PORT NOTE (this test's expectation was stale, not the router): it used to
// signal "weak evidence" through face_similarity = 0.10, but Phase B removed the
// face_similarity predicate from evidenceComplete (FACE_MATCH_ENABLED=false
// makes that metric permanently nil), so the fixture's healthy
// document_in_selfie_decision=approve / score=100 legitimately approved the
// record. The intent — incomplete evidence must route to manual review — is
// unchanged; only the field that carries the incompleteness is.
func TestRouterEvidenceIncompleteRoutesManual(t *testing.T) {
	db := setupEscalationTestDB(t)

	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:          models.ReviewPriorityAutoEligible,
		faceSim:             0.10, // still weak: just no longer a Phase B gate
		docInSelfieDecision: docInSelfieApprove,
		docInSelfieScoreNil: true, // worker never scored the match -> incomplete
		isLive:              true,
		livenessScore:       0.96,
	})

	svc := setupRouterService(db, false, 100)

	if svc.evidenceComplete(&rec) {
		t.Fatal("evidenceComplete = true with an unscored document-in-selfie, want false")
	}

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
		govtIDHash:    "router-auto-approve-hash",
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

	// Approve side effects: the router's SQL approve must also claim the dedup
	// hash, grant the KYC bonus and activate the account (HIGH-severity fix).
	var dedup models.GovtIDDedup
	if err := db.Where("govt_id_hash = ?", "router-auto-approve-hash").First(&dedup).Error; err != nil {
		t.Fatalf("router auto-approve did not register the government-ID dedup hash: %v", err)
	}
	if dedup.UserID != rec.UserID {
		t.Errorf("dedup user = %s, want %s", dedup.UserID, rec.UserID)
	}
	var owner models.User
	if err := db.First(&owner, "id = ?", rec.UserID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if owner.Points < 40 {
		t.Errorf("user points = %d after router auto-approve, want >= 40", owner.Points)
	}
}

func TestRouterGuardRejectsWeakEvidence(t *testing.T) {
	db := setupEscalationTestDB(t)
	dialect := db.Dialector.Name()

	// Weak evidence carried by the field the guard actually reads: the worker's
	// document verdict is "manual" and its score is far below the floor.
	//
	// PORT NOTE (stale expectation, not a router bug): this test signalled
	// weakness with face_similarity = 0.10, which Phase B no longer gates on, so
	// the fixture's healthy document_in_selfie approve/100 approved the record.
	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:          models.ReviewPriorityAutoEligible,
		faceSim:             0.10,
		docInSelfieDecision: docInSelfieManual,
		docInSelfieScore:    ptrRouterInt(40), // manual band, below the 75 floor
		isLive:              true,
		livenessScore:       0.96,
	})

	policy := routerTestPolicy(false, 100)

	// Direct SQL: the approval guard is invoked standalone — no Go-side gate —
	// and must touch zero rows because the evidence is below threshold.
	//
	// The argument inventory below is EXACTLY routerApproveGuardArgCount (8),
	// positionally matching the guard's ? placeholders. There is no
	// face_auto_pass argument: passing the policy's FaceAutoPass here (as this
	// test used to) shifted every binding after it by one slot, so the guard
	// silently compared the wrong values and neither this test nor the router
	// could see it.
	args := []interface{}{
		true,                 // 1 shadow_mode_applied (any value; the WHERE short-circuits)
		"ai",                 // 2 decision_source
		policy.PolicyVersion, // 3 policy_version
		rec.ID,               // 4 id
		// 5 document-in-selfie score floor (individual records only)
		policy.DocInSelfieApproveThreshold,
		policy.RequireQRVerified, // 6 QR switch (both types)
		policy.LivenessPass,      // 7 liveness floor (individual records only)
		policy.OCRPass,           // 8 OCR floor (both types)
	}
	assertRouterApproveGuardArgs(t, args)
	result := db.Exec(routerApproveGuardSQL(dialect), args...)
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
// Phase B: document-in-selfie gate + divergence guard (defense in depth)
//
// These tests previously asserted the Phase 5 Aadhaar gate (floor + nil-block).
// Phase B REPLACED that gate: with the worker running FACE_MATCH_ENABLED=false,
// aadhaar_similarity is always null, so the old assertion could only ever
// describe a permanent block. The gate is now document_in_selfie_decision /
// document_in_selfie_score. The divergence guard is unchanged.
// ---------------------------------------------------------------------------

func TestRouterRequiresAadhaarFloor(t *testing.T) {
	db := setupEscalationTestDB(t)

	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority: models.ReviewPriorityAutoEligible,
		faceSim:    0.70, // face evidence is strong...
		// ...but the document-in-selfie match scored below the approve floor.
		docInSelfieDecision: docInSelfieApprove,
		docInSelfieScore:    ptrRouterInt(50),
		isLive:              true,
		livenessScore:       0.96,
	})

	svc := setupRouterService(db, false, 100)

	// Direct policy assertions (no DB write): the pure functions must reject.
	if svc.evidenceComplete(&rec) {
		t.Error("evidenceComplete = true with document_in_selfie_score below 75, want false")
	}
	if dec := svc.decideRouting(&rec); dec.Effective != models.ReviewPriorityManual || dec.Approve {
		t.Errorf("decideRouting = %+v, want manual review (doc-in-selfie score floor)", dec)
	}

	_, approved, _, manual := runRouter(t, svc, 10)

	if approved != 0 || manual != 1 {
		t.Fatalf("router counted approved=%d manual=%d; want 0 approved, 1 manual", approved, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q (doc-in-selfie score below floor blocks approval)", got.ReviewPriority, models.ReviewPriorityManual)
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
		aiPriority: models.ReviewPriorityAutoEligible,
		faceSim:    0.70, // weaker than the Aadhaar anchor...
		// ...but the document held in the selfie matches the uploaded document
		// at a strong score, which is the Phase B anchor.
		docInSelfieDecision: docInSelfieApprove,
		docInSelfieScore:    ptrRouterInt(100),
		isLive:              true,
		livenessScore:       0.96,
	})

	svc := setupRouterService(db, false, 100)

	// Direct policy assertions: a strong document-in-selfie match clears the gate.
	if !svc.evidenceComplete(&rec) {
		t.Error("evidenceComplete = false with document_in_selfie approve/100, want true")
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
		t.Errorf("status = %q, want %q (strong doc-in-selfie + no divergence can approve)", got.Status, models.VerificationApproved)
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
		aiPriority: models.ReviewPriorityAutoEligible,
		faceSim:    0.70,
		// Phase B: a nil document_in_selfie_score means the worker never
		// scored the match (e.g. no document visible in the selfie) — a
		// missing-verdict case, never an approval.
		docInSelfieDecision: docInSelfieApprove,
		docInSelfieScoreNil: true,
		isLive:              true,
		livenessScore:       0.96,
	})

	svc := setupRouterService(db, false, 100)

	// Direct policy assertions: a nil document_in_selfie_score blocks approval.
	if svc.evidenceComplete(&rec) {
		t.Error("evidenceComplete = true with nil document_in_selfie_score, want false")
	}
	if dec := svc.decideRouting(&rec); dec.Effective != models.ReviewPriorityManual || dec.Approve {
		t.Errorf("decideRouting = %+v, want manual review (nil doc-in-selfie score)", dec)
	}

	_, approved, _, manual := runRouter(t, svc, 10)

	if approved != 0 || manual != 1 {
		t.Fatalf("router counted approved=%d manual=%d; want 0 approved, 1 manual", approved, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q (nil doc-in-selfie score blocks auto)", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationPending)
	}
}

// TestRouterIgnoresNilAadhaarSimilarityWhenFaceMatchDisabled is the Phase B
// regression guard for the bug this phase fixes: the worker runs with
// FACE_MATCH_ENABLED=false, so aadhaar_similarity is ALWAYS null. A record with
// a healthy document-in-selfie verdict must still auto-approve — the removed
// Phase 5 nil-gate would have forced every such record into manual review.
func TestRouterIgnoresNilAadhaarSimilarityWhenFaceMatchDisabled(t *testing.T) {
	db := setupEscalationTestDB(t)

	rec := createRouterTestRecord(t, db, routerRecordOpts{
		aiPriority:    models.ReviewPriorityAutoEligible,
		faceSim:       0.70,
		aadhaarNil:    true, // exactly what a disabled face match writes
		isLive:        true,
		livenessScore: 0.96,
		// document-in-selfie defaults to the healthy approve/100 verdict.
		govtIDHash: "router-nil-aadhaar-approve-hash",
	})

	svc := setupRouterService(db, false, 100)

	if !svc.evidenceComplete(&rec) {
		t.Fatal("evidenceComplete = false with healthy document-in-selfie but nil aadhaar_similarity; the Phase 5 nil-gate is still blocking approvals")
	}

	routed, approved, rejected, manual := runRouter(t, svc, 10)
	if routed != 1 || approved != 1 || rejected != 0 || manual != 0 {
		t.Fatalf("router counted routed=%d approved=%d rejected=%d manual=%d; want 1 approved", routed, approved, rejected, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.Status != models.VerificationApproved {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationApproved)
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
// Phase B: document-in-selfie gate
//
// The worker OCRs the document held in the selfie, compares it against the
// uploaded document, and writes document_in_selfie_decision / _score / _match
// into face_match_result. The router gates auto-approval on that verdict.
// ---------------------------------------------------------------------------

// setupRouterServiceWithPolicy is setupRouterService with a caller-adjusted
// policy, so a test can flip a Phase B knob (QR requirement, reject-on-mismatch)
// without rebuilding the whole service.
func setupRouterServiceWithPolicy(_ *gorm.DB, mutate func(p *config.KYCPolicy)) *VerificationService {
	svc := &VerificationService{
		verificationRecordsRepo: repository.NewVerificationRecordsRepository(),
	}
	policy := routerTestPolicy(false /* shadow off */, 100 /* everyone in cohort */)
	if mutate != nil {
		mutate(&policy)
	}
	svc.SetKYCPolicy(policy)
	return svc
}

// healthyDocInSelfieOpts is the "everything is green" baseline the Phase B
// tests then bend one field at a time.
func healthyDocInSelfieOpts() routerRecordOpts {
	return routerRecordOpts{
		aiPriority:          models.ReviewPriorityAutoEligible,
		faceSim:             0.90, // >= FaceAutoPass
		docInSelfieDecision: docInSelfieApprove,
		docInSelfieScore:    ptrRouterInt(100),
		docInSelfieMatch:    true,
		docInSelfieMatchSet: true,
		isLive:              true,
		livenessScore:       0.96,
	}
}

func TestRouterDocInSelfieApproveGatesApproval(t *testing.T) {
	db := setupEscalationTestDB(t)

	opts := healthyDocInSelfieOpts()
	opts.govtIDHash = "router-doc-in-selfie-approve-hash"
	rec := createRouterTestRecord(t, db, opts)

	svc := setupRouterService(db, false, 100)

	if !svc.evidenceComplete(&rec) {
		t.Fatal("evidenceComplete = false with decision=approve and score=100, want true")
	}
	dec := svc.decideRouting(&rec)
	if !dec.Approve || dec.Effective != models.ReviewPriorityAutoEligible {
		t.Fatalf("decideRouting = %+v, want approve/auto_eligible", dec)
	}

	routed, approved, rejected, manual := runRouter(t, svc, 10)
	if routed != 1 || approved != 1 || rejected != 0 || manual != 0 {
		t.Fatalf("router counted routed=%d approved=%d rejected=%d manual=%d; want 1 approved", routed, approved, rejected, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.Status != models.VerificationApproved {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationApproved)
	}
	if got.DecisionSource != routerSourceAI {
		t.Errorf("decision_source = %q, want %q", got.DecisionSource, routerSourceAI)
	}
}

func TestRouterDocInSelfieScoreBelowThresholdBlocks(t *testing.T) {
	db := setupEscalationTestDB(t)

	opts := healthyDocInSelfieOpts()
	opts.docInSelfieScore = ptrRouterInt(50) // below the 75 threshold
	rec := createRouterTestRecord(t, db, opts)

	svc := setupRouterService(db, false, 100)

	if svc.evidenceComplete(&rec) {
		t.Error("evidenceComplete = true with decision=approve but score=50 (< 75), want false")
	}
	dec := svc.decideRouting(&rec)
	if dec.Effective != models.ReviewPriorityManual || dec.Approve {
		t.Fatalf("decideRouting = %+v, want manual review", dec)
	}

	_, approved, rejected, manual := runRouter(t, svc, 10)
	if approved != 0 || rejected != 0 || manual != 1 {
		t.Fatalf("router counted approved=%d rejected=%d manual=%d; want 0/0/1", approved, rejected, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationPending)
	}
}

func TestRouterDocInSelfieManualBlocksApprove(t *testing.T) {
	db := setupEscalationTestDB(t)

	opts := healthyDocInSelfieOpts()
	opts.docInSelfieDecision = docInSelfieManual
	opts.docInSelfieScore = ptrRouterInt(40) // inside the manual band
	rec := createRouterTestRecord(t, db, opts)

	svc := setupRouterService(db, false, 100)

	if svc.evidenceComplete(&rec) {
		t.Error("evidenceComplete = true with decision=manual, want false")
	}

	// A manual verdict is never a hard signal: it must route to manual review,
	// not reject.
	dec := svc.decideRouting(&rec)
	if dec.Reject {
		t.Errorf("decideRouting rejected on a manual verdict (Reject=%v); want manual only", dec.Reject)
	}
	if dec.Effective != models.ReviewPriorityManual {
		t.Errorf("decideRouting effective = %q, want %q", dec.Effective, models.ReviewPriorityManual)
	}

	_, approved, rejected, manual := runRouter(t, svc, 10)
	if approved != 0 || rejected != 0 || manual != 1 {
		t.Fatalf("router counted approved=%d rejected=%d manual=%d; want 0/0/1", approved, rejected, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityManual)
	}
	if got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q (a manual verdict must never reject)", got.Status, models.VerificationPending)
	}
	if got.RejectedAt != nil {
		t.Errorf("rejected_at = %v, want nil", got.RejectedAt)
	}
}

func TestRouterDocInSelfieRejectTriggersAutoReject(t *testing.T) {
	db := setupEscalationTestDB(t)

	opts := healthyDocInSelfieOpts()
	opts.docInSelfieDecision = docInSelfieReject
	opts.docInSelfieScore = ptrRouterInt(-100) // worker's hard-mismatch sentinel
	opts.docInSelfieMatch = false
	opts.isLive = false // liveness would also reject; the mismatch path must own it
	opts.livenessScore = 0.10
	rec := createRouterTestRecord(t, db, opts)

	svc := setupRouterService(db, false, 100)

	dec := svc.decideRouting(&rec)
	if !dec.Reject {
		t.Fatalf("decideRouting = %+v, want Reject=true", dec)
	}
	if dec.Effective != models.ReviewPriorityAutoReject {
		t.Errorf("effective = %q, want %q", dec.Effective, models.ReviewPriorityAutoReject)
	}
	if dec.Source != routerSourceDocInSelfie {
		t.Errorf("source = %q, want %q", dec.Source, routerSourceDocInSelfie)
	}
	if dec.Reason != routerReasonAadhaarMismatch {
		t.Errorf("reason = %q, want %q", dec.Reason, routerReasonAadhaarMismatch)
	}
	if svc.evidenceComplete(&rec) {
		t.Error("evidenceComplete = true on a hard mismatch, want false")
	}

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
	if got.DecisionSource != routerSourceDocInSelfie {
		t.Errorf("decision_source = %q, want %q", got.DecisionSource, routerSourceDocInSelfie)
	}
	if got.RejectedAt == nil {
		t.Error("rejected_at = nil, want set")
	}
}

func TestRouterRejectOnMismatchDisabledRoutesManual(t *testing.T) {
	db := setupEscalationTestDB(t)

	opts := healthyDocInSelfieOpts()
	opts.docInSelfieDecision = docInSelfieReject
	opts.docInSelfieScore = ptrRouterInt(-100)
	opts.docInSelfieMatch = false
	rec := createRouterTestRecord(t, db, opts)

	// Operator opted out of hard rejection: a mis-OCR must not reject a real user.
	svc := setupRouterServiceWithPolicy(db, func(p *config.KYCPolicy) {
		p.DocInSelfieRejectOnMismatch = false
	})

	dec := svc.decideRouting(&rec)
	if dec.Reject {
		t.Errorf("decideRouting rejected with DocInSelfieRejectOnMismatch=false: %+v", dec)
	}
	if dec.Effective != models.ReviewPriorityManual {
		t.Errorf("effective = %q, want %q", dec.Effective, models.ReviewPriorityManual)
	}

	_, approved, rejected, manual := runRouter(t, svc, 10)
	if approved != 0 || rejected != 0 || manual != 1 {
		t.Fatalf("router counted approved=%d rejected=%d manual=%d; want 0/0/1 (manual, NOT reject)", approved, rejected, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationPending)
	}
	if got.RejectedAt != nil {
		t.Errorf("rejected_at = %v, want nil", got.RejectedAt)
	}
	if got.ReviewPriority != models.ReviewPriorityManual {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityManual)
	}
}

func TestRouterQrRequiredBlocksUnverified(t *testing.T) {
	db := setupEscalationTestDB(t)

	opts := healthyDocInSelfieOpts()
	opts.docValidationOverall = "failed" // UIDAI QR check did not verify
	rec := createRouterTestRecord(t, db, opts)

	svc := setupRouterService(db, false, 100) // RequireQRVerified defaults true

	if svc.evidenceComplete(&rec) {
		t.Error("evidenceComplete = true with RequireQRVerified=true and overall=failed, want false")
	}

	_, approved, rejected, manual := runRouter(t, svc, 10)
	if approved != 0 || rejected != 0 || manual != 1 {
		t.Fatalf("router counted approved=%d rejected=%d manual=%d; want 0/0/1", approved, rejected, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationPending)
	}
}

func TestRouterQrBypassWhenConfigFalse(t *testing.T) {
	db := setupEscalationTestDB(t)

	opts := healthyDocInSelfieOpts()
	opts.docValidationOverall = "failed" // QR unavailable/failed...
	rec := createRouterTestRecord(t, db, opts)
	// ...but every other dimension is green, so the bypass must approve.

	svc := setupRouterServiceWithPolicy(db, func(p *config.KYCPolicy) {
		p.RequireQRVerified = false
	})

	if !svc.evidenceComplete(&rec) {
		t.Fatal("evidenceComplete = false with RequireQRVerified=false and all other evidence green, want true")
	}

	routed, approved, rejected, manual := runRouter(t, svc, 10)
	if routed != 1 || approved != 1 || rejected != 0 || manual != 0 {
		t.Fatalf("router counted routed=%d approved=%d rejected=%d manual=%d; want 1 approved", routed, approved, rejected, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.Status != models.VerificationApproved {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationApproved)
	}
}

// TestRouterQrBypassStillEnforcesOtherGates is the guard-rail for the bypass:
// turning the QR requirement OFF must not disable any other evidence check.
// Liveness is deliberately broken while the QR bypass is on.
func TestRouterQrBypassStillEnforcesOtherGates(t *testing.T) {
	db := setupEscalationTestDB(t)

	opts := healthyDocInSelfieOpts()
	opts.docValidationOverall = "failed" // QR would fail...
	opts.isLive = false                  // ...and liveness is broken too
	opts.livenessScore = 0.96            // (score high; is_live=false is the gate)
	rec := createRouterTestRecord(t, db, opts)

	svc := setupRouterServiceWithPolicy(db, func(p *config.KYCPolicy) {
		p.RequireQRVerified = false
	})

	if svc.evidenceComplete(&rec) {
		t.Error("evidenceComplete = true with QR bypassed but liveness failing, want false")
	}
}

func TestRouterLivenessGatesStillWork(t *testing.T) {
	db := setupEscalationTestDB(t)

	// Liveness must still block approval on its own, with a perfect
	// document-in-selfie verdict and the QR requirement satisfied.
	opts := healthyDocInSelfieOpts()
	opts.isLive = false
	opts.livenessScore = 0.96 // is_live=false is the failing condition
	rec := createRouterTestRecord(t, db, opts)

	svc := setupRouterService(db, false, 100)

	if svc.evidenceComplete(&rec) {
		t.Error("evidenceComplete = true with is_live=false, want false")
	}
	if _, approved, rejected, manual := runRouter(t, svc, 10); approved != 0 || manual != 1 {
		t.Fatalf("router counted approved=%d rejected=%d manual=%d; want 0 approved, 1 manual", approved, rejected, manual)
	}

	// And a low liveness score blocks too.
	db2 := setupEscalationTestDB(t)
	opts2 := healthyDocInSelfieOpts()
	opts2.livenessScore = 0.50 // below the 0.90 pass threshold
	rec2 := createRouterTestRecord(t, db2, opts2)
	svc2 := setupRouterService(db2, false, 100)
	if svc2.evidenceComplete(&rec2) {
		t.Error("evidenceComplete = true with liveness_score below the pass threshold, want false")
	}
}

// TestRouterApproveGuardSqlEnforcesDocInSelfie invokes the approval guard SQL
// standalone (no Go-side gate) and proves the DB predicate itself refuses a
// record whose document-in-selfie score is under the floor.
func TestRouterApproveGuardSqlEnforcesDocInSelfie(t *testing.T) {
	db := setupEscalationTestDB(t)
	dialect := db.Dialector.Name()

	opts := healthyDocInSelfieOpts()
	opts.docInSelfieScore = ptrRouterInt(50)
	rec := createRouterTestRecord(t, db, opts)

	policy := routerTestPolicy(false, 100)

	// Exactly routerApproveGuardArgCount (8) arguments — no face_auto_pass.
	args := []interface{}{
		false,                              // 1 shadow_mode_applied
		"ai",                               // 2 decision_source
		policy.PolicyVersion,               // 3 policy_version
		rec.ID,                             // 4 id
		policy.DocInSelfieApproveThreshold, // 5 doc-in-selfie floor
		policy.RequireQRVerified,           // 6 QR switch
		policy.LivenessPass,                // 7 liveness floor
		policy.OCRPass,                     // 8 OCR floor
	}
	assertRouterApproveGuardArgs(t, args)
	result := db.Exec(routerApproveGuardSQL(dialect), args...)
	if result.Error != nil {
		t.Fatalf("guard SQL failed: %v", result.Error)
	}
	if result.RowsAffected != 0 {
		t.Fatalf("guard affected %d row(s), want 0 (doc-in-selfie score below floor)", result.RowsAffected)
	}
	if got := reloadRouterRecord(t, db, rec.ID); got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q (direct SQL must not approve)", got.Status, models.VerificationPending)
	}
}

// TestRouterApproveGuardSqlQrBypass proves the QR predicate is genuinely
// conditional in SQL, not only in Go: with require_qr=0 an otherwise-green
// record with overall="failed" must be approved by the guard.
func TestRouterApproveGuardSqlQrBypass(t *testing.T) {
	db := setupEscalationTestDB(t)
	dialect := db.Dialector.Name()

	opts := healthyDocInSelfieOpts()
	opts.docValidationOverall = "failed"
	opts.govtIDHash = "router-qr-bypass-guard-hash"
	rec := createRouterTestRecord(t, db, opts)

	policy := routerTestPolicy(false, 100)

	// Exactly routerApproveGuardArgCount (8) arguments, positionally matching the
	// placeholder inventory; no face_auto_pass argument (see the arg list above).
	args := []interface{}{
		false,                              // 1 shadow_mode_applied
		"ai",                               // 2 decision_source
		policy.PolicyVersion,               // 3 policy_version
		rec.ID,                             // 4 id
		policy.DocInSelfieApproveThreshold, // 5 doc-in-selfie floor
		false,                              // 6 require_qr = false: bypass
		policy.LivenessPass,                // 7 liveness floor
		policy.OCRPass,                     // 8 OCR floor
	}
	assertRouterApproveGuardArgs(t, args)
	result := db.Exec(routerApproveGuardSQL(dialect), args...)
	if result.Error != nil {
		t.Fatalf("guard SQL failed: %v", result.Error)
	}
	if result.RowsAffected != 1 {
		t.Fatalf("guard affected %d row(s), want 1 (QR bypassed, other evidence green)", result.RowsAffected)
	}
	got := reloadRouterRecord(t, db, rec.ID)
	if got.Status != models.VerificationApproved {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationApproved)
	}
}

// TestRouterRejectGuardSqlAcceptsDocInSelfieReject proves the new OR branch of
// the reject guard fires on the document-in-selfie mismatch even when liveness
// would not have rejected the record.
func TestRouterRejectGuardSqlAcceptsDocInSelfieReject(t *testing.T) {
	db := setupEscalationTestDB(t)
	dialect := db.Dialector.Name()

	opts := healthyDocInSelfieOpts()
	opts.docInSelfieDecision = docInSelfieReject
	opts.docInSelfieScore = ptrRouterInt(-100)
	opts.isLive = true // liveness is CLEAN: only the mismatch may reject
	rec := createRouterTestRecord(t, db, opts)

	policy := routerTestPolicy(false, 100)

	result := db.Exec(routerRejectGuardSQL(dialect),
		false,
		routerSourceDocInSelfie,
		policy.PolicyVersion,
		rec.ID,
		policy.LivenessReject,
	)
	if result.Error != nil {
		t.Fatalf("reject guard SQL failed: %v", result.Error)
	}
	if result.RowsAffected != 1 {
		t.Fatalf("reject guard affected %d row(s), want 1 (doc-in-selfie mismatch)", result.RowsAffected)
	}
	got := reloadRouterRecord(t, db, rec.ID)
	if got.Status != models.VerificationRejected {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationRejected)
	}
	if got.DecisionSource != routerSourceDocInSelfie {
		t.Errorf("decision_source = %q, want %q", got.DecisionSource, routerSourceDocInSelfie)
	}
}

// TestRouterRejectGuardSqlIgnoresDocInSelfieWithoutSentinel proves the OR
// branch is narrow: a "reject" verdict whose score is NOT the worker's -100
// sentinel must not be rejected by the guard.
func TestRouterRejectGuardSqlIgnoresDocInSelfieWithoutSentinel(t *testing.T) {
	db := setupEscalationTestDB(t)
	dialect := db.Dialector.Name()

	opts := healthyDocInSelfieOpts()
	opts.docInSelfieDecision = docInSelfieReject
	opts.docInSelfieScore = ptrRouterInt(0) // reject verdict, but not the sentinel
	opts.isLive = true
	rec := createRouterTestRecord(t, db, opts)

	policy := routerTestPolicy(false, 100)

	result := db.Exec(routerRejectGuardSQL(dialect),
		false,
		routerSourceDocInSelfie,
		policy.PolicyVersion,
		rec.ID,
		policy.LivenessReject,
	)
	if result.Error != nil {
		t.Fatalf("reject guard SQL failed: %v", result.Error)
	}
	if result.RowsAffected != 0 {
		t.Fatalf("reject guard affected %d row(s), want 0 (no -100 sentinel, liveness clean)", result.RowsAffected)
	}
}

// ---------------------------------------------------------------------------
// Phase E: business KYC evidence gate
//
// A business record has NO selfie and NO liveness_result: its only identity
// anchor is the registry/document review. These tests pin the type-aware gates
// (Go + SQL) and the P0-1 placeholder-count invariant.
// ---------------------------------------------------------------------------

// TestRouterApprovesBusinessWithDocsApproved: a business record whose ML stage
// completed, whose document review says "approve" (in BOTH the nested and the
// flat shape), whose OCR confidence is 0.95 and whose document_validation is
// "verified" must auto-approve — even though it carries no liveness result at
// all, which is precisely what used to make business auto-approval impossible.
func TestRouterApprovesBusinessWithDocsApproved(t *testing.T) {
	db := setupEscalationTestDB(t)

	opts := healthyBusinessOpts()
	opts.govtIDHash = "router-business-docs-approve-hash"
	rec := createRouterTestRecord(t, db, opts)

	if rec.Type != models.VerificationBusiness {
		t.Fatalf("fixture type = %q, want %q", rec.Type, models.VerificationBusiness)
	}

	svc := setupRouterService(db, false, 100)

	if !svc.evidenceComplete(&rec) {
		t.Fatal("evidenceComplete = false for a business record with approved docs, want true (business must not be gated on selfie/liveness)")
	}
	if dec := svc.decideRouting(&rec); !dec.Approve {
		t.Fatalf("decideRouting = %+v, want Approve=true for an approved business record", dec)
	}

	routed, approved, rejected, manual := runRouter(t, svc, 10)
	if routed != 1 || approved != 1 || rejected != 0 || manual != 0 {
		t.Fatalf("router counted routed=%d approved=%d rejected=%d manual=%d; want 1 routed/1 approved", routed, approved, rejected, manual)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.Status != models.VerificationApproved {
		t.Errorf("status = %q, want %q", got.Status, models.VerificationApproved)
	}
	if got.ReviewPriority != models.ReviewPriorityAutoEligible {
		t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityAutoEligible)
	}
	if got.DecisionSource != routerSourceAI {
		t.Errorf("decision_source = %q, want %q", got.DecisionSource, routerSourceAI)
	}
	if got.VerifiedAt == nil {
		t.Error("verified_at = nil, want set (the SQL guard approved the business record)")
	}
}

// TestRouterBusinessDocsNotApprovedRoutesManual covers the fail-closed
// directions of the business gate: an explicit "manual" verdict and a
// face_match_result with NO business_docs keys at all must both land in manual
// review with status still pending — never approved.
func TestRouterBusinessDocsNotApprovedRoutesManual(t *testing.T) {
	cases := []struct {
		name string
		opts routerRecordOpts
	}{
		{
			name: "nested_and_flat_manual",
			opts: func() routerRecordOpts {
				o := healthyBusinessOpts()
				// Both shapes say "manual" (the worker's score is 0 in that case).
				o.faceMatchOverride = businessFaceMatchJSON("manual", ptrRouterInt(0), "manual", ptrRouterInt(0))
				return o
			}(),
		},
		{
			name: "no_business_docs_keys",
			opts: func() routerRecordOpts {
				o := healthyBusinessOpts()
				o.faceMatchOverride = businessFaceMatchJSON("", nil, "", nil)
				o.businessDocsOmitAll = true
				return o
			}(),
		},
		{
			name: "flat_approve_only_without_nested",
			opts: func() routerRecordOpts {
				// Only the flat key is an approval; the nested object is absent.
				// businessDocsApproved accepts EITHER shape, so this must approve
				// the Go gate — the manual variant below proves the AND-nature of
				// the SQL override is exercised separately.
				o := healthyBusinessOpts()
				o.faceMatchOverride = businessFaceMatchJSON("", nil, "approve", ptrRouterInt(100))
				return o
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupEscalationTestDB(t)
			rec := createRouterTestRecord(t, db, tc.opts)
			svc := setupRouterService(db, false, 100)

			wantApprove := tc.name == "flat_approve_only_without_nested"
			if got := svc.evidenceComplete(&rec); got != wantApprove {
				t.Fatalf("evidenceComplete = %v, want %v", got, wantApprove)
			}

			routed, approved, rejected, manual := runRouter(t, svc, 10)
			if routed != 1 {
				t.Fatalf("router routed = %d, want 1", routed)
			}
			if wantApprove {
				if approved != 1 || manual != 0 {
					t.Fatalf("router counted approved=%d manual=%d; want 1 approved", approved, manual)
				}
			} else {
				if approved != 0 || rejected != 0 || manual != 1 {
					t.Fatalf("router counted approved=%d rejected=%d manual=%d; want 0/0/1 (fail-closed to manual)", approved, rejected, manual)
				}
			}

			got := reloadRouterRecord(t, db, rec.ID)
			if wantApprove {
				if got.Status != models.VerificationApproved {
					t.Errorf("status = %q, want %q", got.Status, models.VerificationApproved)
				}
				return
			}
			if got.Status != models.VerificationPending {
				t.Errorf("status = %q, want %q (unapproved business docs must never approve)", got.Status, models.VerificationPending)
			}
			if got.ReviewPriority != models.ReviewPriorityManual {
				t.Errorf("review_priority = %q, want %q", got.ReviewPriority, models.ReviewPriorityManual)
			}
			if got.DecisionSource != routerSourceSystem {
				t.Errorf("decision_source = %q, want %q", got.DecisionSource, routerSourceSystem)
			}
			if got.AutoApprovedAt != nil {
				t.Errorf("auto_approved_at = %v, want nil", got.AutoApprovedAt)
			}
		})
	}
}

// TestRouterBusinessApproveGuardSqlRejectsUnapprovedDocs is the independent-DB
// half of the business gate: calling the approval guard standalone on a business
// record whose document verdict is "manual" must touch ZERO rows, even though
// the record has no liveness_result at all. If the guard trusted the Go-side
// decision instead of re-reading the evidence, this UPDATE would approve it.
func TestRouterBusinessApproveGuardSqlRejectsUnapprovedDocs(t *testing.T) {
	db := setupEscalationTestDB(t)
	dialect := db.Dialector.Name()

	opts := healthyBusinessOpts()
	opts.faceMatchOverride = businessFaceMatchJSON("manual", ptrRouterInt(0), "manual", ptrRouterInt(0))
	opts.livenessOverride = datatypes.JSON([]byte("{}"))
	rec := createRouterTestRecord(t, db, opts)

	if rec.LivenessResult == nil || string(rec.LivenessResult) != "{}" {
		t.Fatalf("fixture liveness_result = %s, want {} (a business record has no liveness)", rec.LivenessResult)
	}

	policy := routerTestPolicy(false, 100)
	args := []interface{}{
		false,                              // 1 shadow_mode_applied
		"ai",                               // 2 decision_source
		policy.PolicyVersion,               // 3 policy_version
		rec.ID,                             // 4 id
		policy.DocInSelfieApproveThreshold, // 5 doc-in-selfie floor
		policy.RequireQRVerified,           // 6 QR switch
		policy.LivenessPass,                // 7 liveness floor
		policy.OCRPass,                     // 8 OCR floor
	}
	assertRouterApproveGuardArgs(t, args)

	result := db.Exec(routerApproveGuardSQL(dialect), args...)
	if result.Error != nil {
		t.Fatalf("guard SQL failed: %v", result.Error)
	}
	if result.RowsAffected != 0 {
		t.Fatalf("guard affected %d row(s), want 0 (business docs decision is 'manual')", result.RowsAffected)
	}

	got := reloadRouterRecord(t, db, rec.ID)
	if got.Status != models.VerificationPending {
		t.Errorf("status = %q, want %q (the DB guard must independently re-verify)", got.Status, models.VerificationPending)
	}
	if got.AutoApprovedAt != nil {
		t.Errorf("auto_approved_at = %v, want nil", got.AutoApprovedAt)
	}

	// Same guard, same record, verdict flipped to approve: it must now match, so
	// the zero above came from the business predicate and not from some other
	// (unintentionally strict) clause.
	approvedJSON := businessFaceMatchJSON("approve", ptrRouterInt(100), "approve", ptrRouterInt(100))
	if err := db.Model(&models.VerificationRecord{}).Where("id = ?", rec.ID).
		UpdateColumn("face_match_result", approvedJSON).Error; err != nil {
		t.Fatalf("flip business verdict: %v", err)
	}
	ok := db.Exec(routerApproveGuardSQL(dialect), args...)
	if ok.Error != nil {
		t.Fatalf("guard SQL failed on the approve flip: %v", ok.Error)
	}
	if ok.RowsAffected != 1 {
		t.Fatalf("guard affected %d row(s) after the verdict became 'approve', want 1 (proves the manual case was blocked by the business predicate)", ok.RowsAffected)
	}
}

// TestRouterApproveGuardPlaceholderRegression is the P0-1 regression guard. The
// bug was a leftover face_auto_pass argument in the tx.Exec call: the statement
// has 8 ? markers, the call passed 9 arguments, and every binding after the
// fifth compared the WRONG value (the doc-in-selfie floor was bound where the QR
// switch belonged, and so on) — silently, with no SQL error, because the extra
// argument is simply ignored by both drivers.
//
// This test fails on that defect without a PostgreSQL server: it counts the real
// ? markers in BOTH dialects and asserts they equal routerApproveGuardArgCount,
// the same constant every call site's argument list is checked against.
func TestRouterApproveGuardPlaceholderRegression(t *testing.T) {
	// The counting helper must not count a ? inside a string literal, otherwise
	// this guard would be meaningless.
	if got := countSQLPlaceholders("SELECT 1 WHERE x = ? AND note = 'why?' AND y = ?"); got != 2 {
		t.Fatalf("countSQLPlaceholders = %d on a statement with 2 real markers, want 2 (string literals must not count)", got)
	}

	for _, dialect := range []string{"postgres", "sqlite"} {
		sql := routerApproveGuardSQL(dialect)
		got := countSQLPlaceholders(sql)
		if got != routerApproveGuardArgCount {
			t.Errorf("%s approve guard has %d ? placeholders, want %d (routerApproveGuardArgCount)", dialect, got, routerApproveGuardArgCount)
		}
		// Reading the placeholders back IN ORDER proves not just the count but the
		// sequence the applyRouting call site must mirror (the P0-1 shape):
		// SET shadow/decision_source/policy_version, then id, then the
		// doc-in-selfie floor, the QR switch, the liveness floor and the OCR floor.
		marks := make([]string, 0, got)
		for i := 1; i <= got; i++ {
			marks = append(marks, fmt.Sprintf("?%d", i))
		}
		seq := sql
		for i, mark := range marks {
			at := nthRealPlaceholder(seq, i)
			if at < 0 {
				t.Fatalf("%s: statement has fewer than %d placeholders", dialect, i+1)
			}
			seq = seq[:at] + mark + seq[at+1:]
		}
		t.Logf("%s approve-guard placeholder sequence: %s", dialect, strings.Join(marks, ", "))
		t.Logf("%s approve-guard statement:\n%s", dialect, seq)
		// A business record's verdict must be re-verified by both dialects.
		for _, needle := range []string{"business_docs", "'business'"} {
			if !strings.Contains(sql, needle) {
				t.Errorf("%s approve guard is missing the type-aware %q predicate", dialect, needle)
			}
		}
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
