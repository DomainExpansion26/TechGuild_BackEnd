package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

// ============================================================
// KYC Phase 3 — Go router
//
// The Python worker writes ML scores + a threshold-only
// ai_recommended_priority and hands the record to this router via
// routing_state='pending'. The router applies the evidence gates and the
// shadow-mode/cohort policy and writes the FINAL review_priority (or
// approves/rejects through SQL guards that independently re-verify the
// evidence inside the UPDATE itself, so no path can approve beneath the
// thresholds — ChatGPT P0 #2).
// ============================================================

// kycCohortHashKey is the HMAC key for the deterministic shadow-off rollout
// cohort. A fixed application constant keeps cohort assignment stable across
// restarts and instances (same user -> same bucket forever, no key rotation).
// Phase 4+ can promote this to a managed env secret if cohort re-shuffling is
// ever wanted deliberately.
const kycCohortHashKey = "KYC_COHORT_V1"

// Decision sources recorded in verification_records.decision_source.
const (
	routerSourceShadow = "shadow" // shadow mode or cohort gate, not the AI score
	routerSourceAI     = "ai"     // Python's threshold decision, re-verified by Go gates
	routerSourceSystem = "system" // Go-side fallback (incomplete evidence, guards)

	// routerSourceDocInSelfie (Phase B): the hard Aadhaar-number mismatch
	// between the uploaded document and the document held in the selfie.
	// Distinct from "system" so an operator can tell a document-in-selfie
	// rejection apart from a guard fallback in decision_source.
	routerSourceDocInSelfie = "doc_in_selfie"
)

// routerReason values appear in logs/audit details (there is no reason column).
const (
	routerReasonShadowMode          = "shadow_mode"
	routerReasonMLIncomplete        = "ml_incomplete"
	routerReasonEvidenceIncomplete  = "evidence_incomplete"
	routerReasonRejectNotHardSignal = "reject_not_hard_signal"
	routerReasonGuardRejected       = "guard_rejected"
	routerReasonRejectGuardRejected = "reject_guard_rejected"

	// routerReasonAadhaarMismatch (Phase B): the document held in the selfie
	// carries a different Aadhaar number than the uploaded document.
	routerReasonAadhaarMismatch = "aadhaar_number_mismatch"
)

// routingActionType marks router decisions in verification_audit_logs.
const routerActionType = "KYC_ROUTED"

// routingDecision is the pure outcome of decideRouting; applyRouting then
// persists it (or downgrades to manual when a guard rejects the write).
type routingDecision struct {
	Effective     models.VerificationReviewPriority
	Source        string // "shadow" | "ai" | "system"
	Reason        string // "" | "shadow_mode" | "evidence_incomplete" | ...
	Approve       bool
	Reject        bool
	PolicyVersion string
}

// RouteMLCompleted claims one batch of ml_completed + routing_state='pending'
// records and applies the routing policy to each. It is called from
// jobs.kyc_router on an interval.
//
// Concurrency: the batch SELECT takes row locks (FOR UPDATE SKIP LOCKED on
// PostgreSQL), and every outcome UPDATE re-checks the record state, so two
// router runs (or a router running against a resubmission) can never apply two
// decisions to one record. Returns (routed, approved, rejected, manual).
func (s *VerificationService) RouteMLCompleted(batchSize int) (routed, approved, rejected, manual int, err error) {
	if batchSize <= 0 {
		batchSize = 50
	}

	err = postgres.DB.Transaction(func(tx *gorm.DB) error {
		var recs []models.VerificationRecord
		q := tx.Clauses(routerLockClauses(tx)...).
			Where("ml_status = ? AND routing_state = ?",
				models.MLStatusCompleted, models.RoutingPending).
			Order("ml_completed_at ASC").
			Limit(batchSize)
		if err := q.Find(&recs).Error; err != nil {
			return err
		}

		for i := range recs {
			if err := s.applyRouting(tx, &recs[i]); err != nil {
				// Log and continue — a stuck record must not hold back the
				// whole batch; the next run picks it up after the tx commits.
				log.Printf("[KYC Router] applyRouting failed for %s: %v",
					recs[i].ID, err)
				continue
			}
			routed++
			switch recs[i].Status {
			case models.VerificationApproved:
				approved++
			case models.VerificationRejected:
				rejected++
			default:
				manual++
			}
		}
		return nil
	})
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return routed, approved, rejected, manual, nil
}

