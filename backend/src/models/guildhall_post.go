package models

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/datatypes"
    "gorm.io/gorm"
)

type GuildhallPostType string

const (
    GuildhallPostDiscussion      GuildhallPostType = "discussion"
    GuildhallPostQuestion        GuildhallPostType = "question"
    GuildhallPostProjectShowcase GuildhallPostType = "project_showcase"
    GuildhallPostPoll            GuildhallPostType = "poll"
    GuildhallPostOpportunity     GuildhallPostType = "opportunity"
    GuildhallPostMedia           GuildhallPostType = "media"
)

type GuildhallPost struct {
    ID uuid.UUID `gorm:"type:uuid;primaryKey"`

    AuthorID uuid.UUID `gorm:"type:uuid;not null;index"`
    Author   User      `gorm:"foreignKey:AuthorID;constraint:OnDelete:CASCADE"`

    Type    GuildhallPostType `gorm:"type:varchar(30);not null;index"`
    Title   string            `gorm:"type:varchar(200)"`
    Content string            `gorm:"type:text;not null"`
    LinkURL string            `gorm:"type:varchar(2048)"`
    Details datatypes.JSON    `gorm:"type:jsonb"`

    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (p *GuildhallPost) BeforeCreate(tx *gorm.DB) error {
    if p.ID == uuid.Nil {
        p.ID = uuid.New()
    }
    return nil
}