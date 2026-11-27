package dto

// ---------- Requests ----------

type CreatePartyRequest struct {
	Name        string `json:"name" huma:"required" example:"Bengaluru Go Guild"`
	Slug        string `json:"slug" huma:"required" example:"bengaluru-go-guild"`
	Description string `json:"description" example:"A 12-engineer party focused on Go, gRPC and cloud-native delivery."`
	LogoURL     string `json:"logo_url" example:"https://cdn.techguild.in/logos/bengaluru-go-guild.png"`
	BannerURL   string `json:"banner_url" example:"https://cdn.techguild.in/banners/bengaluru-go-guild.png"`
	IsHiring    bool   `json:"is_hiring" example:"true"`
}

type UpdatePartyRequest struct {
	Name        string `json:"name" example:"Bengaluru Go Guild"`
	Description string `json:"description" example:"Now 18 engineers; added a DevOps pod."`
	LogoURL     string `json:"logo_url" example:"https://cdn.techguild.in/logos/bengaluru-go-guild.png"`
	BannerURL   string `json:"banner_url" example:"https://cdn.techguild.in/banners/bengaluru-go-guild.png"`
	IsHiring    bool   `json:"is_hiring" example:"false"`
}

type InviteMemberRequest struct {
	UserID  string `json:"user_id" huma:"required" example:"9f8b1c2d-3e4f-4a5b-8c6d-7e8f9a0b1c2d"`
	Message string `json:"message" example:"We'd love to have you join the Bengaluru Go Guild."`
}

type GetAvailablePartiesInput struct {
	Search string `query:"search"`
	Page   int    `query:"page"`
	Limit  int    `query:"limit"`
}

type RejectInvitationRequest struct {
	Reason string `json:"reason" example:"Committed to a client engagement through March; cannot take on party work."`
}

type LeavePartyRequest struct {
	Reason string `json:"reason" example:"Moving to an in-house role; leaving the party on good terms."`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role" huma:"required" example:"admin"`
}

type CreatePortfolioRequest struct {
	Title       string `json:"title" huma:"required" example:"UPI Reconciliation Dashboard"`
	Description string `json:"description" example:"Real-time settlement dashboard for a Pune NBFC, built with Go, React and ClickHouse."`
	ImageURL    string `json:"image_url" example:"https://cdn.techguild.in/portfolio/upi-reconciliation.png"`
	ProjectURL  string `json:"project_url" example:"https://cygnetsolutions.in/case-studies/upi-reconciliation"`
	GithubURL   string `json:"github_url" example:"https://github.com/cygnet-software/upi-reconciliation"`
}

type UpdatePortfolioRequest struct {
	Title       string `json:"title" example:"UPI Reconciliation Dashboard v2"`
	Description string `json:"description" example:"Added multi-bank settlement matching and anomaly alerts."`
	ImageURL    string `json:"image_url" example:"https://cdn.techguild.in/portfolio/upi-reconciliation-v2.png"`
	ProjectURL  string `json:"project_url" example:"https://cygnetsolutions.in/case-studies/upi-reconciliation-v2"`
	GithubURL   string `json:"github_url" example:"https://github.com/cygnet-software/upi-reconciliation-v2"`
}

type AddSkillRequest struct {
	SkillName       string `json:"skill_name" huma:"required" example:"Terraform"`
	ExperienceLevel string `json:"experience_level" example:"expert"`
}

// ---------- Responses ----------

type PartyResponse struct {
	ID          string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string `json:"name" example:"TechGuild Devs"`
	Slug        string `json:"slug" example:"techguild-devs"`
	Description string `json:"description" example:"A party of full-stack developers"`
	LogoURL     string `json:"logo_url" example:"https://cdn.techguild.in/logos/technova-labs.png"`
	BannerURL   string `json:"banner_url" example:"https://cdn.techguild.in/banners/technova-labs.png"`
	LeaderID    string `json:"leader_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	IsHiring    bool   `json:"is_hiring" example:"true"`
	IsVerified  bool   `json:"is_verified" example:"true"`
	// Status      string `json:"status" example:"active"`
	CreatedAt string `json:"created_at" example:"2026-01-15T00:00:00Z"`
	UpdatedAt string `json:"updated_at" example:"2026-01-15T00:00:00Z"`
}

// new
type PartyDetailsResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url"`
	BannerURL   string `json:"banner_url"`
	LeaderID    string `json:"leader_id"`
	IsHiring    bool   `json:"is_hiring"`
	IsVerified  bool   `json:"is_verified"`

	Members   []PartyMemberResponse    `json:"members"`
	Portfolio []PartyPortfolioResponse `json:"portfolio"`
	Skills    []PartySkillResponse     `json:"skills"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// continued
type PartyMemberResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PartyID   string `json:"party_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID    string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Role      string `json:"role" example:"member"`
	Status    string `json:"status" example:"active"`
	JoinedAt  string `json:"joined_at" example:"2026-01-15T00:00:00Z"`
	CreatedAt string `json:"created_at" example:"2026-01-15T00:00:00Z"`
	UpdatedAt string `json:"updated_at" example:"2026-01-15T00:00:00Z"`
}