// routerLockClauses returns FOR UPDATE SKIP LOCKED on PostgreSQL. The SQLite
// test databases reject the clause (and already serialise every statement on
// a single connection), so tests exercise the same decisions without it.
func routerLockClauses(db *gorm.DB) []clause.Expression {
	switch db.Dialector.Name() {
	case "postgres":
		return []clause.Expression{
			clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"},
		}
	default:
		return nil
	}
}

// applyRouting writes the decision for one claimed record inside an open
// transaction. The SQL mutations mirror the decision state back onto rec so
// the caller can count outcomes and tests can assert on the in-memory record.
func (s *VerificationService) applyRouting(tx *gorm.DB, rec *models.VerificationRecord) error {
	policy := s.effectiveKYCPolicy()
	dec := s.decideRouting(rec)
	dialect := tx.Dialector.Name()

	switch {
	case dec.Approve:
		// Defense-in-depth guard: the SQL itself must reject if evidence is
		// missing or below threshold (independently re-parsed by the DB).
		// Exactly routerApproveGuardArgCount arguments, positionally matching the
		// guard's ? placeholders (the constant lives in the test file and
		// TestRouterApproveGuardPlaceholderRegression pins its value on both
		// dialects). A stale extra argument here (the removed FaceAutoPass)
		// silently SHIFTS every following binding, so every threshold would be
		// compared against the wrong value.
		result := tx.Exec(routerApproveGuardSQL(dialect),
			dec.Source == routerSourceShadow, // 1 shadow_mode_applied
			dec.Source,                       // 2 decision_source
			dec.PolicyVersion,                // 3 policy_version
			rec.ID,                           // 4 id
			// Phase B: document-in-selfie score floor replaces the
			// aadhaar_similarity floor that a disabled face match kept nil.
			policy.DocInSelfieApproveThreshold, // 5 doc_in_selfie score floor
			policy.RequireQRVerified,           // 6 require_qr switch
			policy.LivenessPass,                // 7 liveness floor
			policy.OCRPass,                     // 8 ocr floor
		)
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			// Guard rejected the approval — fall back to manual review.
			log.Printf("[KYC Router] ROUTING_GUARD_REJECTED id=%s", rec.ID)
			return s.applyManual(tx, rec, dec, routerReasonGuardRejected)
		}

		// The status flip above is only half the contract: an approved record
		// must also claim its dedup hash, grant the KYC bonus and activate the
		// account. Running it on the same tx means a failure rolls the approval
		// back instead of leaving a record approved with no uniqueness claim.
		if err := s.approveVerificationSideEffectsTx(tx, rec); err != nil {
			log.Printf("[KYC Router] APPROVE_SIDE_EFFECTS_FAILED id=%s: %v", rec.ID, err)
			return err
		}

		rec.Status = models.VerificationApproved
		rec.ReviewPriority = models.ReviewPriorityAutoEligible
		rec.RoutingState = models.RoutingRouted
		rec.ShadowModeApplied = dec.Source == routerSourceShadow
		rec.DecisionSource = dec.Source
		rec.PolicyVersion = dec.PolicyVersion
		s.auditRouterDecision(tx, rec, dec)

	case dec.Reject:
		result := tx.Exec(routerRejectGuardSQL(dialect),
			dec.Source == routerSourceShadow,
			dec.Source,
			dec.PolicyVersion,
			rec.ID,
			policy.LivenessReject,
		)
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			log.Printf("[KYC Router] REJECT_GUARD_REJECTED id=%s (falling back to manual review)", rec.ID)
			return s.applyManual(tx, rec, dec, routerReasonRejectGuardRejected)
		}

		rec.Status = models.VerificationRejected
		rec.ReviewPriority = models.ReviewPriorityAutoReject
		rec.RoutingState = models.RoutingRouted
		rec.ShadowModeApplied = dec.Source == routerSourceShadow
		rec.DecisionSource = dec.Source
		rec.PolicyVersion = dec.PolicyVersion
		s.auditRouterDecision(tx, rec, dec)

	default:
		return s.applyManual(tx, rec, dec, dec.Reason)
	}
	return nil
}

