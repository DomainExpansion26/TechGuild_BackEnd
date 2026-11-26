package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MilestonePaymentRepository interface {
	Create(mp *models.MilestonePayment) error
	GetByID(id uuid.UUID) (*models.MilestonePayment, error)
	// GetByQuestMilestoneID returns the milestone's one non-failed attempt.
	// `failed` attempts are history once a newer attempt exists, so they are
	// filtered out here once, for every caller, rather than at each call site.
	GetByQuestMilestoneID(questMilestoneID uuid.UUID) (*models.MilestonePayment, error)
	GetByQuestMilestoneIDForUpdate(questMilestoneID uuid.UUID) (*models.MilestonePayment, error)

	// GetLatestByQuestMilestoneID returns the milestone's most recent attempt
	// whatever its status, failed ones included. Read paths use it to explain
	// why the newest attempt failed when no live attempt is left.
	GetLatestByQuestMilestoneID(questMilestoneID uuid.UUID) (*models.MilestonePayment, error)

	// GetByPaymentID returns the attempt row created by one specific payment —
	// the only correct lookup once a milestone can hold several attempts.
	GetByPaymentID(paymentID uuid.UUID) (*models.MilestonePayment, error)
	Save(mp *models.MilestonePayment) error
	UpdateStatus(id uuid.UUID, status models.MilestonePaymentStatus) error
	UpdateEscrowRef(id uuid.UUID, escrowID uuid.UUID) error

	// SumReleasedNetByQuest totals the freelancer net across milestones of the quest
	// whose money has actually been released (used for the wallet hold unlock).
	SumReleasedNetByQuest(questID uuid.UUID) (int64, error)

	WithTransaction(fn func(txRepo MilestonePaymentRepository) error) error
}

type milestonePaymentRepository struct {
	db *gorm.DB
}

func NewMilestonePaymentRepository() MilestonePaymentRepository {
	return &milestonePaymentRepository{db: postgres.DB}
}

func NewMilestonePaymentRepositoryTx(tx *gorm.DB) MilestonePaymentRepository {
	return &milestonePaymentRepository{db: tx}
}

func (r *milestonePaymentRepository) WithTransaction(fn func(txRepo MilestonePaymentRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewMilestonePaymentRepositoryTx(tx))
	})
}

func (r *milestonePaymentRepository) Create(mp *models.MilestonePayment) error {
	return r.db.Create(mp).Error
}

func (r *milestonePaymentRepository) GetByID(id uuid.UUID) (*models.MilestonePayment, error) {
	var mp models.MilestonePayment
	err := r.db.Where("id = ?", id).First(&mp).Error
	if err != nil {
		return nil, err
	}
	return &mp, nil
}

func (r *milestonePaymentRepository) GetByQuestMilestoneID(questMilestoneID uuid.UUID) (*models.MilestonePayment, error) {
	var mp models.MilestonePayment
	err := r.db.
		Where("quest_milestone_id = ? AND status <> ?", questMilestoneID, models.MilestonePaymentFailed).
		First(&mp).Error
	if err != nil {
		return nil, err
	}
	return &mp, nil
}

func (r *milestonePaymentRepository) GetLatestByQuestMilestoneID(questMilestoneID uuid.UUID) (*models.MilestonePayment, error) {
	var mp models.MilestonePayment
	err := r.db.
		Where("quest_milestone_id = ?", questMilestoneID).
		Order("created_at DESC, id DESC").
		First(&mp).Error
	if err != nil {
		return nil, err
	}
	return &mp, nil
}

// GetByQuestMilestoneIDForUpdate — row-level lock (SELECT ... FOR UPDATE)
// inside an existing transaction. Used for concurrent fund/release safety.
func (r *milestonePaymentRepository) GetByQuestMilestoneIDForUpdate(questMilestoneID uuid.UUID) (*models.MilestonePayment, error) {
	var mp models.MilestonePayment
	err := r.db.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("quest_milestone_id = ? AND status <> ?", questMilestoneID, models.MilestonePaymentFailed).
		First(&mp).Error
	if err != nil {
		return nil, err
	}
	return &mp, nil
}

func (r *milestonePaymentRepository) GetByPaymentID(paymentID uuid.UUID) (*models.MilestonePayment, error) {
	var mp models.MilestonePayment
	err := r.db.Where("payment_id = ?", paymentID).First(&mp).Error
	if err != nil {
		return nil, err
	}
	return &mp, nil
}

func (r *milestonePaymentRepository) Save(mp *models.MilestonePayment) error {
	return r.db.Save(mp).Error
}

func (r *milestonePaymentRepository) UpdateStatus(id uuid.UUID, status models.MilestonePaymentStatus) error {
	res := r.db.
		Model(&models.MilestonePayment{}).
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

func (r *milestonePaymentRepository) UpdateEscrowRef(id uuid.UUID, escrowID uuid.UUID) error {
	res := r.db.
		Model(&models.MilestonePayment{}).
		Where("id = ?", id).
		Update("escrow_transaction_id", escrowID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SumReleasedNetByQuest returns the total freelancer net (paise) across every
// milestone payment of the quest that has actually been released.
func (r *milestonePaymentRepository) SumReleasedNetByQuest(questID uuid.UUID) (int64, error) {
	var total int64
	err := r.db.Model(&models.MilestonePayment{}).
		Where("quest_id = ? AND status = ?", questID, models.MilestonePaymentReleased).
		Select("COALESCE(SUM(freelancer_net_minor), 0)").
		Scan(&total).Error
	return total, err
}
