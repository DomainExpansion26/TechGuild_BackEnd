package utils

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// PostmanCollectionSchema is the Postman Collection v2.1.0 schema URI.
const PostmanCollectionSchema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

// defaultBaseURL is used when the OpenAPI document declares no servers.
const defaultBaseURL = "http://localhost:8080"

// PostmanCollection converts the runtime OpenAPI document exposed by api into
// a Postman Collection v2.1.0 JSON document.
//
// Operations are grouped into 2-level nested folders derived from each
// operation's "<Parent> / <Subfolder>" tag (e.g. "Client / Quest Board"):
//   - Level 1: <Parent> (Client, Freelancer, Agency, Common, Admin), ordered
//     exactly as declared in the OpenAPI document's `x-tagGroups` extension.
//   - Level 2: <Subfolder> (Quest Board, Milestones & Escrow, ...).
//   - Level 3: request items named after operation.Summary
//     (falling back to operation.OperationID).
//
// Endpoints whose tag does not follow the "<Parent> / <Subfolder>" pattern,
// or whose parent is not declared in `x-tagGroups`, are placed under a level-1
// folder named after the parent in discovery order.
func PostmanCollection(api huma.API) ([]byte, error) {
	oapi := api.OpenAPI()
	if oapi == nil {
		return nil, fmt.Errorf("postman export: openapi document is nil")
	}

	tree := &folderTree{
		l1:            []*l1Folder{},
		l1ByName:      map[string]*l1Folder{},
		l1Order:       []string{},
		unknownL1:     []*l1Folder{},
		unknownByName: map[string]*l1Folder{},
	}

	// Level-1 folders and their order come from x-tagGroups. The extension is
	// whatever the API config declared (typically []map[string]any when it is set
	// in Go, []any once it has been through JSON), so normalise before iterating.
	if groups := tagGroups(oapi.Extensions["x-tagGroups"]); len(groups) > 0 {
		for _, g := range groups {
			m, ok := g.(map[string]any)
			if !ok {
				continue
			}
			name, _ := m["name"].(string)
			if name == "" {
				continue
			}
			if _, exists := tree.l1ByName[name]; exists {
				continue
			}
			l1 := &l1Folder{name: name, subByName: map[string]*l2Folder{}}
			tree.l1 = append(tree.l1, l1)
			tree.l1ByName[name] = l1
			tree.l1Order = append(tree.l1Order, name)
		}
	}

	// Sort paths for deterministic output.
	paths := make([]string, 0, len(oapi.Paths))
	for path := range oapi.Paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		pathItem := oapi.Paths[path]
		if pathItem == nil {
			continue
		}
		if err := tree.addPath(path, pathItem); err != nil {
			return nil, err
		}
	}

	collection := postmanCollection{
		Info: postmanInfo{
			Name:        infoTitle(oapi),
			Schema:      PostmanCollectionSchema,
			Description: infoDescription(oapi),
		},
		Variable: []postmanVariable{
			{Key: "baseUrl", Value: baseURL(oapi), Description: "Base URL of the API"},
			{Key: "token", Value: "", Description: "Bearer token used with the Authorization header"},
		},
		Item: tree.items(),
	}

	return json.MarshalIndent(collection, "", "  ")
}

func infoTitle(oapi *huma.OpenAPI) string {
	if oapi.Info != nil && oapi.Info.Title != "" {
		return oapi.Info.Title
	}
	return "TechGuild Backend API"
}

func infoDescription(oapi *huma.OpenAPI) string {
	if oapi.Info != nil {
		return oapi.Info.Description
	}
	return ""
}

func baseURL(oapi *huma.OpenAPI) string {
	if len(oapi.Servers) > 0 && oapi.Servers[0] != nil && oapi.Servers[0].URL != "" {
		return oapi.Servers[0].URL
	}
	return defaultBaseURL
}

// -------- folder tree --------

type l2Folder struct {
	name     string
	requests []postmanItem
}

type l1Folder struct {
	name       string
	subfolders []*l2Folder
	subByName  map[string]*l2Folder
	direct     []postmanItem // requests with no level-2 subfolder
}

type folderTree struct {
	l1            []*l1Folder
	l1ByName      map[string]*l1Folder
	l1Order       []string
	unknownL1     []*l1Folder
	unknownByName map[string]*l1Folder
}

