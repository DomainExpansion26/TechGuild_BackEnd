package models
import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)
type DocumentType string
const (
	DocumentTypeTerms     DocumentType = "terms"
	DocumentTypePrivacy   DocumentType = "privacy"
	DocumentTypeBusiness  DocumentType = "business"
	DocumentTypeCommunity DocumentType = "community"
)
type RuleDocument struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`
	Title string `gorm:"size:255;not null"`
	DocumentType DocumentType `gorm:"size:50;not null"`
	Version string `gorm:"size:20;not null"`
	Content string `gorm:"type:text;not null"`
	IsActive bool `gorm:"default:true"`
	PublishedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
func (r *RuleDocument) BeforeCreate(tx *gorm.DB) error {
	r.ID = uuid.New()
	return nil
}