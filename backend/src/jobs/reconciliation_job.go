package jobs

import (
	"log"
	"time"

	"techguild-backend/src/config"
	"techguild-backend/src/services"
)

func StartReconciliationJob(cfg *config.Config, fundingSvc *services.MilestoneFundingService) {
	reconSvc := services.NewReconciliationService(cfg, fundingSvc)

	interval := cfg.ReconciliationIntervalSeconds
	if interval <= 0 {
		interval = 180
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)

	go func() {
		for range ticker.C {
			reconSvc.ReconcilePendingPayments()
		}
	}()

	log.Printf("[Job] Reconciliation job started (interval %ds)", interval)
}
