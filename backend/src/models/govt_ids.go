package models
// GovernmentID stores the verified government identity of a user.
// This table only contains successfully verified IDs.
// Raw government ID numbers are never stored.
import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GovernmentIDType string

const (
	IDAadhaar         GovernmentIDType = "aadhaar"
	IDPAN             GovernmentIDType = "pan"
	IDPassport        GovernmentIDType = "passport"
	IDDrivingLicense  GovernmentIDType = "driving_license"
)

type GovernmentID struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	UserID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"`
	User   User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`

	IDType GovernmentIDType `gorm:"size:30;not null"`
	// SHA-256 hash of the government ID number
	GovtIDHash string `gorm:"type:varchar(255);uniqueIndex;not null"`
	// Last 4 digits displayed to the user
	LastFourDigits string `gorm:"size:4"`
	// Verification record that approved this ID
	VerificationRecordID uuid.UUID `gorm:"type:uuid;not null"`

	VerifiedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (g *GovernmentID) BeforeCreate(tx *gorm.DB) error {
	g.ID = uuid.New()
	return nil
}