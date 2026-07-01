package models
import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)
type PolicyChangeNotification struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;index;not null"`
	RuleDocumentID uuid.UUID `gorm:"type:uuid;not null"`
	Message string `gorm:"type:text"`
	IsRead bool `gorm:"default:false"`
	SentAt time.Time
	ReadAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
func (p *PolicyChangeNotification) BeforeCreate(tx *gorm.DB) error {
	p.ID = uuid.New()
	return nil
}