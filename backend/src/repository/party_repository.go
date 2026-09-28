package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
)

type PartyRepository struct {
}

func NewPartyRepository() *PartyRepository {
	return &PartyRepository{}
}

// Create Party
func (r *PartyRepository) CreateParty(party *models.Party) error {
	return postgres.DB.Create(party).Error
}

// Update Party
func (r *PartyRepository) UpdateParty(party *models.Party) error {
	return postgres.DB.Save(party).Error
}

// Delete Party
func (r *PartyRepository) DeleteParty(party *models.Party) error {
	return postgres.DB.Delete(party).Error
}
//finding the party by ID
func (r *PartyRepository) FindByID(
	partyID uuid.UUID,
) (*models.Party, error) {

	var party models.Party

	err := postgres.DB.
		Preload("Leader").
		Preload("Members").
		Preload("Members.User").
		Preload("Invitations").
		Preload("Portfolio").
		Preload("Skills").
		First(&party, "id = ?", partyID).Error

	if err != nil {
		return nil, err
	}

	return &party, nil
}
//by uuid
func (r *PartyRepository) FindByUUID(
	partyID string,
) (*models.Party, error) {

	id, err := uuid.Parse(partyID)
	if err != nil {
		return nil, err
	}

	return r.FindByID(id)
}
//finding party by leader
func (r *PartyRepository) FindByLeader(
	leaderID uuid.UUID,
) ([]models.Party, error) {

	var parties []models.Party

	err := postgres.DB.
		Where("leader_id = ?", leaderID).
		Order("created_at DESC").
		Find(&parties).Error

	return parties, err
}

func (r *PartyRepository) FindAvailableForUser(
    userID uuid.UUID,
    search string,
    offset int,
    limit int,
) ([]models.Party, int64, error) {
    var parties []models.Party
    var total int64

    query := postgres.DB.
        Model(&models.Party{}).
        // Where("parties.status = ?", models.PartyActive).
        Where("parties.is_hiring = ?", true).
        Where(`
            NOT EXISTS (
                SELECT 1
                FROM party_members
                WHERE party_members.party_id = parties.id
                AND party_members.user_id = ?
                AND party_members.status = ?
            )
        `, userID, models.MemberActive).
        Where(`
            NOT EXISTS (
                SELECT 1
                FROM party_invitations
                WHERE party_invitations.party_id = parties.id
                AND party_invitations.invited_user_id = ?
                AND party_invitations.status = ?
                AND party_invitations.expires_at > NOW()
            )
        `, userID, models.InvitationPending)

    if search != "" {
        pattern := "%" + search + "%"

        query = query.Where(
            "(parties.name ILIKE ? OR parties.slug ILIKE ? OR parties.description ILIKE ?)",
            pattern,
            pattern,
            pattern,
        )
    }

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    if err := query.
        Order("parties.created_at DESC").
        Offset(offset).
        Limit(limit).
        Find(&parties).Error; err != nil {
        return nil, 0, err
    }

    return parties, total, nil
}
//memeber functions 
func (r *PartyRepository) AddMember(
	member *models.PartyMember,
) error {

	return postgres.DB.Create(member).Error
}

func (r *PartyRepository) UpdateMember(
	member *models.PartyMember,
) error {

	return postgres.DB.Save(member).Error
}

func (r *PartyRepository) DeleteMember(
	member *models.PartyMember,
) error {

	return postgres.DB.Delete(member).Error
}

func (r *PartyRepository) FindMember(
	partyID uuid.UUID,
	userID uuid.UUID,
) (*models.PartyMember, error) {

	var member models.PartyMember

	err := postgres.DB.
		Where("party_id = ? AND user_id = ?", partyID, userID).
		First(&member).Error

	if err != nil {
		return nil, err
	}

	return &member, nil
}

func (r *PartyRepository) GetMembers(
	partyID uuid.UUID,
) ([]models.PartyMember, error) {

	var members []models.PartyMember

	err := postgres.DB.
		Where("party_id = ?", partyID).
		Preload("User").
		Find(&members).Error

	return members, err
}
//invitation function
func (r *PartyRepository) CreateInvitation(
	invitation *models.PartyInvitation,
) error {

	return postgres.DB.Create(invitation).Error
}

