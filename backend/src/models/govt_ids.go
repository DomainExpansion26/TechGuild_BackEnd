package models
import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)
type GovernmentIDType string

const (
	IDAadhaar GovernmentIDType = "aadhaar"
	IDPAN     GovernmentIDType = "pan"
	IDPassport GovernmentIDType = "passport"
	IDDrivingLicense GovernmentIDType = "driving_license"
)
type GovernmentID struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	IDType GovernmentIDType `gorm:"size:30;not null"`
	DocumentHash string `gorm:"size:255;uniqueIndex;not null"`
	LastFourDigits string `gorm:"size:4"`
	IsVerified bool `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (g *GovernmentID) BeforeCreate(tx *gorm.DB) error {
	g.ID = uuid.New()
	return nil
}