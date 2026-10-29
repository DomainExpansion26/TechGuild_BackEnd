package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"techguild-backend/src/dto"
	"techguild-backend/src/models"
	"techguild-backend/src/repository"
)

var (
	ErrGuildhallInvalidUserID     = errors.New("invalid user id")
	ErrGuildhallUserNotFound      = errors.New("user not found")
	ErrGuildhallInactiveUser      = errors.New("active user account required")
	ErrGuildhallInvalidPost       = errors.New("invalid Guildhall post")
	ErrGuildhallInvalidPostID     = errors.New("invalid Guildhall post id")
	ErrGuildhallPostNotFound      = errors.New("Guildhall post not found")
	ErrGuildhallPostNotOwned      = errors.New("only the post author can modify or delete it")
	ErrGuildhallInvalidComment    = errors.New("invalid Guildhall comment")
	ErrGuildhallInvalidCommentID  = errors.New("invalid Guildhall comment id")
	ErrGuildhallCommentNotFound   = errors.New("Guildhall comment not found")
	ErrGuildhallCommentNotOwned   = errors.New("only the comment author can modify or delete it")
	ErrGuildhallInvalidPoll       = errors.New("Guildhall post is not a valid poll")
	ErrGuildhallPollNotVoted      = errors.New("vote in this poll to view its results")
	ErrGuildhallPollOptionsLocked = errors.New("poll options cannot be changed after voting starts")
)

type GuildhallService struct {
	postRepo     *repository.GuildhallRepository
	commentRepo  *repository.GuildhallCommentRepository
	reactionRepo *repository.GuildhallReactionRepository
	voteRepo     *repository.GuildhallVoteRepository
	userRepo     repository.UserRepository
}

func NewGuildhallService() *GuildhallService {
	return &GuildhallService{
		postRepo:     repository.NewGuildhallRepository(),
		commentRepo:  repository.NewGuildhallCommentRepository(),
		reactionRepo: repository.NewGuildhallReactionRepository(),
		voteRepo:     repository.NewGuildhallVoteRepository(),
		userRepo:     repository.NewUserRepository(),
	}
}

func (s *GuildhallService) CreatePost(
	authorID string,
	req dto.CreateGuildhallPostBody,
) (*dto.GuildhallPostResponse, error) {
	authorUUID, err := uuid.Parse(authorID)
	if err != nil {
		return nil, ErrGuildhallInvalidUserID
	}

	author, err := s.userRepo.GetUserByID(authorID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGuildhallUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, ErrGuildhallUserNotFound
	}
	if author.Status != models.StatusActive {
		return nil, ErrGuildhallInactiveUser
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, fmt.Errorf("%w: content is required", ErrGuildhallInvalidPost)
	}

	title := strings.TrimSpace(req.Title)
	if len(title) > 200 {
		return nil, fmt.Errorf("%w: title must be at most 200 characters", ErrGuildhallInvalidPost)
	}

	postType := models.GuildhallPostType(req.Type)
	switch postType {
	case models.GuildhallPostDiscussion,
		models.GuildhallPostQuestion,
		models.GuildhallPostProjectShowcase,
		models.GuildhallPostPoll,
		models.GuildhallPostOpportunity,
		models.GuildhallPostMedia:
	default:
		return nil, fmt.Errorf("%w: unsupported post type", ErrGuildhallInvalidPost)
	}

	if postType == models.GuildhallPostPoll {
		if req.Details == nil {
			return nil, fmt.Errorf("%w: poll details are required", ErrGuildhallInvalidPost)
		}

		options, ok := req.Details["options"].([]any)
		if !ok || len(options) < 2 {
			return nil, fmt.Errorf("%w: a poll requires at least two options", ErrGuildhallInvalidPost)
		}

		seenOptions := make(map[string]struct{}, len(options))
		for _, value := range options {
			option, ok := value.(string)
			if !ok || strings.TrimSpace(option) == "" {
				return nil, fmt.Errorf("%w: poll options must be non-empty strings", ErrGuildhallInvalidPost)
			}

			normalizedOption := strings.ToLower(strings.TrimSpace(option))
			if _, exists := seenOptions[normalizedOption]; exists {
				return nil, fmt.Errorf("%w: poll options must be unique", ErrGuildhallInvalidPost)
			}
			seenOptions[normalizedOption] = struct{}{}
		}
	}

	linkURL := strings.TrimSpace(req.LinkURL)
	if len(linkURL) > 2048 {
		return nil, fmt.Errorf("%w: link_url must be at most 2048 characters", ErrGuildhallInvalidPost)
	}
	if linkURL != "" {
		parsedURL, err := url.ParseRequestURI(linkURL)
		if err != nil ||
			(parsedURL.Scheme != "http" && parsedURL.Scheme != "https") ||
			parsedURL.Host == "" {
			return nil, fmt.Errorf("%w: link_url must be an absolute HTTP or HTTPS URL", ErrGuildhallInvalidPost)
		}
	}

	var detailsJSON []byte
	if req.Details != nil {
		detailsJSON, err = json.Marshal(req.Details)
		if err != nil {
			return nil, fmt.Errorf("%w: details must be valid JSON", ErrGuildhallInvalidPost)
		}
		if len(detailsJSON) > 32*1024 {
			return nil, fmt.Errorf("%w: details must not exceed 32 KB", ErrGuildhallInvalidPost)
		}
	}

	post := models.GuildhallPost{
		AuthorID: authorUUID,
		Type:     postType,
		Title:    title,
		Content:  content,
		LinkURL:  linkURL,
		Details:  detailsJSON,
	}

	if err := s.postRepo.CreatePost(&post); err != nil {
		return nil, err
	}

	var responseDetails map[string]any
	if len(post.Details) > 0 {
		if err := json.Unmarshal(post.Details, &responseDetails); err != nil {
			return nil, fmt.Errorf("decode Guildhall post details: %w", err)
		}
	}

	return &dto.GuildhallPostResponse{
		ID:        post.ID.String(),
		Type:      string(post.Type),
		Title:     post.Title,
		Content:   post.Content,
		LinkURL:   post.LinkURL,
		Details:   responseDetails,
		AuthorID:  post.AuthorID.String(),
		CreatedAt: post.CreatedAt,
	}, nil
}

