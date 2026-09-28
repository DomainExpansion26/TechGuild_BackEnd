package services

import (
	"errors"
	"time"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"

	"github.com/google/uuid"
)
var (
    ErrIndividualAccountRequired = errors.New("individual account required")
    ErrActiveAccountRequired     = errors.New("active account required")
)

type PartyService struct {
	partyRepo *repository.PartyRepository
	userRepo repository.UserRepository
}

func NewPartyService() *PartyService {
	return &PartyService{
		partyRepo: repository.NewPartyRepository(),
		userRepo: repository.NewUserRepository(),
	}
}


//get members(new)
func (s *PartyService) GetMembers(
    partyID string,
) (*dto.PartyMemberListResponse, error) {
    id, err := uuid.Parse(partyID)
    if err != nil {
        return nil, errors.New("invalid party id")
    }

    if _, err := s.partyRepo.FindByID(id); err != nil {
        return nil, errors.New("party not found")
    }

    members, err := s.partyRepo.GetMembers(id)
    if err != nil {
        return nil, err
    }

    response := &dto.PartyMemberListResponse{
        Members: make([]dto.PartyMemberResponse, 0, len(members)),
        Total:   len(members),
    }

    for _, member := range members {
        item := dto.PartyMemberResponse{
            ID:        member.ID.String(),
            PartyID:   member.PartyID.String(),
            UserID:    member.UserID.String(),
            Role:      string(member.Role),
            Status:    string(member.Status),
            CreatedAt: member.CreatedAt.Format(time.RFC3339),
            UpdatedAt: member.UpdatedAt.Format(time.RFC3339),
        }

        if member.JoinedAt != nil {
            item.JoinedAt = member.JoinedAt.Format(time.RFC3339)
        }

        response.Members = append(response.Members, item)
    }

    return response, nil
}

//get invitations(new)
func (s *PartyService) GetInvitations(
    partyID string,
) (*dto.PartyInvitationListResponse, error) {
    id, err := uuid.Parse(partyID)
    if err != nil {
        return nil, errors.New("invalid party id")
    }

    party, err := s.partyRepo.FindByID(id)
    if err != nil {
        return nil, errors.New("party not found")
    }

    invitations, err := s.partyRepo.GetInvitations(party.ID)
    if err != nil {
        return nil, err
    }

    response := &dto.PartyInvitationListResponse{
        Invitations: make([]dto.PartyInvitationResponse, 0, len(invitations)),
        Total:       len(invitations),
    }

    for _, invitation := range invitations {
        item := dto.PartyInvitationResponse{
            ID:            invitation.ID.String(),
            PartyID:       invitation.PartyID.String(),
            InvitedByID:   invitation.InvitedByID.String(),
            InvitedUserID: invitation.InvitedUserID.String(),
            Message:       invitation.Message,
            Status:        string(invitation.Status),
            ExpiresAt:     invitation.ExpiresAt.Format(time.RFC3339),
            CreatedAt:     invitation.CreatedAt.Format(time.RFC3339),
        }

        if invitation.RespondedAt != nil {
            item.RespondedAt = invitation.RespondedAt.Format(time.RFC3339)
        }

        response.Invitations = append(response.Invitations, item)
    }

    return response, nil
}

//get portfolio(new)
func (s *PartyService) GetPortfolio(
    partyID string,
) (*dto.PartyPortfolioListResponse, error) {
    id, err := uuid.Parse(partyID)
    if err != nil {
        return nil, errors.New("invalid party id")
    }

    if _, err := s.partyRepo.FindByID(id); err != nil {
        return nil, errors.New("party not found")
    }

    portfolio, err := s.partyRepo.GetPortfolio(id)
    if err != nil {
        return nil, err
    }

    response := &dto.PartyPortfolioListResponse{
        Portfolio: make([]dto.PartyPortfolioResponse, 0, len(portfolio)),
        Total:     len(portfolio),
    }

    for _, item := range portfolio {
        response.Portfolio = append(response.Portfolio, dto.PartyPortfolioResponse{
            ID:          item.ID.String(),
            PartyID:     item.PartyID.String(),
            Title:       item.Title,
            Description: item.Description,
            ImageURL:    item.ImageURL,
            ProjectURL:  item.ProjectURL,
            GithubURL:   item.GithubURL,
            CreatedAt:   item.CreatedAt.Format(time.RFC3339),
            UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
        })
    }

    return response, nil
}

