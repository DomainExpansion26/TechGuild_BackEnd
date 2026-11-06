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
)

// routerReason values appear in logs/audit details (there is no reason column).
const (
	routerReasonShadowMode          = "shadow_mode"
	routerReasonMLIncomplete        = "ml_incomplete"
	routerReasonEvidenceIncomplete  = "evidence_incomplete"
	routerReasonRejectNotHardSignal = "reject_not_hard_signal"
	routerReasonGuardRejected       = "guard_rejected"
	routerReasonRejectGuardRejected = "reject_guard_rejected"
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
		result := tx.Exec(routerApproveGuardSQL(dialect),
			dec.Source == routerSourceShadow,
			dec.Source,
			dec.PolicyVersion,
			rec.ID,
			policy.FaceAutoPass,
			policy.AadhaarFloor,
			policy.LivenessPass,
			policy.OCRPass,
		)
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			// Guard rejected the approval — fall back to manual review.
			log.Printf("[KYC Router] ROUTING_GUARD_REJECTED id=%s", rec.ID)
			return s.applyManual(tx, rec, dec, routerReasonGuardRejected)
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

	// 3. AI says eligible -> re-check the evidence against the thresholds.
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

	// 4. AI says reject -> only a clean liveness spoof backfills that.
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

	// 5. Default: the AI recommendation is manual (or unknown) -> manual.
	dec.Effective = models.ReviewPriorityManual
	dec.Source = routerSourceAI
	return dec
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
	if face.Error != "" || face.FaceSimilarity == nil ||
		float64(*face.FaceSimilarity) < policy.FaceAutoPass {
		return false
	}

	// Phase 5 — Aadhaar floor even when PAN wins the multi-reference match:
	// a fake PAN cannot carry the score for an unverified Aadhaar.
	if face.AadhaarSimilarity == nil {
		return false // no usable Aadhaar photo — cannot auto-approve
	}
	if float64(*face.AadhaarSimilarity) < policy.AadhaarFloor {
		return false // Aadhaar too weak — cannot auto-approve
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
	if ocr.DocumentValidation == nil ||
		ocr.DocumentValidation.Overall != "verified" {
		return false
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
	AadhaarSimilarity         *flexibleFloat `json:"aadhaar_similarity"`
	SecondReferenceDivergence flexibleBool   `json:"second_reference_divergence"`
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
		   AND (face_match_result->>'face_similarity')::float >= ?
		   AND (face_match_result->>'aadhaar_similarity')::float >= ?
		   AND (face_match_result->>'second_reference_divergence')::boolean IS NOT TRUE
		   AND liveness_result->>'is_live' = 'true'
		   AND (liveness_result->>'liveness_score')::float >= ?
		   AND (ocr_ocr_result->>'ocr_confidence')::float >= ?
		   AND ocr_ocr_result->'document_validation'->>'overall' = 'verified'`
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
		   AND CAST(json_extract(face_match_result, '$.face_similarity') AS REAL) >= ?
		   AND CAST(json_extract(face_match_result, '$.aadhaar_similarity') AS REAL) >= ?
		   AND COALESCE(json_extract(face_match_result, '$.second_reference_divergence'), 0) = 0
		   AND json_extract(liveness_result, '$.is_live') = 1
		   AND CAST(json_extract(liveness_result, '$.liveness_score') AS REAL) >= ?
		   AND CAST(json_extract(ocr_ocr_result, '$.ocr_confidence') AS REAL) >= ?
		   AND json_extract(ocr_ocr_result, '$.document_validation.overall') = 'verified'`
}

// routerRejectGuardSQL mirrors the approval guard for the auto-reject path.
// Only a clean spoof signal (is_live=false and a score below the reject
// threshold) reaches 'rejected'; everything else is protected by the metric
// predicate falling through into the manual-review fallback.
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
		   AND liveness_result->>'is_live' = 'false'
		   AND (liveness_result->>'liveness_score')::float < ?`
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
		   AND json_extract(liveness_result, '$.is_live') = 0
		   AND CAST(json_extract(liveness_result, '$.liveness_score') AS REAL) < ?`
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
