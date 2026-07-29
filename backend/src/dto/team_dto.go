package dto

type CreateTeamRequest struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug" binding:"required"`
	Description string `json:"description"`

	LogoURL   string `json:"logo_url"`
	BannerURL string `json:"banner_url"`

	IsHiring bool `json:"is_hiring"`
}

type UpdateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	LogoURL   string `json:"logo_url"`
	BannerURL string `json:"banner_url"`

	IsHiring bool `json:"is_hiring"`
}

type InviteMemberRequest struct {
	UserID  string `json:"user_id" binding:"required"`
	Message string `json:"message"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

type CreatePortfolioRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`

	ImageURL   string `json:"image_url"`
	ProjectURL string `json:"project_url"`
	GithubURL  string `json:"github_url"`
}

type UpdatePortfolioRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`

	ImageURL   string `json:"image_url"`
	ProjectURL string `json:"project_url"`
	GithubURL  string `json:"github_url"`
}

type AddSkillRequest struct {
	SkillName       string `json:"skill_name" binding:"required"`
	ExperienceLevel string `json:"experience_level"`
}

type TeamResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`

	LogoURL   string `json:"logo_url"`
	BannerURL string `json:"banner_url"`

	LeaderID string `json:"leader_id"`

	IsHiring   bool `json:"is_hiring"`
	IsVerified bool `json:"is_verified"`

	Status string `json:"status"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type TeamMemberResponse struct {
	ID string `json:"id"`

	TeamID string `json:"team_id"`
	UserID string `json:"user_id"`

	Role   string `json:"role"`
	Status string `json:"status"`

	JoinedAt string `json:"joined_at"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type TeamInvitationResponse struct {
	ID string `json:"id"`

	TeamID string `json:"team_id"`

	InvitedByID string `json:"invited_by_id"`

	InvitedUserID string `json:"invited_user_id"`

	Message string `json:"message"`

	Status string `json:"status"`

	ExpiresAt string `json:"expires_at"`

	RespondedAt string `json:"responded_at"`

	CreatedAt string `json:"created_at"`
}

type TeamPortfolioResponse struct {
	ID string `json:"id"`

	TeamID string `json:"team_id"`

	Title       string `json:"title"`
	Description string `json:"description"`

	ImageURL string `json:"image_url"`

	ProjectURL string `json:"project_url"`

	GithubURL string `json:"github_url"`

	CreatedAt string `json:"created_at"`

	UpdatedAt string `json:"updated_at"`
}

type TeamSkillResponse struct {
	ID string `json:"id"`

	TeamID string `json:"team_id"`

	SkillName string `json:"skill_name"`

	ExperienceLevel string `json:"experience_level"`

	CreatedAt string `json:"created_at"`

	UpdatedAt string `json:"updated_at"`
}

type TeamListResponse struct {
	Teams []TeamResponse `json:"teams"`

	Total int `json:"total"`
}

type TeamMemberListResponse struct {
	Members []TeamMemberResponse `json:"members"`

	Total int `json:"total"`
}

type TeamInvitationListResponse struct {
	Invitations []TeamInvitationResponse `json:"invitations"`

	Total int `json:"total"`
}

type TeamPortfolioListResponse struct {
	Portfolio []TeamPortfolioResponse `json:"portfolio"`

	Total int `json:"total"`
}

type TeamSkillListResponse struct {
	Skills []TeamSkillResponse `json:"skills"`

	Total int `json:"total"`
}