//get skills(new)
func (s *PartyService) GetSkills(
    partyID string,
) (*dto.PartySkillListResponse, error) {
    id, err := uuid.Parse(partyID)
    if err != nil {
        return nil, errors.New("invalid party id")
    }

    if _, err := s.partyRepo.FindByID(id); err != nil {
        return nil, errors.New("party not found")
    }

    skills, err := s.partyRepo.GetSkills(id)
    if err != nil {
        return nil, err
    }

    response := &dto.PartySkillListResponse{
        Skills: make([]dto.PartySkillResponse, 0, len(skills)),
        Total:  len(skills),
    }

    for _, skill := range skills {
        response.Skills = append(response.Skills, dto.PartySkillResponse{
            ID:              skill.ID.String(),
            PartyID:          skill.PartyID.String(),
            SkillName:       skill.SkillName,
            ExperienceLevel: skill.ExperienceLevel,
            CreatedAt:       skill.CreatedAt.Format(time.RFC3339),
            UpdatedAt:       skill.UpdatedAt.Format(time.RFC3339),
        })
    }

    return response, nil
}

//create party
func (s *PartyService) CreateParty(
	leaderID string,
	req dto.CreatePartyRequest,
) (*dto.PartyResponse, error) {

	userUUID, err := uuid.Parse(leaderID)
	if err != nil {
		return nil, errors.New("invalid leader id")
	}

	_, err = s.userRepo.GetUserByID(userUUID.String())
		if err != nil {
			return nil, errors.New("leader not found")
		}

	party := models.Party{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		LogoURL:     req.LogoURL,
		BannerURL:   req.BannerURL,
		LeaderID:    userUUID,
		IsHiring:    req.IsHiring,
		// Status:      models.PartyPending,
	}

	if err := s.partyRepo.CreateParty(&party); err != nil {
		return nil, err
	}

	member := models.PartyMember{
		PartyID:   party.ID,
		UserID:   userUUID,
		Role: models.PartyRoleLeader,
		Status:   models.MemberActive,
		JoinedAt: func() *time.Time { t := time.Now(); return &t }(),
	}

	if err := s.partyRepo.AddMember(&member); err != nil {
		return nil, err
	}

	return &dto.PartyResponse{
		ID:          party.ID.String(),
		Name:        party.Name,
		Slug:        party.Slug,
		Description: party.Description,
		LogoURL:     party.LogoURL,
		BannerURL:   party.BannerURL,
		LeaderID:    party.LeaderID.String(),
		IsHiring:    party.IsHiring,
		IsVerified:  party.IsVerified,
		// Status:      string(party.Status),
		CreatedAt:   party.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   party.UpdatedAt.Format(time.RFC3339),
	}, nil
}

//update the party
func (s *PartyService) UpdateParty(
	leaderID string,
	partyID string,
	req dto.UpdatePartyRequest,
) error {

	party, err := s.partyRepo.FindByUUID(partyID)
	if err != nil {
		return errors.New("party not found")
	}

	if party.LeaderID.String() != leaderID {
		return errors.New("only party leader can update the party")
	}

	if req.Name != "" {
		party.Name = req.Name
	}

	if req.Description != "" {
		party.Description = req.Description
	}

	if req.LogoURL != "" {
		party.LogoURL = req.LogoURL
	}

	if req.BannerURL != "" {
		party.BannerURL = req.BannerURL
	}

	party.IsHiring = req.IsHiring

	return s.partyRepo.UpdateParty(party)
}

//delete party
func (s *PartyService) DeleteParty(
	leaderID string,
	partyID string,
) error {

	party, err := s.partyRepo.FindByUUID(partyID)
	if err != nil {
		return errors.New("party not found")
	}

	if party.LeaderID.String() != leaderID {
		return errors.New("only party leader can delete the party")
	}

	return s.partyRepo.DeleteParty(party)
}

