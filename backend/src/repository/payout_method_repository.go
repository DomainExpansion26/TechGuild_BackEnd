package repository

import (
	"errors"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PayoutMethodRepository interface {
	Create(method *models.PayoutMethod) error
	GetByID(id uuid.UUID) (*models.PayoutMethod, error)
	GetByIDAndUser(id, userID uuid.UUID) (*models.PayoutMethod, error)
	ListByUser(userID uuid.UUID) ([]models.PayoutMethod, error)

	// SetPrimary atomically un-sets any existing primary of the same
	// (purpose, type) for this user, then sets target as primary.
	SetPrimary(methodID, userID uuid.UUID) error

	Delete(id, userID uuid.UUID) error
	ExistsByProviderMethodID(provider, providerMethodID string) (bool, error)
	// NEW — webhook ke liye
	GetByProviderMethodID(provider, providerMethodID string) (*models.PayoutMethod, error)
	UpdateStatus(id uuid.UUID, status models.PayoutMethodStatus) error
	GetPrimaryByUser(userID uuid.UUID, purpose models.PayoutMethodPurpose) (*models.PayoutMethod, error)

	WithTransaction(fn func(txRepo PayoutMethodRepository) error) error
}

type payoutMethodRepository struct {
	db *gorm.DB
}

func NewPayoutMethodRepository() PayoutMethodRepository {
	return &payoutMethodRepository{db: postgres.DB}
}

func NewPayoutMethodRepositoryTx(tx *gorm.DB) PayoutMethodRepository {
	return &payoutMethodRepository{db: tx}
}

func (r *payoutMethodRepository) WithTransaction(fn func(txRepo PayoutMethodRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewPayoutMethodRepositoryTx(tx))
	})
}

func (r *payoutMethodRepository) Create(method *models.PayoutMethod) error {
	return r.db.Create(method).Error
}

func (r *payoutMethodRepository) GetByID(id uuid.UUID) (*models.PayoutMethod, error) {
	var method models.PayoutMethod
	err := r.db.Where("id = ?", id).First(&method).Error
	if err != nil {
		return nil, err
	}
	return &method, nil
}

func (r *payoutMethodRepository) GetByIDAndUser(id, userID uuid.UUID) (*models.PayoutMethod, error) {
	var method models.PayoutMethod
	err := r.db.
		Where("id = ? AND user_id = ?", id, userID).
		First(&method).Error
	if err != nil {
		return nil, err
	}
	return &method, nil
}

func (r *payoutMethodRepository) ListByUser(userID uuid.UUID) ([]models.PayoutMethod, error) {
	var methods []models.PayoutMethod
	err := r.db.
		Where("user_id = ?", userID).
		Order("is_primary DESC, created_at DESC").
		Find(&methods).Error
	return methods, err
}

func (r *payoutMethodRepository) SetPrimary(methodID, userID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Lock + verify ownership
		var method models.PayoutMethod
		err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", methodID, userID).
			First(&method).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return gorm.ErrRecordNotFound
			}
			return err
		}

		// Only verified methods can become primary
		if method.Status != models.PayoutMethodStatusVerified {
			return errors.New("payout method must be verified before setting as primary")
		}

		// Un-set existing primary for same (user, purpose, type)
		if err := tx.
			Model(&models.PayoutMethod{}).
			Where("user_id = ? AND purpose = ? AND type = ? AND is_primary = true AND id <> ?",
				userID, method.Purpose, method.Type, methodID).
			Update("is_primary", false).Error; err != nil {
			return err
		}

		// Set target as primary
		return tx.
			Model(&models.PayoutMethod{}).
			Where("id = ?", methodID).
			Update("is_primary", true).Error
	})
}

func (r *payoutMethodRepository) Delete(id, userID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: unset is_primary if this was the primary method
		if err := tx.
			Model(&models.PayoutMethod{}).
			Where("id = ? AND user_id = ? AND is_primary = true", id, userID).
			Update("is_primary", false).Error; err != nil {
			return err
		}

		// Step 2: soft delete (GORM sets deleted_at)
		res := tx.
			Where("id = ? AND user_id = ?", id, userID).
			Delete(&models.PayoutMethod{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *payoutMethodRepository) ExistsByProviderMethodID(provider, providerMethodID string) (bool, error) {
	var count int64
	err := r.db.
		Model(&models.PayoutMethod{}).
		Where("provider = ? AND provider_method_id = ?", provider, providerMethodID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *payoutMethodRepository) GetByProviderMethodID(provider, providerMethodID string) (*models.PayoutMethod, error) {
	var method models.PayoutMethod
	err := r.db.
		Where("provider = ? AND provider_method_id = ?", provider, providerMethodID).
		First(&method).Error
	if err != nil {
		return nil, err
	}
	return &method, nil
}

func (r *payoutMethodRepository) UpdateStatus(id uuid.UUID, status models.PayoutMethodStatus) error {
	res := r.db.
		Model(&models.PayoutMethod{}).
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

func (r *payoutMethodRepository) GetPrimaryByUser(userID uuid.UUID, purpose models.PayoutMethodPurpose) (*models.PayoutMethod, error) {
	var method models.PayoutMethod
	err := r.db.
		Where("user_id = ? AND purpose = ? AND is_primary = true", userID, purpose).
		First(&method).Error
	if err != nil {
		return nil, err
	}
	return &method, nil
}
