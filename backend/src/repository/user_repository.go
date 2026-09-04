package repository

import (
	"errors"
	"time"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"
	"techguild-backend/src/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// defining the lists of methods signatures so that it will talk to db
type UserRepository interface {
	CreateUser(user *models.User) error
	CreateProfile(profile interface{}) error
	CreateVerification(record *models.VerificationRecord) error
	GetUserByID(userID string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	DeleteUser(userID string) error
	UpdateUserStatus(userID string, status string) error
	UpdateEmailVerified(userID string, verified bool) error
	UpdateAccountType(userID string, accountType models.AccountType) error
	AddUserPoints(userID string, points int) error

	CreateSession(session *models.UserSession) error
	GetSession(refreshToken string) (*models.UserSession, error)
	RevokeSession(refreshToken string) error
	UpdateRefreshToken(oldToken, newToken string) error
	UpdatePassword(userID string, passwordHash string) error
	RevokeAllSessions(userID string) error
	GetIndividualProfileByUserID(userID string) (*models.IndividualProfile, error)
	UpdateIndividualProfile(profile *models.IndividualProfile) error

	GetAgencyProfileByUserID(userID string) (*models.AgencyProfile, error)
	UpdateAgencyProfile(profile *models.AgencyProfile) error

	GetClientProfileByUserID(userID string) (*models.ClientProfile, error)
	UpdateClientProfile(profile *models.ClientProfile) error

	GetIndividualProfileBySlug(slug string) (*models.IndividualProfile, error)
	GetAgencyProfileBySlug(slug string) (*models.AgencyProfile, error)
	GetClientProfileBySlug(slug string) (*models.ClientProfile, error)
	GetVerificationRecordByUserID(userID string) (*models.VerificationRecord, error)
	UpdateUser(user *models.User) error

	GetSessionByToken(refreshToken string) (*models.UserSession, error)
	RevokeSessionByID(sessionID uuid.UUID) error

	WithTransaction(fn func(txRepo UserRepository) error) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository() UserRepository {
	return &userRepository{db: postgres.DB}
}

// NEW: returns a repo bound to a transaction
func NewUserRepositoryTx(tx *gorm.DB) UserRepository {
	return &userRepository{db: tx}
}

func (r *userRepository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) WithTransaction(fn func(txRepo UserRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := NewUserRepositoryTx(tx)
		return fn(txRepo)
	})
}

func (r *userRepository) CreateProfile(profile interface{}) error {
	return r.db.Create(profile).Error
}

func (r *userRepository) CreateVerification(record *models.VerificationRecord) error {
	return r.db.Create(record).Error
}

func (r *userRepository) GetUserByID(userID string) (*models.User, error) {

	var user models.User

	err := r.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}
func (r *userRepository) GetUserByEmail(email string) (*models.User, error) {

	var user models.User

	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) UpdateUserStatus(userID string, status string) error {

	return r.db.
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("status", status).Error
}

func (r *userRepository) UpdateEmailVerified(userID string, verified bool) error {

	return r.db.
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("email_verified", verified).Error
}

func (r *userRepository) DeleteUser(userID string) error {
	return r.db.Where("id = ?", userID).Delete(&models.User{}).Error
}

// UpdateAccountType sets account_type AND resets rank to "F" —
// intentional side effect for fresh account-type assignment; do not
// call this for account_type changes where rank should be preserved.
func (r *userRepository) UpdateAccountType(userID string, accountType models.AccountType) error {

	return r.db.
		Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"account_type": accountType,
			"rank":         "F",
		}).Error
}

func (r *userRepository) AddUserPoints(userID string, points int) error {
	return r.db.
		Model(&models.User{}).
		Where("id = ?", userID).
		UpdateColumn("points", gorm.Expr("points + ?", points)).Error
}