//to get party info
func (s *PartyService) GetParty(
    partyID string,
) (*dto.PartyDetailsResponse, error) {
    party, err := s.partyRepo.FindByUUID(partyID)
    if err != nil {
        return nil, errors.New("party not found")
    }

    response := &dto.PartyDetailsResponse{
        ID:          party.ID.String(),
        Name:        party.Name,
        Slug:        party.Slug,
        Description: party.Description,
        LogoURL:     party.LogoURL,
        BannerURL:   party.BannerURL,
        LeaderID:    party.LeaderID.String(),
        IsHiring:    party.IsHiring,
        IsVerified:  party.IsVerified,
        Members:     make([]dto.PartyMemberResponse, 0, len(party.Members)),
        Portfolio:   make([]dto.PartyPortfolioResponse, 0, len(party.Portfolio)),
        Skills:      make([]dto.PartySkillResponse, 0, len(party.Skills)),
        CreatedAt:   party.CreatedAt.Format(time.RFC3339),
        UpdatedAt:   party.UpdatedAt.Format(time.RFC3339),
    }

    for _, member := range party.Members {
        memberResponse := dto.PartyMemberResponse{
            ID:        member.ID.String(),
            PartyID:   member.PartyID.String(),
            UserID:    member.UserID.String(),
            Role:      string(member.Role),
            Status:    string(member.Status),
            CreatedAt: member.CreatedAt.Format(time.RFC3339),
            UpdatedAt: member.UpdatedAt.Format(time.RFC3339),
        }

        if member.JoinedAt != nil {
            memberResponse.JoinedAt = member.JoinedAt.Format(time.RFC3339)
        }

        response.Members = append(response.Members, memberResponse)
    }

    for _, portfolio := range party.Portfolio {
        response.Portfolio = append(
            response.Portfolio,
            dto.PartyPortfolioResponse{
                ID:          portfolio.ID.String(),
                PartyID:     portfolio.PartyID.String(),
                Title:       portfolio.Title,
                Description: portfolio.Description,
                ImageURL:    portfolio.ImageURL,
                ProjectURL:  portfolio.ProjectURL,
                GithubURL:   portfolio.GithubURL,
                CreatedAt:   portfolio.CreatedAt.Format(time.RFC3339),
                UpdatedAt:   portfolio.UpdatedAt.Format(time.RFC3339),
            },
        )
    }

    for _, skill := range party.Skills {
        response.Skills = append(
            response.Skills,
            dto.PartySkillResponse{
                ID:              skill.ID.String(),
                PartyID:          skill.PartyID.String(),
                SkillName:       skill.SkillName,
                ExperienceLevel: skill.ExperienceLevel,
                CreatedAt:       skill.CreatedAt.Format(time.RFC3339),
                UpdatedAt:       skill.UpdatedAt.Format(time.RFC3339),
            },
        )
    }

    return response, nil
}

//get my Parties
func (s *PartyService) GetMyParties(
	userID string,
) (*dto.PartyListResponse, error) {

	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user id")
	}

	members, err := s.partyRepo.FindPartyByMember(id)
	if err != nil {
		return nil, err
	}

	response := dto.PartyListResponse{}

	for _, member := range members {

		response.Parties = append(response.Parties, dto.PartyResponse{
			ID:          member.Party.ID.String(),
			Name:        member.Party.Name,
			Slug:        member.Party.Slug,
			Description: member.Party.Description,
			LogoURL:     member.Party.LogoURL,
			BannerURL:   member.Party.BannerURL,
			LeaderID:    member.Party.LeaderID.String(),
			IsHiring:    member.Party.IsHiring,
			IsVerified:  member.Party.IsVerified,
			// Status:      string(member.Party.Status),
			CreatedAt:   member.Party.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   member.Party.UpdatedAt.Format(time.RFC3339),
		})
	}

	response.Total = len(response.Parties)

	return &response, nil
}

func (s *PartyService) GetAvailableParties(
    userID string,
    search string,
    page int,
    limit int,
) (*dto.AvailablePartyListResponse, error) {
    userUUID, err := uuid.Parse(userID)
    if err != nil {
        return nil, errors.New("invalid user id")
    }

    user, err := s.userRepo.GetUserByID(userID)
    if err != nil {
        return nil, errors.New("user not found")
    }

    if user.AccountType == nil ||
        *user.AccountType != models.AccountTypeIndividual {
        return nil, ErrIndividualAccountRequired
    }

    if user.Status != models.StatusActive {
        return nil, ErrActiveAccountRequired
    }

    if page < 1 {
        page = 1
    }

    if limit < 1 {
        limit = 20
    }

    if limit > 100 {
        limit = 100
    }

    offset := (page - 1) * limit

    parties, total, err := s.partyRepo.FindAvailableForUser(
        userUUID,
        search,
        offset,
        limit,
    )
    if err != nil {
        return nil, err
    }

    response := &dto.AvailablePartyListResponse{
        Parties: make([]dto.PartyResponse, 0, len(parties)),
        Total:   int(total),
        Page:    page,
        Limit:   limit,
    }

    for _, party := range parties {
        response.Parties = append(response.Parties, dto.PartyResponse{
            ID:          party.ID.String(),
            Name:        party.Name,
            Slug:        party.Slug,
            Description: party.Description,
            LogoURL:     party.LogoURL,
            BannerURL:   party.BannerURL,
            LeaderID:    party.LeaderID.String(),
            IsHiring:    party.IsHiring,
            IsVerified:  party.IsVerified,
            // Status:      string(party.Status),
            CreatedAt:   party.CreatedAt.Format(time.RFC3339),
            UpdatedAt:   party.UpdatedAt.Format(time.RFC3339),
        })
    }

    return response, nil
}