// applyManual stamps the routing columns without changing status: the record
// stays 'pending' and moves into the (human) manual review queue.
func (s *VerificationService) applyManual(tx *gorm.DB, rec *models.VerificationRecord, dec routingDecision, reason string) error {
	shadow := dec.Source == routerSourceShadow
	dialect := tx.Dialector.Name()
	nowFunc := "NOW()"
	if dialect != "postgres" {
		nowFunc = "CURRENT_TIMESTAMP"
	}
	sql := `
        UPDATE verification_records
           SET review_priority = 'manual_review',
               routing_state = 'routed',
               shadow_mode_applied = ?,
               decision_source = ?,
               policy_version = ?,
               updated_at = ` + nowFunc + `
         WHERE id = ?
           AND status = 'pending'
           AND ml_status = 'ml_completed'
           AND system_escalated_at IS NULL`
	result := tx.Exec(sql,
		shadow, dec.Source, dec.PolicyVersion, rec.ID,
	)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// The record changed state between claim and write (escalated or
		// already routed) — leave the DB value untouched.
		log.Printf("[KYC Router] manual write skipped id=%s reason=%s (state changed under lock)", rec.ID, reason)
		return nil
	}
	if reason != "" {
		log.Printf("[KYC Router] manual id=%s source=%s reason=%s", rec.ID, dec.Source, reason)
	}
	rec.ReviewPriority = models.ReviewPriorityManual
	rec.RoutingState = models.RoutingRouted
	rec.ShadowModeApplied = shadow
	rec.DecisionSource = dec.Source
	rec.PolicyVersion = dec.PolicyVersion
	return nil
}

// decideRouting is the pure policy step: it reads nothing but the record and
// the loaded policy, so unit tests can call it directly.
func (s *VerificationService) decideRouting(rec *models.VerificationRecord) routingDecision {
	policy := s.effectiveKYCPolicy()
	dec := routingDecision{PolicyVersion: policy.PolicyVersion}

	// 1. Shadow policy first: it outranks every AI recommendation.
	if s.shouldEnforceShadow(rec) {
		dec.Effective = models.ReviewPriorityManual
		dec.Source = routerSourceShadow
		dec.Reason = routerReasonShadowMode
		return dec
	}

	// 2. ML not actually complete (should be impossible after the batch query,
	//    but the record may have been hand-built in a test).
	if rec.MLStatus != models.MLStatusCompleted || rec.MLCompletedAt == nil {
		dec.Effective = models.ReviewPriorityManual
		dec.Source = routerSourceSystem
		dec.Reason = routerReasonMLIncomplete
		return dec
	}

	// 3. Phase B hard signal: the document held in the selfie carries a
	//    different Aadhaar number than the uploaded document. Like the
	//    liveness spoof, this is a "hard" signal and outranks the AI
	//    recommendation — but it is operator-gated, because a mis-OCR must
	//    never silently reject a legitimate user. Checked before the
	//    liveness-spoof path so the audit records the mismatch reason.
	if hardDocInSelfieMismatch(rec) && policy.DocInSelfieRejectOnMismatch {
		dec.Effective = models.ReviewPriorityAutoReject
		dec.Source = routerSourceDocInSelfie
		dec.Reason = routerReasonAadhaarMismatch
		dec.Reject = true
		return dec
	}

	// 4. AI says eligible -> re-check the evidence against the thresholds.
	if rec.AIRecommendedPriority == models.ReviewPriorityAutoEligible {
		if !s.evidenceComplete(rec) {
			dec.Effective = models.ReviewPriorityManual
			dec.Source = routerSourceSystem
			dec.Reason = routerReasonEvidenceIncomplete
			return dec
		}
		dec.Effective = models.ReviewPriorityAutoEligible
		dec.Source = routerSourceAI
		dec.Approve = true
		return dec
	}

	// 5. AI says reject -> only a clean liveness spoof backfills that.
	if rec.AIRecommendedPriority == models.ReviewPriorityAutoReject {
		if !s.isLivenessSpoof(rec) {
			dec.Effective = models.ReviewPriorityManual
			dec.Source = routerSourceSystem
			dec.Reason = routerReasonRejectNotHardSignal
			return dec
		}
		dec.Effective = models.ReviewPriorityAutoReject
		dec.Source = routerSourceAI
		dec.Reject = true
		return dec
	}

	// 6. Default: the AI recommendation is manual (or unknown) -> manual.
	dec.Effective = models.ReviewPriorityManual
	dec.Source = routerSourceAI
	return dec
}