func (t *folderTree) l1For(parent string) *l1Folder {
	if l1, ok := t.l1ByName[parent]; ok && l1 != nil {
		return l1
	}
	if l1, ok := t.unknownByName[parent]; ok && l1 != nil {
		return l1
	}
	l1 := &l1Folder{name: parent, subByName: map[string]*l2Folder{}}
	t.unknownL1 = append(t.unknownL1, l1)
	t.unknownByName[parent] = l1
	return l1
}

func (t *folderTree) addPath(path string, item *huma.PathItem) error {
	// Method order is fixed so output is deterministic regardless of map order.
	ops := []*huma.Operation{}
	if item.Get != nil {
		ops = append(ops, item.Get)
	}
	if item.Post != nil {
		ops = append(ops, item.Post)
	}
	if item.Put != nil {
		ops = append(ops, item.Put)
	}
	if item.Patch != nil {
		ops = append(ops, item.Patch)
	}
	if item.Delete != nil {
		ops = append(ops, item.Delete)
	}
	if item.Options != nil {
		ops = append(ops, item.Options)
	}
	if item.Head != nil {
		ops = append(ops, item.Head)
	}
	if item.Trace != nil {
		ops = append(ops, item.Trace)
	}

	// Path-level parameters apply to every operation on the path.
	pathParams := paramMap(item.Parameters)

	for _, op := range ops {
		if op == nil || op.Hidden || op.Method == "" {
			continue
		}
		parent, subfolder := splitTag(op.Tags)
		l1 := t.l1For(parent)

		request, err := requestItem(path, item, op, pathParams)
		if err != nil {
			return err
		}

		if subfolder == "" {
			l1.direct = append(l1.direct, request)
		} else {
			l2, ok := l1.subByName[subfolder]
			if !ok {
				l2 = &l2Folder{name: subfolder}
				l1.subfolders = append(l1.subfolders, l2)
				l1.subByName[subfolder] = l2
			}
			l2.requests = append(l2.requests, request)
		}
	}
	return nil
}

func paramMap(params []*huma.Param) map[string]*huma.Param {
	out := map[string]*huma.Param{}
	for _, p := range params {
		if p == nil || p.Name == "" {
			continue
		}
		out[p.Name] = p
	}
	return out
}

