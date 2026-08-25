package dto

// ---------- Requests ----------

type CreateProjectRequest struct {
	Title       string `json:"title" huma:"required" example:"Build a Web Application"`
	Description string `json:"description" huma:"required" example:"Need a full-stack web application with user authentication and dashboard"`
	Category    string `json:"category" huma:"required" example:"web-development"`

	BudgetType string  `json:"budget_type" huma:"required,enum=fixed;hourly" example:"fixed"`
	MinBudget  float64 `json:"min_budget" huma:"required" example:"1000.00"`
	MaxBudget  float64 `json:"max_budget" huma:"required" example:"5000.00"`
	Currency   string  `json:"currency" huma:"" example:"USD"`

	ExperienceLevel string   `json:"experience_level" huma:"" example:"intermediate"`
	ProjectType     string   `json:"project_type" huma:"" example:"remote"`
	Duration        string   `json:"duration" huma:"" example:"2 months"`
	RequiredSkills  []string `json:"required_skills" huma:""`

	Visibility string `json:"visibility" huma:"" example:"public"`

	ApplicationDeadline string `json:"application_deadline" huma:"" example:"2026-03-01T00:00:00Z"`
	EstimatedStartDate  string `json:"estimated_start_date" huma:"" example:"2026-03-15T00:00:00Z"`
	EstimatedEndDate    string `json:"estimated_end_date" huma:"" example:"2026-05-15T00:00:00Z"`

	MaxApplications int `json:"max_applications" huma:"" example:"10"`

	IsFeatured bool `json:"is_featured" huma:"" example:"false"`
	IsUrgent   bool `json:"is_urgent" huma:"" example:"false"`
}

type UpdateProjectRequest struct {
	Title       string `json:"title" huma:"" example:"Build a Web Application v2"`
	Description string `json:"description" huma:"" example:"Updated requirements for the web application"`
	Category    string `json:"category" huma:"" example:"web-development"`

	BudgetType string  `json:"budget_type" huma:"" example:"fixed"`
	MinBudget  float64 `json:"min_budget" huma:"" example:"2000.00"`
	MaxBudget  float64 `json:"max_budget" huma:"" example:"8000.00"`
	Currency   string  `json:"currency" huma:"" example:"USD"`

	ExperienceLevel string   `json:"experience_level" huma:"" example:"senior"`
	ProjectType     string   `json:"project_type" huma:"" example:"remote"`
	Duration        string   `json:"duration" huma:"" example:"3 months"`
	RequiredSkills  []string `json:"required_skills" huma:""`

	Visibility string `json:"visibility" huma:"" example:"public"`

	ApplicationDeadline string `json:"application_deadline" huma:"" example:"2026-03-01T00:00:00Z"`
	EstimatedStartDate  string `json:"estimated_start_date" huma:"" example:"2026-03-15T00:00:00Z"`
	EstimatedEndDate    string `json:"estimated_end_date" huma:"" example:"2026-05-15T00:00:00Z"`

	MaxApplications int `json:"max_applications" huma:"" example:"15"`

	IsFeatured bool `json:"is_featured" huma:"" example:"true"`
	IsUrgent   bool `json:"is_urgent" huma:"" example:"false"`
}

type SearchProjectRequest struct {
	Keyword         string  `query:"keyword" example:"web development"`
	Category        string  `query:"category" example:"web-development"`
	MinBudget       float64 `query:"min_budget" example:"1000.00"`
	MaxBudget       float64 `query:"max_budget" example:"5000.00"`
	ExperienceLevel string  `query:"experience_level" example:"intermediate"`
	ProjectType     string  `query:"project_type" example:"remote"`
	Visibility      string  `query:"visibility" example:"public"`
	Page            int     `query:"page" example:"1"`
	Limit           int     `query:"limit" example:"20"`
}

// ---------- Responses ----------