// hardDocInSelfieMismatch reports whether face_match_result carries the
// worker's hard document-in-selfie rejection: document_in_selfie_decision ==
// "reject". Unparseable evidence is never a hard signal — a malformed column
// must fall through to manual review, not auto-reject a user.
//
// Phase E: a business record has NO document-in-selfie stage, so a "reject"
// value there could only come from a leaked/copied column — and this path leads
// to an auto-REJECT. It is therefore gated on rec.Type == VerificationIndividual
// so a business record can never be auto-rejected on an individual-only signal.
func hardDocInSelfieMismatch(rec *models.VerificationRecord) bool {
	if rec.Type != models.VerificationIndividual {
		return false
	}
	if len(rec.FaceMatchResult) == 0 {
		return false
	}
	var face faceEvidenceJSON
	if err := json.Unmarshal(rec.FaceMatchResult, &face); err != nil {
		return false
	}
	return face.DocumentInSelfieDecision == "reject"
}

// ============================================================
// Evidence gates
// ============================================================

// evidenceComplete re-checks the ACTUAL VALUES in the JSON columns, not just
// field presence (ChatGPT P0 #2). Every dimension must independently verify:
// face, liveness, OCR confidence, document validation, ML version. Records the
// worker flagged auto_eligible but whose evidence fails here route to manual
// review instead of approval.
func (s *VerificationService) evidenceComplete(rec *models.VerificationRecord) bool {
	policy := s.effectiveKYCPolicy()

	var face faceEvidenceJSON
	if err := json.Unmarshal(rec.FaceMatchResult, &face); err != nil {
		return false
	}

	// Phase E — business KYC is a DIFFERENT evidence contract, so it branches
	// BEFORE any selfie/liveness predicate. A business record carries no selfie
	// and no liveness_result at all: gating it on document_in_selfie_*,
	// second_reference_divergence or liveness would make auto-approval
	// impossible. Its anchors are the ML model version, the document review
	// verdict and (when the policy asks for it) the UIDAI QR check.
	if rec.Type == models.VerificationBusiness {
		if rec.MLModelVersion == "" {
			return false // no model version: the ML stage never completed
		}
		if !businessDocsApproved(face) {
			return false // documents not approved (or verdict absent)
		}
		var ocr ocrEvidenceJSON
		if err := json.Unmarshal(rec.OCROcrResult, &ocr); err != nil {
			return false
		}
		// OCR confidence must be present and above the floor: it is the quality
		// proof of the document/registry read the verdict above depends on.
		if ocr.OCRConfidence == nil || float64(*ocr.OCRConfidence) < policy.OCRPass {
			return false
		}
		if policy.RequireQRVerified {
			if ocr.DocumentValidation == nil ||
				ocr.DocumentValidation.Overall != "verified" {
				return false // same operator escape hatch as the individual path
			}
		}
		return true
	}

	// // 	if face.Error != "" || face.FaceSimilarity == nil ||
	// // 		float64(*face.FaceSimilarity) < policy.FaceAutoPass {
	// // 		return false
	// // 	}

	// Phase B — document-in-selfie is the identity anchor now that face match
	// is disabled (FACE_MATCH_ENABLED=false), which makes aadhaar_similarity
	// permanently nil. The worker OCRs the selfie and compares the document
	// held in it against the uploaded document; only a clean "approve" verdict
	// with a score at/above the threshold may auto-approve. "manual" and
	// "reject" both fall through to manual review here — the hard reject is
	// decided in decideRouting, which outranks this evidence gate.
	if face.DocumentInSelfieDecision != "approve" {
		return false // no (or failed) document-in-selfie verification
	}
	if face.DocumentInSelfieScore == nil {
		return false // worker never scored the match — cannot auto-approve
	}
	if *face.DocumentInSelfieScore < policy.DocInSelfieApproveThreshold {
		return false // match too weak — cannot auto-approve
	}

	// Phase 5 — divergence guard: PAN-only strong match with a weak Aadhaar
	// routes to manual review regardless of the winning score.
	if face.SecondReferenceDivergence {
		return false
	}

	var live livenessEvidenceJSON
	if err := json.Unmarshal(rec.LivenessResult, &live); err != nil {
		return false
	}
	if live.Error != "" || live.IsLive == nil || !*live.IsLive ||
		live.LivenessScore == nil || float64(*live.LivenessScore) < policy.LivenessPass {
		return false
	}

	var ocr ocrEvidenceJSON
	if err := json.Unmarshal(rec.OCROcrResult, &ocr); err != nil {
		return false
	}
	if ocr.OCRConfidence == nil || float64(*ocr.OCRConfidence) < policy.OCRPass {
		return false
	}
	// QR gate (Phase B): require document_validation.overall == "verified"
	// only when the policy asks for it. THRESHOLD_REQUIRE_QR=false is the
	// operator escape hatch for a UIDAI QR-certificate outage — every other
	// gate above and below stays enforced.
	if policy.RequireQRVerified {
		if ocr.DocumentValidation == nil ||
			ocr.DocumentValidation.Overall != "verified" {
			return false
		}
	}

	// user_id_validation is optional; when present it must be explicitly valid.
	if len(rec.UserIDValidation) > 0 {
		var uid userIDValidationJSON
		if err := json.Unmarshal(rec.UserIDValidation, &uid); err != nil {
			return false
		}
		if uid.Overall != "valid" {
			return false
		}
	}

	return rec.MLModelVersion != ""
}

