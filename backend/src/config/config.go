package config

// KYCPolicy configures the Go KYC router (verification_router.go).
//
// It MUST mirror the Python worker's config block (kyc-ml-service/config.py):
// the router re-checks the evidence Python already gated on, so any drift
// between the two env files silently changes who gets auto-approved.
// backend/.env.example and kyc-ml-service/.env.example carry the same keys.
type KYCPolicy struct {
	// ShadowMode: when true, the router routes EVERY record to
	// manual_review (Python writes ai_recommended_priority, Go decides).
	// Default true, matching Python's SHADOW_MODE.
	ShadowMode bool

	// RolloutCohortPct: deterministic shadow-off rollout cohort. 0 = nobody,
	// 100 = everybody. Bucket = HMAC-SHA256(user_id) % 100.
	RolloutCohortPct int

	// PolicyVersion stamps every router decision for auditability.
	PolicyVersion string

	// Thresholds — identical names/values to the Python settings.
	FaceAutoPass   float64
	FaceReview     float64
	FaceReject     float64
	LivenessPass   float64
	LivenessReject float64
	OCRPass        float64

	// AadhaarFloor (Phase 5): minimum aadhaar_similarity required for
	// auto-approval, even when PAN wins the multi-reference match. A fake
	// PAN cannot carry the score for an unverified Aadhaar. Mirrors the
	// Python-side divergence guardrail (defense in depth on both sides).
	//
	// NOTE (Phase B): the router no longer GATES on this value. The worker
	// runs with FACE_MATCH_ENABLED=false, so aadhaar_similarity is always nil
	// and a gate on it blocked every auto-approval. Phase B gates on
	// DocInSelfie* instead. The field is retained because the guard SQL and
	// existing deployments still carry THRESHOLD_AADHAAR_FLOOR.
	AadhaarFloor float64

	// Phase B — document-in-selfie gate. The worker OCRs the selfie, compares
	// the held document against the uploaded one, and writes
	// document_in_selfie_decision ("approve" | "manual" | "reject"),
	// document_in_selfie_score (int) and document_in_selfie_match (bool) into
	// face_match_result. This replaces aadhaar_similarity as the identity
	// anchor for auto-approval while face match is disabled.
	//
	// DocInSelfieApproveThreshold: minimum document_in_selfie_score that may
	// auto-approve (worker's "approve" verdict is a necessary condition too).
	DocInSelfieApproveThreshold int
	// DocInSelfieManualThreshold: score floor below which the verdict is
	// treated as manual review rather than approval. Reserved for the worker's
	// manual band; the router only enforces the approve threshold.
	DocInSelfieManualThreshold int
	// DocInSelfieRejectOnMismatch: when true (default), a hard
	// document_in_selfie_decision="reject" (Aadhaar number mismatch between the
	// uploaded document and the document held in the selfie) is an auto-reject
	// hard signal. When false it routes to manual review instead, so a
	// mis-OCR can never silently reject a legitimate user.
	DocInSelfieRejectOnMismatch bool
	// RequireQRVerified: when true (default), auto-approval additionally
	// requires document_validation.overall == "verified" (UIDAI QR
	// certificate). Set false when UIDAI QR certs are unavailable, so a QR
	// outage does not stall every legitimate approval. All other evidence
	// gates stay enforced either way.
	RequireQRVerified bool
}

// DefaultKYCPolicy returns the hardcoded defaults used when a service is
// constructed without LoadConfig (e.g. tests or internal tooling). They match
// the envDefault values registered in env.go.
func DefaultKYCPolicy() KYCPolicy {
	return KYCPolicy{
		ShadowMode:       true,
		RolloutCohortPct: 0,
		PolicyVersion:    "phase3-v1",
		FaceAutoPass:     0.363,
		FaceReview:       0.300,
		FaceReject:       0.240,
		LivenessPass:     0.90,
		LivenessReject:   0.50,
		OCRPass:          0.80,
		AadhaarFloor:     0.35,

		// Phase B document-in-selfie defaults (mirror env.go + the .env
		// templates). RequireQRVerified defaults true so a service built
		// without LoadConfig keeps the strict QR requirement.
		DocInSelfieApproveThreshold: 75,
		DocInSelfieManualThreshold:  25,
		DocInSelfieRejectOnMismatch: true,
		RequireQRVerified:           true,
	}
}

type Config struct {
	AppName string
	AppEnv  string
	Port    string

	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	RedisHost     string
	RedisPort     string
	RedisUsername string
	RedisPassword string

	JWTSecret        string
	JWTRefreshSecret string

	FrontendURL      string
	ZudokuURL        string
	GeoNamesUsername string

	SMTPHost     string
	SMTPPort     string
	SMTPEmail    string
	SMTPPassword string

	// Provider selection
	PaymentProvider string // "mock" | "razorpay"

	// Razorpay credentials
	// Razorpay credentials
	RazorpayKeyID         string
	RazorpayKeySecret     string
	RazorpayWebhookSecret string

	// RazorpayX (payouts)
	RazorpayXKeyID         string
	RazorpayXKeySecret     string
	RazorpayXAccountNumber string
	// NEW — Fee configuration
	ClientProcessingFeeBps     int64 // 500 = 5.00%
	FreelancerProcessingFeeBps int64 // 500 = 5.00%

	// KYC Phase 3: Go router policy + interval. Loaded from env in env.go.
	KYCPolicy             KYCPolicy
	RouterIntervalSeconds int

	// KYC Phase 4: client-IP resolution. Loaded from env in env.go.
	KYCIPPepper    string // env: KYC_IP_HASH_PEPPER, required in production
	TrustedProxies string // env: KYC_TRUSTED_PROXY_CIDRS, default ""

	// KYCDocRetentionDays is how long KYC document images (Aadhaar, PAN,
	// selfie, ...) may be kept after the owning record reached a terminal
	// state (approved/rejected). DPDP Act compliance. Loaded from env in
	// env.go; a non-positive value disables the cleanup job entirely.
	KYCDocRetentionDays int // env: KYC_DOC_RETENTION_DAYS, default 30
}
