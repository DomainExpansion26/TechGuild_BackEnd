package services

import (
	"errors"
	"time"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"

	"github.com/google/uuid"
)

type TeamService struct {
	teamRepo *repository.TeamRepository
	userRepo repository.UserRepository
}

func NewTeamService() *TeamService {
	return &TeamService{
		teamRepo: repository.NewTeamRepository(),
		userRepo: repository.NewUserRepository(),
	}
}


//create team
func (s *TeamService) CreateTeam(
	leaderID string,
	req dto.CreateTeamRequest,
) (*dto.TeamResponse, error) {

	userUUID, err := uuid.Parse(leaderID)
	if err != nil {
		return nil, errors.New("invalid leader id")
	}

	_, err = s.userRepo.GetUserByID(userUUID.String())
		if err != nil {
			return nil, errors.New("leader not found")
		}

	team := models.Team{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		LogoURL:     req.LogoURL,
		BannerURL:   req.BannerURL,
		LeaderID:    userUUID,
		IsHiring:    req.IsHiring,
		Status:      models.TeamPending,
	}

	if err := s.teamRepo.CreateTeam(&team); err != nil {
		return nil, err
	}

	member := models.TeamMember{
		TeamID:   team.ID,
		UserID:   userUUID,
		Role: models.TeamRoleLeader,
		Status:   models.MemberActive,
		JoinedAt: func() *time.Time { t := time.Now(); return &t }(),
	}

	if err := s.teamRepo.AddMember(&member); err != nil {
		return nil, err
	}

	return &dto.TeamResponse{
		ID:          team.ID.String(),
		Name:        team.Name,
		Slug:        team.Slug,
		Description: team.Description,
		LogoURL:     team.LogoURL,
		BannerURL:   team.BannerURL,
		LeaderID:    team.LeaderID.String(),
		IsHiring:    team.IsHiring,
		IsVerified:  team.IsVerified,
		Status:      string(team.Status),
		CreatedAt:   team.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   team.UpdatedAt.Format(time.RFC3339),
	}, nil
}

//update the team
func (s *TeamService) UpdateTeam(
	leaderID string,
	teamID string,
	req dto.UpdateTeamRequest,
) error {

	team, err := s.teamRepo.FindByUUID(teamID)
	if err != nil {
		return errors.New("team not found")
	}

	if team.LeaderID.String() != leaderID {
		return errors.New("only team leader can update the team")
	}

	if req.Name != "" {
		team.Name = req.Name
	}

	if req.Description != "" {
		team.Description = req.Description
	}

	if req.LogoURL != "" {
		team.LogoURL = req.LogoURL
	}

	if req.BannerURL != "" {
		team.BannerURL = req.BannerURL
	}

	team.IsHiring = req.IsHiring

	return s.teamRepo.UpdateTeam(team)
}

//delete team
func (s *TeamService) DeleteTeam(
	leaderID string,
	teamID string,
) error {

	team, err := s.teamRepo.FindByUUID(teamID)
	if err != nil {
		return errors.New("team not found")
	}

	if team.LeaderID.String() != leaderID {
		return errors.New("only team leader can delete the team")
	}

	return s.teamRepo.DeleteTeam(team)
}

//to get team info
func (s *TeamService) GetTeam(
	teamID string,
) (*dto.TeamResponse, error) {

	team, err := s.teamRepo.FindByUUID(teamID)
	if err != nil {
		return nil, err
	}

	return &dto.TeamResponse{
		ID:          team.ID.String(),
		Name:        team.Name,
		Slug:        team.Slug,
		Description: team.Description,
		LogoURL:     team.LogoURL,
		BannerURL:   team.BannerURL,
		LeaderID:    team.LeaderID.String(),
		IsHiring:    team.IsHiring,
		IsVerified:  team.IsVerified,
		Status:      string(team.Status),
		CreatedAt:   team.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   team.UpdatedAt.Format(time.RFC3339),
	}, nil
}

