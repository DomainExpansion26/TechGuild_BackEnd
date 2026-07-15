package dto

type CreateIndividualProfileRequest struct {
	// Basic Info
	Phone             *string `json:"phone" form:"phone"`
	DateOfBirth       *string `json:"date_of_birth" form:"date_of_birth"` // typically parseable date string
	Gender            string  `json:"gender" form:"gender"`
	AvatarURL         string  `json:"avatar_url" form:"avatar_url"`
	Bio               string  `json:"bio" binding:"max=500" form:"bio"`
	Country           string  `json:"country" form:"country"`
	State             string  `json:"state" form:"state"`
	City              string  `json:"city" form:"city"`
	Headline          string  `json:"headline" binding:"max=150" form:"headline"`
	PreferredLanguage string  `json:"preferred_language" form:"preferred_language"`
	TimeZone          string  `json:"timezone" form:"timezone"`
	CountryCode       string  `json:"country_code" form:"country_code"`

	// Freelancer Specific Fields
	ExperienceLevel   string   `json:"experience_level" form:"experience_level"`
	Availability      string   `json:"availability" form:"availability"`
	Skills            []string `json:"skills" form:"skills"`
	ToolsTechnologies []string `json:"tools_technologies" form:"tools_technologies"`
	ServiceCategories []string `json:"service_categories" form:"service_categories"`

	PortfolioURL string `json:"portfolio_url" form:"portfolio_url"`
	GithubURL    string `json:"github_url" form:"github_url"`
	LinkedinURL  string `json:"linkedin_url" form:"linkedin_url"`
	ResumeURL    string `json:"resume_url" form:"resume_url"`

	TermsConfirmed    bool   `json:"terms_confirmed" form:"terms_confirmed"`
	ProfileVisibility string `json:"profile_visibility" form:"profile_visibility"`
}

type CreateAgencyProfileRequest struct {
	AgencyName      string   `json:"agency_name" binding:"required" form:"agency_name"`
	LogoURL         string   `json:"logo_url" form:"logo_url"`
	Description     string   `json:"description" binding:"max=1000" form:"description"`
	WebsiteURL      string   `json:"website_url" form:"website_url"`
	ServicesOffered []string `json:"services_offered" form:"services_offered"`
	Industries      []string `json:"industries" form:"industries"`
	TeamSize        string   `json:"team_size" form:"team_size"`

	ContactName    string  `json:"contact_name" form:"contact_name"`
	Phone          *string `json:"phone" form:"phone"`
	RegistrationNo string  `json:"registration_no" form:"registration_no"`
	Country        string  `json:"country" form:"country"`
	State          string  `json:"state" form:"state"`
	City           string  `json:"city" form:"city"`
	TimeZone       string  `json:"timezone" form:"timezone"`
	CountryCode    string  `json:"country_code" form:"country_code"`
}

type CreateClientProfileRequest struct {
	CompanyName  string   `json:"company_name" binding:"required" form:"company_name"`
	LogoURL      string   `json:"logo_url" form:"logo_url"`
	Industry     string   `json:"industry" form:"industry"`
	WebsiteURL   string   `json:"website_url" form:"website_url"`
	ProjectTypes []string `json:"project_types" form:"project_types"`
	BudgetRange  string   `json:"budget_range" form:"budget_range"`
	TeamSize     string   `json:"team_size" form:"team_size"`

	ContactName string  `json:"contact_name" form:"contact_name"`
	Phone       *string `json:"phone" form:"phone"`
	Country     string  `json:"country" form:"country"`
	State       string  `json:"state" form:"state"`
	City        string  `json:"city" form:"city"`
	TimeZone    string  `json:"timezone" form:"timezone"`
	CountryCode string  `json:"country_code" form:"country_code"`
}

type CreateProfileResponse struct {
	Message       string `json:"message"`
	PublicUrlSlug string `json:"public_url_slug"` // Primarily for individual profile, can be empty for others for now
}

type SetAccountTypeRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required"`
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

type ExportResponse struct {
	Message     string `json:"message"`
	DownloadURL string `json:"download_url"`
	ExpiresIn   string `json:"expires_in"`
}

type CheckSlugResponse struct {
	Available    bool     `json:"available"`
	Alternatives []string `json:"alternatives,omitempty"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required"`
}