//InviteMember
func (s *PartyService) InviteMember(
	leaderID string,
	partyID string,
	req dto.InviteMemberRequest,
) error {

	party, err := s.partyRepo.FindByUUID(partyID)
	if err != nil {
		return errors.New("party not found")
	}

	if party.LeaderID.String() != leaderID {
		return errors.New("only party leader can invite members")
	}

	user, err := s.userRepo.GetUserByID(req.UserID)
	if err != nil {
		return errors.New("user not found")
	}

	_, err = s.partyRepo.FindMember(party.ID, user.ID)
	if err == nil {
		return errors.New("user is already a party member")
	}

	_, err = s.partyRepo.FindInvitationByUser(party.ID, user.ID)
	if err == nil {
		return errors.New("invitation already sent")
	}

	invitation := models.PartyInvitation{
		PartyID:         party.ID,
		InvitedByID:    party.LeaderID,
		InvitedUserID:  user.ID,
		Message:        req.Message,
		Status:         models.InvitationPending,
	}

	return s.partyRepo.CreateInvitation(&invitation)
}

//accept and reject the invite

func (s *PartyService) AcceptInvitation(
	userID string,
	invitationID string,
) error {

	id, err := uuid.Parse(invitationID)
	if err != nil {
		return errors.New("invalid invitation id")
	}

	invitation, err := s.partyRepo.FindInvitationByID(id)
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

	if err := s.partyRepo.UpdateInvitation(invitation); err != nil {
		return err
	}

	member := models.PartyMember{
		PartyID: invitation.PartyID,
		UserID: invitation.InvitedUserID,
		Role:models.PartyRoleMember,
		Status: models.MemberActive,
		JoinedAt: &now,
	}

	return s.partyRepo.AddMember(&member)
}

func (s *PartyService) RejectInvitation(
	userID string,
	invitationID string,
) error {

	id, err := uuid.Parse(invitationID)
	if err != nil {
		return errors.New("invalid invitation id")
	}

	invitation, err := s.partyRepo.FindInvitationByID(id)
	if err != nil {
		return errors.New("invitation not found")
	}

	if invitation.InvitedUserID.String() != userID {
		return errors.New("unauthorized")
	}

	now := time.Now()

	invitation.Status = models.InvitationRejected
	invitation.RespondedAt = &now

	return s.partyRepo.UpdateInvitation(invitation)
}

// to remove the member
func (s *PartyService) RemoveMember(
	leaderID string,
	partyID string,
	memberID string,
) error {

	party, err := s.partyRepo.FindByUUID(partyID)
	if err != nil {
		return errors.New("party not found")
	}

	if party.LeaderID.String() != leaderID {
		return errors.New("only leader can remove members")
	}

	userUUID, err := uuid.Parse(memberID)
	if err != nil {
		return errors.New("invalid member id")
	}

	member, err := s.partyRepo.FindMember(party.ID, userUUID)
	if err != nil {
		return errors.New("member not found")
	}

	return s.partyRepo.DeleteMember(member)
}


//leave the party
func (s *PartyService) LeaveParty(
	userID string,
	partyID string,
) error {

	party, err := s.partyRepo.FindByUUID(partyID)
	if err != nil {
		return errors.New("party not found")
	}

	if party.LeaderID.String() == userID {
		return errors.New("party leader cannot leave the party")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user id")
	}

	member, err := s.partyRepo.FindMember(party.ID, userUUID)
	if err != nil {
		return errors.New("member not found")
	}

	return s.partyRepo.DeleteMember(member)
}

