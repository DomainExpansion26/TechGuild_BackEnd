package dto

// ---------- Requests ----------

type CreateTeamRequest struct {
	Name        string `json:"name" huma:"required" example:"TechGuild Devs"`
	Slug        string `json:"slug" huma:"required" example:"techguild-devs"`
	Description string `json:"description" example:"A team of full-stack developers"`
	LogoURL     string `json:"logo_url" example:"https://storage.example.com/logos/team.png"`
	BannerURL   string `json:"banner_url" example:"https://storage.example.com/banners/team.png"`
	IsHiring    bool   `json:"is_hiring" example:"true"`
}

type UpdateTeamRequest struct {
	Name        string `json:"name" example:"TechGuild Devs v2"`
	Description string `json:"description" example:"Updated team description"`
	LogoURL     string `json:"logo_url" example:"https://storage.example.com/logos/team-v2.png"`
	BannerURL   string `json:"banner_url" example:"https://storage.example.com/banners/team-v2.png"`
	IsHiring    bool   `json:"is_hiring" example:"false"`
}

type InviteMemberRequest struct {
	UserID  string `json:"user_id" huma:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Message string `json:"message" example:"We'd love to have you on our team!"`
}

type RejectInvitationRequest struct {
	Reason string `json:"reason" example:"Currently busy with other projects"`
}

type LeaveTeamRequest struct {
	Reason string `json:"reason" example:"Joining another team"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" huma:"required" example:"admin"`
}

type CreatePortfolioRequest struct {
	Title       string `json:"title" huma:"required" example:"E-commerce Platform"`
	Description string `json:"description" example:"Built a full-stack e-commerce platform with React and Go"`
	ImageURL    string `json:"image_url" example:"https://storage.example.com/portfolio/project.png"`
	ProjectURL  string `json:"project_url" example:"https://example.com/project"`
	GithubURL   string `json:"github_url" example:"https://github.com/user/project"`
}

type UpdatePortfolioRequest struct {
	Title       string `json:"title" example:"E-commerce Platform v2"`
	Description string `json:"description" example:"Updated project with payment integration"`
	ImageURL    string `json:"image_url" example:"https://storage.example.com/portfolio/project-v2.png"`
	ProjectURL  string `json:"project_url" example:"https://example.com/project-v2"`
	GithubURL   string `json:"github_url" example:"https://github.com/user/project-v2"`
}

type AddSkillRequest struct {
	SkillName       string `json:"skill_name" huma:"required" example:"Go"`
	ExperienceLevel string `json:"experience_level" example:"intermediate"`
}

// ---------- Responses ----------

type TeamResponse struct {
	ID          string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string `json:"name" example:"TechGuild Devs"`
	Slug        string `json:"slug" example:"techguild-devs"`
	Description string `json:"description" example:"A team of full-stack developers"`
	LogoURL     string `json:"logo_url" example:"https://storage.example.com/logos/team.png"`
	BannerURL   string `json:"banner_url" example:"https://storage.example.com/banners/team.png"`
	LeaderID    string `json:"leader_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	IsHiring    bool   `json:"is_hiring" example:"true"`
	IsVerified  bool   `json:"is_verified" example:"true"`
	Status      string `json:"status" example:"active"`
	CreatedAt   string `json:"created_at" example:"2026-01-15T00:00:00Z"`
	UpdatedAt   string `json:"updated_at" example:"2026-01-15T00:00:00Z"`
}

type TeamMemberResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TeamID    string `json:"team_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID    string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Role      string `json:"role" example:"member"`
	Status    string `json:"status" example:"active"`
	JoinedAt  string `json:"joined_at" example:"2026-01-15T00:00:00Z"`
	CreatedAt string `json:"created_at" example:"2026-01-15T00:00:00Z"`
	UpdatedAt string `json:"updated_at" example:"2026-01-15T00:00:00Z"`
}

type TeamInvitationResponse struct {
	ID            string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TeamID        string `json:"team_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	InvitedByID   string `json:"invited_by_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	InvitedUserID string `json:"invited_user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Message       string `json:"message" example:"We'd love to have you on our team!"`
	Status        string `json:"status" example:"pending"`
	ExpiresAt     string `json:"expires_at" example:"2026-02-15T00:00:00Z"`
	RespondedAt   string `json:"responded_at" example:"2026-01-20T00:00:00Z"`
	CreatedAt     string `json:"created_at" example:"2026-01-15T00:00:00Z"`
}

