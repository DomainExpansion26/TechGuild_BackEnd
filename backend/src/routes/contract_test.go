package routes_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"techguild-backend/src/routes"
	"techguild-backend/src/utils"
)

// registerAllRoutes mirrors main()'s registration block. Handlers are never
// invoked here, so nil controllers are sufficient to render the contract.
func registerAllRoutes(api huma.API) {
	routes.RegisterAuthRoutes(api, nil)
	routes.RegisterProfileRoutes(api, nil)
	routes.RegisterLocationRoutes(api, nil)
	routes.RegisterSettingsRoutes(api, nil)
	routes.RegisterOAuthRoutes(api, nil)
	routes.RegisterPartyRoutes(api)
	routes.RegisterVerificationRoutes(api, nil)
	routes.RegisterTwoFactorRoutes(api)
	routes.RegisterQuestApplicationRoutes(api)
	routes.RegisterQuestRoutes(api, nil)
	routes.RegisterTaskManagementRoutes(api)
	routes.RegisterGuildhallRoutes(api)
	routes.RegisterQuestMilestoneRoutes(api, nil)
	routes.RegisterMilestoneAttachmentRoutes(api)
	routes.RegisterPartyTaskManagementRoutes(api)
	routes.RegisterMilestoneFundingRoutes(api, nil)
	routes.RegisterMilestoneReleaseRoutes(api, nil)
	routes.RegisterWalletRoutes(api, nil)
	routes.RegisterWalletWithdrawalRoutes(api, nil)
	routes.RegisterRefundRoutes(api, nil)
	routes.RegisterPaymentMethodRoutes(api, nil)
	routes.RegisterPayoutMethodRoutes(api, nil)
	routes.RegisterWebhookRoutes(api, nil)
}

func buildTestAPI(t *testing.T) huma.API {
	t.Helper()
	gin.SetMode(gin.TestMode)
	api := humagin.New(gin.New(), routes.APIConfig())
	registerAllRoutes(api)
	return api
}