//create protfolio
func (s *PartyService) CreatePortfolio(
	leaderID string,
	partyID string,
	req dto.CreatePortfolioRequest,
) error {

	party, err := s.partyRepo.FindByUUID(partyID)
	if err != nil {
		return errors.New("party not found")
	}

	if party.LeaderID.String() != leaderID {
		return errors.New("only party leader can add portfolio")
	}

	portfolio := models.PartyPortfolio{
		PartyID:      party.ID,
		Title:       req.Title,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		ProjectURL:  req.ProjectURL,
		GithubURL:   req.GithubURL,
	}

	return s.partyRepo.CreatePortfolio(&portfolio)
}

//update portfolio
func (s *PartyService) UpdatePortfolio(
	leaderID string,
	portfolioID string,
	req dto.UpdatePortfolioRequest,
) error {

	id, err := uuid.Parse(portfolioID)
	if err != nil {
		return errors.New("invalid portfolio id")
	}

	portfolio, err := s.partyRepo.FindPortfolioByID(id)
	if err != nil {
		return errors.New("portfolio not found")
	}

	party, err := s.partyRepo.FindByID(portfolio.PartyID)
	if err != nil {
		return err
	}

	if party.LeaderID.String() != leaderID {
		return errors.New("unauthorized")
	}

	if req.Title != "" {
		portfolio.Title = req.Title
	}

	if req.Description != "" {
		portfolio.Description = req.Description
	}

	if req.ImageURL != "" {
		portfolio.ImageURL = req.ImageURL
	}

	if req.ProjectURL != "" {
		portfolio.ProjectURL = req.ProjectURL
	}

	if req.GithubURL != "" {
		portfolio.GithubURL = req.GithubURL
	}

	return s.partyRepo.UpdatePortfolio(portfolio)
}

//delete portfolio
func (s *PartyService) DeletePortfolio(
	leaderID string,
	portfolioID string,
) error {

	id, err := uuid.Parse(portfolioID)
	if err != nil {
		return errors.New("invalid portfolio id")
	}

	portfolio, err := s.partyRepo.FindPortfolioByID(id)
	if err != nil {
		return errors.New("portfolio not found")
	}

	party, err := s.partyRepo.FindByID(portfolio.PartyID)
	if err != nil {
		return err
	}

	if party.LeaderID.String() != leaderID {
		return errors.New("unauthorized")
	}

	return s.partyRepo.DeletePortfolio(portfolio)
}

//skill services 
func (s *PartyService) AddSkill(
	leaderID string,
	partyID string,
	req dto.AddSkillRequest,
) error {

	party, err := s.partyRepo.FindByUUID(partyID)
	if err != nil {
		return errors.New("party not found")
	}

	if party.LeaderID.String() != leaderID {
		return errors.New("unauthorized")
	}

	skill := models.PartySkill{
		PartyID:          party.ID,
		SkillName:       req.SkillName,
		ExperienceLevel: req.ExperienceLevel,
	}

	return s.partyRepo.AddSkill(&skill)
}

func (s *PartyService) UpdateSkill(
	leaderID string,
	skillID string,
	req dto.AddSkillRequest,
) error {

	id, err := uuid.Parse(skillID)
	if err != nil {
		return errors.New("invalid skill id")
	}

	skill, err := s.partyRepo.FindSkillByID(id)
	if err != nil {
		return errors.New("skill not found")
	}

	party, err := s.partyRepo.FindByID(skill.PartyID)
	if err != nil {
		return err
	}

	if party.LeaderID.String() != leaderID {
		return errors.New("unauthorized")
	}

	skill.SkillName = req.SkillName
	skill.ExperienceLevel = req.ExperienceLevel

	return s.partyRepo.UpdateSkill(skill)
}

func (s *PartyService) DeleteSkill(
	leaderID string,
	skillID string,
) error {

	id, err := uuid.Parse(skillID)
	if err != nil {
		return errors.New("invalid skill id")
	}

	skill, err := s.partyRepo.FindSkillByID(id)
	if err != nil {
		return errors.New("skill not found")
	}

	party, err := s.partyRepo.FindByID(skill.PartyID)
	if err != nil {
		return err
	}

	if party.LeaderID.String() != leaderID {
		return errors.New("unauthorized")
	}

	return s.partyRepo.DeleteSkill(skill)
}