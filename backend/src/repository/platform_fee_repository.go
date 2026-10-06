package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlatformFeeRepository interface {
	Create(fee *models.PlatformFee) error
	GetByID(id uuid.UUID) (*models.PlatformFee, error)
	GetByMilestonePaymentID(mpID uuid.UUID) (*models.PlatformFee, error)
	UpdateStatus(id uuid.UUID, status models.PlatformFeeStatus) error
	MarkSettled(id uuid.UUID) error

	WithTransaction(fn func(txRepo PlatformFeeRepository) error) error
}

type platformFeeRepository struct {
	db *gorm.DB
}

func NewPlatformFeeRepository() PlatformFeeRepository {
	return &platformFeeRepository{db: postgres.DB}
}

func NewPlatformFeeRepositoryTx(tx *gorm.DB) PlatformFeeRepository {
	return &platformFeeRepository{db: tx}
}

func (r *platformFeeRepository) WithTransaction(fn func(txRepo PlatformFeeRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewPlatformFeeRepositoryTx(tx))
	})
}

func (r *platformFeeRepository) Create(fee *models.PlatformFee) error {
	return r.db.Create(fee).Error
}

func (r *platformFeeRepository) GetByID(id uuid.UUID) (*models.PlatformFee, error) {
	var f models.PlatformFee
	err := r.db.Where("id = ?", id).First(&f).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *platformFeeRepository) GetByMilestonePaymentID(mpID uuid.UUID) (*models.PlatformFee, error) {
	var f models.PlatformFee
	err := r.db.Where("milestone_payment_id = ?", mpID).First(&f).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *platformFeeRepository) UpdateStatus(id uuid.UUID, status models.PlatformFeeStatus) error {
	res := r.db.
		Model(&models.PlatformFee{}).
		Where("id = ?", id).
		Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *platformFeeRepository) MarkSettled(id uuid.UUID) error {
	res := r.db.
		Model(&models.PlatformFee{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     models.PlatformFeeSettled,
			"settled_at": gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
