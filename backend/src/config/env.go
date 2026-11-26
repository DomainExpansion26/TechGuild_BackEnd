package config

import (
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

func LoadConfig() (*Config, error) {

	// Defaults MUST be registered before reading .env so a partial .env still
	// yields safe values. KYC_SHADOW_MODE defaults to TRUE, matching the
	// Python worker's SHADOW_MODE default. If one side is flipped without the
	// other the router overrides Python's decision incorrectly —
	// LogKYCShadowModeAssertion warns about that at startup.
	viper.SetDefault("KYC_SHADOW_MODE", true)
	viper.SetDefault("KYC_ROLLOUT_COHORT_PCT", 0)
	viper.SetDefault("KYC_POLICY_VERSION", "phase3-v1")
	viper.SetDefault("KYC_ROUTER_INTERVAL_SECONDS", 60)
	// DPDP Act: KYC document images are deleted 30 days after the record
	// reaches a terminal state. A non-positive value disables the job.
	viper.SetDefault("KYC_DOC_RETENTION_DAYS", 30)

	viper.SetDefault("THRESHOLD_FACE_AUTO_PASS", 0.363)
	viper.SetDefault("THRESHOLD_FACE_REVIEW", 0.300)
	viper.SetDefault("THRESHOLD_FACE_REJECT", 0.240)
	viper.SetDefault("THRESHOLD_LIVE_PASS", 0.90)
	viper.SetDefault("THRESHOLD_LIVE_REJECT", 0.50)
	viper.SetDefault("THRESHOLD_OCR_PASS", 0.80)
	viper.SetDefault("THRESHOLD_AADHAAR_FLOOR", 0.35)

	// Phase B document-in-selfie gate. Registered here for the same reason as
	// the thresholds above: a partial .env must still yield safe values.
	// DOC_IN_SELFIE_REJECT_ON_MISMATCH and THRESHOLD_REQUIRE_QR default TRUE,
	// the strict behaviour; THRESHOLD_REQUIRE_QR=false is an explicit operator
	// opt-out for QR-certificate outages.
	viper.SetDefault("DOC_IN_SELFIE_APPROVE_THRESHOLD", 75)
	viper.SetDefault("DOC_IN_SELFIE_MANUAL_THRESHOLD", 25)
	viper.SetDefault("DOC_IN_SELFIE_REJECT_ON_MISMATCH", true)
	viper.SetDefault("THRESHOLD_REQUIRE_QR", true)

	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()

	cfg := &Config{
		AppName: viper.GetString("APP_NAME"),
		AppEnv:  viper.GetString("APP_ENV"),
		Port:    viper.GetString("PORT"),

		DBHost:     viper.GetString("DB_HOST"),
		DBPort:     viper.GetString("DB_PORT"),
		DBName:     viper.GetString("DB_NAME"),
		DBUser:     viper.GetString("DB_USER"),
		DBPassword: viper.GetString("DB_PASSWORD"),
		DBSSLMode:  viper.GetString("DB_SSLMODE"),

		RedisHost:     viper.GetString("REDIS_HOST"),
		RedisPort:     viper.GetString("REDIS_PORT"),
		RedisUsername: viper.GetString("REDIS_USERNAME"),
		RedisPassword: viper.GetString("REDIS_PASSWORD"),

		FrontendURL:      viper.GetString("FRONTEND_URL"),
		ZudokuURL:        viper.GetString("ZUDOKU_URL"),
		GeoNamesUsername: viper.GetString("GEONAMES_USERNAME"),

		SMTPHost:     viper.GetString("SMTP_HOST"),
		SMTPPort:     viper.GetString("SMTP_PORT"),
		SMTPEmail:    viper.GetString("SMTP_EMAIL"),
		SMTPPassword: viper.GetString("SMTP_PASSWORD"),

		PaymentProvider:       viper.GetString("PAYMENT_PROVIDER"),
		RazorpayKeyID:         viper.GetString("RAZORPAY_KEY_ID"),
		RazorpayKeySecret:     viper.GetString("RAZORPAY_KEY_SECRET"),
		RazorpayWebhookSecret: viper.GetString("RAZORPAY_WEBHOOK_SECRET"),

		RazorpayXKeyID:         viper.GetString("RAZORPAYX_KEY_ID"),
		RazorpayXKeySecret:     viper.GetString("RAZORPAYX_KEY_SECRET"),
		RazorpayXAccountNumber: viper.GetString("RAZORPAYX_ACCOUNT_NUMBER"),

		ClientProcessingFeeBps:     viper.GetInt64("CLIENT_PROCESSING_FEE_BPS"),
		FreelancerProcessingFeeBps: viper.GetInt64("FREELANCER_PROCESSING_FEE_BPS"),

		KYCPolicy: KYCPolicy{
			ShadowMode:       viper.GetBool("KYC_SHADOW_MODE"),
			RolloutCohortPct: viper.GetInt("KYC_ROLLOUT_COHORT_PCT"),
			PolicyVersion:    viper.GetString("KYC_POLICY_VERSION"),
			FaceAutoPass:     viper.GetFloat64("THRESHOLD_FACE_AUTO_PASS"),
			FaceReview:       viper.GetFloat64("THRESHOLD_FACE_REVIEW"),
			FaceReject:       viper.GetFloat64("THRESHOLD_FACE_REJECT"),
			LivenessPass:     viper.GetFloat64("THRESHOLD_LIVE_PASS"),
			LivenessReject:   viper.GetFloat64("THRESHOLD_LIVE_REJECT"),
			OCRPass:          viper.GetFloat64("THRESHOLD_OCR_PASS"),
			AadhaarFloor:     viper.GetFloat64("THRESHOLD_AADHAAR_FLOOR"),

			DocInSelfieApproveThreshold: getIntWithDefault("DOC_IN_SELFIE_APPROVE_THRESHOLD", 75),
			DocInSelfieManualThreshold:  getIntWithDefault("DOC_IN_SELFIE_MANUAL_THRESHOLD", 25),
			DocInSelfieRejectOnMismatch: viper.GetBool("DOC_IN_SELFIE_REJECT_ON_MISMATCH"),
			RequireQRVerified:           viper.GetBool("THRESHOLD_REQUIRE_QR"),
		},
		RouterIntervalSeconds: viper.GetInt("KYC_ROUTER_INTERVAL_SECONDS"),

		// getIntWithDefault, not GetInt: an unset value must not collapse to 0
		// and busy-loop the reconciliation job.
		ReconciliationIntervalSeconds: getIntWithDefault("RECONCILIATION_INTERVAL_SECONDS", 180),

		KYCIPPepper:    os.Getenv("KYC_IP_HASH_PEPPER"),
		TrustedProxies: os.Getenv("KYC_TRUSTED_PROXY_CIDRS"),

		// DPDP Act retention window. getIntWithDefault matters here: an
		// unset KYC_DOC_RETENTION_DAYS must NOT collapse to 0, which would
		// silently disable the cleanup job. A declared 0 still disables it
		// deliberately (see jobs.StartKYCDocumentCleanup).
		KYCDocRetentionDays: getIntWithDefault("KYC_DOC_RETENTION_DAYS", 30),
	}

	// Phase 4: fail closed in production when the IP-hash pepper is absent.
	// Development (and test) processes may run without it.
	if cfg.KYCIPPepper == "" && isProductionEnv() {
		log.Fatal(
			"[config] KYC_IP_HASH_PEPPER must be set in production " +
				"(generate with: openssl rand -hex 32)",
		)
	}

	LogKYCShadowModeAssertion(cfg.KYCPolicy)

	return cfg, nil
}

// isProductionEnv reports whether the process is running in production.
// Production is signalled by APP_ENV=production (see .env.production.example)
// or GIN_MODE=release.
func isProductionEnv() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("GIN_MODE")), "release")
}

