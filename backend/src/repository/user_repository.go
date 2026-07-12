package repository

import (
	"errors"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
)

// defining the lists of methods signatures so that it will talk to db
type UserRepository interface {
	CreateUser(user *models.User) error
	CreateProfile(profile *models.UserProfile) error
	CreateVerification(record *models.VerificationRecord) error
	GetUserByID(userID string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)

	UpdateUserStatus(userID string, status string) error
	UpdateEmailVerified(userID string, verified bool) error
	UpdateAccountType(userID string, accountType models.AccountType) error

	CreateSession(session *models.UserSession) error
	GetSession(refreshToken string) (*models.UserSession, error)
	RevokeSession(refreshToken string) error
	UpdateRefreshToken(oldToken, newToken string) error
	UpdatePassword(userID string, passwordHash string) error
	RevokeAllSessions(userID string) error
	GetProfileByUserID(userID string) (*models.UserProfile, error)
	UpdateProfile(profile *models.UserProfile) error

	GetProfileBySlug(slug string) (*models.UserProfile, error)
	GetVerificationRecordByUserID(userID string) (*models.VerificationRecord, error)
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

func (r *userRepository) GetUserByID(userID string) (*models.User, error) {

	var user models.User

	err := postgres.DB.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}
func (r *userRepository) GetUserByEmail(email string) (*models.User, error) {

	var user models.User

	err := postgres.DB.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) UpdateUserStatus(userID string, status string) error {

	return postgres.DB.
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("status", status).Error
}

func (r *userRepository) UpdateEmailVerified(userID string, verified bool) error {

	return postgres.DB.
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("email_verified", verified).Error
}

func (r *userRepository) UpdateAccountType(userID string, accountType models.AccountType) error {

	return postgres.DB.
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("account_type", accountType).Error
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

func (r *userRepository) UpdatePassword(userID string, passwordHash string) error {

	return postgres.DB.
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("password_hash", passwordHash).Error
}

func (r *userRepository) RevokeAllSessions(userID string) error {

	return postgres.DB.
		Model(&models.UserSession{}).
		Where("user_id = ?", userID).
		Update("is_revoked", true).Error
}

// attach this function with userRepo struct
func (r *userRepository) GetProfileBySlug(slug string) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := postgres.DB.Where("public_url_slug = ?", slug).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *userRepository) GetVerificationRecordByUserID(userID string) (*models.VerificationRecord, error) {
	var record models.VerificationRecord
	err := postgres.DB.Where("user_id = ?", userID).First(&record).Error
	if err != nil {
		return nil, err

	}
	return &record, nil
}

func (r *userRepository) GetProfileByUserID(userID string) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := postgres.DB.Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *userRepository) UpdateProfile(profile *models.UserProfile) error {
	// GORM's Save method automatically runs UPDATE if the record exists
	return postgres.DB.Save(profile).Error
}