func (r *userRepository) CreateSession(session *models.UserSession) error {
	session.RefreshToken = utils.HashToken(session.RefreshToken)
	return r.db.Create(session).Error
}
func (r *userRepository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}
func (r *userRepository) GetSession(refreshToken string) (*models.UserSession, error) {

	var session models.UserSession
	hashed := utils.HashToken(refreshToken)

	err := r.db.
		Where("refresh_token = ? AND expires_at > ?", hashed, time.Now()).
		First(&session).Error
	// NOTE: is_revoked filter removed from WHERE (fixes Bug 3 reuse-detection too) —
	// caller must check session.IsRevoked explicitly after fetch.
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *userRepository) GetSessionByToken(refreshToken string) (*models.UserSession, error) {
	var session models.UserSession
	hashed := utils.HashToken(refreshToken)
	err := r.db.Where("refresh_token = ?", hashed).First(&session).Error
	if err != nil {
		return nil, errors.New("session not found")
	}
	return &session, nil
}

func (r *userRepository) RevokeSession(refreshToken string) error {
	hashed := utils.HashToken(refreshToken)
	return r.db.
		Model(&models.UserSession{}).
		Where("refresh_token = ?", hashed).
		Update("is_revoked", true).Error
}

func (r *userRepository) RevokeSessionByID(sessionID uuid.UUID) error {
	return r.db.
		Model(&models.UserSession{}).
		Where("id = ?", sessionID).
		Update("is_revoked", true).Error
}

// FIX (Bug 2): hash the NEW token before storing
func (r *userRepository) UpdateRefreshToken(oldToken, newToken string) error {
	oldHashed := utils.HashToken(oldToken)
	newHashed := utils.HashToken(newToken)
	return r.db.
		Model(&models.UserSession{}).
		Where("refresh_token = ? AND is_revoked = false", oldHashed).
		Update("refresh_token", newHashed).Error
}

func (r *userRepository) UpdatePassword(userID string, passwordHash string) error {

	return r.db.
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("password_hash", passwordHash).Error
}

func (r *userRepository) RevokeAllSessions(userID string) error {

	return r.db.
		Model(&models.UserSession{}).
		Where("user_id = ?", userID).
		Update("is_revoked", true).Error
}

// attach this function with userRepo struct
func (r *userRepository) GetIndividualProfileBySlug(slug string) (*models.IndividualProfile, error) {
	var profile models.IndividualProfile
	err := r.db.Where("public_url_slug = ?", slug).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *userRepository) GetAgencyProfileBySlug(slug string) (*models.AgencyProfile, error) {
	var profile models.AgencyProfile
	err := r.db.Where("public_url_slug = ?", slug).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *userRepository) GetClientProfileBySlug(slug string) (*models.ClientProfile, error) {
	var profile models.ClientProfile
	err := r.db.Where("public_url_slug = ?", slug).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *userRepository) GetVerificationRecordByUserID(userID string) (*models.VerificationRecord, error) {
	var record models.VerificationRecord
	err := r.db.Where("user_id = ?", userID).First(&record).Error
	if err != nil {
		return nil, err

	}
	return &record, nil
}

func (r *userRepository) GetIndividualProfileByUserID(userID string) (*models.IndividualProfile, error) {
	var profile models.IndividualProfile
	err := r.db.Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *userRepository) UpdateIndividualProfile(profile *models.IndividualProfile) error {
	// GORM's Save method automatically runs UPDATE if the record exists
	return r.db.Save(profile).Error
}

func (r *userRepository) GetAgencyProfileByUserID(userID string) (*models.AgencyProfile, error) {
	var profile models.AgencyProfile
	err := r.db.Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *userRepository) UpdateAgencyProfile(profile *models.AgencyProfile) error {
	return r.db.Save(profile).Error
}

func (r *userRepository) GetClientProfileByUserID(userID string) (*models.ClientProfile, error) {
	var profile models.ClientProfile
	err := r.db.Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *userRepository) UpdateClientProfile(profile *models.ClientProfile) error {
	return r.db.Save(profile).Error
}
