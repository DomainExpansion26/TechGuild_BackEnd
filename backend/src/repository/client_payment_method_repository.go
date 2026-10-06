package repository

import (
	"errors"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ClientPaymentMethodRepository interface {
	Create(method *models.ClientPaymentMethod) error
	GetByID(id uuid.UUID) (*models.ClientPaymentMethod, error)
	GetByIDAndUser(id, userID uuid.UUID) (*models.ClientPaymentMethod, error)
	ListByUser(userID uuid.UUID) ([]models.ClientPaymentMethod, error)

	// SetPrimary atomically un-sets any existing primary of the same type
	// and sets the target method as primary. Uses DB transaction + partial
	// unique index for correctness.
	SetPrimary(methodID, userID uuid.UUID) error

	Delete(id, userID uuid.UUID) error
	ExistsByProviderMethodID(provider, providerMethodID string) (bool, error)

	// WithTransaction exposes tx-scoped repo, matches UserRepository pattern.
	WithTransaction(fn func(txRepo ClientPaymentMethodRepository) error) error
}

type clientPaymentMethodRepository struct {
	db *gorm.DB
}

func NewClientPaymentMethodRepository() ClientPaymentMethodRepository {
	return &clientPaymentMethodRepository{db: postgres.DB}
}

func NewClientPaymentMethodRepositoryTx(tx *gorm.DB) ClientPaymentMethodRepository {
	return &clientPaymentMethodRepository{db: tx}
}

func (r *clientPaymentMethodRepository) WithTransaction(fn func(txRepo ClientPaymentMethodRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewClientPaymentMethodRepositoryTx(tx))
	})
}

func (r *clientPaymentMethodRepository) Create(method *models.ClientPaymentMethod) error {
	return r.db.Create(method).Error
}

func (r *clientPaymentMethodRepository) GetByID(id uuid.UUID) (*models.ClientPaymentMethod, error) {
	var method models.ClientPaymentMethod
	err := r.db.Where("id = ?", id).First(&method).Error
	if err != nil {
		return nil, err
	}
	return &method, nil
}

func (r *clientPaymentMethodRepository) GetByIDAndUser(id, userID uuid.UUID) (*models.ClientPaymentMethod, error) {
	var method models.ClientPaymentMethod
	err := r.db.
		Where("id = ? AND user_id = ?", id, userID).
		First(&method).Error
	if err != nil {
		return nil, err
	}
	return &method, nil
}

func (r *clientPaymentMethodRepository) ListByUser(userID uuid.UUID) ([]models.ClientPaymentMethod, error) {
	var methods []models.ClientPaymentMethod
	err := r.db.
		Where("user_id = ?", userID).
		Order("is_primary DESC, created_at DESC").
		Find(&methods).Error
	return methods, err
}

func (r *clientPaymentMethodRepository) SetPrimary(methodID, userID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Lock target row + verify ownership
		var method models.ClientPaymentMethod
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

		// Un-set any existing primary of the same type for this user
		if err := tx.
			Model(&models.ClientPaymentMethod{}).
			Where("user_id = ? AND type = ? AND is_primary = true AND id <> ?", userID, method.Type, methodID).
			Update("is_primary", false).Error; err != nil {
			return err
		}

		// Set target as primary
		return tx.
			Model(&models.ClientPaymentMethod{}).
			Where("id = ?", methodID).
			Update("is_primary", true).Error
	})
}

func (r *clientPaymentMethodRepository) Delete(id, userID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: unset is_primary if this was the primary method
		if err := tx.
			Model(&models.ClientPaymentMethod{}).
			Where("id = ? AND user_id = ? AND is_primary = true", id, userID).
			Update("is_primary", false).Error; err != nil {
			return err
		}

		// Step 2: soft delete (GORM sets deleted_at)
		res := tx.
			Where("id = ? AND user_id = ?", id, userID).
			Delete(&models.ClientPaymentMethod{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *clientPaymentMethodRepository) ExistsByProviderMethodID(provider, providerMethodID string) (bool, error) {
	var count int64
	err := r.db.
		Model(&models.ClientPaymentMethod{}).
		Where("provider = ? AND provider_method_id = ?", provider, providerMethodID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