// businessDocsApproved reports whether the business document review verdict in
// face_match_result is an approval. The worker writes the verdict in several
// shapes and different producers may write only one of them, so ALL are
// accepted:
//
//	nested: {"business_docs": {"decision": "approve", "score": 100}}
//	flat:   {"business_docs_decision": "approve", "business_docs_score": 100}
//	flat:   {"business_decision": "approve"}            (Phase G canonical key)
//
// Any one confirming "approve" is enough. Anything else — "manual", "reject",
// an empty string, or an absent key — reads as NOT approved, so a missing field
// can never fail open into an auto-approval.
//
// Phase G: the verdict now means "the single selected business document matched
// the form value AND the cancelled cheque matched the bank details" — the
// worker only writes "approve" when BOTH checks passed (see
// validators/document_match.compare_business_documents). A mismatch or an OCR
// failure writes "manual", which this gate reads as not-approved and the router
// then holds for a human. The gate itself is unchanged in strength: it still
// requires exactly the string "approve".
func businessDocsApproved(face faceEvidenceJSON) bool {
	if strings.TrimSpace(string(face.BusinessDecision)) == "approve" {
		return true
	}
	if strings.TrimSpace(string(face.BusinessDocsDecision)) == "approve" {
		return true
	}
	if face.BusinessDocs != nil &&
		strings.TrimSpace(string(face.BusinessDocs.Decision)) == "approve" {
		return true
	}
	return false
}

// isLivenessSpoof identifies the ONLY auto-reject signal: a successful
// liveness inference confirming a spoof. If liveness itself errored, this is
// missing evidence (manual review), never a fraud conviction.
func (s *VerificationService) isLivenessSpoof(rec *models.VerificationRecord) bool {
	policy := s.effectiveKYCPolicy()

	var live livenessEvidenceJSON
	if err := json.Unmarshal(rec.LivenessResult, &live); err != nil {
		return false
	}
	return live.Error == "" &&
		live.IsLive != nil && !bool(*live.IsLive) &&
		live.LivenessScore != nil &&
		float64(*live.LivenessScore) < policy.LivenessReject
}

// shouldEnforceShadow implements the rollout cohort policy (ChatGPT P0 #4):
//
//	shadow_mode=true            -> everybody, cohort irrelevant
//	cohort_pct >= 100           -> nobody (full rollout)
//	cohort_pct <= 0             -> everybody (nothing released yet)
//	else                        -> shadow unless HMAC(user_id)%100 < pct
func (s *VerificationService) shouldEnforceShadow(rec *models.VerificationRecord) bool {
	policy := s.effectiveKYCPolicy()
	if policy.ShadowMode {
		return true
	}

	cohortPct := policy.RolloutCohortPct
	if cohortPct >= 100 {
		return false
	}
	if cohortPct <= 0 {
		return true
	}
	bucket := int(kycCohortBucket(rec.UserID))
	return bucket >= cohortPct // not in cohort => shadow enforced
}

// kycCohortBucket maps a user onto the [0, 100) cohort spectrum
// deterministically: HMAC-SHA256 over the UUID string, first 8 bytes as a
// big-endian word mod 100.
func kycCohortBucket(userID uuid.UUID) uint64 {
	mac := hmac.New(sha256.New, []byte(kycCohortHashKey))
	mac.Write([]byte(userID.String()))
	return binary.BigEndian.Uint64(mac.Sum(nil)[:8]) % 100
}

// ============================================================
// Evidence JSON decoding
// ============================================================

// The worker serialises both numbers and (legacy/mock) numeric strings. The
// flexible scalars accept both; anything else makes Unmarshal fail, which
// reads as "evidence incomplete" (fail-safe direction, never fail-open).

type flexibleFloat float64

