package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Document Types Constants
//
// Phase 5 splits the individual government ID into its components: aadhaar is
// the QR-verified trust anchor, pan_card is an optional corroborating
// reference. govt_id remains a legacy alias accepted for backward
// compatibility (no migration of historical rows).
const (
	DocTypeAadhaar         = "aadhaar" // NEW (Phase 5): QR-verified trust anchor
	DocTypeGovtID          = "govt_id" // legacy single-ID alias; back-compat
	DocTypeSelfie          = "selfie"
	DocTypeGSTCert         = "gst_certificate"
	DocTypeIncCert         = "incorporation_certificate"
	DocTypePANCard         = "pan_card"
	DocTypeBizReg          = "business_registration"
	DocTypeCancelledCheque = "cancelled_cheque"
)

// RequiredDocTypesFor returns the set of doc types that must be present for a
// given verification type to be considered complete (Phase 5).
//
// Individual KYC: Aadhaar is required; PAN is optional (a second reference
// improves face matching but can never replace the QR-verified anchor).
// Business KYC: PAN is required (existing behaviour).
//
// Legacy records with doc_type='govt_id' are intentionally NOT migrated — the
// worker treats govt_id as "unknown, single-reference fallback" and the router
// routes them to manual review. Phase 6 evaluation decides any backfill.
func RequiredDocTypesFor(verificationType VerificationType) []string {
	switch verificationType {
	case VerificationIndividual:
		return []string{DocTypeAadhaar}
	case VerificationBusiness:
		return []string{DocTypePANCard}
	default:
		return nil
	}
}

type VerificationDocument struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	VerificationRecordID uuid.UUID          `gorm:"type:uuid;index;not null"`
	VerificationRecord   VerificationRecord `gorm:"foreignKey:VerificationRecordID;constraint:OnDelete:CASCADE"`

	DocumentType string `gorm:"size:50;not null"`

	// Cloudinary secure URL
	FileURL string `gorm:"type:text;not null"`

	// Cloudinary Public ID (used for delete/update)
	PublicID string `gorm:"size:255"`

	// image/jpeg, image/png, application/pdf
	ContentType string `gorm:"size:100"`

	CreatedAt time.Time
}

func (v *VerificationDocument) BeforeCreate(tx *gorm.DB) error {
	v.ID = uuid.New()
	return nil
}
