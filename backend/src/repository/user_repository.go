package repository

import (
	"errors"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

type UserRepository interface {
	CreateUser(user *models.User) error
	CreateProfile(profile *models.UserProfile) error
	CreateVerification(record *models.VerificationRecord) error

	GetUserByEmail(email string) (*models.User, error)
	GetUserByPhone(phone string) (*models.User, error)

	UpdateUserStatus(userID string, status string) error
}

type userRepository struct{}

func NewUserRepository() UserRepository {
	return &userRepository{}
}

func (r *userRepository) CreateUser(user *models.User) error {
	return postgres.DB.Create(user).Error
}

func (r *userRepository) CreateProfile(profile *models.UserProfile) error {
	return postgres.DB.Create(profile).Error
}

func (r *userRepository) CreateVerification(record *models.VerificationRecord) error {
	return postgres.DB.Create(record).Error
}

func (r *userRepository) GetUserByEmail(email string) (*models.User, error) {

	var user models.User

	err := postgres.DB.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, errors.New("user not found")
	}

	return &user, nil
}

func (r *userRepository) GetUserByPhone(phone string) (*models.User, error) {

	var user models.User

	err := postgres.DB.Where("phone = ?", phone).First(&user).Error
	if err != nil {
		return nil, errors.New("user not found")
	}

	return &user, nil
}

func (r *userRepository) UpdateUserStatus(userID string, status string) error {

	return postgres.DB.Model(&models.User{}).
		Where("id = ?", userID).
		Update("status", status).Error
}