// splitTag splits the first "<Parent> / <Subfolder>" tag. Operations without
// such a tag return (tag, "").
func splitTag(tags []string) (string, string) {
	if len(tags) == 0 || tags[0] == "" {
		return "", ""
	}
	parts := strings.SplitN(tags[0], " / ", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return tags[0], ""
}

func (t *folderTree) items() []postmanItem {
	out := []postmanItem{}
	for _, name := range t.l1Order {
		l1, ok := t.l1ByName[name]
		if !ok || l1 == nil {
			// Unknown parent tag: the operation was already routed through
			// l1For's fallback folder, so skip rather than panic on nil.
			continue
		}
		out = append(out, l1.item())
	}
	for _, l1 := range t.unknownL1 {
		out = append(out, l1.item())
	}
	return out
}

func (l1 *l1Folder) item() postmanItem {
	children := []postmanItem{}
	for _, l2 := range l1.subfolders {
		children = append(children, l2.item())
	}
	for _, r := range l1.direct {
		children = append(children, r)
	}
	return postmanItem{Name: l1.name, Item: children}
}

func (l2 *l2Folder) item() postmanItem {
	reqs := []postmanItem{}
	for _, r := range l2.requests {
		reqs = append(reqs, r)
	}
	return postmanItem{Name: l2.name, Item: reqs}
}

// -------- request item --------

func requestItem(path string, item *huma.PathItem, op *huma.Operation, pathParams map[string]*huma.Param) (postmanItem, error) {
	name := op.Summary
	if name == "" {
		name = op.OperationID
	}

	request := postmanRequest{
		Method: op.Method,
		URL:    urlItem(path, op, pathParams),
	}
	if op.Description != "" {
		request.Description = op.Description
	}

	headers := []postmanHeader{}

	// Request body. Multipart uploads become a Postman `formdata` body; JSON
	// ops get a raw example, preferring the singular `example` and falling back
	// to the first named `examples` entry.
	if op.RequestBody != nil {
		if mediaType, ok := op.RequestBody.Content["multipart/form-data"]; ok && mediaType != nil {
			request.Body = multipartBody(mediaType)
		} else if mediaType, ok := op.RequestBody.Content["application/json"]; ok && mediaType != nil {
			example := mediaType.Example
			if example == nil {
				example = firstExample(mediaType.Examples)
			}
			if example != nil {
				raw, err := json.MarshalIndent(example, "", "  ")
				if err != nil {
					return postmanItem{}, fmt.Errorf("postman export: marshal request body example for %s: %w", op.OperationID, err)
				}
				request.Body = &postmanBody{
					Mode:    "raw",
					Raw:     string(raw),
					Options: &postmanRawOptions{Raw: postmanRawLanguage{Language: "json"}},
				}
				headers = append(headers, postmanHeader{
					Key:         "Content-Type",
					Value:       "application/json",
					Description: "JSON request body",
				})
			}
		}
	}

	// Authorization header for secured operations.
	if secured(op) {
		headers = append(headers, postmanHeader{
			Key:         "Authorization",
			Value:       "Bearer {{token}}",
			Description: "Bearer token",
		})
	}
	if len(headers) > 0 {
		request.Header = headers
	}

	return postmanItem{Name: name, Request: &request}, nil
}

// firstExample returns the value of the alphabetically-first named example so
// the export stays deterministic. It backs the singular `example` field: when
// that is absent, named OpenAPI `examples` still reach Postman.
func firstExample(examples map[string]*huma.Example) any {
	if len(examples) == 0 {
		return nil
	}
	names := make([]string, 0, len(examples))
	for name := range examples {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if ex := examples[name]; ex != nil && ex.Value != nil {
			return ex.Value
		}
	}
	return nil
}

// multipartBody converts a `multipart/form-data` media type into a Postman
// `formdata` body. File-typed schema fields (format "binary", including arrays
// of binaries) become `type:"file"`; every other field becomes `type:"text"`
// with a sample value where the schema provides one. No Content-Type header is
// added: Postman generates the multipart boundary itself.
func multipartBody(mediaType *huma.MediaType) *postmanBody {
	body := &postmanBody{Mode: "formdata"}
	if mediaType.Schema == nil {
		return body
	}

	names := make([]string, 0, len(mediaType.Schema.Properties))
	for name := range mediaType.Schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)

	required := make(map[string]bool, len(mediaType.Schema.Required))
	for _, name := range mediaType.Schema.Required {
		required[name] = true
	}

	for _, name := range names {
		prop := mediaType.Schema.Properties[name]
		if prop == nil {
			continue
		}
		field := postmanFormParam{
			Key:         name,
			Description: prop.Description,
		}
		if required[name] {
			field.Description = strings.TrimSpace("Required. " + field.Description)
		}
		if isFileSchema(prop) {
			empty := ""
			field.Type = "file"
			field.Src = &empty
		} else {
			value := sampleValue(prop)
			field.Type = "text"
			field.Value = &value
		}
		body.Formdata = append(body.Formdata, field)
	}
	return body
}

// isFileSchema reports whether a multipart field is an uploaded file.
func isFileSchema(s *huma.Schema) bool {
	if s == nil {
		return false
	}
	if s.Format == "binary" {
		return true
	}
	return s.Type == "array" && s.Items != nil && s.Items.Format == "binary"
}

// sampleValue picks a cheap placeholder for a text form field.
func sampleValue(s *huma.Schema) string {
	if s == nil {
		return ""
	}
	if len(s.Examples) > 0 && s.Examples[0] != nil {
		return fmt.Sprintf("%v", s.Examples[0])
	}
	if s.Default != nil {
		return fmt.Sprintf("%v", s.Default)
	}
	if len(s.Enum) > 0 && s.Enum[0] != nil {
		return fmt.Sprintf("%v", s.Enum[0])
	}
	return ""
}

func secured(op *huma.Operation) bool {
	return len(op.Security) > 0
}

