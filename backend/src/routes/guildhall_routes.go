package routes

import (
	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"
)

func stringPointer(value string) *string {
	return &value
}

func RegisterGuildhallRoutes(api huma.API) {
	authMiddleware := huma.Middlewares{
		middleware.AuthMiddlewareHuma(api),
	}

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "list-guildhall-posts",
		Method:      "GET",
		Path:        "/v1/guildhall/posts",
		Tags:        []string{"Guildhall"},
		Summary:     "List Guildhall posts",
		Middlewares: authMiddleware,
	}, controllers.ListGuildhallPostsHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-guildhall-post",
		Method:      "GET",
		Path:        "/v1/guildhall/posts/{post_id}",
		Tags:        []string{"Guildhall"},
		Summary:     "Get a Guildhall post by ID",
		Middlewares: authMiddleware,
	}, controllers.GetGuildhallPostHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-guildhall-post",
		Method:      "PATCH",
		Path:        "/v1/guildhall/posts/{post_id}",
		Tags:        []string{"Guildhall"},
		Summary:     "Update a Guildhall post",
		Middlewares: authMiddleware,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateGuildhallPostBody{
						Content: stringPointer("Updated post content"),
					},
				},
			},
		},
	}, controllers.UpdateGuildhallPostHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "delete-guildhall-post",
		Method:      "DELETE",
		Path:        "/v1/guildhall/posts/{post_id}",
		Tags:        []string{"Guildhall"},
		Summary:     "Delete a Guildhall post",
		Middlewares: authMiddleware,
	}, controllers.DeleteGuildhallPostHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "like-guildhall-post",
		Method:      "PUT",
		Path:        "/v1/guildhall/posts/{post_id}/like",
		Tags:        []string{"Guildhall"},
		Summary:     "Like a Guildhall post",
		Middlewares: authMiddleware,
	}, controllers.LikeGuildhallPostHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "unlike-guildhall-post",
		Method:      "DELETE",
		Path:        "/v1/guildhall/posts/{post_id}/like",
		Tags:        []string{"Guildhall"},
		Summary:     "Remove your like from a Guildhall post",
		Middlewares: authMiddleware,
	}, controllers.UnlikeGuildhallPostHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "cast-guildhall-poll-vote",
		Method:      "POST",
		Path:        "/v1/guildhall/posts/{post_id}/votes",
		Tags:        []string{"Guildhall"},
		Summary:     "Vote in a Guildhall poll or change your vote",
		Middlewares: authMiddleware,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CastGuildhallVoteBody{Option: "PostgreSQL"},
				},
			},
		},
	}, controllers.CastGuildhallVoteHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-guildhall-poll-results",
		Method:      "GET",
		Path:        "/v1/guildhall/posts/{post_id}/results",
		Tags:        []string{"Guildhall"},
		Summary:     "Get results for a Guildhall poll after voting",
		Middlewares: authMiddleware,
	}, controllers.GetGuildhallPollResultsHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "create-guildhall-post",
		Method:      "POST",
		Path:        "/v1/guildhall/posts",
		Tags:        []string{"Guildhall"},
		Summary:     "Create a Guildhall post",
		Middlewares: authMiddleware,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateGuildhallPostBody{
						Type:    "question",
						Title:   "How do you approach API versioning?",
						Content: "I'm comparing a few approaches and would appreciate your experience.",
					},
				},
			},
		},
	}, controllers.CreateGuildhallPostHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "create-guildhall-comment",
		Method:      "POST",
		Path:        "/v1/guildhall/posts/{post_id}/comments",
		Tags:        []string{"Guildhall"},
		Summary:     "Add a comment to a Guildhall post",
		Middlewares: authMiddleware,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.CreateGuildhallCommentBody{Content: "Thanks for sharing this!"},
				},
			},
		},
	}, controllers.CreateGuildhallCommentHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "list-guildhall-comments",
		Method:      "GET",
		Path:        "/v1/guildhall/posts/{post_id}/comments",
		Tags:        []string{"Guildhall"},
		Summary:     "List comments on a Guildhall post",
		Middlewares: authMiddleware,
	}, controllers.ListGuildhallCommentsHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-guildhall-comment",
		Method:      "PATCH",
		Path:        "/v1/guildhall/comments/{comment_id}",
		Tags:        []string{"Guildhall"},
		Summary:     "Update a Guildhall comment",
		Middlewares: authMiddleware,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateGuildhallCommentBody{Content: "Updated comment text."},
				},
			},
		},
	}, controllers.UpdateGuildhallCommentHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "delete-guildhall-comment",
		Method:      "DELETE",
		Path:        "/v1/guildhall/comments/{comment_id}",
		Tags:        []string{"Guildhall"},
		Summary:     "Soft-delete a Guildhall comment",
		Middlewares: authMiddleware,
	}, controllers.DeleteGuildhallCommentHandler)
}
