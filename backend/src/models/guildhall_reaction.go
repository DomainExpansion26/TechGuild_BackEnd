package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GuildhallReaction struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	PostID uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex:uidx_guildhall_reaction_post_user,priority:1"`
	Post   GuildhallPost `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`

	UserID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uidx_guildhall_reaction_post_user,priority:2"`
	User   User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

	CreatedAt time.Time
}

func (r *GuildhallReaction) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
