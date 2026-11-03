package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"
)

func CreateGuildhallPostHandler(
	ctx context.Context,
	input *dto.CreateGuildhallPostInput,
) (*dto.CreateGuildhallPostOutput, error) {
	authorID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	post, err := services.NewGuildhallService().CreatePost(authorID, input.Body)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGuildhallInvalidUserID),
			errors.Is(err, services.ErrGuildhallUserNotFound):
			return nil, huma.Error401Unauthorized(err.Error())
		case errors.Is(err, services.ErrGuildhallInactiveUser):
			return nil, huma.Error403Forbidden(err.Error())
		case errors.Is(err, services.ErrGuildhallInvalidPost):
			return nil, huma.Error400BadRequest(err.Error())
		default:
			return nil, huma.Error500InternalServerError("failed to create Guildhall post")
		}
	}

	return &dto.CreateGuildhallPostOutput{
		Status: http.StatusCreated,
		Body:   *post,
	}, nil
}

func GetGuildhallPostHandler(
	ctx context.Context,
	input *dto.GetGuildhallPostInput,
) (*dto.GetGuildhallPostOutput, error) {
	viewerID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	post, err := services.NewGuildhallService().GetPost(input.PostID, viewerID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGuildhallInvalidUserID):
			return nil, huma.Error401Unauthorized(err.Error())
		case errors.Is(err, services.ErrGuildhallInvalidPostID):
			return nil, huma.Error400BadRequest(err.Error())
		case errors.Is(err, services.ErrGuildhallPostNotFound):
			return nil, huma.Error404NotFound(err.Error())
		default:
			return nil, huma.Error500InternalServerError("failed to load Guildhall post")
		}
	}

	return &dto.GetGuildhallPostOutput{Body: *post}, nil
}

func ListGuildhallPostsHandler(
	ctx context.Context,
	input *dto.ListGuildhallPostsInput,
) (*dto.ListGuildhallPostsOutput, error) {
	viewerID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	posts, err := services.NewGuildhallService().ListPosts(
		input.Type,
		input.Page,
		input.Limit,
		viewerID,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGuildhallInvalidUserID):
			return nil, huma.Error401Unauthorized(err.Error())
		case errors.Is(err, services.ErrGuildhallInvalidPost):
			return nil, huma.Error400BadRequest(err.Error())
		default:
			return nil, huma.Error500InternalServerError("failed to list Guildhall posts")
		}
	}

	return &dto.ListGuildhallPostsOutput{Body: *posts}, nil
}

func UpdateGuildhallPostHandler(
	ctx context.Context,
	input *dto.UpdateGuildhallPostInput,
) (*dto.UpdateGuildhallPostOutput, error) {
	authorID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	post, err := services.NewGuildhallService().UpdatePost(
		authorID,
		input.PostID,
		input.Body,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGuildhallInvalidUserID):
			return nil, huma.Error401Unauthorized(err.Error())
		case errors.Is(err, services.ErrGuildhallInvalidPostID),
			errors.Is(err, services.ErrGuildhallInvalidPost):
			return nil, huma.Error400BadRequest(err.Error())
		case errors.Is(err, services.ErrGuildhallInvalidPoll):
			return nil, huma.Error400BadRequest(err.Error())
		case errors.Is(err, services.ErrGuildhallPollOptionsLocked):
			return nil, huma.Error409Conflict(err.Error())
		case errors.Is(err, services.ErrGuildhallPostNotFound):
			return nil, huma.Error404NotFound(err.Error())
		case errors.Is(err, services.ErrGuildhallPostNotOwned):
			return nil, huma.Error403Forbidden(err.Error())
		default:
			return nil, huma.Error500InternalServerError("failed to update Guildhall post")
		}
	}

	return &dto.UpdateGuildhallPostOutput{Body: *post}, nil
}

func DeleteGuildhallPostHandler(
	ctx context.Context,
	input *dto.DeleteGuildhallPostInput,
) (*dto.DeleteGuildhallPostOutput, error) {
	authorID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	err = services.NewGuildhallService().DeletePost(authorID, input.PostID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGuildhallInvalidUserID):
			return nil, huma.Error401Unauthorized(err.Error())
		case errors.Is(err, services.ErrGuildhallInvalidPostID):
			return nil, huma.Error400BadRequest(err.Error())
		case errors.Is(err, services.ErrGuildhallPostNotFound):
			return nil, huma.Error404NotFound(err.Error())
		case errors.Is(err, services.ErrGuildhallPostNotOwned):
			return nil, huma.Error403Forbidden(err.Error())
		default:
			return nil, huma.Error500InternalServerError("failed to delete Guildhall post")
		}
	}

	return &dto.DeleteGuildhallPostOutput{
		Body: dto.DeleteGuildhallPostResponse{
			Message: "Guildhall post deleted successfully",
		},
	}, nil
}

