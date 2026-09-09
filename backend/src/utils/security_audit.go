package utils

import (
	"log"
	"time"
)

type SecurityEventType string

const (
	EventTwoFAEnabled        SecurityEventType = "2fa_enabled"
	EventTwoFADisabled       SecurityEventType = "2fa_disabled"
	EventTwoFASetupInitiated SecurityEventType = "2fa_setup_initiated"
	EventTwoFAVerifyFailed   SecurityEventType = "2fa_verify_failed"
	EventTwoFALoginSuccess   SecurityEventType = "2fa_login_success"
	EventRecoveryCodeUsed    SecurityEventType = "2fa_recovery_code_used"
	EventRecoveryCodesRegen  SecurityEventType = "2fa_recovery_codes_regenerated"
)

// LogSecurityEvent writes a structured audit line — never pass secrets, OTP codes, or raw tokens here
func LogSecurityEvent(eventType SecurityEventType, userID string, meta string) {
	log.Printf(
		"[SECURITY_AUDIT] time=%s event%s user_id=%s meta=%s",
		time.Now().UTC().Format(time.RFC3339),
		eventType,
		userID,
		meta,
	)
}
