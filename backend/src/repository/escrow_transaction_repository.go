package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EscrowTransactionRepository interface {
	Create(escrow *models.EscrowTransaction) error
	GetByID(id uuid.UUID) (*models.EscrowTransaction, error)
	GetByMilestoneID(milestoneID uuid.UUID) (*models.EscrowTransaction, error)
	GetByReleaseIdempotencyKey(key string) (*models.EscrowTransaction, error)
	UpdateStatus(id uuid.UUID, status models.EscrowStatus) error
	MarkHeld(id uuid.UUID, providerEscrowID string) error
	MarkReleased(id uuid.UUID) error
	MarkRefunded(id uuid.UUID) error
	MarkFailed(id uuid.UUID, reason string) error
	GetByMilestoneIDForUpdate(milestoneID uuid.UUID) (*models.EscrowTransaction, error)
	Save(escrow *models.EscrowTransaction) error

	WithTransaction(fn func(txRepo EscrowTransactionRepository) error) error
}

type escrowTransactionRepository struct {
	db *gorm.DB
}

func NewEscrowTransactionRepository() EscrowTransactionRepository {
	return &escrowTransactionRepository{db: postgres.DB}
}

func NewEscrowTransactionRepositoryTx(tx *gorm.DB) EscrowTransactionRepository {
	return &escrowTransactionRepository{db: tx}
}

func (r *escrowTransactionRepository) WithTransaction(fn func(txRepo EscrowTransactionRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewEscrowTransactionRepositoryTx(tx))
	})
}

func (r *escrowTransactionRepository) Create(escrow *models.EscrowTransaction) error {
	return r.db.Create(escrow).Error
}

func (r *escrowTransactionRepository) GetByID(id uuid.UUID) (*models.EscrowTransaction, error) {
	var e models.EscrowTransaction
	err := r.db.Where("id = ?", id).First(&e).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *escrowTransactionRepository) GetByMilestoneID(milestoneID uuid.UUID) (*models.EscrowTransaction, error) {
	var e models.EscrowTransaction
	err := r.db.Where("milestone_id = ?", milestoneID).First(&e).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *escrowTransactionRepository) GetByReleaseIdempotencyKey(key string) (*models.EscrowTransaction, error) {
	var e models.EscrowTransaction
	err := r.db.Where("release_idempotency_key = ?", key).First(&e).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *escrowTransactionRepository) GetByMilestoneIDForUpdate(milestoneID uuid.UUID) (*models.EscrowTransaction, error) {
	var e models.EscrowTransaction
	err := r.db.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("milestone_id = ?", milestoneID).
		First(&e).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *escrowTransactionRepository) Save(escrow *models.EscrowTransaction) error {
	return r.db.Save(escrow).Error
}

func (r *escrowTransactionRepository) UpdateStatus(id uuid.UUID, status models.EscrowStatus) error {
	res := r.db.
		Model(&models.EscrowTransaction{}).
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

func (r *escrowTransactionRepository) MarkHeld(id uuid.UUID, providerEscrowID string) error {
	res := r.db.
		Model(&models.EscrowTransaction{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":             models.EscrowStatusHeld,
			"provider_escrow_id": providerEscrowID,
			"held_at":            gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *escrowTransactionRepository) MarkReleased(id uuid.UUID) error {
	res := r.db.
		Model(&models.EscrowTransaction{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      models.EscrowStatusReleased,
			"released_at": gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *escrowTransactionRepository) MarkRefunded(id uuid.UUID) error {
	res := r.db.
		Model(&models.EscrowTransaction{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      models.EscrowStatusRefunded,
			"refunded_at": gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *escrowTransactionRepository) MarkFailed(id uuid.UUID, reason string) error {
	res := r.db.
		Model(&models.EscrowTransaction{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         models.EscrowStatusFailed,
			"failure_reason": reason,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