func CreateGuildhallCommentHandler(
	ctx context.Context,
	input *dto.CreateGuildhallCommentInput,
) (*dto.CreateGuildhallCommentOutput, error) {
	authorID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	comment, err := services.NewGuildhallService().CreateComment(authorID, input.PostID, input.Body)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGuildhallInvalidUserID),
			errors.Is(err, services.ErrGuildhallUserNotFound):
			return nil, huma.Error401Unauthorized(err.Error())
		case errors.Is(err, services.ErrGuildhallInactiveUser):
			return nil, huma.Error403Forbidden(err.Error())
		case errors.Is(err, services.ErrGuildhallInvalidPostID),
			errors.Is(err, services.ErrGuildhallInvalidComment):
			return nil, huma.Error400BadRequest(err.Error())
		case errors.Is(err, services.ErrGuildhallPostNotFound):
			return nil, huma.Error404NotFound(err.Error())
		default:
			return nil, huma.Error500InternalServerError("failed to create Guildhall comment")
		}
	}

	return &dto.CreateGuildhallCommentOutput{
		Status: http.StatusCreated,
		Body:   *comment,
	}, nil
}

func ListGuildhallCommentsHandler(
	ctx context.Context,
	input *dto.ListGuildhallCommentsInput,
) (*dto.ListGuildhallCommentsOutput, error) {
	if _, err := utils.GetUserIDFromHumaContext(ctx); err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	comments, err := services.NewGuildhallService().ListComments(input.PostID, input.Page, input.Limit)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGuildhallInvalidPostID):
			return nil, huma.Error400BadRequest(err.Error())
		case errors.Is(err, services.ErrGuildhallPostNotFound):
			return nil, huma.Error404NotFound(err.Error())
		default:
			return nil, huma.Error500InternalServerError("failed to list Guildhall comments")
		}
	}
	return &dto.ListGuildhallCommentsOutput{Body: *comments}, nil
}

func UpdateGuildhallCommentHandler(
	ctx context.Context,
	input *dto.UpdateGuildhallCommentInput,
) (*dto.UpdateGuildhallCommentOutput, error) {
	authorID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	comment, err := services.NewGuildhallService().UpdateComment(authorID, input.CommentID, input.Body)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGuildhallInvalidUserID):
			return nil, huma.Error401Unauthorized(err.Error())
		case errors.Is(err, services.ErrGuildhallInvalidCommentID),
			errors.Is(err, services.ErrGuildhallInvalidComment):
			return nil, huma.Error400BadRequest(err.Error())
		case errors.Is(err, services.ErrGuildhallCommentNotFound):
			return nil, huma.Error404NotFound(err.Error())
		case errors.Is(err, services.ErrGuildhallCommentNotOwned):
			return nil, huma.Error403Forbidden(err.Error())
		default:
			return nil, huma.Error500InternalServerError("failed to update Guildhall comment")
		}
	}
	return &dto.UpdateGuildhallCommentOutput{Body: *comment}, nil
}

func DeleteGuildhallCommentHandler(
	ctx context.Context,
	input *dto.DeleteGuildhallCommentInput,
) (*dto.DeleteGuildhallCommentOutput, error) {
	authorID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	err = services.NewGuildhallService().DeleteComment(authorID, input.CommentID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGuildhallInvalidUserID):
			return nil, huma.Error401Unauthorized(err.Error())
		case errors.Is(err, services.ErrGuildhallInvalidCommentID):
			return nil, huma.Error400BadRequest(err.Error())
		case errors.Is(err, services.ErrGuildhallCommentNotFound):
			return nil, huma.Error404NotFound(err.Error())
		case errors.Is(err, services.ErrGuildhallCommentNotOwned):
			return nil, huma.Error403Forbidden(err.Error())
		default:
			return nil, huma.Error500InternalServerError("failed to delete Guildhall comment")
		}
	}
	return &dto.DeleteGuildhallCommentOutput{
		Body: dto.DeleteGuildhallCommentResponse{Message: "Guildhall comment deleted successfully"},
	}, nil
}

