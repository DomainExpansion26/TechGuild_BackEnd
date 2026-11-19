package dto

import "time"

type CreateGuildhallPostBody struct {
	Type    string         `json:"type" huma:"required,enum=discussion;question;project_showcase;poll;opportunity;media" example:"question"`
	Title   string         `json:"title,omitempty" huma:"maxLength=200" example:"How do you approach API versioning?"`
	Content string         `json:"content" huma:"required,minLength=1" example:"I'm comparing a few approaches and would appreciate your experience."`
	LinkURL string         `json:"link_url,omitempty" huma:"maxLength=2048" example:"https://example.com/project"`
	Details map[string]any `json:"details,omitempty"`
}

type CreateGuildhallPostInput struct {
	Body CreateGuildhallPostBody
}

type GuildhallPostResponse struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Title     string         `json:"title,omitempty"`
	Content   string         `json:"content"`
	LinkURL   string         `json:"link_url,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
	AuthorID  string         `json:"author_id"`
	LikeCount int64          `json:"like_count"`
	LikedByMe bool           `json:"liked_by_me"`
	CreatedAt time.Time      `json:"created_at"`
}

type CreateGuildhallPostOutput struct {
	Status int `json:"-" huma:"status:201"`
	Body   GuildhallPostResponse
}

type GetGuildhallPostInput struct {
	PostID string `path:"post_id"`
}

type GetGuildhallPostOutput struct {
	Body GuildhallPostResponse
}

type ListGuildhallPostsInput struct {
	Type  string `query:"type"`
	Page  int    `query:"page" default:"1"`
	Limit int    `query:"limit" default:"20"`
}

type ListGuildhallPostsResponse struct {
	Posts []GuildhallPostResponse `json:"posts"`
	Total int64                   `json:"total"`
	Page  int                     `json:"page"`
	Limit int                     `json:"limit"`
}

type ListGuildhallPostsOutput struct {
	Body ListGuildhallPostsResponse
}

type UpdateGuildhallPostBody struct {
	Title   *string         `json:"title,omitempty" huma:"maxLength=200"`
	Content *string         `json:"content,omitempty" huma:"minLength=1"`
	LinkURL *string         `json:"link_url,omitempty" huma:"maxLength=2048"`
	Details *map[string]any `json:"details,omitempty"`
}

type UpdateGuildhallPostInput struct {
	PostID string `path:"post_id"`
	Body   UpdateGuildhallPostBody
}

type UpdateGuildhallPostOutput struct {
	Body GuildhallPostResponse
}

type DeleteGuildhallPostInput struct {
	PostID string `path:"post_id"`
}

type DeleteGuildhallPostResponse struct {
	Message string `json:"message"`
}

type DeleteGuildhallPostOutput struct {
	Body DeleteGuildhallPostResponse
}

type CreateGuildhallCommentBody struct {
	Content string `json:"content" huma:"required,minLength=1,maxLength=5000" example:"Thanks for sharing this!"`
}

type CreateGuildhallCommentInput struct {
	PostID string `path:"post_id"`
	Body   CreateGuildhallCommentBody
}

type GuildhallCommentResponse struct {
	ID        string    `json:"id"`
	PostID    string    `json:"post_id"`
	AuthorID  string    `json:"author_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateGuildhallCommentOutput struct {
	Status int `json:"-" huma:"status:201"`
	Body   GuildhallCommentResponse
}

type ListGuildhallCommentsInput struct {
	PostID string `path:"post_id"`
	Page   int    `query:"page" default:"1"`
	Limit  int    `query:"limit" default:"20"`
}

type ListGuildhallCommentsResponse struct {
	Comments []GuildhallCommentResponse `json:"comments"`
	Total    int64                      `json:"total"`
	Page     int                        `json:"page"`
	Limit    int                        `json:"limit"`
}

type ListGuildhallCommentsOutput struct {
	Body ListGuildhallCommentsResponse
}

type UpdateGuildhallCommentBody struct {
	Content string `json:"content" huma:"required,minLength=1,maxLength=5000" example:"Updated comment text."`
}

type UpdateGuildhallCommentInput struct {
	CommentID string `path:"comment_id"`
	Body      UpdateGuildhallCommentBody
}

type UpdateGuildhallCommentOutput struct {
	Body GuildhallCommentResponse
}

type DeleteGuildhallCommentInput struct {
	CommentID string `path:"comment_id"`
}

type DeleteGuildhallCommentResponse struct {
	Message string `json:"message"`
}

type DeleteGuildhallCommentOutput struct {
	Body DeleteGuildhallCommentResponse
}

type GuildhallLikeResponse struct {
	PostID    string `json:"post_id"`
	Liked     bool   `json:"liked"`
	LikeCount int64  `json:"like_count"`
}

type LikeGuildhallPostInput struct {
	PostID string `path:"post_id"`
}

type LikeGuildhallPostOutput struct {
	Body GuildhallLikeResponse
}

type UnlikeGuildhallPostInput struct {
	PostID string `path:"post_id"`
}

type UnlikeGuildhallPostOutput struct {
	Body GuildhallLikeResponse
}

type CastGuildhallVoteBody struct {
	Option string `json:"option" huma:"required,minLength=1" example:"PostgreSQL"`
}

type CastGuildhallVoteInput struct {
	PostID string `path:"post_id"`
	Body   CastGuildhallVoteBody
}

type GuildhallPollOptionResult struct {
	Option     string  `json:"option"`
	Votes      int64   `json:"votes"`
	Percentage float64 `json:"percentage"`
}

type GuildhallPollResultsResponse struct {
	PostID     string                      `json:"post_id"`
	TotalVotes int64                       `json:"total_votes"`
	UserVote   string                      `json:"user_vote"`
	Options    []GuildhallPollOptionResult `json:"options"`
}

type CastGuildhallVoteOutput struct {
	Body GuildhallPollResultsResponse
}

type GetGuildhallPollResultsInput struct {
	PostID string `path:"post_id"`
}

type GetGuildhallPollResultsOutput struct {
	Body GuildhallPollResultsResponse
}
