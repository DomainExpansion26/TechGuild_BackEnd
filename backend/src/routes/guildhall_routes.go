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
		Tags:        []string{"Common / Guildhall"},
		Summary:     "[Common] List Guildhall posts",
		Middlewares: authMiddleware,
	}, controllers.ListGuildhallPostsHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "get-guildhall-post",
		Method:      "GET",
		Path:        "/v1/guildhall/posts/{post_id}",
		Tags:        []string{"Common / Guildhall"},
		Summary:     "[Common] Get a Guildhall post by ID",
		Middlewares: authMiddleware,
	}, controllers.GetGuildhallPostHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-guildhall-post",
		Method:      "PATCH",
		Path:        "/v1/guildhall/posts/{post_id}",
		Tags:        []string{"Common / Guildhall"},
		Summary:     "[Common] Update a Guildhall post",
		Middlewares: authMiddleware,
		RequestBody: &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.UpdateGuildhallPostBody{
						Title:   stringPointer("Updated: How do you approach API versioning?"),
						Content: stringPointer("Editing to add our contract-first workflow and the deprecation policy we now use."),
						LinkURL: stringPointer("https://techguild.in/blog/api-versioning"),
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
		Tags:        []string{"Common / Guildhall"},
		Summary:     "[Common] Delete a Guildhall post",
		Middlewares: authMiddleware,
	}, controllers.DeleteGuildhallPostHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "like-guildhall-post",
		Method:      "PUT",
		Path:        "/v1/guildhall/posts/{post_id}/like",
		Tags:        []string{"Common / Guildhall"},
		Summary:     "[Common] Like a Guildhall post",
		Description: "Toggles the like on the post and takes no request body.",
		Middlewares: authMiddleware,
	}, controllers.LikeGuildhallPostHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "unlike-guildhall-post",
		Method:      "DELETE",
		Path:        "/v1/guildhall/posts/{post_id}/like",
		Tags:        []string{"Common / Guildhall"},
		Summary:     "[Common] Remove your like from a Guildhall post",
		Middlewares: authMiddleware,
	}, controllers.UnlikeGuildhallPostHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "cast-guildhall-poll-vote",
		Method:      "POST",
		Path:        "/v1/guildhall/posts/{post_id}/votes",
		Tags:        []string{"Common / Guildhall"},
		Summary:     "[Common] Vote in a Guildhall poll or change your vote",
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
		Tags:        []string{"Common / Guildhall"},
		Summary:     "[Common] Get results for a Guildhall poll after voting",
		Middlewares: authMiddleware,
	}, controllers.GetGuildhallPollResultsHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "create-guildhall-post",
		Method:      "POST",
		Path:        "/v1/guildhall/posts",
		Tags:        []string{"Common / Guildhall"},
		Summary:     "[Common] Create a Guildhall post",
		Description: "The details object is type-specific. For opportunity posts it carries: engagement_type, location, budget_inr (rupees, major units — e.g. 250000 = ₹2,50,000), budget_period.",
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
		Tags:        []string{"Common / Guildhall"},
		Summary:     "[Common] Add a comment to a Guildhall post",
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
		Tags:        []string{"Common / Guildhall"},
		Summary:     "[Common] List comments on a Guildhall post",
		Middlewares: authMiddleware,
	}, controllers.ListGuildhallCommentsHandler)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "update-guildhall-comment",
		Method:      "PATCH",
		Path:        "/v1/guildhall/comments/{comment_id}",
		Tags:        []string{"Common / Guildhall"},
		Summary:     "[Common] Update a Guildhall comment",
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
		Tags:        []string{"Common / Guildhall"},
		Summary:     "[Common] Soft-delete a Guildhall comment",
		Middlewares: authMiddleware,
	}, controllers.DeleteGuildhallCommentHandler)
}
