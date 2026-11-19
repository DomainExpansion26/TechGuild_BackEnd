package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GuildhallComment struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	PostID uuid.UUID     `gorm:"type:uuid;not null;index"`
	Post   GuildhallPost `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`

	AuthorID uuid.UUID `gorm:"type:uuid;not null;index"`
	Author   User      `gorm:"foreignKey:AuthorID;constraint:OnDelete:CASCADE"`

	Content string `gorm:"type:varchar(5000);not null"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (c *GuildhallComment) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
