package jobs

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"
)

// StartChatCleanupJob initializes background cleanup routines for the ChatSystem
func StartChatCleanupJob(db *gorm.DB) func() {
	stopChan := make(chan struct{})

	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		log.Println("[ChatSystem Jobs] Cleanup job worker started (runs daily)")

		for {
			select {
			case <-ticker.C:
				runCleanup(db)
			case <-stopChan:
				log.Println("[ChatSystem Jobs] Cleanup job worker stopped")
				return
			}
		}
	}()

	return func() {
		close(stopChan)
	}
}

func runCleanup(db *gorm.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	log.Println("[ChatSystem Jobs] Running chat system maintenance...")

	// 1. Clean up abandoned draft channels or orphaned records older than 30 days
	cutoff := time.Now().AddDate(0, 0, -30)
	res := db.WithContext(ctx).
		Exec(`DELETE FROM message_reactions WHERE message_id IN (SELECT id FROM communication_records WHERE deleted = true AND created_at < ?)`, cutoff)
	if res.Error != nil {
		log.Printf("[ChatSystem Jobs] Error cleaning orphaned reactions: %v", res.Error)
	}

	log.Println("[ChatSystem Jobs] Maintenance finished")
}