func (s *GuildhallService) GetPost(postID, viewerID string) (*dto.GuildhallPostResponse, error) {
	postUUID, err := uuid.Parse(postID)
	if err != nil {
		return nil, ErrGuildhallInvalidPostID
	}
	viewerUUID, err := uuid.Parse(viewerID)
	if err != nil {
		return nil, ErrGuildhallInvalidUserID
	}

	post, err := s.postRepo.FindPostByID(postUUID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGuildhallPostNotFound
	}
	if err != nil {
		return nil, err
	}

	summaries, err := s.reactionRepo.Summaries([]uuid.UUID{postUUID}, viewerUUID)
	if err != nil {
		return nil, err
	}
	return buildGuildhallPostResponse(post, summaries[postUUID])
}

func (s *GuildhallService) ListPosts(
	postType string,
	page int,
	limit int,
	viewerID string,
) (*dto.ListGuildhallPostsResponse, error) {
	viewerUUID, err := uuid.Parse(viewerID)
	if err != nil {
		return nil, ErrGuildhallInvalidUserID
	}

	if postType != "" {
		switch models.GuildhallPostType(postType) {
		case models.GuildhallPostDiscussion,
			models.GuildhallPostQuestion,
			models.GuildhallPostProjectShowcase,
			models.GuildhallPostPoll,
			models.GuildhallPostOpportunity,
			models.GuildhallPostMedia:
		default:
			return nil, fmt.Errorf("%w: unsupported post type", ErrGuildhallInvalidPost)
		}
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
	posts, total, err := s.postRepo.ListPosts(postType, offset, limit)
	if err != nil {
		return nil, err
	}
	postIDs := make([]uuid.UUID, 0, len(posts))
	for _, post := range posts {
		postIDs = append(postIDs, post.ID)
	}
	summaries, err := s.reactionRepo.Summaries(postIDs, viewerUUID)
	if err != nil {
		return nil, err
	}

	response := &dto.ListGuildhallPostsResponse{
		Posts: make([]dto.GuildhallPostResponse, 0, len(posts)),
		Total: total,
		Page:  page,
		Limit: limit,
	}

	for _, post := range posts {
		postResponse, err := buildGuildhallPostResponse(&post, summaries[post.ID])
		if err != nil {
			return nil, err
		}
		response.Posts = append(response.Posts, *postResponse)
	}

	return response, nil
}

func (s *GuildhallService) UpdatePost(
	authorID string,
	postID string,
	req dto.UpdateGuildhallPostBody,
) (*dto.GuildhallPostResponse, error) {
	authorUUID, err := uuid.Parse(authorID)
	if err != nil {
		return nil, ErrGuildhallInvalidUserID
	}

	postUUID, err := uuid.Parse(postID)
	if err != nil {
		return nil, ErrGuildhallInvalidPostID
	}

	post, err := s.postRepo.FindPostByID(postUUID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGuildhallPostNotFound
	}
	if err != nil {
		return nil, err
	}

	if post.AuthorID != authorUUID {
		return nil, ErrGuildhallPostNotOwned
	}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if len(title) > 200 {
			return nil, fmt.Errorf("%w: title must be at most 200 characters", ErrGuildhallInvalidPost)
		}
		post.Title = title
	}

	if req.Content != nil {
		content := strings.TrimSpace(*req.Content)
		if content == "" {
			return nil, fmt.Errorf("%w: content cannot be empty", ErrGuildhallInvalidPost)
		}
		post.Content = content
	}

	if req.LinkURL != nil {
		linkURL := strings.TrimSpace(*req.LinkURL)
		if len(linkURL) > 2048 {
			return nil, fmt.Errorf("%w: link_url must be at most 2048 characters", ErrGuildhallInvalidPost)
		}
		if linkURL != "" {
			parsedURL, err := url.ParseRequestURI(linkURL)
			if err != nil ||
				(parsedURL.Scheme != "http" && parsedURL.Scheme != "https") ||
				parsedURL.Host == "" {
				return nil, fmt.Errorf("%w: link_url must be an absolute HTTP or HTTPS URL", ErrGuildhallInvalidPost)
			}
		}
		post.LinkURL = linkURL
	}

	if req.Details != nil {
		if post.Type == models.GuildhallPostPoll {
			newOptions, err := guildhallPollOptionsFromMap(*req.Details)
			if err != nil {
				return nil, err
			}
			currentOptions, err := guildhallPollOptions(post.Details)
			if err != nil {
				return nil, err
			}
			if !sameGuildhallPollOptions(currentOptions, newOptions) {
				hasVotes, err := s.voteRepo.HasVotes(postUUID)
				if err != nil {
					return nil, err
				}
				if hasVotes {
					return nil, ErrGuildhallPollOptionsLocked
				}
			}
		}

		detailsJSON, err := json.Marshal(*req.Details)
		if err != nil {
			return nil, fmt.Errorf("%w: details must be valid JSON", ErrGuildhallInvalidPost)
		}
		if len(detailsJSON) > 32*1024 {
			return nil, fmt.Errorf("%w: details must not exceed 32 KB", ErrGuildhallInvalidPost)
		}
		post.Details = detailsJSON
	}

	if err := s.postRepo.UpdatePost(post); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGuildhallPostNotFound
		}
		return nil, err
	}

	summaries, err := s.reactionRepo.Summaries([]uuid.UUID{postUUID}, authorUUID)
	if err != nil {
		return nil, err
	}
	return buildGuildhallPostResponse(post, summaries[postUUID])
}