func (f *flexibleFloat) UnmarshalJSON(v []byte) error {
	s := strings.TrimSpace(string(v))
	if s == "null" {
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	parsed, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("not a number: %s", s)
	}
	*f = flexibleFloat(parsed)
	return nil
}

type flexibleBool bool

func (b *flexibleBool) UnmarshalJSON(v []byte) error {
	switch strings.TrimSpace(string(v)) {
	case "true", "1":
		*b = true
	case "false", "0":
		*b = false
	case `"true"`, `"True"`, `"TRUE"`:
		*b = true
	case `"false"`, `"False"`, `"FALSE"`:
		*b = false
	default:
		return fmt.Errorf("not a boolean: %s", strings.TrimSpace(string(v)))
	}
	return nil
}

type faceEvidenceJSON struct {
	FaceSimilarity *flexibleFloat `json:"face_similarity"`
	Error          flexibleString `json:"error"`
	// Phase 5 multi-reference fields. AadhaarSimilarity is nil when no
	// usable Aadhaar photo existed (nil vs 0 must be distinguishable).
	// SecondReferenceDivergence is a plain bool (not pointer): absent means
	// the record predates Phase 5, which evidenceComplete already blocks
	// via the nil/nil-floor Aadhaar gate.
	//
	// Phase B: the router no longer GATES on AadhaarSimilarity — with
	// FACE_MATCH_ENABLED=false the worker always writes nil, which would
	// block every approval. The fields stay so the divergence guardrail and
	// the SQL guards keep a readable contract. Auto-approval now gates on the
	// DocumentInSelfie* fields below.
	AadhaarSimilarity         *flexibleFloat `json:"aadhaar_similarity"`
	SecondReferenceDivergence flexibleBool   `json:"second_reference_divergence"`

	// Phase B — document-in-selfie signal, written by the worker into
	// face_match_result. Decision is "approve" | "manual" | "reject"; Score is
	// a pointer so "field absent" is distinguishable from a legitimate 0, and
	// Match is the worker's boolean cross-check of the two OCR reads.
	DocumentInSelfieDecision string `json:"document_in_selfie_decision"`
	DocumentInSelfieScore    *int   `json:"document_in_selfie_score"`
	DocumentInSelfieMatch    bool   `json:"document_in_selfie_match"`

	// Phase E — business KYC verdicts. A business record has no selfie and no
	// liveness result, so its only identity anchor is the registry/document
	// review. The Python worker writes the verdict in BOTH shapes (a nested
	// business_docs object plus a flat business_docs_* mirror) and other writers
	// may emit only one of them, so both are decoded here and EITHER one
	// confirming "approve" is accepted by businessDocsApproved. Every key is
	// optional: absent means NOT approved (fail-closed), never fail-open.
	//
	// Phase G adds the canonical flat alias "business_decision": the worker now
	// emits it alongside the two Phase E shapes, so a Phase G producer that
	// writes only the documented key still gates correctly. All three are read
	// and any single "approve" is sufficient; the value of the Phase G verdict
	// is what business_docs.decision carries.
	BusinessDocsDecision flexibleString    `json:"business_docs_decision"`
	BusinessDocsScore    *int              `json:"business_docs_score"`
	BusinessDocs         *businessDocsJSON `json:"business_docs"`
	BusinessDecision     flexibleString    `json:"business_decision"`
}

// businessDocsJSON is the NESTED business verdict object the worker writes
// into face_match_result: {"business_docs": {"decision": "approve",
// "score": 100}}. The pointer on faceEvidenceJSON keeps "no nested object"
// distinguishable from "object present but decision absent" — both read as
// not-approved.
type businessDocsJSON struct {
	Decision flexibleString `json:"decision"`
	Score    *int           `json:"score"`
}

type livenessEvidenceJSON struct {
	IsLive        *flexibleBool  `json:"is_live"`
	LivenessScore *flexibleFloat `json:"liveness_score"`
	Error         flexibleString `json:"error"`
}

type ocrEvidenceJSON struct {
	OCRConfidence      *flexibleFloat          `json:"ocr_confidence"`
	DocumentValidation *documentValidationJSON `json:"document_validation"`
}

type documentValidationJSON struct {
	Overall string `json:"overall"`
}

type userIDValidationJSON struct {
	Overall string `json:"overall"`
}

type flexibleString string

func (s *flexibleString) UnmarshalJSON(v []byte) error {
	raw := strings.TrimSpace(string(v))
	if raw == "null" {
		*s = "" // explicit null == absent for the "is there an error" check
		return nil
	}
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		*s = flexibleString(raw[1 : len(raw)-1])
		return nil
	}
	// A non-string error payload means the stage produced structured output
	// we do not recognise; treat the value as opaque (non-empty).
	*s = flexibleString(raw)
	return nil
}

