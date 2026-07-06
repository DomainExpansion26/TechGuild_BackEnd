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

	CreateSession(session *models.UserSession) error
	GetSession(refreshToken string) (*models.UserSession, error)
	RevokeSession(refreshToken string) error
	UpdateRefreshToken(oldToken, newToken string) error
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
func (r *userRepository) CreateSession(session *models.UserSession) error {
	return postgres.DB.Create(session).Error
}

func (r *userRepository) GetSession(refreshToken string) (*models.UserSession, error) {

	var session models.UserSession

	err := postgres.DB.
		Where("refresh_token = ? AND is_revoked = false", refreshToken).
		First(&session).Error

	if err != nil {
		return nil, errors.New("session not found")
	}

	return &session, nil
}

func (r *userRepository) RevokeSession(refreshToken string) error {

	return postgres.DB.
		Model(&models.UserSession{}).
		Where("refresh_token = ?", refreshToken).
		Update("is_revoked", true).Error
}
func (r *userRepository) UpdateRefreshToken(oldToken, newToken string) error {

	return postgres.DB.
		Model(&models.UserSession{}).
		Where("refresh_token = ? AND is_revoked = false", oldToken).
		Update("refresh_token", newToken).Error
}