//get my Teams
func (s *TeamService) GetMyTeams(
	userID string,
) (*dto.TeamListResponse, error) {

	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user id")
	}

	members, err := s.teamRepo.FindTeamByMember(id)
	if err != nil {
		return nil, err
	}

	response := dto.TeamListResponse{}

	for _, member := range members {

		response.Teams = append(response.Teams, dto.TeamResponse{
			ID:          member.Team.ID.String(),
			Name:        member.Team.Name,
			Slug:        member.Team.Slug,
			Description: member.Team.Description,
			LogoURL:     member.Team.LogoURL,
			BannerURL:   member.Team.BannerURL,
			LeaderID:    member.Team.LeaderID.String(),
			IsHiring:    member.Team.IsHiring,
			IsVerified:  member.Team.IsVerified,
			Status:      string(member.Team.Status),
			CreatedAt:   member.Team.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   member.Team.UpdatedAt.Format(time.RFC3339),
		})
	}

	response.Total = len(response.Teams)

	return &response, nil
}

//InviteMember
func (s *TeamService) InviteMember(
	leaderID string,
	teamID string,
	req dto.InviteMemberRequest,
) error {

	team, err := s.teamRepo.FindByUUID(teamID)
	if err != nil {
		return errors.New("team not found")
	}

	if team.LeaderID.String() != leaderID {
		return errors.New("only team leader can invite members")
	}

	user, err := s.userRepo.GetUserByID(req.UserID)
	if err != nil {
		return errors.New("user not found")
	}

	_, err = s.teamRepo.FindMember(team.ID, user.ID)
	if err == nil {
		return errors.New("user is already a team member")
	}

	_, err = s.teamRepo.FindInvitationByUser(team.ID, user.ID)
	if err == nil {
		return errors.New("invitation already sent")
	}

	invitation := models.TeamInvitation{
		TeamID:         team.ID,
		InvitedByID:    team.LeaderID,
		InvitedUserID:  user.ID,
		Message:        req.Message,
		Status:         models.InvitationPending,
	}

	return s.teamRepo.CreateInvitation(&invitation)
}

//accept and reject the invite

func (s *TeamService) AcceptInvitation(
	userID string,
	invitationID string,
) error {

	id, err := uuid.Parse(invitationID)
	if err != nil {
		return errors.New("invalid invitation id")
	}

	invitation, err := s.teamRepo.FindInvitationByID(id)
	if err != nil {
		return errors.New("invitation not found")
	}

	if invitation.InvitedUserID.String() != userID {
		return errors.New("unauthorized")
	}

	if invitation.Status != models.InvitationPending {
		return errors.New("invitation already processed")
	}

	now := time.Now()

	invitation.Status = models.InvitationAccepted
	invitation.RespondedAt = &now

	if err := s.teamRepo.UpdateInvitation(invitation); err != nil {
		return err
	}

	member := models.TeamMember{
		TeamID: invitation.TeamID,
		UserID: invitation.InvitedUserID,
		Role:models.TeamRoleMember,
		Status: models.MemberActive,
		JoinedAt: &now,
	}

	return s.teamRepo.AddMember(&member)
}

func (s *TeamService) RejectInvitation(
	userID string,
	invitationID string,
) error {

	id, err := uuid.Parse(invitationID)
	if err != nil {
		return errors.New("invalid invitation id")
	}

	invitation, err := s.teamRepo.FindInvitationByID(id)
	if err != nil {
		return errors.New("invitation not found")
	}

	if invitation.InvitedUserID.String() != userID {
		return errors.New("unauthorized")
	}

	now := time.Now()

	invitation.Status = models.InvitationRejected
	invitation.RespondedAt = &now

	return s.teamRepo.UpdateInvitation(invitation)
}

// to remove the member
func (s *TeamService) RemoveMember(
	leaderID string,
	teamID string,
	memberID string,
) error {

	team, err := s.teamRepo.FindByUUID(teamID)
	if err != nil {
		return errors.New("team not found")
	}

	if team.LeaderID.String() != leaderID {
		return errors.New("only leader can remove members")
	}

	userUUID, err := uuid.Parse(memberID)
	if err != nil {
		return errors.New("invalid member id")
	}

	member, err := s.teamRepo.FindMember(team.ID, userUUID)
	if err != nil {
		return errors.New("member not found")
	}

	return s.teamRepo.DeleteMember(member)
}


//leave the team
func (s *TeamService) LeaveTeam(
	userID string,
	teamID string,
) error {

	team, err := s.teamRepo.FindByUUID(teamID)
	if err != nil {
		return errors.New("team not found")
	}

	if team.LeaderID.String() == userID {
		return errors.New("team leader cannot leave the team")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user id")
	}

	member, err := s.teamRepo.FindMember(team.ID, userUUID)
	if err != nil {
		return errors.New("member not found")
	}

	return s.teamRepo.DeleteMember(member)
}