// ============================================================
// SQL guards + audit
// ============================================================

// routerApproveGuardSQL builds the approval UPDATE with evidence predicates in
// the database itself. PostgreSQL uses ->>, SQLite (tests) uses the JSON1
// extension; both yield the same decision. The guard additionally requires
// system_escalated_at IS NULL so an escalated record can never be approved by
// a router instance racing the escalator.
//
// Phase B: the aadhaar_similarity floor predicate is GONE. With
// FACE_MATCH_ENABLED=false the worker writes aadhaar_similarity as null, so
// that predicate could never be satisfied and blocked every approval. The
// documented identity anchor is now document_in_selfie_decision/score, which
// Phase A's worker writes. The QR predicate is now conditional on
// THRESHOLD_REQUIRE_QR so a UIDAI cert outage can be bypassed deliberately.
//
// Phase E: the identity predicates are TYPE-AWARE. A business record has no
// selfie and no liveness result at all, so gating it on
// document_in_selfie_*/divergence/liveness made business auto-approval
// impossible. For type='business' the identity block is replaced by the
// business-documents verdict (flat business_decision, flat
// business_docs_decision, OR nested business_docs.decision == 'approve') and
// the liveness block is exempted. Individual records keep the exact same
// predicates as before.
//
// Phase G only widens the accepted KEY NAMES for that same verdict (the worker
// now writes the canonical "business_decision"); the required VALUE is still
// exactly 'approve', so a mismatch or an OCR failure can never pass this guard.
//
// Parameter order — EXACTLY routerApproveGuardArgCount (8) placeholders, no more:
//
//	1 shadow_mode_applied  (bool)
//	2 decision_source      (string)
//	3 policy_version       (string)
//	4 id                   (uuid)
//	5 doc_in_selfie_approve_threshold (int)  — individual only
//	6 require_qr           (bool)            — both types
//	7 liveness_pass        (float)           — individual only
//	8 ocr_pass             (float)           — both types
//
// The removed face_auto_pass argument is NOT part of this statement: passing it
// again shifts every binding after it by one position.
func routerApproveGuardSQL(dialect string) string {
	if dialect == "postgres" {
		return `
		UPDATE verification_records
		   SET status = 'approved',
		       verified_at = NOW(),
		       auto_approved_at = NOW(),
		       review_priority = 'auto_eligible',
		       routing_state = 'routed',
		       shadow_mode_applied = ?,
		       decision_source = ?,
		       policy_version = ?,
		       updated_at = NOW()
		 WHERE id = ?
		   AND status = 'pending'
		   AND ml_status = 'ml_completed'
		   AND system_escalated_at IS NULL
		   AND CASE WHEN type = 'business' THEN
		              COALESCE(face_match_result->>'business_decision', '') = 'approve'
		              OR COALESCE(face_match_result->>'business_docs_decision', '') = 'approve'
		              OR COALESCE(face_match_result->'business_docs'->>'decision', '') = 'approve'
		            ELSE (
		              (face_match_result->>'document_in_selfie_decision') = 'approve'
		              AND (face_match_result->>'document_in_selfie_score')::int >= ?
		              AND (face_match_result->>'second_reference_divergence')::boolean IS NOT TRUE
		            ) END
		   AND (CASE WHEN ? THEN
		              COALESCE(ocr_ocr_result->'document_validation'->>'overall', '') = 'verified'
		            ELSE TRUE END)
		   AND CASE WHEN type = 'business' THEN TRUE
		            ELSE (liveness_result->>'is_live' = 'true'
		                  AND (liveness_result->>'liveness_score')::float >= ?)
		       END
		   AND (ocr_ocr_result->>'ocr_confidence')::float >= ?`
	}
	return `
		UPDATE verification_records
		   SET status = 'approved',
		       verified_at = CURRENT_TIMESTAMP,
		       auto_approved_at = CURRENT_TIMESTAMP,
		       review_priority = 'auto_eligible',
		       routing_state = 'routed',
		       shadow_mode_applied = ?,
		       decision_source = ?,
		       policy_version = ?,
		       updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?
		   AND status = 'pending'
		   AND ml_status = 'ml_completed'
		   AND system_escalated_at IS NULL
		   AND CASE WHEN type = 'business' THEN
		              COALESCE(json_extract(face_match_result, '$.business_decision'), '') = 'approve'
		              OR COALESCE(json_extract(face_match_result, '$.business_docs_decision'), '') = 'approve'
		              OR COALESCE(json_extract(face_match_result, '$.business_docs.decision'), '') = 'approve'
		            ELSE
		              (json_extract(face_match_result, '$.document_in_selfie_decision') = 'approve'
		               AND CAST(json_extract(face_match_result, '$.document_in_selfie_score') AS INTEGER) >= ?
		               AND COALESCE(json_extract(face_match_result, '$.second_reference_divergence'), 0) = 0)
		            END = 1
		   AND (CASE WHEN ? <> 0 THEN
		              COALESCE(json_extract(ocr_ocr_result, '$.document_validation.overall'), '') = 'verified'
		            ELSE 1 END) = 1
		   AND (CASE WHEN type = 'business' THEN 1
		             ELSE (json_extract(liveness_result, '$.is_live') = 1
		                   AND CAST(json_extract(liveness_result, '$.liveness_score') AS REAL) >= ?)
		        END) = 1
		   AND CAST(json_extract(ocr_ocr_result, '$.ocr_confidence') AS REAL) >= ?`
}

