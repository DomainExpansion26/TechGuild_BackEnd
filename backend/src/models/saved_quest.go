package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SavedQuest is a freelancer's bookmark on a quest. One row per (user, quest).
type SavedQuest struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	UserID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uniq_saved_quest_user,priority:1" json:"userId"`
	QuestID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uniq_saved_quest_user,priority:2" json:"questId"`

	CreatedAt time.Time `json:"createdAt"`
}

func (s *SavedQuest) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