type TeamPortfolioResponse struct {
	ID          string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TeamID      string `json:"team_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Title       string `json:"title" example:"E-commerce Platform"`
	Description string `json:"description" example:"Built a full-stack e-commerce platform with React and Go"`
	ImageURL    string `json:"image_url" example:"https://storage.example.com/portfolio/project.png"`
	ProjectURL  string `json:"project_url" example:"https://example.com/project"`
	GithubURL   string `json:"github_url" example:"https://github.com/user/project"`
	CreatedAt   string `json:"created_at" example:"2026-01-15T00:00:00Z"`
	UpdatedAt   string `json:"updated_at" example:"2026-01-15T00:00:00Z"`
}

type TeamSkillResponse struct {
	ID              string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TeamID          string `json:"team_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SkillName       string `json:"skill_name" example:"Go"`
	ExperienceLevel string `json:"experience_level" example:"intermediate"`
	CreatedAt       string `json:"created_at" example:"2026-01-15T00:00:00Z"`
	UpdatedAt       string `json:"updated_at" example:"2026-01-15T00:00:00Z"`
}

type TeamListResponse struct {
	Teams []TeamResponse `json:"teams"`
	Total int            `json:"total" example:"1"`
}

type TeamMemberListResponse struct {
	Members []TeamMemberResponse `json:"members"`
	Total   int                  `json:"total" example:"1"`
}

type TeamInvitationListResponse struct {
	Invitations []TeamInvitationResponse `json:"invitations"`
	Total       int                      `json:"total" example:"1"`
}

type TeamPortfolioListResponse struct {
	Portfolio []TeamPortfolioResponse `json:"portfolio"`
	Total     int                     `json:"total" example:"1"`
}

type TeamSkillListResponse struct {
	Skills []TeamSkillResponse `json:"skills"`
	Total  int                 `json:"total" example:"1"`
}

type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

// ---------- Huma Input/Output wrapper structs ----------

type CreateTeamInput struct {
	Body CreateTeamRequest
}
type CreateTeamOutput struct {
	Body TeamResponse
}

type UpdateTeamInput struct {
	ID   string `path:"team_id" doc:"Team ID"`
	Body UpdateTeamRequest
}
type UpdateTeamOutput struct {
	Body MessageResponse
}

type DeleteTeamInput struct {
	ID string `path:"team_id" doc:"Team ID"`
}
type DeleteTeamOutput struct {
	Body MessageResponse
}

type GetTeamInput struct {
	ID string `path:"team_id" doc:"Team ID"`
}
type GetTeamOutput struct {
	Body TeamResponse
}

type GetMyTeamsInput struct{}
type GetMyTeamsOutput struct {
	Body TeamListResponse
}

type InviteMemberInput struct {
	TeamID string `path:"team_id" doc:"Team ID"`
	Body   InviteMemberRequest
}
type InviteMemberOutput struct {
	Body MessageResponse
}

type AcceptInvitationInput struct {
	InvitationID string `path:"invitation_id" doc:"Invitation ID"`
}
type AcceptInvitationOutput struct {
	Body MessageResponse
}

type RejectInvitationInput struct {
	InvitationID string `path:"invitation_id" doc:"Invitation ID"`
	Body         RejectInvitationRequest
}
type RejectInvitationOutput struct {
	Body MessageResponse
}

type RemoveMemberInput struct {
	TeamID   string `path:"team_id" doc:"Team ID"`
	MemberID string `path:"member_id" doc:"Member ID"`
}
type RemoveMemberOutput struct {
	Body MessageResponse
}

type LeaveTeamInput struct {
	TeamID string `path:"team_id" doc:"Team ID"`
	Body   LeaveTeamRequest
}
type LeaveTeamOutput struct {
	Body MessageResponse
}

type CreatePortfolioInput struct {
	TeamID string `path:"team_id" doc:"Team ID"`
	Body   CreatePortfolioRequest
}
type CreatePortfolioOutput struct {
	Body MessageResponse
}

type UpdatePortfolioInput struct {
	PortfolioID string `path:"portfolio_id" doc:"Portfolio ID"`
	Body        UpdatePortfolioRequest
}
type UpdatePortfolioOutput struct {
	Body MessageResponse
}

type DeletePortfolioInput struct {
	PortfolioID string `path:"portfolio_id" doc:"Portfolio ID"`
}
type DeletePortfolioOutput struct {
	Body MessageResponse
}

type AddSkillInput struct {
	TeamID string `path:"team_id" doc:"Team ID"`
	Body   AddSkillRequest
}
type AddSkillOutput struct {
	Body MessageResponse
}

type UpdateSkillInput struct {
	SkillID string `path:"skill_id" doc:"Skill ID"`
	Body    AddSkillRequest
}
type UpdateSkillOutput struct {
	Body MessageResponse
}

type DeleteSkillInput struct {
	SkillID string `path:"skill_id" doc:"Skill ID"`
}
type DeleteSkillOutput struct {
	Body MessageResponse
}
