package services

import (
	"errors"
	"fmt"
	"testing"
)

// Guards the branch that decides whether a failed milestone refund blocks the
// whole quest cancel. A real provider failure must abort; "nothing to refund
// here" must not.
func TestSkippableCancelRefundError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"refund already in progress", ErrRefStateRace, true},
		{"already refunded", ErrRefAlreadyRefunded, true},
		{"state moved on concurrently", ErrRefInvalidState, true},
		{"escrow never created", ErrRefEscrowNotFound, true},
		{"only failed funding attempts", ErrRefPaymentNotFound, true},
		{"escrow not held", ErrRefEscrowNotHeld, true},
		{"wrapped race still skippable", fmt.Errorf("milestone x: %w", ErrRefStateRace), true},
		{"provider failure must abort", ErrRefProviderFailed, false},
		{"unknown error must abort", errors.New("boom"), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := skippableCancelRefundError(tc.err); got != tc.want {
				t.Fatalf("skippableCancelRefundError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
