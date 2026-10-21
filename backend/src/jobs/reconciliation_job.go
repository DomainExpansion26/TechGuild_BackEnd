package jobs

import (
	"log"
	"time"

	"techguild-backend/src/services"
)

func StartReconciliationJob(fundingSvc *services.MilestoneFundingService) {
	reconSvc := services.NewReconciliationService(fundingSvc)

	// Run every 10 minutes
	ticker := time.NewTicker(10 * time.Minute)

	go func() {
		for range ticker.C {
			reconSvc.ReconcilePendingPayments()
		}
	}()

	log.Println("[Job] Reconciliation job started (Interval: 10 mins)")
}