// TestOpenAPIContract renders the real spec + Postman collection and asserts the
// documented request-body contract for every mutating operation.
func TestOpenAPIContract(t *testing.T) {
	api := buildTestAPI(t)
	oapi := api.OpenAPI()

	if len(oapi.Paths) == 0 {
		t.Fatal("openapi spec has no paths")
	}

	// 1. Spec must not panic and must expose x-tagGroups.
	rawGroups, err := json.Marshal(oapi.Extensions["x-tagGroups"])
	if err != nil {
		t.Fatalf("x-tagGroups marshal: %v", err)
	}
	var groups []struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(rawGroups, &groups); err != nil || len(groups) == 0 {
		t.Fatalf("x-tagGroups missing or wrong type: %v (raw %s)", err, rawGroups)
	}
	declared := map[string]bool{}
	for _, g := range groups {
		if g.Name == "" || len(g.Tags) == 0 {
			t.Errorf("x-tagGroups entry has empty name or tags: %+v", g)
		}
		for _, tg := range g.Tags {
			declared[tg] = true
		}
	}
	used := map[string]bool{}
	for _, item := range oapi.Paths {
		for _, op := range opsOf(item) {
			for _, tag := range op.Tags {
				used[tag] = true
			}
		}
	}
	for name := range used {
		if !declared[name] {
			t.Errorf("tag %q is used by at least one operation but absent from x-tagGroups", name)
		}
	}
	for name := range declared {
		if !used[name] {
			t.Errorf("tag %q is declared in x-tagGroups but no operation uses it", name)
		}
	}

	// 2. Every operation has exactly one tag.
	mutating := 0
	for path, item := range oapi.Paths {
		for method, op := range map[string]*huma.Operation{
			"GET": item.Get, "POST": item.Post, "PUT": item.Put,
			"PATCH": item.Patch, "DELETE": item.Delete,
		} {
			if op == nil {
				continue
			}
			if len(op.Tags) != 1 {
				t.Errorf("%s %s (%s) has %d tags, want exactly 1: %v", method, path, op.OperationID, len(op.Tags), op.Tags)
			}
			if method == "POST" || method == "PUT" || method == "PATCH" {
				mutating++
			}
		}
	}
	if mutating < 90 {
		t.Errorf("expected at least 90 mutating operations, got %d", mutating)
	}

	// 3. The Postman collection must render, and every JSON/multipart mutating
	//    operation must carry a non-empty body.
	data, err := utils.PostmanCollection(api)
	if err != nil {
		t.Fatalf("PostmanCollection: %v", err)
	}
	var coll struct {
		Item []postmanItem `json:"item"`
	}
	if err := json.Unmarshal(data, &coll); err != nil {
		t.Fatalf("postman json: %v", err)
	}
	byName := map[string]postmanItem{}
	var walk func(items []postmanItem)
	walk = func(items []postmanItem) {
		for _, it := range items {
			if len(it.Item) > 0 {
				walk(it.Item)
				continue
			}
			byName[it.Name] = it
		}
	}
	walk(coll.Item)

	// Every POST/PUT/PATCH operation in the spec must appear in the collection.
	checked, withBody, formdata := 0, 0, 0
	for path, item := range oapi.Paths {
		for method, op := range map[string]*huma.Operation{"POST": item.Post, "PUT": item.Put, "PATCH": item.Patch} {
			if op == nil || op.Deprecated {
				continue
			}
			checked++
			it, ok := byName[op.Summary]
			if _, bodyless := bodylessOps[op.OperationID]; bodyless {
				continue
			}
			if !ok {
				t.Errorf("%s %s: operation %q missing from Postman collection", method, path, op.OperationID)
				continue
			}
			if it.Request.Body == nil {
				t.Errorf("%s %s: Postman request has no body", method, path)
				continue
			}
			if it.Request.Body.Mode == "formdata" && len(it.Request.Body.Formdata) > 0 {
				formdata++
				withBody++
				continue
			}
			if strings.TrimSpace(it.Request.Body.Raw) == "" || (strings.TrimSpace(it.Request.Body.Raw) == "{}" && op.OperationID != "publish-quest") {
				t.Errorf("%s %s (%s): empty Postman body %q", method, path, op.OperationID, it.Request.Body.Raw)
				continue
			}
			withBody++
		}
	}
	t.Logf("contract OK: %d paths, %d mutating, %d checked, %d with body (%d multipart), zero panics",
		len(oapi.Paths), mutating, checked, withBody, formdata)
}

type postmanItem struct {
	Name    string        `json:"name"`
	Item    []postmanItem `json:"item"`
	Request postmanReq    `json:"request"`
}

type postmanReq struct {
	Body *postmanBody `json:"body"`
}

type postmanBody struct {
	Mode     string              `json:"mode"`
	Raw      string              `json:"raw"`
	Formdata []map[string]string `json:"formdata"`
}

// bodylessOps are mutating operations that legitimately carry no request body
// (path/header/auth only, or an intentionally empty object for publish-quest).
var bodylessOps = map[string]struct{}{
	"save-quest":                        {},
	"accept-quest-application":          {},
	"withdraw-quest-application":        {},
	"2fa-setup":                         {},
	"like-guildhall-post":               {},
	"export-profile":                    {},
	"accept-invitation":                 {},
	"submit-verification":               {},
	"set-primary-client-payment-method": {},
	"set-primary-payout-method":         {},
	"publish-quest":                     {},
}

func opsOf(item *huma.PathItem) []*huma.Operation {
	all := []*huma.Operation{item.Get, item.Post, item.Put, item.Patch, item.Delete}
	out := make([]*huma.Operation, 0, len(all))
	for _, op := range all {
		if op != nil {
			out = append(out, op)
		}
	}
	return out
}
