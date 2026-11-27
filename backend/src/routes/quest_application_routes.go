package routes

import (
	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/controllers"
	"techguild-backend/src/dto"
	"techguild-backend/src/middleware"
)

func RegisterQuestApplicationRoutes(api huma.API) {
	generalAuthMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	clientAuthMw := huma.Middlewares{
		middleware.AuthMiddlewareHuma(api),
		middleware.ClientMiddlewareHuma(api),
	}

	// Freelancer applies (individual or agency)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "apply-to-quest",
		Method:      "POST",
		Path:        "/v1/quests/{id}/apply",
		Tags:        []string{"Freelancer / Quest Board"},
		Summary:     "[Freelancer] Apply/bid to a quest",
		Middlewares: generalAuthMw,
		RequestBody: &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.ApplyQuestBody{
						CoverLetter:       "I have 5+ years in UI/UX and shipped multiple FinTech dashboards.",
						ProposedBudget:    72000,
						Currency:          "INR",
						EstimatedDuration: "18 days",
						PartyID:           "8f14e45f-ceea-467a-9a3b-1c2d3e4f5a6b",
					},
				},
			},
		},
	}, controllers.ApplyQuestHandler)

	// Client views applications; freelancer views own
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "list-quest-applications",
		Method:      "GET",
		Path:        "/v1/quests/{id}/applications",
		Tags:        []string{"Common / Quest Discovery"},
		Summary:     "[Common] List applications (client sees all, freelancer sees own)",
		Middlewares: generalAuthMw,
	}, controllers.ListApplicationsHandler)

	// Client accepts application (hires winner)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "accept-quest-application",
		Method:      "POST",
		Path:        "/v1/quests/{id}/applications/{application_id}/accept",
		Tags:        []string{"Client / Quest Board"},
		Summary:     "[Client] Accept application (client only)",
		Description: "Accepts the selected application for this quest. This operation takes no request body; it is identified by the quest and application path parameters only.",
		Middlewares: clientAuthMw,
	}, controllers.AcceptApplicationHandler)

	// Client rejects application
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "reject-quest-application",
		Method:      "POST",
		Path:        "/v1/quests/{id}/applications/{application_id}/reject",
		Tags:        []string{"Client / Quest Board"},
		Summary:     "[Client] Reject application (client only)",
		Middlewares: clientAuthMw,
		RequestBody: &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Example: dto.RejectApplicationBody{
						Reason: "Looking for more React experience",
					},
				},
			},
		},
	}, controllers.RejectApplicationHandler)

	// Freelancer withdraws
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "withdraw-quest-application",
		Method:      "POST",
		Path:        "/v1/quests/{id}/applications/{application_id}/withdraw",
		Tags:        []string{"Freelancer / Quest Board"},
		Summary:     "[Freelancer] Withdraw own pending application",
		Description: "Withdraws the caller's own pending application for this quest. This operation takes no request body; it is identified by the quest and application path parameters only.",
		Middlewares: generalAuthMw,
	}, controllers.WithdrawApplicationHandler)

	// Freelancer's task-management view
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "my-work",
		Method:      "GET",
		Path:        "/v1/quests/my-work",
		Tags:        []string{"Freelancer / Quest Board"},
		Summary:     "[Freelancer] List quests assigned to me (dashboard)",
		Middlewares: generalAuthMw,
	}, controllers.MyWorkHandler)
}