type PartyInvitationResponse struct {
	ID            string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PartyID       string `json:"party_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	InvitedByID   string `json:"invited_by_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	InvitedUserID string `json:"invited_user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Message       string `json:"message" example:"We'd love to have you on our party!"`
	Status        string `json:"status" example:"pending"`
	ExpiresAt     string `json:"expires_at" example:"2026-02-15T00:00:00Z"`
	RespondedAt   string `json:"responded_at" example:"2026-01-20T00:00:00Z"`
	CreatedAt     string `json:"created_at" example:"2026-01-15T00:00:00Z"`
}

type PartyPortfolioResponse struct {
	ID          string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PartyID     string `json:"party_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Title       string `json:"title" example:"E-commerce Platform"`
	Description string `json:"description" example:"Built a full-stack e-commerce platform with React and Go"`
	ImageURL    string `json:"image_url" example:"https://cdn.techguild.in/portfolio/technova-labs-dashboard.png"`
	ProjectURL  string `json:"project_url" example:"https://technova-labs.in/case-studies/fintech-dashboard"`
	GithubURL   string `json:"github_url" example:"https://github.com/user/project"`
	CreatedAt   string `json:"created_at" example:"2026-01-15T00:00:00Z"`
	UpdatedAt   string `json:"updated_at" example:"2026-01-15T00:00:00Z"`
}

type PartySkillResponse struct {
	ID              string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PartyID         string `json:"party_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SkillName       string `json:"skill_name" example:"Go"`
	ExperienceLevel string `json:"experience_level" example:"intermediate"`
	CreatedAt       string `json:"created_at" example:"2026-01-15T00:00:00Z"`
	UpdatedAt       string `json:"updated_at" example:"2026-01-15T00:00:00Z"`
}

type AvailablePartyListResponse struct {
	Parties []PartyResponse `json:"parties"`
	Total   int             `json:"total"`
	Page    int             `json:"page"`
	Limit   int             `json:"limit"`
}

type PartyListResponse struct {
	Parties []PartyResponse `json:"parties"`
	Total   int             `json:"total" example:"1"`
}

type PartyMemberListResponse struct {
	Members []PartyMemberResponse `json:"members"`
	Total   int                   `json:"total" example:"1"`
}

type PartyInvitationListResponse struct {
	Invitations []PartyInvitationResponse `json:"invitations"`
	Total       int                       `json:"total" example:"1"`
}

type PartyPortfolioListResponse struct {
	Portfolio []PartyPortfolioResponse `json:"portfolio"`
	Total     int                      `json:"total" example:"1"`
}

type PartySkillListResponse struct {
	Skills []PartySkillResponse `json:"skills"`
	Total  int                  `json:"total" example:"1"`
}

type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

// ---------- Huma Input/Output wrapper structs ----------

type CreatePartyInput struct {
	Body CreatePartyRequest
}
type CreatePartyOutput struct {
	Body PartyResponse
}

type UpdatePartyInput struct {
	ID   string `path:"party_id" doc:"Party ID"`
	Body UpdatePartyRequest
}
type UpdatePartyOutput struct {
	Body MessageResponse
}

type DeletePartyInput struct {
	ID string `path:"party_id" doc:"Party ID"`
}
type DeletePartyOutput struct {
	Body MessageResponse
}
type GetAvailablePartiesOutput struct {
	Body AvailablePartyListResponse
}
type GetPartyInput struct {
	ID string `path:"party_id" doc:"Party ID"`
}
type GetPartyOutput struct {
	Body PartyDetailsResponse
}

// New
type GetPartyMembersInput struct {
	PartyID string `path:"party_id" doc:"Party ID"`
}

type GetPartyMembersOutput struct {
	Body PartyMemberListResponse
}

type GetPartyInvitationsInput struct {
	PartyID string `path:"party_id" doc:"Party ID"`
}

type GetPartyInvitationsOutput struct {
	Body PartyInvitationListResponse
}

type GetPartyPortfolioInput struct {
	PartyID string `path:"party_id" doc:"Party ID"`
}

type GetPartyPortfolioOutput struct {
	Body PartyPortfolioListResponse
}

type GetPartySkillsInput struct {
	PartyID string `path:"party_id" doc:"Party ID"`
}

type GetPartySkillsOutput struct {
	Body PartySkillListResponse
}

// continued
type GetMyPartiesInput struct{}
type GetMyPartiesOutput struct {
	Body PartyListResponse
}

type InviteMemberInput struct {
	PartyID string `path:"party_id" doc:"Party ID"`
	Body    InviteMemberRequest
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
	PartyID  string `path:"party_id" doc:"Party ID"`
	MemberID string `path:"member_id" doc:"Member ID"`
}
type RemoveMemberOutput struct {
	Body MessageResponse
}

type LeavePartyInput struct {
	PartyID string `path:"party_id" doc:"Party ID"`
	Body    LeavePartyRequest
}
type LeavePartyOutput struct {
	Body MessageResponse
}

type CreatePortfolioInput struct {
	PartyID string `path:"party_id" doc:"Party ID"`
	Body    CreatePortfolioRequest
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
	PartyID string `path:"party_id" doc:"Party ID"`
	Body    AddSkillRequest
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
