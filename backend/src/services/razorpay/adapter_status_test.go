package razorpay

import (
	"testing"

	"techguild-backend/src/models"
)

func TestNormalizePayoutFundAccountStatus(t *testing.T) {
	cases := map[string]string{
		"active":  string(models.PayoutMethodStatusVerified),
		"":        string(models.PayoutMethodStatusPendingVerification),
		"created": string(models.PayoutMethodStatusPendingVerification),
		"failed":  string(models.PayoutMethodStatusPendingVerification),
	}
	for raw, want := range cases {
		if got := normalizePayoutFundAccountStatus(raw); got != want {
			t.Errorf("normalizePayoutFundAccountStatus(%q) = %q, want %q", raw, got, want)
		}
	}
}