type CreateProjectResponse struct {
	Message   string `json:"message" example:"Project created successfully"`
	ProjectID string `json:"project_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type UpdateProjectResponse struct {
	Message string `json:"message" example:"Project updated successfully"`
}

type DeleteProjectResponse struct {
	Message string `json:"message" example:"Project deleted successfully"`
}

type PublishProjectResponse struct {
	Message string `json:"message" example:"Project published successfully"`
}

type CloseProjectResponse struct {
	Message string `json:"message" example:"Project closed successfully"`
}

type CloseProjectRequest struct {
	Reason string `json:"reason" example:"Project requirements changed"`
}

type ReopenProjectRequest struct {
	Reason string `json:"reason" example:"Client needs additional changes"`
}

type ReopenProjectResponse struct {
	Message string `json:"message" example:"Project reopened successfully"`
}

type ProjectResponse struct {
	ID       string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ClientID string `json:"client_id" example:"550e8400-e29b-41d4-a716-446655440000"`

	Title       string `json:"title" example:"Build a Web Application"`
	Description string `json:"description" example:"Need a full-stack web application with user authentication and dashboard"`
	Category    string `json:"category" example:"web-development"`

	BudgetType string  `json:"budget_type" example:"fixed"`
	MinBudget  float64 `json:"min_budget" example:"1000.00"`
	MaxBudget  float64 `json:"max_budget" example:"5000.00"`
	Currency   string  `json:"currency" example:"USD"`

	ExperienceLevel string   `json:"experience_level" example:"intermediate"`
	ProjectType     string   `json:"project_type" example:"remote"`
	Duration        string   `json:"duration" example:"2 months"`
	RequiredSkills  []string `json:"required_skills"`

	Visibility string `json:"visibility" example:"public"`
	Status     string `json:"status" example:"open"`

	ApplicationDeadline string `json:"application_deadline,omitempty" example:"2026-03-01T00:00:00Z"`
	EstimatedStartDate  string `json:"estimated_start_date,omitempty" example:"2026-03-15T00:00:00Z"`
	EstimatedEndDate    string `json:"estimated_end_date,omitempty" example:"2026-05-15T00:00:00Z"`
	PublishedAt         string `json:"published_at,omitempty" example:"2026-01-20T00:00:00Z"`

	MaxApplications int `json:"max_applications" example:"10"`

	IsFeatured bool `json:"is_featured" example:"false"`
	IsUrgent   bool `json:"is_urgent" example:"false"`

	CreatedAt string `json:"created_at" example:"2026-01-15T00:00:00Z"`
	UpdatedAt string `json:"updated_at" example:"2026-01-15T00:00:00Z"`
}

type ProjectListResponse struct {
	Projects []ProjectResponse `json:"projects"`
	Total    int               `json:"total" example:"1"`
}

// ---------- Huma Input/Output wrapper structs ----------

type CreateProjectInput struct {
	Body CreateProjectRequest
}
type CreateProjectOutput struct {
	Body CreateProjectResponse
}

type UpdateProjectInput struct {
	ID   string `path:"project_id" doc:"Project ID"`
	Body UpdateProjectRequest
}
type UpdateProjectOutput struct {
	Body UpdateProjectResponse
}

type DeleteProjectInput struct {
	ID string `path:"project_id" doc:"Project ID"`
}
type DeleteProjectOutput struct {
	Body DeleteProjectResponse
}

type PublishProjectInput struct {
	ID string `path:"project_id" doc:"Project ID"`
}
type PublishProjectOutput struct {
	Body PublishProjectResponse
}

type CloseProjectInput struct {
	ID   string `path:"project_id" doc:"Project ID"`
	Body CloseProjectRequest
}
type CloseProjectOutput struct {
	Body CloseProjectResponse
}

type ReopenProjectInput struct {
	ID   string `path:"project_id" doc:"Project ID"`
	Body ReopenProjectRequest
}
type ReopenProjectOutput struct {
	Body ReopenProjectResponse
}

type GetProjectByIDInput struct {
	ID string `path:"project_id" doc:"Project ID"`
}
type GetProjectByIDOutput struct {
	Body ProjectResponse
}

type GetMyProjectsInput struct{}
type GetMyProjectsOutput struct {
	Body ProjectListResponse
}

type BrowseProjectsInput struct{}
type BrowseProjectsOutput struct {
	Body ProjectListResponse
}

type SearchProjectsInput struct {
	Keyword         string  `query:"keyword" example:"web development"`
	Category        string  `query:"category" example:"web-development"`
	MinBudget       float64 `query:"min_budget" example:"1000.00"`
	MaxBudget       float64 `query:"max_budget" example:"5000.00"`
	ExperienceLevel string  `query:"experience_level" example:"intermediate"`
	ProjectType     string  `query:"project_type" example:"remote"`
	Visibility      string  `query:"visibility" example:"public"`
	Page            int     `query:"page" example:"1"`
	Limit           int     `query:"limit" example:"20"`
}
type SearchProjectsOutput struct {
	Body ProjectListResponse
}
