package repository

import (
	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"github.com/google/uuid"
)

type TeamRepository struct {
}

func NewTeamRepository() *TeamRepository {
	return &TeamRepository{}
}

// Create Team
func (r *TeamRepository) CreateTeam(team *models.Team) error {
	return postgres.DB.Create(team).Error
}

// Update Team
func (r *TeamRepository) UpdateTeam(team *models.Team) error {
	return postgres.DB.Save(team).Error
}

// Delete Team
func (r *TeamRepository) DeleteTeam(team *models.Team) error {
	return postgres.DB.Delete(team).Error
}
//finding the team by ID
func (r *TeamRepository) FindByID(
	teamID uuid.UUID,
) (*models.Team, error) {

	var team models.Team

	err := postgres.DB.
		Preload("Leader").
		Preload("Members").
		Preload("Invitations").
		Preload("Portfolio").
		Preload("Skills").
		First(&team, "id = ?", teamID).Error

	if err != nil {
		return nil, err
	}

	return &team, nil
}
//by uuid
func (r *TeamRepository) FindByUUID(
	teamID string,
) (*models.Team, error) {

	id, err := uuid.Parse(teamID)
	if err != nil {
		return nil, err
	}

	return r.FindByID(id)
}
//finding team by leader
func (r *TeamRepository) FindByLeader(
	leaderID uuid.UUID,
) ([]models.Team, error) {

	var teams []models.Team

	err := postgres.DB.
		Where("leader_id = ?", leaderID).
		Order("created_at DESC").
		Find(&teams).Error

	return teams, err
}
//memeber functions 
func (r *TeamRepository) AddMember(
	member *models.TeamMember,
) error {

	return postgres.DB.Create(member).Error
}

func (r *TeamRepository) UpdateMember(
	member *models.TeamMember,
) error {

	return postgres.DB.Save(member).Error
}

func (r *TeamRepository) DeleteMember(
	member *models.TeamMember,
) error {

	return postgres.DB.Delete(member).Error
}

func (r *TeamRepository) FindMember(
	teamID uuid.UUID,
	userID uuid.UUID,
) (*models.TeamMember, error) {

	var member models.TeamMember

	err := postgres.DB.
		Where("team_id = ? AND user_id = ?", teamID, userID).
		First(&member).Error

	if err != nil {
		return nil, err
	}

	return &member, nil
}

func (r *TeamRepository) GetMembers(
	teamID uuid.UUID,
) ([]models.TeamMember, error) {

	var members []models.TeamMember

	err := postgres.DB.
		Where("team_id = ?", teamID).
		Preload("User").
		Find(&members).Error

	return members, err
}
//invitation function
func (r *TeamRepository) CreateInvitation(
	invitation *models.TeamInvitation,
) error {

	return postgres.DB.Create(invitation).Error
}

func (r *TeamRepository) UpdateInvitation(
	invitation *models.TeamInvitation,
) error {

	return postgres.DB.Save(invitation).Error
}

func (r *TeamRepository) FindInvitationByID(
	invitationID uuid.UUID,
) (*models.TeamInvitation, error) {

	var invitation models.TeamInvitation

	err := postgres.DB.
		Preload("Team").
		Preload("InvitedBy").
		Preload("InvitedUser").
		First(&invitation, "id = ?", invitationID).Error

	if err != nil {
		return nil, err
	}

	return &invitation, nil
}

func (r *TeamRepository) GetInvitations(
	teamID uuid.UUID,
) ([]models.TeamInvitation, error) {

	var invitations []models.TeamInvitation

	err := postgres.DB.
		Where("team_id = ?", teamID).
		Preload("InvitedUser").
		Find(&invitations).Error

	return invitations, err
}

func (r *TeamRepository) CreatePortfolio(
	portfolio *models.TeamPortfolio,
) error {

	return postgres.DB.Create(portfolio).Error
}

func (r *TeamRepository) UpdatePortfolio(
	portfolio *models.TeamPortfolio,
) error {

	return postgres.DB.Save(portfolio).Error
}

func (r *TeamRepository) DeletePortfolio(
	portfolio *models.TeamPortfolio,
) error {

	return postgres.DB.Delete(portfolio).Error
}
//to update and add portifolio and skill of team 
func (r *TeamRepository) GetPortfolio(
	teamID uuid.UUID,
) ([]models.TeamPortfolio, error) {

	var portfolio []models.TeamPortfolio

	err := postgres.DB.
		Where("team_id = ?", teamID).
		Find(&portfolio).Error

	return portfolio, err
}

func (r *TeamRepository) AddSkill(
	skill *models.TeamSkill,
) error {

	return postgres.DB.Create(skill).Error
}

func (r *TeamRepository) UpdateSkill(
	skill *models.TeamSkill,
) error {

	return postgres.DB.Save(skill).Error
}

func (r *TeamRepository) DeleteSkill(
	skill *models.TeamSkill,
) error {

	return postgres.DB.Delete(skill).Error
}

func (r *TeamRepository) GetSkills(
	teamID uuid.UUID,
) ([]models.TeamSkill, error) {

	var skills []models.TeamSkill

	err := postgres.DB.
		Where("team_id = ?", teamID).
		Find(&skills).Error

	return skills, err
}
func (r *TeamRepository) FindBySlug(
	slug string,
) (*models.Team, error) {

	var team models.Team

	err := postgres.DB.
		Where("slug = ?", slug).
		First(&team).Error

	if err != nil {
		return nil, err
	}

	return &team, nil
}

func (r *TeamRepository) FindPortfolioByID(
	portfolioID uuid.UUID,
) (*models.TeamPortfolio, error) {

	var portfolio models.TeamPortfolio

	err := postgres.DB.
		Preload("Team").
		First(&portfolio, "id = ?", portfolioID).Error

	if err != nil {
		return nil, err
	}

	return &portfolio, nil
}

func (r *TeamRepository) FindSkillByID(
	skillID uuid.UUID,
) (*models.TeamSkill, error) {

	var skill models.TeamSkill

	err := postgres.DB.
		Preload("Team").
		First(&skill, "id = ?", skillID).Error

	if err != nil {
		return nil, err
	}

	return &skill, nil
}

func (r *TeamRepository) FindInvitationByUser(
	teamID uuid.UUID,
	userID uuid.UUID,
) (*models.TeamInvitation, error) {

	var invitation models.TeamInvitation

	err := postgres.DB.
		Where("team_id = ? AND invited_user_id = ?", teamID, userID).
		First(&invitation).Error

	if err != nil {
		return nil, err
	}

	return &invitation, nil
}

func (r *TeamRepository) FindTeamByMember(
	userID uuid.UUID,
) ([]models.TeamMember, error) {

	var members []models.TeamMember

	err := postgres.DB.
		Where("user_id = ? AND status = ?", userID, models.MemberActive).
		Preload("Team").
		Find(&members).Error

	return members, err
}