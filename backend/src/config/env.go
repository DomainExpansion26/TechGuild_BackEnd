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

	viper.SetDefault("THRESHOLD_FACE_AUTO_PASS", 0.363)
	viper.SetDefault("THRESHOLD_FACE_REVIEW", 0.300)
	viper.SetDefault("THRESHOLD_FACE_REJECT", 0.240)
	viper.SetDefault("THRESHOLD_LIVE_PASS", 0.90)
	viper.SetDefault("THRESHOLD_LIVE_REJECT", 0.50)
	viper.SetDefault("THRESHOLD_OCR_PASS", 0.80)
	viper.SetDefault("THRESHOLD_AADHAAR_FLOOR", 0.35)

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

		JWTSecret:        viper.GetString("JWT_SECRET"),
		JWTRefreshSecret: viper.GetString("JWT_REFRESH_SECRET"),

		FrontendURL: viper.GetString("FRONTEND_URL"),
		ZudokuURL:   viper.GetString("ZUDOKU_URL"),

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
		},
		RouterIntervalSeconds: viper.GetInt("KYC_ROUTER_INTERVAL_SECONDS"),
	}

	LogKYCShadowModeAssertion(cfg.KYCPolicy)

	return cfg, nil
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