// routerRejectGuardSQL mirrors the approval guard for the auto-reject path.
// Two independent hard signals can reach 'rejected': a clean liveness spoof
// (is_live=false and a score below the reject threshold) or, since Phase B, a
// hard document-in-selfie Aadhaar-number mismatch, which the worker marks with
// document_in_selfie_score = -100. Everything else is protected by the metric
// predicates falling through into the manual-review fallback.
func routerRejectGuardSQL(dialect string) string {
	if dialect == "postgres" {
		return `
		UPDATE verification_records
		   SET status = 'rejected',
		       rejected_at = NOW(),
		       review_priority = 'auto_reject',
		       routing_state = 'routed',
		       shadow_mode_applied = ?,
		       decision_source = ?,
		       policy_version = ?,
		       updated_at = NOW()
		 WHERE id = ?
		   AND status = 'pending'
		   AND ml_status = 'ml_completed'
		   AND system_escalated_at IS NULL
		   AND (
		         (
		           liveness_result->>'is_live' = 'false'
		           AND (liveness_result->>'liveness_score')::float < ?
		         )
		         OR (
		           face_match_result->>'document_in_selfie_decision' = 'reject'
		           AND (face_match_result->>'document_in_selfie_score')::int = -100
		         )
		       )`
	}
	return `
		UPDATE verification_records
		   SET status = 'rejected',
		       rejected_at = CURRENT_TIMESTAMP,
		       review_priority = 'auto_reject',
		       routing_state = 'routed',
		       shadow_mode_applied = ?,
		       decision_source = ?,
		       policy_version = ?,
		       updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?
		   AND status = 'pending'
		   AND ml_status = 'ml_completed'
		   AND system_escalated_at IS NULL
		   AND (
		         (
		           json_extract(liveness_result, '$.is_live') = 0
		           AND CAST(json_extract(liveness_result, '$.liveness_score') AS REAL) < ?
		         )
		         OR (
		           json_extract(face_match_result, '$.document_in_selfie_decision') = 'reject'
		           AND CAST(json_extract(face_match_result, '$.document_in_selfie_score') AS INTEGER) = -100
		         )
		       )`
}

// auditRouterDecision records one KYC_ROUTED entry per approve/reject outcome.
// Manual outcomes are not audited here (the brief keeps audit noise bounded);
// their trail is review_priority/decision_source on the record itself.
func (s *VerificationService) auditRouterDecision(tx *gorm.DB, rec *models.VerificationRecord, dec routingDecision) {
	details, err := json.Marshal(map[string]interface{}{
		"verification_id": rec.ID,
		"user_id":         rec.UserID,
		"decision":        dec.Source,
		"reason":          dec.Reason,
		"effective":       string(dec.Effective),
		"policy_version":  dec.PolicyVersion,
	})
	if err != nil {
		log.Printf("[KYC Router] audit details marshal failed for %s: %v", rec.ID, err)
		return
	}
	if err := tx.Create(&models.VerificationAuditLog{
		VerificationID: rec.ID,
		UserID:         rec.UserID,
		ActionType:     routerActionType,
		Details:        datatypes.JSON(details),
	}).Error; err != nil {
		log.Printf("[KYC Router] audit log write failed for %s: %v", rec.ID, err)
	}
}
