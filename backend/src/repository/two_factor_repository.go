// repository/two_factor_repository.go
package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TwoFactorRepository interface {
	GetByUserID(userID uuid.UUID) (*models.UserTwoFactorAuthentication, error)
	Upsert(record *models.UserTwoFactorAuthentication) error
	UpdateStatus(userID uuid.UUID, updates map[string]interface{}) error

	CreateRecoveryCodes(codes []models.UserRecoveryCode) error
	GetUnusedRecoveryCodes(userID uuid.UUID) ([]models.UserRecoveryCode, error)
	MarkRecoveryCodeUsed(id uuid.UUID) error
	InvalidateRecoveryCodes(userID uuid.UUID) error
}

type twoFactorRepository struct {
	db *gorm.DB
}

func NewTwoFactorRepository() TwoFactorRepository {
	return &twoFactorRepository{db: postgres.DB}
}

func (r *twoFactorRepository) GetByUserID(userID uuid.UUID) (*models.UserTwoFactorAuthentication, error) {
	var record models.UserTwoFactorAuthentication
	err := r.db.Where("user_id = ?", userID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *twoFactorRepository) Upsert(record *models.UserTwoFactorAuthentication) error {
	return r.db.Save(record).Error
}

func (r *twoFactorRepository) UpdateStatus(userID uuid.UUID, updates map[string]interface{}) error {
	return r.db.Model(&models.UserTwoFactorAuthentication{}).
		Where("user_id = ?", userID).
		Updates(updates).Error
}

func (r *twoFactorRepository) CreateRecoveryCodes(codes []models.UserRecoveryCode) error {
	return r.db.Create(&codes).Error
}

func (r *twoFactorRepository) GetUnusedRecoveryCodes(userID uuid.UUID) ([]models.UserRecoveryCode, error) {
	var codes []models.UserRecoveryCode
	err := r.db.Where("user_id = ? AND is_used = false", userID).Find(&codes).Error
	return codes, err
}

func (r *twoFactorRepository) MarkRecoveryCodeUsed(id uuid.UUID) error {
	return r.db.Model(&models.UserRecoveryCode{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_used": true,
			"used_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *twoFactorRepository) InvalidateRecoveryCodes(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.UserRecoveryCode{}).Error
}
