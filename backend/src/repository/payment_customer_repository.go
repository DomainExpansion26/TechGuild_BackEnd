package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentCustomerRepository interface {
	Create(customer *models.PaymentCustomer) error
	GetByUserAndProvider(userID uuid.UUID, provider string) (*models.PaymentCustomer, error)
	GetByProviderCustomerID(provider, providerCustomerID string) (*models.PaymentCustomer, error)
}

type paymentCustomerRepository struct {
	db *gorm.DB
}

func NewPaymentCustomerRepository() PaymentCustomerRepository {
	return &paymentCustomerRepository{db: postgres.DB}
}

func NewPaymentCustomerRepositoryTx(tx *gorm.DB) PaymentCustomerRepository {
	return &paymentCustomerRepository{db: tx}
}

func (r *paymentCustomerRepository) Create(customer *models.PaymentCustomer) error {
	return r.db.Create(customer).Error
}

func (r *paymentCustomerRepository) GetByUserAndProvider(userID uuid.UUID, provider string) (*models.PaymentCustomer, error) {
	var customer models.PaymentCustomer
	err := r.db.
		Where("user_id = ? AND provider = ?", userID, provider).
		First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

func (r *paymentCustomerRepository) GetByProviderCustomerID(provider, providerCustomerID string) (*models.PaymentCustomer, error) {
	var customer models.PaymentCustomer
	err := r.db.
		Where("provider = ? AND provider_customer_id = ?", provider, providerCustomerID).
		First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}
