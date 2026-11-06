package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type VerificationType string

const (
	VerificationIndividual VerificationType = "individual"
	VerificationBusiness   VerificationType = "business"
)

type VerificationStatus string

const (
	VerificationDraft    VerificationStatus = "draft"
	VerificationPending  VerificationStatus = "pending"
	VerificationReview   VerificationStatus = "under_review"
	VerificationApproved VerificationStatus = "approved"
	VerificationRejected VerificationStatus = "rejected"
)

// ---- AI/ML Pipeline Decision States ----

type VerificationReviewPriority string

const (
	ReviewPriorityAutoEligible  VerificationReviewPriority = "auto_eligible"   // AI-high confidence; eligible for bulk approve
	ReviewPriorityManual        VerificationReviewPriority = "manual_review"   // borderline; human must check
	ReviewPriorityAutoReject    VerificationReviewPriority = "auto_reject"     // hard fraud signal; auto-fail
	ReviewPriorityAutoApproved  VerificationReviewPriority = "auto_approved"   // passed the 24-48h auto-approve window
)

type MLVerificationStatus string

const (
	MLStatusIdle       MLVerificationStatus = "idle"
	MLStatusQueued     MLVerificationStatus = "queued"
	MLStatusProcessing MLVerificationStatus = "processing"
	MLStatusCompleted  MLVerificationStatus = "ml_completed"
	MLStatusFailed     MLVerificationStatus = "ml_failed"
)

// RoutingState values (Phase 3): the Go router owns the queued -> routed
// transition. 'pending' records await the router; 'routed' records have been
// processed by the router, the stale-pending escalator, or a pre-Phase-3
// writer (the default keeps existing rows invisible to the router).
const (
	RoutingPending string = "pending"
	RoutingRouted  string = "routed"
)

type VerificationRecord struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	UserID uuid.UUID `gorm:"type:uuid;index;not null"`
	User   User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

	Type   VerificationType   `gorm:"size:30;not null"`
	Status VerificationStatus `gorm:"size:30;default:'draft'"`

	Vendor            string `gorm:"size:100"`
	VendorReferenceID string `gorm:"size:255"`

	// Deduplication Hashes (Unique constraints lagane ke liye)
	GovtIDHash      string `gorm:"size:64;index"`
	BusinessPANHash string `gorm:"size:64;index"`

	// Business Information
	BusinessName       string `gorm:"size:255"`
	GSTNumber          string `gorm:"size:50"`
	RegistrationNumber string `gorm:"size:100"`
	Website            string `gorm:"size:255"`
	Country            string `gorm:"size:100"`

	// Bank Details (Security: Account number MUST be encrypted)
	BankName          string `gorm:"size:100"`
	AccountHolderName string `gorm:"size:255"`
	AccountNumberEnc  string `gorm:"size:255"`
	IFSCCode          string `gorm:"size:50"`

	RejectionReason  string     `gorm:"type:text"`
	PreviousRecordID *uuid.UUID `gorm:"type:uuid"`
	VerifiedAt       *time.Time
	SubmittedAt      *time.Time

	// ---- NEW: AI/ML Pipeline Tracking ----
	MLStatus         MLVerificationStatus `gorm:"size:20;default:'idle'"`
	MLJobID          string               `gorm:"size:100"`
	MLSubmittedAt    *time.Time
	MLCompletedAt    *time.Time
	MLProcessingTime int                  `gorm:"default:0"`
	MLVendor         string               `gorm:"size:100"`
	MLModelVersion   string               `gorm:"size:50"`

	// ---- AI Score Results (JSONB) ----
	OCROcrResult      datatypes.JSON `gorm:"type:jsonb"`
	FaceMatchResult   datatypes.JSON `gorm:"type:jsonb"`
	LivenessResult    datatypes.JSON `gorm:"type:jsonb"`

	// ---- Human-in-the-Loop Decision State ----
	ReviewPriority         VerificationReviewPriority `gorm:"size:20;default:'manual_review'"`
	BulkReviewRequestedAt  *time.Time                   `gorm:"type:timestamp;index"`
	AdminReviewRequestedBy *uuid.UUID                   `gorm:"type:uuid"`
	AdminReviewRequestedAt *time.Time                   `gorm:"type:timestamp"`

	// ---- Phase 3: Go router state ----
	// Explicit column tags: the Python worker writes these columns by name,
	// so the GORM mapping must be unambiguous (no heuristic toDBName snake
	// casing on the AI* prefix).
	AIRecommendedPriority VerificationReviewPriority `gorm:"column:ai_recommended_priority;size:20;index"`
	RoutingState          string                     `gorm:"column:routing_state;size:20;not null;default:'routed';index"`
	ShadowModeApplied     bool                       `gorm:"column:shadow_mode_applied;not null;default:false"`
	DecisionSource        string                     `gorm:"column:decision_source;size:30"`
	PolicyVersion         string                     `gorm:"column:policy_version;size:50"`
	SystemEscalatedAt     *time.Time                 `gorm:"column:system_escalated_at;type:timestamp;index"`
	UserIDValidation      datatypes.JSON             `gorm:"column:user_id_validation;type:jsonb"`

	// ---- Auto-Decision Flags (24-48h window) ----
	AutoApprovedAt *time.Time `gorm:"type:timestamp"`
	AutoRejectedAt *time.Time `gorm:"type:timestamp"`
	BulkApprovedAt *time.Time `gorm:"type:timestamp"`
	RejectedAt     *time.Time `gorm:"type:timestamp"`

	// ---- Security & Audit ----
	MLOtherReasonCode string `gorm:"size:50"`

	// Documents is loaded on demand (e.g. GetVerificationDetail), not by default.
	Documents []VerificationDocument `gorm:"foreignKey:VerificationRecordID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (v *VerificationRecord) BeforeCreate(tx *gorm.DB) error {
	v.ID = uuid.New()
	return nil
}
