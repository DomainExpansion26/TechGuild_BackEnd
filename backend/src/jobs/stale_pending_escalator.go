package jobs

import (
	"log"
	"time"

	"techguild-backend/src/services"
)

// stalePendingEscalationInterval: how often the escalator scans for auto_eligible
// records whose hold window expired without a decision. 15 minutes keeps the
// manual-review handoff prompt without ever auto-approving a record.
const stalePendingEscalationInterval = 15 * time.Minute

// StartStalePendingEscalator launches the KYC stale-pending escalator cron in a
// background goroutine: run once at boot (clears overnight backlog), then every
// stalePendingEscalationInterval. Non-blocking — the server keeps starting while
// this runs.
func StartStalePendingEscalator(verificationService *services.VerificationService) {
	log.Printf("[KYC Escalator] starting stale-pending escalator job (every %s)", stalePendingEscalationInterval)

	go func() {
		runStalePendingEscalator(verificationService)

		ticker := time.NewTicker(stalePendingEscalationInterval)
		defer ticker.Stop()
		for range ticker.C {
			runStalePendingEscalator(verificationService)
		}
	}()
}

// runStalePendingEscalator is the tick body; panic recovery so one bad record
// can never kill the job loop for the lifetime of the process.
func runStalePendingEscalator(verificationService *services.VerificationService) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[KYC Escalator] panic recovered, job will continue next tick: %v", r)
		}
	}()

	verificationService.ProcessStalePendingEscalations()
}