func (s *GuildhallService) DeletePost(authorID, postID string) error {
	authorUUID, err := uuid.Parse(authorID)
	if err != nil {
		return ErrGuildhallInvalidUserID
	}

	postUUID, err := uuid.Parse(postID)
	if err != nil {
		return ErrGuildhallInvalidPostID
	}

	post, err := s.postRepo.FindPostByID(postUUID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrGuildhallPostNotFound
	}
	if err != nil {
		return err
	}

	if post.AuthorID != authorUUID {
		return ErrGuildhallPostNotOwned
	}

	if err := s.postRepo.DeletePost(postUUID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGuildhallPostNotFound
		}
		return err
	}

	return nil
}

func (s *GuildhallService) CreateComment(
	authorID string,
	postID string,
	req dto.CreateGuildhallCommentBody,
) (*dto.GuildhallCommentResponse, error) {
	authorUUID, err := uuid.Parse(authorID)
	if err != nil {
		return nil, ErrGuildhallInvalidUserID
	}
	postUUID, err := uuid.Parse(postID)
	if err != nil {
		return nil, ErrGuildhallInvalidPostID
	}

	author, err := s.userRepo.GetUserByID(authorID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGuildhallUserNotFound
	}
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, ErrGuildhallUserNotFound
	}
	if author.Status != models.StatusActive {
		return nil, ErrGuildhallInactiveUser
	}

	if _, err := s.postRepo.FindPostByID(postUUID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGuildhallPostNotFound
	} else if err != nil {
		return nil, err
	}

	content, err := validateGuildhallCommentContent(req.Content)
	if err != nil {
		return nil, err
	}

	comment := models.GuildhallComment{
		PostID:   postUUID,
		AuthorID: authorUUID,
		Content:  content,
	}
	if err := s.commentRepo.Create(&comment); err != nil {
		return nil, err
	}

	return guildhallCommentResponse(&comment), nil
}

