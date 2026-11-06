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
	AadhaarFloor float64
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

	FrontendURL string
	ZudokuURL   string

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
}