func urlItem(path string, op *huma.Operation, pathParams map[string]*huma.Param) *postmanURL {
	rawSegments := []string{}
	pathParamNames := []string{}
	for _, seg := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if seg == "" {
			continue
		}
		if len(seg) >= 2 && seg[0] == '{' && seg[len(seg)-1] == '}' {
			name := seg[1 : len(seg)-1]
			rawSegments = append(rawSegments, ":"+name)
			pathParamNames = append(pathParamNames, name)
		} else {
			rawSegments = append(rawSegments, seg)
		}
	}

	url := &postmanURL{
		Raw:  "{{baseUrl}}/" + strings.Join(rawSegments, "/"),
		Host: []string{"{{baseUrl}}"},
		Path: rawSegments,
	}

	// Path parameters become Postman url variables (sorted for determinism).
	sort.Strings(pathParamNames)
	seenVar := map[string]bool{}
	for _, name := range pathParamNames {
		if seenVar[name] {
			continue
		}
		seenVar[name] = true
		url.Variable = append(url.Variable, postmanVariable{
			Key:         name,
			Value:       "",
			Description: paramDescription(pathParams[name]),
		})
	}
	if len(url.Variable) == 0 {
		url.Variable = nil
	}

	// Query parameters, operation-level first (declared order), then path-level.
	url.Query = queryParams(op.Parameters, pathParams)
	if len(url.Query) == 0 {
		url.Query = nil
	}

	return url
}

func queryParams(opParams []*huma.Param, pathParams map[string]*huma.Param) []postmanQuery {
	seen := map[string]bool{}
	out := []postmanQuery{}
	add := func(p *huma.Param) {
		if p == nil || p.In != "query" || seen[p.Name] {
			return
		}
		seen[p.Name] = true
		out = append(out, postmanQuery{
			Key:         p.Name,
			Value:       "",
			Description: p.Description,
		})
	}
	for _, p := range opParams {
		add(p)
	}
	// Path-level params last, sorted by name for determinism.
	names := []string{}
	for name := range pathParams {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		add(pathParams[name])
	}
	return out
}

func paramDescription(p *huma.Param) string {
	if p == nil {
		return ""
	}
	return p.Description
}

// -------- Postman collection types --------

type postmanCollection struct {
	Info     postmanInfo       `json:"info"`
	Variable []postmanVariable `json:"variable,omitempty"`
	Item     []postmanItem     `json:"item"`
}

type postmanInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"`
}

type postmanVariable struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

type postmanItem struct {
	Name    string          `json:"name"`
	Item    []postmanItem   `json:"item,omitempty"`
	Request *postmanRequest `json:"request,omitempty"`
}

type postmanRequest struct {
	Method      string          `json:"method"`
	Header      []postmanHeader `json:"header,omitempty"`
	Body        *postmanBody    `json:"body,omitempty"`
	URL         *postmanURL     `json:"url"`
	Description string          `json:"description,omitempty"`
}

type postmanHeader struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

type postmanBody struct {
	Mode     string             `json:"mode"`
	Raw      string             `json:"raw,omitempty"`
	Options  *postmanRawOptions `json:"options,omitempty"`
	Formdata []postmanFormParam `json:"formdata,omitempty"`
}

// postmanFormParam is one field of a Postman `formdata` body. File uploads use
// Type "file" with Src; text fields use Type "text" with Value.
type postmanFormParam struct {
	Key         string  `json:"key"`
	Value       *string `json:"value,omitempty"`
	Type        string  `json:"type"`
	Src         *string `json:"src,omitempty"`
	Description string  `json:"description,omitempty"`
}

type postmanRawOptions struct {
	Raw postmanRawLanguage `json:"raw"`
}

type postmanRawLanguage struct {
	Language string `json:"language"`
}

type postmanURL struct {
	Raw      string            `json:"raw"`
	Host     []string          `json:"host"`
	Path     []string          `json:"path"`
	Query    []postmanQuery    `json:"query,omitempty"`
	Variable []postmanVariable `json:"variable,omitempty"`
}

type postmanQuery struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

// tagGroups normalises the x-tagGroups extension to []any regardless of whether
// it was declared in Go as []map[string]any or decoded from JSON as []any.
func tagGroups(ext any) []any {
	switch v := ext.(type) {
	case []any:
		return v
	case []map[string]any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = v[i]
		}
		return out
	case nil:
		return nil
	}
	data, err := json.Marshal(ext)
	if err != nil {
		return nil
	}
	var out []any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}
