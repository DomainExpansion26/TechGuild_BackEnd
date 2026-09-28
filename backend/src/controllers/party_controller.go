package controllers

import (
	"context"
    "errors"

    "techguild-backend/src/dto"
    "techguild-backend/src/services"
    "techguild-backend/src/utils"

    "github.com/danielgtaylor/huma/v2"
)

// ---------- Party CRUD ----------

func CreatePartyHandler(ctx context.Context, input *dto.CreatePartyInput) (*dto.CreatePartyOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	res, err := partyService.CreateParty(userID, input.Body)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.CreatePartyOutput{Body: *res}, nil
}

func UpdatePartyHandler(ctx context.Context, input *dto.UpdatePartyInput) (*dto.UpdatePartyOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	if err := partyService.UpdateParty(userID, input.ID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.UpdatePartyOutput{Body: dto.MessageResponse{Message: "Party updated successfully"}}, nil
}

func DeletePartyHandler(ctx context.Context, input *dto.DeletePartyInput) (*dto.DeletePartyOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	if err := partyService.DeleteParty(userID, input.ID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.DeletePartyOutput{Body: dto.MessageResponse{Message: "Party deleted successfully"}}, nil
}

//By PartyID
func GetPartyHandler(ctx context.Context,input *dto.GetPartyInput,) (*dto.GetPartyOutput, error) {
    partyService := services.NewPartyService()

    res, err := partyService.GetParty(input.ID)
    if err != nil {
        return nil, huma.Error404NotFound(err.Error())
    }

    return &dto.GetPartyOutput{
        Body: *res,
    }, nil
}

//Available Parties
func GetMyPartiesHandler(ctx context.Context, input *dto.GetMyPartiesInput) (*dto.GetMyPartiesOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	res, err := partyService.GetMyParties(userID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.GetMyPartiesOutput{Body: *res}, nil
}

func GetAvailablePartiesHandler(ctx context.Context,input *dto.GetAvailablePartiesInput,) (*dto.GetAvailablePartiesOutput, error) {
    userID, err := utils.GetUserIDFromHumaContext(ctx)
    if err != nil {
        return nil, huma.Error401Unauthorized(err.Error())
    }

    partyService := services.NewPartyService()

    res, err := partyService.GetAvailableParties(
        userID,
        input.Search,
        input.Page,
        input.Limit,
    )
    if err != nil {
        if errors.Is(err, services.ErrIndividualAccountRequired) ||
            errors.Is(err, services.ErrActiveAccountRequired) {
            return nil, huma.Error403Forbidden(err.Error())
        }

        return nil, huma.Error400BadRequest(err.Error())
    }

    return &dto.GetAvailablePartiesOutput{
        Body: *res,
    }, nil
}

//new
func GetPartyInvitationsHandler(
    ctx context.Context,
    input *dto.GetPartyInvitationsInput,
) (*dto.GetPartyInvitationsOutput, error) {
    if _, err := utils.GetUserIDFromHumaContext(ctx); err != nil {
        return nil, huma.Error401Unauthorized(err.Error())
    }

    partyService := services.NewPartyService()

    res, err := partyService.GetInvitations(input.PartyID)
    if err != nil {
        return nil, huma.Error404NotFound(err.Error())
    }

    return &dto.GetPartyInvitationsOutput{
        Body: *res,
    }, nil
}


// ---------- Members ----------

func InviteMemberHandler(ctx context.Context, input *dto.InviteMemberInput) (*dto.InviteMemberOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	if err := partyService.InviteMember(userID, input.PartyID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.InviteMemberOutput{Body: dto.MessageResponse{Message: "Invitation sent successfully"}}, nil
}

func AcceptInvitationHandler(ctx context.Context, input *dto.AcceptInvitationInput) (*dto.AcceptInvitationOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	if err := partyService.AcceptInvitation(userID, input.InvitationID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.AcceptInvitationOutput{Body: dto.MessageResponse{Message: "Invitation accepted successfully"}}, nil
}

func RejectInvitationHandler(ctx context.Context, input *dto.RejectInvitationInput) (*dto.RejectInvitationOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	if err := partyService.RejectInvitation(userID, input.InvitationID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.RejectInvitationOutput{Body: dto.MessageResponse{Message: "Invitation rejected successfully"}}, nil
}

func RemoveMemberHandler(ctx context.Context, input *dto.RemoveMemberInput) (*dto.RemoveMemberOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	if err := partyService.RemoveMember(userID, input.PartyID, input.MemberID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.RemoveMemberOutput{Body: dto.MessageResponse{Message: "Member removed successfully"}}, nil
}

func LeavePartyHandler(ctx context.Context, input *dto.LeavePartyInput) (*dto.LeavePartyOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	if err := partyService.LeaveParty(userID, input.PartyID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.LeavePartyOutput{Body: dto.MessageResponse{Message: "Left Party successfully"}}, nil
}

//new
func GetPartyMembersHandler(
    ctx context.Context,
    input *dto.GetPartyMembersInput,
) (*dto.GetPartyMembersOutput, error) {
    if _, err := utils.GetUserIDFromHumaContext(ctx); err != nil {
        return nil, huma.Error401Unauthorized(err.Error())
    }

    partyService := services.NewPartyService()

    res, err := partyService.GetMembers(input.PartyID)
    if err != nil {
        return nil, huma.Error404NotFound(err.Error())
    }

    return &dto.GetPartyMembersOutput{
        Body: *res,
    }, nil
}

// ---------- Portfolio ----------

func CreatePortfolioHandler(ctx context.Context, input *dto.CreatePortfolioInput) (*dto.CreatePortfolioOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	if err := partyService.CreatePortfolio(userID, input.PartyID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.CreatePortfolioOutput{Body: dto.MessageResponse{Message: "Portfolio created successfully"}}, nil
}

func UpdatePortfolioHandler(ctx context.Context, input *dto.UpdatePortfolioInput) (*dto.UpdatePortfolioOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	if err := partyService.UpdatePortfolio(userID, input.PortfolioID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.UpdatePortfolioOutput{Body: dto.MessageResponse{Message: "Portfolio updated successfully"}}, nil
}

func DeletePortfolioHandler(ctx context.Context, input *dto.DeletePortfolioInput) (*dto.DeletePortfolioOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	if err := partyService.DeletePortfolio(userID, input.PortfolioID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.DeletePortfolioOutput{Body: dto.MessageResponse{Message: "Portfolio deleted successfully"}}, nil
}

//new
func GetPartyPortfolioHandler(
    ctx context.Context,
    input *dto.GetPartyPortfolioInput,
) (*dto.GetPartyPortfolioOutput, error) {
    if _, err := utils.GetUserIDFromHumaContext(ctx); err != nil {
        return nil, huma.Error401Unauthorized(err.Error())
    }

    partyService := services.NewPartyService()

    res, err := partyService.GetPortfolio(input.PartyID)
    if err != nil {
        return nil, huma.Error404NotFound(err.Error())
    }

    return &dto.GetPartyPortfolioOutput{
        Body: *res,
    }, nil
}


// ---------- Skills ----------

func AddSkillHandler(ctx context.Context, input *dto.AddSkillInput) (*dto.AddSkillOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	if err := partyService.AddSkill(userID, input.PartyID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.AddSkillOutput{Body: dto.MessageResponse{Message: "Skill added successfully"}}, nil
}

func UpdateSkillHandler(ctx context.Context, input *dto.UpdateSkillInput) (*dto.UpdateSkillOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	if err := partyService.UpdateSkill(userID, input.SkillID, input.Body); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.UpdateSkillOutput{Body: dto.MessageResponse{Message: "Skill updated successfully"}}, nil
}

func DeleteSkillHandler(ctx context.Context, input *dto.DeleteSkillInput) (*dto.DeleteSkillOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}

	partyService := services.NewPartyService()
	if err := partyService.DeleteSkill(userID, input.SkillID); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}

	return &dto.DeleteSkillOutput{Body: dto.MessageResponse{Message: "Skill deleted successfully"}}, nil
}

//new
func GetPartySkillsHandler(
    ctx context.Context,
    input *dto.GetPartySkillsInput,
) (*dto.GetPartySkillsOutput, error) {
    if _, err := utils.GetUserIDFromHumaContext(ctx); err != nil {
        return nil, huma.Error401Unauthorized(err.Error())
    }

    partyService := services.NewPartyService()

    res, err := partyService.GetSkills(input.PartyID)
    if err != nil {
        return nil, huma.Error404NotFound(err.Error())
    }

    return &dto.GetPartySkillsOutput{
        Body: *res,
    }, nil
}
