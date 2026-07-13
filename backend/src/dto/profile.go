package dto

type CreateIndividualProfileRequest struct {
	// Basic Info
	Phone             *string `json:"phone"`
	DateOfBirth       *string `json:"date_of_birth"` // typically parseable date string
	Gender            string  `json:"gender"`
	AvatarURL         string  `json:"avatar_url"`
	Bio               string  `json:"bio" binding:"max=500"`
	Country           string  `json:"country"`
	State             string  `json:"state"`
	City              string  `json:"city"`
	Headline          string  `json:"headline" binding:"max=150"`
	PreferredLanguage string  `json:"preferred_language"`
	TimeZone          string  `json:"timezone"`
	CountryCode       string  `json:"country_code"`

	// Freelancer Specific Fields
	ExperienceLevel   string   `json:"experience_level"`
	Availability      string   `json:"availability"`
	Skills            []string `json:"skills"`
	ToolsTechnologies []string `json:"tools_technologies"`
	ServiceCategories []string `json:"service_categories"`

	PortfolioURL string `json:"portfolio_url"`
	GithubURL    string `json:"github_url"`
	LinkedinURL  string `json:"linkedin_url"`
	ResumeURL    string `json:"resume_url"`

	TermsConfirmed    bool   `json:"terms_confirmed"`
	ProfileVisibility string `json:"profile_visibility"`
}

type CreateAgencyProfileRequest struct {
	AgencyName      string   `json:"agency_name" binding:"required"`
	LogoURL         string   `json:"logo_url"`
	Description     string   `json:"description" binding:"max=1000"`
	WebsiteURL      string   `json:"website_url"`
	ServicesOffered []string `json:"services_offered"`
	Industries      []string `json:"industries"`
	TeamSize        string   `json:"team_size"`

	ContactName    string  `json:"contact_name"`
	Phone          *string `json:"phone"`
	RegistrationNo string  `json:"registration_no"`
	Country        string  `json:"country"`
	State          string  `json:"state"`
	City           string  `json:"city"`
	TimeZone       string  `json:"timezone"`
	CountryCode    string  `json:"country_code"`
}

type CreateClientProfileRequest struct {
	CompanyName  string   `json:"company_name" binding:"required"`
	LogoURL      string   `json:"logo_url"`
	Industry     string   `json:"industry"`
	WebsiteURL   string   `json:"website_url"`
	ProjectTypes []string `json:"project_types"`
	BudgetRange  string   `json:"budget_range"`
	TeamSize     string   `json:"team_size"`

	ContactName string  `json:"contact_name"`
	Phone       *string `json:"phone"`
	Country     string  `json:"country"`
	State       string  `json:"state"`
	City        string  `json:"city"`
	TimeZone    string  `json:"timezone"`
	CountryCode string  `json:"country_code"`
}

type CreateProfileResponse struct {
	Message       string `json:"message"`
	PublicUrlSlug string `json:"public_url_slug"` // Primarily for individual profile, can be empty for others for now
}

type SetAccountTypeRequest struct {
	AccountType string `json:"account_type" binding:"required,oneof=individual agency client"`
}

type PublicProfileResponse struct {
	// User info
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	AccountType string `json:"account_type"`
	Points      int    `json:"points"`

	// Profile info
	Phone             string   `json:"phone,omitempty"`
	DateOfBirth       string   `json:"date_of_birth,omitempty"`
	Gender            string   `json:"gender,omitempty"`
	AvatarURL         string   `json:"avatar_url,omitempty"`
	Bio               string   `json:"bio,omitempty"`
	Country           string   `json:"country,omitempty"`
	State             string   `json:"state,omitempty"`
	City              string   `json:"city,omitempty"`
	Headline          string   `json:"headline,omitempty"`
	PreferredLanguage string   `json:"preferred_language,omitempty"`
	TimeZone          string   `json:"timezone,omitempty"`
	CountryCode       string   `json:"country_code,omitempty"`
	PublicUrlSlug     string   `json:"public_url_slug"`
	ExperienceLevel   string   `json:"experience_level,omitempty"`
	Availability      string   `json:"availability,omitempty"`
	Skills            []string `json:"skills,omitempty"`
	ToolsTechnologies []string `json:"tools_technologies,omitempty"`
	ServiceCategories []string `json:"service_categories,omitempty"`
	PortfolioURL      string   `json:"portfolio_url,omitempty"`
	GithubURL         string   `json:"github_url,omitempty"`
	LinkedinURL       string   `json:"linkedin_url,omitempty"`
	ResumeURL         string   `json:"resume_url,omitempty"`
	ProfileVisibility string   `json:"profile_visibility,omitempty"`
	MemberSince       string   `json:"member_since"`
}

type UserPointsResponse struct {
	Points          int    `json:"points"`
	AccountType     string `json:"account_type"`
	ProfileComplete bool   `json:"profile_complete"`
}
