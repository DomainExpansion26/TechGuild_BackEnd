package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GuildhallVote struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	PostID uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex:uidx_guildhall_vote_post_user,priority:1"`
	Post   GuildhallPost `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`

	UserID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uidx_guildhall_vote_post_user,priority:2"`
	User   User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

	Option    string `gorm:"type:text;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (v *GuildhallVote) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}