// getIntWithDefault reads an int key, falling back to the registered default
// when the key is absent from BOTH the environment and the .env file.
//
// viper.IsSet() reports true only for keys supplied by a config file or the
// environment; a value that exists solely as a viper.SetDefault is NOT "set".
// So an operator who never declares DOC_IN_SELFIE_APPROVE_THRESHOLD gets the
// explicit fallback (75) rather than GetInt's zero — a zero threshold would
// silently reduce the approval bar to "score > 0". A declared 0 is honoured.
func getIntWithDefault(key string, fallback int) int {
	if !viper.IsSet(key) {
		return fallback
	}
	return viper.GetInt(key)
}

// LogKYCShadowModeAssertion prints the Go shadow-mode value next to the Python
// worker's SHADOW_MODE whenever the backend process carries it (shared-env
// deployments). The two .env files live in separate directories, so this only
// catches mismatches that are visible to this process — operators must still
// keep backend/.env and kyc-ml-service/.env in sync by hand.
func LogKYCShadowModeAssertion(p KYCPolicy) {
	log.Printf(
		"[KYC Router] policy shadow_mode=%v rollout_cohort_pct=%d policy_version=%s thresholds(face_auto_pass=%.3f live_pass=%.2f live_reject=%.2f ocr_pass=%.2f aadhaar_floor=%.2f)",
		p.ShadowMode, p.RolloutCohortPct, p.PolicyVersion,
		p.FaceAutoPass, p.LivenessPass, p.LivenessReject, p.OCRPass, p.AadhaarFloor,
	)

	if raw, ok := os.LookupEnv("SHADOW_MODE"); ok {
		pyTrue := raw == "1" || strings.EqualFold(strings.TrimSpace(raw), "true") ||
			strings.EqualFold(strings.TrimSpace(raw), "yes") ||
			strings.EqualFold(strings.TrimSpace(raw), "on")
		if pyTrue != p.ShadowMode {
			log.Printf(
				"[KYC Router] SHADOW MODE MISMATCH: go KYC_SHADOW_MODE=%v vs python SHADOW_MODE=%s — fix before enabling rollout, the router would override Python's routing",
				p.ShadowMode, raw,
			)
		}
	}

	if !p.ShadowMode {
		log.Printf(
			"[KYC Router] WARNING: shadow mode is OFF — the router will approve/reject records for rollout cohort %d%%; ensure kyc-ml-service SHADOW_MODE is set to the same value",
			p.RolloutCohortPct,
		)
	}
}
