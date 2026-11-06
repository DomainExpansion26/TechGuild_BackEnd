package models

import (
	"log"

	"gorm.io/gorm"
)

// AutoMigrate applies schema migrations for all ChatSystem models
func AutoMigrate(db *gorm.DB) error {
	log.Println("[ChatSystem] Running database auto-migration...")

	err := db.AutoMigrate(
		&ChatChannel{},
		&ChannelMember{},
		&CommunicationRecord{},
		&MessageReaction{},
	)
	if err != nil {
		log.Printf("[ChatSystem] Migration error: %v", err)
		return err
	}

	// Ensure composite unique index for message_reactions (message_id, user_id, emoji)
	_ = db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_message_user_emoji 
		ON message_reactions (message_id, user_id, emoji);
	`).Error

	// Ensure composite unique index for channel_members (channel_id, user_id)
	_ = db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_channel_user 
		ON channel_members (channel_id, user_id);
	`).Error

	// Index for pinned messages retrieval
	_ = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_comm_channel_pinned 
		ON communication_records (channel_id, is_pinned) 
		WHERE is_pinned = true AND deleted = false;
	`).Error

	// Index for cursor pagination (channel_id, created_at DESC)
	_ = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_comm_channel_created 
		ON communication_records (channel_id, created_at DESC);
	`).Error

	log.Println("[ChatSystem] Database auto-migration completed successfully")
	return nil
}

