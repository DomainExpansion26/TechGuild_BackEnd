package jobs

import (
	"log"
	"time"

	"techguild-backend/src/services"
)

func StartReconciliationJob(fundingSvc *services.MilestoneFundingService) {
	reconSvc := services.NewReconciliationService(fundingSvc)

	// Production me 10 * time.Minute karna
	ticker := time.NewTicker(10 * time.Second)

	go func() {
		for range ticker.C {
			reconSvc.ReconcilePendingPayments()
		}
	}()

	log.Println("[Job] Reconciliation job started")
}