func (r *PartyRepository) UpdateInvitation(
	invitation *models.PartyInvitation,
) error {

	return postgres.DB.Save(invitation).Error
}

func (r *PartyRepository) FindInvitationByID(
	invitationID uuid.UUID,
) (*models.PartyInvitation, error) {

	var invitation models.PartyInvitation

	err := postgres.DB.
		Preload("Party").
		Preload("InvitedBy").
		Preload("InvitedUser").
		First(&invitation, "id = ?", invitationID).Error

	if err != nil {
		return nil, err
	}

	return &invitation, nil
}

func (r *PartyRepository) GetInvitations(
	partyID uuid.UUID,
) ([]models.PartyInvitation, error) {

	var invitations []models.PartyInvitation

	err := postgres.DB.
		Where("party_id = ?", partyID).
		Preload("InvitedUser").
		Find(&invitations).Error

	return invitations, err
}

func (r *PartyRepository) CreatePortfolio(
	portfolio *models.PartyPortfolio,
) error {

	return postgres.DB.Create(portfolio).Error
}

func (r *PartyRepository) UpdatePortfolio(
	portfolio *models.PartyPortfolio,
) error {

	return postgres.DB.Save(portfolio).Error
}

func (r *PartyRepository) DeletePortfolio(
	portfolio *models.PartyPortfolio,
) error {

	return postgres.DB.Delete(portfolio).Error
}
//to update and add portifolio and skill of party 
func (r *PartyRepository) GetPortfolio(
	partyID uuid.UUID,
) ([]models.PartyPortfolio, error) {

	var portfolio []models.PartyPortfolio

	err := postgres.DB.
		Where("party_id = ?", partyID).
		Find(&portfolio).Error

	return portfolio, err
}

func (r *PartyRepository) AddSkill(
	skill *models.PartySkill,
) error {

	return postgres.DB.Create(skill).Error
}

func (r *PartyRepository) UpdateSkill(
	skill *models.PartySkill,
) error {

	return postgres.DB.Save(skill).Error
}

func (r *PartyRepository) DeleteSkill(
	skill *models.PartySkill,
) error {

	return postgres.DB.Delete(skill).Error
}

func (r *PartyRepository) GetSkills(
	partyID uuid.UUID,
) ([]models.PartySkill, error) {

	var skills []models.PartySkill

	err := postgres.DB.
		Where("party_id = ?", partyID).
		Find(&skills).Error

	return skills, err
}
func (r *PartyRepository) FindBySlug(
	slug string,
) (*models.Party, error) {

	var party models.Party

	err := postgres.DB.
		Where("slug = ?", slug).
		First(&party).Error

	if err != nil {
		return nil, err
	}

	return &party, nil
}

func (r *PartyRepository) FindPortfolioByID(
	portfolioID uuid.UUID,
) (*models.PartyPortfolio, error) {

	var portfolio models.PartyPortfolio

	err := postgres.DB.
		Preload("Party").
		First(&portfolio, "id = ?", portfolioID).Error

	if err != nil {
		return nil, err
	}

	return &portfolio, nil
}

func (r *PartyRepository) FindSkillByID(
	skillID uuid.UUID,
) (*models.PartySkill, error) {

	var skill models.PartySkill

	err := postgres.DB.
		Preload("Party").
		First(&skill, "id = ?", skillID).Error

	if err != nil {
		return nil, err
	}

	return &skill, nil
}

func (r *PartyRepository) FindInvitationByUser(
	partyID uuid.UUID,
	userID uuid.UUID,
) (*models.PartyInvitation, error) {

	var invitation models.PartyInvitation

	err := postgres.DB.
		Where("party_id = ? AND invited_user_id = ?", partyID, userID).
		First(&invitation).Error

	if err != nil {
		return nil, err
	}

	return &invitation, nil
}

func (r *PartyRepository) FindPartyByMember(
	userID uuid.UUID,
) ([]models.PartyMember, error) {

	var members []models.PartyMember

	err := postgres.DB.
		Where("user_id = ? AND status = ?", userID, models.MemberActive).
		Preload("Party").
		Find(&members).Error

	return members, err
}