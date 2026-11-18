package jobs

import (
	"context"
	"log"
	"time"

	"techguild-backend/src/services"
)

// kycRouterBatchSize caps one RouteMLCompleted claim per tick (records are
// row-locked for the duration of the batch transaction).
const kycRouterBatchSize = 50

// kycRouterDefaultIntervalSeconds stays in one place so a zero/negative env
// value cannot spin the router into a hot loop.
const kycRouterDefaultIntervalSeconds = 60

// StartKYCRouter launches the Phase 3 KYC routing cron in a background
// goroutine: run once at boot (clears the backlog that accumulated while
// records were queued), then every intervalSeconds. Non-blocking — the server
// keeps starting while this runs (matches StartStalePendingEscalator's
// pattern). ctx lets a host that owns shutdown cancel the loop.
func StartKYCRouter(ctx context.Context, verificationService *services.VerificationService, intervalSeconds, batchSize int) {
	if ctx == nil {
		ctx = context.Background()
	}
	if intervalSeconds <= 0 {
		intervalSeconds = kycRouterDefaultIntervalSeconds
	}
	if batchSize <= 0 {
		batchSize = kycRouterBatchSize
	}

	log.Printf("[KYC Router] starting router job (every %ds, batch=%d)", intervalSeconds, batchSize)

	go func() {
		runKYCRouterTick(ctx, verificationService, batchSize)

		ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Printf("[KYC Router] context done; router job stopping")
				return
			case <-ticker.C:
				runKYCRouterTick(ctx, verificationService, batchSize)
			}
		}
	}()
}

// runKYCRouterTick is the tick body; panic recovery so one bad record can
// never kill the job loop for the lifetime of the process.
func runKYCRouterTick(ctx context.Context, verificationService *services.VerificationService, batchSize int) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[KYC Router] panic recovered, job will continue next tick: %v", r)
		}
	}()

	select {
	case <-ctx.Done():
		return
	default:
	}

	routed, approved, rejected, manual, err := verificationService.RouteMLCompleted(batchSize)
	if err != nil {
		log.Printf("[KYC Router] batch error: %v", err)
		return
	}
	if routed > 0 {
		log.Printf("[KYC Router] batch routed=%d approved=%d rejected=%d manual=%d",
			routed, approved, rejected, manual)
	}
}