func (s *GuildhallService) ListComments(
	postID string,
	page int,
	limit int,
) (*dto.ListGuildhallCommentsResponse, error) {
	postUUID, err := uuid.Parse(postID)
	if err != nil {
		return nil, ErrGuildhallInvalidPostID
	}
	if _, err := s.postRepo.FindPostByID(postUUID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGuildhallPostNotFound
	} else if err != nil {
		return nil, err
	}

	page, limit = normalizeGuildhallPagination(page, limit)
	comments, total, err := s.commentRepo.ListByPostID(postUUID, (page-1)*limit, limit)
	if err != nil {
		return nil, err
	}

	response := &dto.ListGuildhallCommentsResponse{
		Comments: make([]dto.GuildhallCommentResponse, 0, len(comments)),
		Total:    total,
		Page:     page,
		Limit:    limit,
	}
	for i := range comments {
		response.Comments = append(response.Comments, *guildhallCommentResponse(&comments[i]))
	}
	return response, nil
}

func (s *GuildhallService) UpdateComment(
	authorID string,
	commentID string,
	req dto.UpdateGuildhallCommentBody,
) (*dto.GuildhallCommentResponse, error) {
	authorUUID, err := uuid.Parse(authorID)
	if err != nil {
		return nil, ErrGuildhallInvalidUserID
	}
	commentUUID, err := uuid.Parse(commentID)
	if err != nil {
		return nil, ErrGuildhallInvalidCommentID
	}

	comment, err := s.commentRepo.FindByID(commentUUID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGuildhallCommentNotFound
	}
	if err != nil {
		return nil, err
	}
	if comment.AuthorID != authorUUID {
		return nil, ErrGuildhallCommentNotOwned
	}

	comment.Content, err = validateGuildhallCommentContent(req.Content)
	if err != nil {
		return nil, err
	}
	if err := s.commentRepo.Update(comment); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGuildhallCommentNotFound
		}
		return nil, err
	}
	return guildhallCommentResponse(comment), nil
}

func (s *GuildhallService) DeleteComment(authorID, commentID string) error {
	authorUUID, err := uuid.Parse(authorID)
	if err != nil {
		return ErrGuildhallInvalidUserID
	}
	commentUUID, err := uuid.Parse(commentID)
	if err != nil {
		return ErrGuildhallInvalidCommentID
	}

	comment, err := s.commentRepo.FindByID(commentUUID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrGuildhallCommentNotFound
	}
	if err != nil {
		return err
	}
	if comment.AuthorID != authorUUID {
		return ErrGuildhallCommentNotOwned
	}
	if err := s.commentRepo.Delete(commentUUID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGuildhallCommentNotFound
		}
		return err
	}
	return nil
}

func validateGuildhallCommentContent(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("%w: content is required", ErrGuildhallInvalidComment)
	}
	if len([]rune(content)) > 5000 {
		return "", fmt.Errorf("%w: content must be at most 5000 characters", ErrGuildhallInvalidComment)
	}
	return content, nil
}

func guildhallCommentResponse(comment *models.GuildhallComment) *dto.GuildhallCommentResponse {
	return &dto.GuildhallCommentResponse{
		ID:        comment.ID.String(),
		PostID:    comment.PostID.String(),
		AuthorID:  comment.AuthorID.String(),
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}
}

