package routes

import (
	"github.com/danielgtaylor/huma/v2"

	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"
)

func RegisterQuestApplicationRoutes(api huma.API) {
	generalAuthMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}
	clientAuthMw := huma.Middlewares{
		middleware.AuthMiddlewareHuma(api),
		middleware.ClientMiddlewareHuma(api),
	}
	tags := []string{"Quest Applications"}

	// Freelancer applies (individual or agency)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "apply-to-quest",
		Method:      "POST",
		Path:        "/v1/quests/{id}/apply",
		Tags:        tags,
		Summary:     "Apply/bid to a quest (freelancer or party lead)",
		Middlewares: generalAuthMw,
	}, controllers.ApplyQuestHandler)

	// Client views applications; freelancer views own
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "list-quest-applications",
		Method:      "GET",
		Path:        "/v1/quests/{id}/applications",
		Tags:        tags,
		Summary:     "List applications (client sees all, freelancer sees own)",
		Middlewares: generalAuthMw,
	}, controllers.ListApplicationsHandler)

	// Client accepts application (hires winner)
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "accept-quest-application",
		Method:      "POST",
		Path:        "/v1/quests/{id}/applications/{application_id}/accept",
		Tags:        tags,
		Summary:     "Accept application (client only) — assigns freelancer, quest→in_progress, auto-rejects others",
		Middlewares: clientAuthMw,
	}, controllers.AcceptApplicationHandler)

	// Client rejects application
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "reject-quest-application",
		Method:      "POST",
		Path:        "/v1/quests/{id}/applications/{application_id}/reject",
		Tags:        tags,
		Summary:     "Reject application (client only)",
		Middlewares: clientAuthMw,
	}, controllers.RejectApplicationHandler)

	// Freelancer withdraws
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "withdraw-quest-application",
		Method:      "POST",
		Path:        "/v1/quests/{id}/applications/{application_id}/withdraw",
		Tags:        tags,
		Summary:     "Withdraw own pending application (freelancer)",
		Middlewares: generalAuthMw,
	}, controllers.WithdrawApplicationHandler)

	// Freelancer's task-management view
	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"BearerAuth": {}}},
		OperationID: "my-work",
		Method:      "GET",
		Path:        "/v1/quests/my-work",
		Tags:        tags,
		Summary:     "List quests assigned to me (freelancer dashboard)",
		Middlewares: generalAuthMw,
	}, controllers.MyWorkHandler)
}
