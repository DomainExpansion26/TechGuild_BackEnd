package dto


// Create Project


type CreateProjectRequest struct {
	Title       string   `form:"title" binding:"required"`
	Description string   `form:"description" binding:"required"`
	Category    string   `form:"category" binding:"required"`

	BudgetType string  `form:"budget_type" binding:"required,oneof=fixed hourly"`
	MinBudget  float64 `form:"min_budget" binding:"required"`
	MaxBudget  float64 `form:"max_budget" binding:"required"`
	Currency   string  `form:"currency"`

	ExperienceLevel string   `form:"experience_level"`
	ProjectType     string   `form:"project_type"`
	Duration        string   `form:"duration"`
	RequiredSkills  []string `form:"required_skills"`

	Visibility string `form:"visibility"`

	ApplicationDeadline string `form:"application_deadline"`
	EstimatedStartDate  string `form:"estimated_start_date"`
	EstimatedEndDate    string `form:"estimated_end_date"`

	MaxApplications int `form:"max_applications"`

	IsFeatured bool `form:"is_featured"`
	IsUrgent   bool `form:"is_urgent"`
}

// Update Project


type UpdateProjectRequest struct {
	Title       string   `form:"title"`
	Description string   `form:"description"`
	Category    string   `form:"category"`

	BudgetType string  `form:"budget_type"`
	MinBudget  float64 `form:"min_budget"`
	MaxBudget  float64 `form:"max_budget"`
	Currency   string  `form:"currency"`

	ExperienceLevel string   `form:"experience_level"`
	ProjectType     string   `form:"project_type"`
	Duration        string   `form:"duration"`
	RequiredSkills  []string `form:"required_skills"`

	Visibility string `form:"visibility"`

	ApplicationDeadline string `form:"application_deadline"`
	EstimatedStartDate  string `form:"estimated_start_date"`
	EstimatedEndDate    string `form:"estimated_end_date"`

	MaxApplications int `form:"max_applications"`

	IsFeatured bool `form:"is_featured"`
	IsUrgent   bool `form:"is_urgent"`
}

// Create Project Response


type CreateProjectResponse struct {
	Message   string `json:"message"`
	ProjectID string `json:"project_id"`
}


type UpdateProjectResponse struct {
	Message string `json:"message"`
}


type DeleteProjectResponse struct {
	Message string `json:"message"`
}

// publish project response

type PublishProjectResponse struct {
	Message string `json:"message"`
}


type CloseProjectResponse struct {
	Message string `json:"message"`
}

//reopen project response

type ReopenProjectResponse struct {
	Message string `json:"message"`
}

//project response

type ProjectResponse struct {
	ID          string `json:"id"`
	ClientID    string `json:"client_id"`

	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`

	BudgetType string  `json:"budget_type"`
	MinBudget  float64 `json:"min_budget"`
	MaxBudget  float64 `json:"max_budget"`
	Currency   string  `json:"currency"`

	ExperienceLevel string   `json:"experience_level"`
	ProjectType     string   `json:"project_type"`
	Duration        string   `json:"duration"`
	RequiredSkills  []string `json:"required_skills"`

	Visibility string `json:"visibility"`
	Status     string `json:"status"`

	ApplicationDeadline string `json:"application_deadline,omitempty"`
	EstimatedStartDate  string `json:"estimated_start_date,omitempty"`
	EstimatedEndDate    string `json:"estimated_end_date,omitempty"`
	PublishedAt         string `json:"published_at,omitempty"`

	MaxApplications int `json:"max_applications"`

	IsFeatured bool `json:"is_featured"`
	IsUrgent   bool `json:"is_urgent"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

//project list response

type ProjectListResponse struct {
	Projects []ProjectResponse `json:"projects"`
	Total    int               `json:"total"`
}

//search project request

type SearchProjectRequest struct {
	Keyword string `form:"keyword"`

	Category string `form:"category"`

	MinBudget float64 `form:"min_budget"`
	MaxBudget float64 `form:"max_budget"`

	ExperienceLevel string `form:"experience_level"`

	ProjectType string `form:"project_type"`

	Visibility string `form:"visibility"`

	Page  int `form:"page"`
	Limit int `form:"limit"`
}