func normalizeGuildhallPagination(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func buildGuildhallPostResponse(
	post *models.GuildhallPost,
	reactionSummary repository.GuildhallReactionSummary,
) (*dto.GuildhallPostResponse, error) {
	var details map[string]any
	if len(post.Details) > 0 {
		if err := json.Unmarshal(post.Details, &details); err != nil {
			return nil, fmt.Errorf("decode Guildhall post details: %w", err)
		}
	}

	return &dto.GuildhallPostResponse{
		ID:        post.ID.String(),
		Type:      string(post.Type),
		Title:     post.Title,
		Content:   post.Content,
		LinkURL:   post.LinkURL,
		Details:   details,
		AuthorID:  post.AuthorID.String(),
		LikeCount: reactionSummary.LikeCount,
		LikedByMe: reactionSummary.LikedByMe,
		CreatedAt: post.CreatedAt,
	}, nil
}

func (s *GuildhallService) LikePost(userID, postID string) (*dto.GuildhallLikeResponse, error) {
	userUUID, postUUID, err := s.parseReactionIDs(userID, postID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureActiveUser(userID); err != nil {
		return nil, err
	}
	if _, err := s.postRepo.FindPostByID(postUUID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGuildhallPostNotFound
	} else if err != nil {
		return nil, err
	}
	if err := s.reactionRepo.Like(postUUID, userUUID); err != nil {
		return nil, err
	}
	return s.guildhallLikeResponse(postUUID, userUUID)
}

func (s *GuildhallService) UnlikePost(userID, postID string) (*dto.GuildhallLikeResponse, error) {
	userUUID, postUUID, err := s.parseReactionIDs(userID, postID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureActiveUser(userID); err != nil {
		return nil, err
	}
	if _, err := s.postRepo.FindPostByID(postUUID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGuildhallPostNotFound
	} else if err != nil {
		return nil, err
	}
	if err := s.reactionRepo.Unlike(postUUID, userUUID); err != nil {
		return nil, err
	}
	return s.guildhallLikeResponse(postUUID, userUUID)
}

func (s *GuildhallService) parseReactionIDs(userID, postID string) (uuid.UUID, uuid.UUID, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrGuildhallInvalidUserID
	}
	postUUID, err := uuid.Parse(postID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrGuildhallInvalidPostID
	}
	return userUUID, postUUID, nil
}

func (s *GuildhallService) ensureActiveUser(userID string) error {
	user, err := s.userRepo.GetUserByID(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrGuildhallUserNotFound
	}
	if err != nil {
		return err
	}
	if user == nil {
		return ErrGuildhallUserNotFound
	}
	if user.Status != models.StatusActive {
		return ErrGuildhallInactiveUser
	}
	return nil
}

func (s *GuildhallService) guildhallLikeResponse(
	postID uuid.UUID,
	userID uuid.UUID,
) (*dto.GuildhallLikeResponse, error) {
	summaries, err := s.reactionRepo.Summaries([]uuid.UUID{postID}, userID)
	if err != nil {
		return nil, err
	}
	summary := summaries[postID]
	return &dto.GuildhallLikeResponse{
		PostID:    postID.String(),
		Liked:     summary.LikedByMe,
		LikeCount: summary.LikeCount,
	}, nil
}

func (s *GuildhallService) CastPollVote(
	userID string,
	postID string,
	req dto.CastGuildhallVoteBody,
) (*dto.GuildhallPollResultsResponse, error) {
	userUUID, postUUID, err := s.parseReactionIDs(userID, postID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureActiveUser(userID); err != nil {
		return nil, err
	}

	post, err := s.postRepo.FindPostByID(postUUID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGuildhallPostNotFound
	}
	if err != nil {
		return nil, err
	}
	if post.Type != models.GuildhallPostPoll {
		return nil, ErrGuildhallInvalidPoll
	}

	options, err := guildhallPollOptions(post.Details)
	if err != nil {
		return nil, err
	}
	selectedOption, err := findGuildhallPollOption(options, req.Option)
	if err != nil {
		return nil, err
	}
	if err := s.voteRepo.Cast(postUUID, userUUID, selectedOption); err != nil {
		return nil, err
	}

	return s.guildhallPollResults(post, userUUID)
}

func (s *GuildhallService) GetPollResults(
	userID string,
	postID string,
) (*dto.GuildhallPollResultsResponse, error) {
	userUUID, postUUID, err := s.parseReactionIDs(userID, postID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureActiveUser(userID); err != nil {
		return nil, err
	}

	post, err := s.postRepo.FindPostByID(postUUID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrGuildhallPostNotFound
	}
	if err != nil {
		return nil, err
	}
	if post.Type != models.GuildhallPostPoll {
		return nil, ErrGuildhallInvalidPoll
	}

	hasVoted, err := s.voteRepo.HasUserVoted(postUUID, userUUID)
	if err != nil {
		return nil, err
	}
	if !hasVoted {
		return nil, ErrGuildhallPollNotVoted
	}

	return s.guildhallPollResults(post, userUUID)
}

func (s *GuildhallService) guildhallPollResults(
	post *models.GuildhallPost,
	userID uuid.UUID,
) (*dto.GuildhallPollResultsResponse, error) {
	options, err := guildhallPollOptions(post.Details)
	if err != nil {
		return nil, err
	}
	voteCounts, totalVotes, err := s.voteRepo.Counts(post.ID)
	if err != nil {
		return nil, err
	}
	userVote, err := s.voteRepo.UserOption(post.ID, userID)
	if err != nil {
		return nil, err
	}

	countsByOption := make(map[string]int64, len(voteCounts))
	for _, count := range voteCounts {
		countsByOption[normalizeGuildhallPollOption(count.Option)] = count.Votes
	}

	response := &dto.GuildhallPollResultsResponse{
		PostID:     post.ID.String(),
		TotalVotes: totalVotes,
		UserVote:   userVote,
		Options:    make([]dto.GuildhallPollOptionResult, 0, len(options)),
	}
	for _, option := range options {
		count := countsByOption[normalizeGuildhallPollOption(option)]
		percentage := 0.0
		if totalVotes > 0 {
			percentage = math.Round(float64(count)*10000/float64(totalVotes)) / 100
		}
		response.Options = append(response.Options, dto.GuildhallPollOptionResult{
			Option:     option,
			Votes:      count,
			Percentage: percentage,
		})
	}
	return response, nil
}

func guildhallPollOptions(details []byte) ([]string, error) {
	var detailsMap map[string]any
	if err := json.Unmarshal(details, &detailsMap); err != nil {
		return nil, fmt.Errorf("%w: poll details are invalid", ErrGuildhallInvalidPoll)
	}
	return guildhallPollOptionsFromMap(detailsMap)
}

func guildhallPollOptionsFromMap(details map[string]any) ([]string, error) {
	rawOptions, ok := details["options"].([]any)
	if !ok || len(rawOptions) < 2 {
		return nil, fmt.Errorf("%w: a poll requires at least two options", ErrGuildhallInvalidPoll)
	}

	options := make([]string, 0, len(rawOptions))
	seen := make(map[string]struct{}, len(rawOptions))
	for _, rawOption := range rawOptions {
		option, ok := rawOption.(string)
		option = strings.TrimSpace(option)
		if !ok || option == "" {
			return nil, fmt.Errorf("%w: poll options must be non-empty strings", ErrGuildhallInvalidPoll)
		}
		key := normalizeGuildhallPollOption(option)
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("%w: poll options must be unique", ErrGuildhallInvalidPoll)
		}
		seen[key] = struct{}{}
		options = append(options, option)
	}
	return options, nil
}

func findGuildhallPollOption(options []string, selected string) (string, error) {
	selected = strings.TrimSpace(selected)
	if selected == "" {
		return "", fmt.Errorf("%w: option is required", ErrGuildhallInvalidPoll)
	}
	selectedKey := normalizeGuildhallPollOption(selected)
	for _, option := range options {
		if normalizeGuildhallPollOption(option) == selectedKey {
			return option, nil
		}
	}
	return "", fmt.Errorf("%w: selected option does not exist", ErrGuildhallInvalidPoll)
}

func normalizeGuildhallPollOption(option string) string {
	return strings.ToLower(strings.TrimSpace(option))
}

func sameGuildhallPollOptions(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftOptions := make(map[string]int, len(left))
	for _, option := range left {
		leftOptions[normalizeGuildhallPollOption(option)]++
	}
	for _, option := range right {
		key := normalizeGuildhallPollOption(option)
		if leftOptions[key] == 0 {
			return false
		}
		leftOptions[key]--
	}
	return true
}