func LikeGuildhallPostHandler(
	ctx context.Context,
	input *dto.LikeGuildhallPostInput,
) (*dto.LikeGuildhallPostOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	result, err := services.NewGuildhallService().LikePost(userID, input.PostID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGuildhallInvalidUserID),
			errors.Is(err, services.ErrGuildhallUserNotFound):
			return nil, huma.Error401Unauthorized(err.Error())
		case errors.Is(err, services.ErrGuildhallInactiveUser):
			return nil, huma.Error403Forbidden(err.Error())
		case errors.Is(err, services.ErrGuildhallInvalidPostID):
			return nil, huma.Error400BadRequest(err.Error())
		case errors.Is(err, services.ErrGuildhallPostNotFound):
			return nil, huma.Error404NotFound(err.Error())
		default:
			return nil, huma.Error500InternalServerError("failed to like Guildhall post")
		}
	}
	return &dto.LikeGuildhallPostOutput{Body: *result}, nil
}

func UnlikeGuildhallPostHandler(
	ctx context.Context,
	input *dto.UnlikeGuildhallPostInput,
) (*dto.UnlikeGuildhallPostOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	result, err := services.NewGuildhallService().UnlikePost(userID, input.PostID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGuildhallInvalidUserID),
			errors.Is(err, services.ErrGuildhallUserNotFound):
			return nil, huma.Error401Unauthorized(err.Error())
		case errors.Is(err, services.ErrGuildhallInactiveUser):
			return nil, huma.Error403Forbidden(err.Error())
		case errors.Is(err, services.ErrGuildhallInvalidPostID):
			return nil, huma.Error400BadRequest(err.Error())
		case errors.Is(err, services.ErrGuildhallPostNotFound):
			return nil, huma.Error404NotFound(err.Error())
		default:
			return nil, huma.Error500InternalServerError("failed to unlike Guildhall post")
		}
	}
	return &dto.UnlikeGuildhallPostOutput{Body: *result}, nil
}

func CastGuildhallVoteHandler(
	ctx context.Context,
	input *dto.CastGuildhallVoteInput,
) (*dto.CastGuildhallVoteOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	results, err := services.NewGuildhallService().CastPollVote(userID, input.PostID, input.Body)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGuildhallInvalidUserID),
			errors.Is(err, services.ErrGuildhallUserNotFound):
			return nil, huma.Error401Unauthorized(err.Error())
		case errors.Is(err, services.ErrGuildhallInactiveUser):
			return nil, huma.Error403Forbidden(err.Error())
		case errors.Is(err, services.ErrGuildhallInvalidPostID),
			errors.Is(err, services.ErrGuildhallInvalidPoll):
			return nil, huma.Error400BadRequest(err.Error())
		case errors.Is(err, services.ErrGuildhallPostNotFound):
			return nil, huma.Error404NotFound(err.Error())
		default:
			return nil, huma.Error500InternalServerError("failed to cast Guildhall poll vote")
		}
	}
	return &dto.CastGuildhallVoteOutput{Body: *results}, nil
}

func GetGuildhallPollResultsHandler(
	ctx context.Context,
	input *dto.GetGuildhallPollResultsInput,
) (*dto.GetGuildhallPollResultsOutput, error) {
	userID, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized("user is not authenticated")
	}

	results, err := services.NewGuildhallService().GetPollResults(userID, input.PostID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrGuildhallInvalidUserID),
			errors.Is(err, services.ErrGuildhallUserNotFound):
			return nil, huma.Error401Unauthorized(err.Error())
		case errors.Is(err, services.ErrGuildhallInactiveUser),
			errors.Is(err, services.ErrGuildhallPollNotVoted):
			return nil, huma.Error403Forbidden(err.Error())
		case errors.Is(err, services.ErrGuildhallInvalidPostID),
			errors.Is(err, services.ErrGuildhallInvalidPoll):
			return nil, huma.Error400BadRequest(err.Error())
		case errors.Is(err, services.ErrGuildhallPostNotFound):
			return nil, huma.Error404NotFound(err.Error())
		default:
			return nil, huma.Error500InternalServerError("failed to load Guildhall poll results")
		}
	}
	return &dto.GetGuildhallPollResultsOutput{Body: *results}, nil
}
