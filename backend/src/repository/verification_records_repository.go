package repository

import (
	"time"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

type VerificationRecordsRepository interface {

	// Verification Records
	CreateVerification(record *models.VerificationRecord) error
	GetVerificationByUser(userID string) (*models.VerificationRecord, error)
	GetVerificationByID(id string) (*models.VerificationRecord, error)
	UpdateStatus(id string, status models.VerificationStatus, reason string) error
	GetPendingVerifications() ([]models.VerificationRecord, error)
	// Govt ID Dedup
	CreateGovtHash(hash *models.GovtIDDedup) error
	GovtHashExists(hash string) (bool, error)

	// Business PAN Dedup
	CreateBusinessHash(hash *models.BusinessPANDedup) error
	BusinessHashExists(hash string) (bool, error)

	// Verification Documents
	CreateDocument(document *models.VerificationDocument) error
	GetDocuments(recordID string) ([]models.VerificationDocument, error)
	//for admin for verification
	GetVerificationQueue() ([]models.VerificationRecord, error)
	ApproveVerification(id string) error
	RejectVerification(id string, reason string) error
}

type verificationRecordsRepository struct{}

func NewVerificationRecordsRepository() VerificationRecordsRepository {
	return &verificationRecordsRepository{}
}

// ---------------- Verification Records ----------------

func (r *verificationRecordsRepository) CreateVerification(record *models.VerificationRecord) error {
	return postgres.DB.Create(record).Error
}
func (r *verificationRecordsRepository) GetPendingVerifications() ([]models.VerificationRecord, error) {

	var records []models.VerificationRecord

	err := postgres.DB.
		Where("status = ?", models.VerificationReview).
		Order("created_at ASC").
		Find(&records).Error

	return records, err
}
func (r *verificationRecordsRepository) GetVerificationByUser(userID string) (*models.VerificationRecord, error) {

	var record models.VerificationRecord

	err := postgres.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&record).Error

	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *verificationRecordsRepository) GetVerificationByID(id string) (*models.VerificationRecord, error) {

	var record models.VerificationRecord

	err := postgres.DB.
		First(&record, "id = ?", id).Error

	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *verificationRecordsRepository) UpdateStatus(
	id string,
	status models.VerificationStatus,
	reason string,
) error {

	update := map[string]interface{}{
		"status": status,
	}

	if reason != "" {
		update["rejection_reason"] = reason
	}

	if status == models.VerificationApproved {
		now := time.Now()
		update["verified_at"] = &now
	}

	return postgres.DB.
		Model(&models.VerificationRecord{}).
		Where("id = ?", id).
		Updates(update).Error
}

// ---------------- Govt ID Dedup ----------------

func (r *verificationRecordsRepository) CreateGovtHash(hash *models.GovtIDDedup) error {
	return postgres.DB.Create(hash).Error
}

func (r *verificationRecordsRepository) GovtHashExists(hash string) (bool, error) {

	var count int64

	err := postgres.DB.
		Model(&models.GovtIDDedup{}).
		Where("govt_id_hash = ?", hash).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ---------------- Business PAN Dedup ----------------

func (r *verificationRecordsRepository) CreateBusinessHash(hash *models.BusinessPANDedup) error {
	return postgres.DB.Create(hash).Error
}

func (r *verificationRecordsRepository) BusinessHashExists(hash string) (bool, error) {

	var count int64

	err := postgres.DB.
		Model(&models.BusinessPANDedup{}).
		Where("business_pan_hash = ?", hash).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ---------------- Verification Documents ----------------

func (r *verificationRecordsRepository) CreateDocument(document *models.VerificationDocument) error {
	return postgres.DB.Create(document).Error
}

func (r *verificationRecordsRepository) GetDocuments(recordID string) ([]models.VerificationDocument, error) {

	var documents []models.VerificationDocument

	err := postgres.DB.
		Where("verification_record_id = ?", recordID).
		Find(&documents).Error

	if err != nil {
		return nil, err
	}

	return documents, nil
}
//method for admin to get the verification queue
func (r *verificationRecordsRepository) GetVerificationQueue() ([]models.VerificationRecord, error) {

	var records []models.VerificationRecord

	err := postgres.DB.
		Where("status = ?", models.VerificationReview).
		Preload("User").
		Order("created_at ASC").
		Find(&records).Error

	return records, err
}
//method for admin to approve and reject the verification
func (r *verificationRecordsRepository) ApproveVerification(id string) error {

	return postgres.DB.Model(&models.VerificationRecord{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      models.VerificationApproved,
			"verified_at": time.Now(),
		}).Error
}

func (r *verificationRecordsRepository) RejectVerification(id string, reason string) error {

	return postgres.DB.Model(&models.VerificationRecord{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":            models.VerificationRejected,
			"rejection_reason":  reason,
		}).Error
}