package dto

import (
	"github.com/danielgtaylor/huma/v2"
)

// ============================================================
// CREATE REQUESTS
// ============================================================

type CreateIndividualProfileRequest struct {
	Phone             *string `json:"phone,omitempty" maxLength:"20" example:"+919812345678"`
	DateOfBirth       *string `json:"date_of_birth,omitempty" format:"date" maxLength:"10" example:"1990-01-15"`
	Gender            *string `json:"gender,omitempty" enum:"male,female,other" example:"female"`
	AvatarURL         *string `json:"avatar_url,omitempty" format:"uri" maxLength:"2048" example:"https://cdn.techguild.in/avatars/ananya-iyer.png"`
	Bio               *string `json:"bio,omitempty" maxLength:"500" example:"Backend engineer with 8 years building Go services for fintech and logistics."`
	Country           *string `json:"country,omitempty" maxLength:"100" example:"India"`
	City              *string `json:"city,omitempty" maxLength:"100" example:"Bengaluru"`
	Headline          *string `json:"headline,omitempty" maxLength:"150" example:"Lead Backend Engineer"`
	PreferredLanguage *string `json:"preferred_language,omitempty" maxLength:"20" example:"en"`
	TimeZone          *string `json:"timezone,omitempty" maxLength:"100" example:"Asia/Kolkata"`

	ExperienceLevel   *string   `json:"experience_level,omitempty" enum:"junior,mid,senior,lead" example:"senior"`
	Availability      *string   `json:"availability,omitempty" enum:"full-time,part-time,contract" example:"full-time"`
	Skills            *[]string `json:"skills,omitempty" maxItems:"50" maxLength:"100"`
	ToolsTechnologies *[]string `json:"tools_technologies,omitempty" maxItems:"50" maxLength:"100"`
	ServiceCategories *[]string `json:"service_categories,omitempty" maxItems:"50" maxLength:"100"`

	PortfolioURL *string `json:"portfolio_url,omitempty" format:"uri" maxLength:"255" example:"https://cdn.techguild.in/portfolio/ananya-iyer"`
	GithubURL    *string `json:"github_url,omitempty" format:"uri" maxLength:"255" example:"https://github.com/ananya-iyer"`
	LinkedinURL  *string `json:"linkedin_url,omitempty" format:"uri" maxLength:"255" example:"https://linkedin.com/in/ananya-iyer"`
	ResumeURL    *string `json:"resume_url,omitempty" format:"uri" maxLength:"255" example:"https://cdn.techguild.in/resumes/ananya-iyer.pdf"`

	TermsConfirmed *bool `json:"terms_confirmed,omitempty" example:"true"`
}

type UpdateIndividualProfileRequest struct {
	Phone             *string `json:"phone,omitempty" maxLength:"20" example:"+919812345678"`
	DateOfBirth       *string `json:"date_of_birth,omitempty" format:"date" maxLength:"10" example:"1990-01-15"`
	Gender            *string `json:"gender,omitempty" enum:"male,female,other" example:"female"`
	AvatarURL         *string `json:"avatar_url,omitempty" format:"uri" maxLength:"2048" example:"https://cdn.techguild.in/avatars/ananya-iyer.png"`
	Bio               *string `json:"bio,omitempty" maxLength:"500" example:"Backend engineer with 8 years building Go services for fintech and logistics."`
	Country           *string `json:"country,omitempty" maxLength:"100" example:"India"`
	City              *string `json:"city,omitempty" maxLength:"100" example:"Bengaluru"`
	Headline          *string `json:"headline,omitempty" maxLength:"150" example:"Lead Backend Engineer"`
	PreferredLanguage *string `json:"preferred_language,omitempty" maxLength:"20" example:"en"`
	TimeZone          *string `json:"timezone,omitempty" maxLength:"100" example:"Asia/Kolkata"`

	ExperienceLevel   *string   `json:"experience_level,omitempty" enum:"junior,mid,senior,lead" example:"senior"`
	Availability      *string   `json:"availability,omitempty" enum:"full-time,part-time,contract" example:"full-time"`
	Skills            *[]string `json:"skills,omitempty" maxItems:"50" maxLength:"100"`
	ToolsTechnologies *[]string `json:"tools_technologies,omitempty" maxItems:"50" maxLength:"100"`
	ServiceCategories *[]string `json:"service_categories,omitempty" maxItems:"50" maxLength:"100"`

	PortfolioURL *string `json:"portfolio_url,omitempty" format:"uri" maxLength:"255" example:"https://cdn.techguild.in/portfolio/ananya-iyer"`
	GithubURL    *string `json:"github_url,omitempty" format:"uri" maxLength:"255" example:"https://github.com/ananya-iyer"`
	LinkedinURL  *string `json:"linkedin_url,omitempty" format:"uri" maxLength:"255" example:"https://linkedin.com/in/ananya-iyer"`
	ResumeURL    *string `json:"resume_url,omitempty" format:"uri" maxLength:"255" example:"https://cdn.techguild.in/resumes/ananya-iyer.pdf"`

	TermsConfirmed *bool `json:"terms_confirmed,omitempty" example:"true"`
}

type CreateAgencyProfileRequest struct {
	Phone           *string   `json:"phone,omitempty" maxLength:"20" example:"+919820012345"`
	AgencyName      string    `json:"agency_name" required:"true" minLength:"1" maxLength:"255" example:"Cygnet Software Solutions Pvt Ltd"`
	LogoURL         *string   `json:"logo_url,omitempty" format:"uri" maxLength:"255" example:"https://cdn.techguild.in/logos/cygnet.png"`
	Description     *string   `json:"description,omitempty" maxLength:"1000" example:"Pune-based product engineering agency for fintech and logistics clients."`
	WebsiteURL      *string   `json:"website_url,omitempty" format:"uri" maxLength:"255" example:"https://cygnetsolutions.in"`
	ServicesOffered *[]string `json:"services_offered,omitempty" maxItems:"50" maxLength:"100"`
	Industries      *[]string `json:"industries,omitempty" maxItems:"50" maxLength:"100"`
	TeamSize        *string   `json:"team_size,omitempty" enum:"1-10,10-50,50-200,200+" example:"50-200"`

	ContactName *string `json:"contact_name,omitempty" maxLength:"100" example:"Rohan Mehta"`
	Country     *string `json:"country,omitempty" maxLength:"100" example:"India"`
	City        *string `json:"city,omitempty" maxLength:"100" example:"Pune"`
	TimeZone    *string `json:"timezone,omitempty" maxLength:"100" example:"Asia/Kolkata"`
}

type UpdateAgencyProfileRequest struct {
	Phone           *string   `json:"phone,omitempty" maxLength:"20" example:"+919820012345"`
	AgencyName      *string   `json:"agency_name,omitempty" minLength:"1" maxLength:"255" example:"Cygnet Software Solutions Pvt Ltd"`
	LogoURL         *string   `json:"logo_url,omitempty" format:"uri" maxLength:"255" example:"https://cdn.techguild.in/logos/cygnet.png"`
	Description     *string   `json:"description,omitempty" maxLength:"1000" example:"Pune-based product engineering agency for fintech and logistics clients."`
	WebsiteURL      *string   `json:"website_url,omitempty" format:"uri" maxLength:"255" example:"https://cygnetsolutions.in"`
	ServicesOffered *[]string `json:"services_offered,omitempty" maxItems:"50" maxLength:"100"`
	Industries      *[]string `json:"industries,omitempty" maxItems:"50" maxLength:"100"`
	TeamSize        *string   `json:"team_size,omitempty" enum:"1-10,10-50,50-200,200+" example:"50-200"`

	ContactName *string `json:"contact_name,omitempty" maxLength:"100" example:"Rohan Mehta"`
	Country     *string `json:"country,omitempty" maxLength:"100" example:"India"`
	City        *string `json:"city,omitempty" maxLength:"100" example:"Pune"`
	TimeZone    *string `json:"timezone,omitempty" maxLength:"100" example:"Asia/Kolkata"`
}

type CreateClientProfileRequest struct {
	Phone        *string   `json:"phone,omitempty" maxLength:"20" example:"+919833012345"`
	CompanyName  string    `json:"company_name" required:"true" minLength:"1" maxLength:"255" example:"Sundaram Logistics Pvt Ltd"`
	LogoURL      *string   `json:"logo_url,omitempty" format:"uri" maxLength:"255" example:"https://cdn.techguild.in/logos/sundaram.png"`
	Industry     *string   `json:"industry,omitempty" maxLength:"100" example:"logistics"`
	WebsiteURL   *string   `json:"website_url,omitempty" format:"uri" maxLength:"255" example:"https://sundaramlogistics.in"`
	ProjectTypes *[]string `json:"project_types,omitempty" maxItems:"50" maxLength:"100"`
	BudgetRange  *string   `json:"budget_range,omitempty" maxLength:"50" example:"INR 5,00,000 - 20,00,000" doc:"Free-text INR budget band. Prefix the value with INR; amounts are in rupees (major units)."`
	TeamSize     *string   `json:"team_size,omitempty" enum:"1-10,10-50,50-200,200+" example:"50-200"`

	Country  *string `json:"country,omitempty" maxLength:"100" example:"India"`
	City     *string `json:"city,omitempty" maxLength:"100" example:"Mumbai"`
	TimeZone *string `json:"timezone,omitempty" maxLength:"100" example:"Asia/Kolkata"`
}

type UpdateClientProfileRequest struct {
	Phone        *string   `json:"phone,omitempty" maxLength:"20" example:"+919833012345"`
	CompanyName  *string   `json:"company_name,omitempty" minLength:"1" maxLength:"255" example:"Sundaram Logistics Pvt Ltd"`
	LogoURL      *string   `json:"logo_url,omitempty" format:"uri" maxLength:"255" example:"https://cdn.techguild.in/logos/sundaram.png"`
	Industry     *string   `json:"industry,omitempty" maxLength:"100" example:"logistics"`
	WebsiteURL   *string   `json:"website_url,omitempty" format:"uri" maxLength:"255" example:"https://sundaramlogistics.in"`
	ProjectTypes *[]string `json:"project_types,omitempty" maxItems:"50" maxLength:"100"`
	BudgetRange  *string   `json:"budget_range,omitempty" maxLength:"50" example:"INR 5,00,000 - 20,00,000" doc:"Free-text INR budget band. Prefix the value with INR; amounts are in rupees (major units)."`
	TeamSize     *string   `json:"team_size,omitempty" enum:"1-10,10-50,50-200,200+" example:"50-200"`

	Country  *string `json:"country,omitempty" maxLength:"100" example:"India"`
	City     *string `json:"city,omitempty" maxLength:"100" example:"Mumbai"`
	TimeZone *string `json:"timezone,omitempty" maxLength:"100" example:"Asia/Kolkata"`
}

// ============================================================
// RESPONSES
// ============================================================

type CreateProfileResponse struct {
	Message       string `json:"message" example:"Profile created successfully"`
	PublicUrlSlug string `json:"public_url_slug" example:"john-doe"`
}

// PublicIndividualProfile is the public-facing shape for individual profiles.
type PublicIndividualProfile struct {
	FirstName         string   `json:"first_name" example:"John"`
	LastName          string   `json:"last_name" example:"Doe"`
	AvatarURL         string   `json:"avatar_url,omitempty"`
	Bio               string   `json:"bio,omitempty"`
	Country           string   `json:"country,omitempty"`
	Headline          string   `json:"headline,omitempty"`
	PreferredLanguage string   `json:"preferred_language,omitempty"`
	TimeZone          string   `json:"timezone,omitempty"`
	PublicUrlSlug     string   `json:"public_url_slug" example:"john-doe"`
	ExperienceLevel   string   `json:"experience_level,omitempty"`
	Availability      string   `json:"availability,omitempty"`
	Skills            []string `json:"skills,omitempty"`
	ToolsTechnologies []string `json:"tools_technologies,omitempty"`
	ServiceCategories []string `json:"service_categories,omitempty"`
	PortfolioURL      string   `json:"portfolio_url,omitempty"`
	GithubURL         string   `json:"github_url,omitempty"`
	LinkedinURL       string   `json:"linkedin_url,omitempty"`
	MemberSince       string   `json:"member_since" example:"August 2026"`
}

// PublicAgencyProfile is the public-facing shape for agency profiles.
type PublicAgencyProfile struct {
	FirstName       string   `json:"first_name" example:"Jane"`
	LastName        string   `json:"last_name" example:"Smith"`
	AgencyName      string   `json:"agency_name" example:"Acme Digital Agency"`
	LogoURL         string   `json:"logo_url,omitempty"`
	Description     string   `json:"description,omitempty"`
	WebsiteURL      string   `json:"website_url,omitempty"`
	ServicesOffered []string `json:"services_offered,omitempty"`
	Industries      []string `json:"industries,omitempty"`
	TeamSize        string   `json:"team_size,omitempty"`
	Country         string   `json:"country,omitempty"`
	TimeZone        string   `json:"timezone,omitempty"`
	PublicUrlSlug   string   `json:"public_url_slug" example:"acme-digital-agency"`
	MemberSince     string   `json:"member_since" example:"August 2026"`
}

// PublicClientProfile is the public-facing shape for client profiles.
type PublicClientProfile struct {
	FirstName     string   `json:"first_name" example:"Jane"`
	LastName      string   `json:"last_name" example:"Smith"`
	CompanyName   string   `json:"company_name" example:"TechStart Inc"`
	LogoURL       string   `json:"logo_url,omitempty"`
	Industry      string   `json:"industry,omitempty"`
	WebsiteURL    string   `json:"website_url,omitempty"`
	ProjectTypes  []string `json:"project_types,omitempty"`
	TeamSize      string   `json:"team_size,omitempty"`
	Country       string   `json:"country,omitempty"`
	TimeZone      string   `json:"timezone,omitempty"`
	PublicUrlSlug string   `json:"public_url_slug" example:"techstart-inc"`
	MemberSince   string   `json:"member_since" example:"August 2026"`
}

type GetPublicProfileResponse struct {
	AccountType string                   `json:"account_type" enum:"individual,agency,client" example:"individual"`
	Individual  *PublicIndividualProfile `json:"individual,omitempty"`
	Agency      *PublicAgencyProfile     `json:"agency,omitempty"`
	Client      *PublicClientProfile     `json:"client,omitempty"`
}

type UserPointsResponse struct {
	Points          int    `json:"points" example:"150"`
	AccountType     string `json:"account_type" enum:"individual,agency,client" example:"individual"`
	ProfileComplete bool   `json:"profile_complete" example:"true"`
}

type ExportResponse struct {
	Message     string `json:"message" example:"Export started"`
	DownloadURL string `json:"download_url" example:"https://cdn.techguild.in/exports/aarav-sharma-data.zip"`
	ExpiresAt   string `json:"expires_at" example:"2026-01-16T00:00:00Z"`
}

type CheckSlugResponse struct {
	Available    bool     `json:"available" example:"true"`
	Alternatives []string `json:"alternatives,omitempty"`
}

// ============================================================
// HUMA INPUT/OUTPUT WRAPPERS
// ============================================================

type IndividualProfileInput struct {
	Body CreateIndividualProfileRequest
}
type IndividualProfileOutput struct {
	Body CreateProfileResponse
}

type UpdateIndividualProfileInput struct {
	Body UpdateIndividualProfileRequest
}
type UpdateIndividualProfileOutput struct {
	Body CreateProfileResponse
}

type AgencyProfileInput struct {
	Body CreateAgencyProfileRequest
}
type AgencyProfileOutput struct {
	Body CreateProfileResponse
}

type UpdateAgencyProfileInput struct {
	Body UpdateAgencyProfileRequest
}
type UpdateAgencyProfileOutput struct {
	Body CreateProfileResponse
}

type ClientProfileInput struct {
	Body CreateClientProfileRequest
}
type ClientProfileOutput struct {
	Body CreateProfileResponse
}

type UpdateClientProfileInput struct {
	Body UpdateClientProfileRequest
}
type UpdateClientProfileOutput struct {
	Body CreateProfileResponse
}

type GetMyProfileInput struct{}
type GetMyProfileOutput struct {
	Body GetMyProfileResponse
}

// GetMyProfileResponse is a typed union of the three profile shapes.
// Only the field corresponding to the authenticated user's account type is populated.
type GetMyProfileResponse struct {
	// --- User-level (always present) ---
	AccountType      string      `json:"account_type,omitempty" example:"individual"`
	ProfileCompleted bool        `json:"profile_completed" example:"false"`
	User             UserSummary `json:"user"`

	// --- Profile-level (only when profile exists) ---
	Individual *MyIndividualProfile `json:"individual,omitempty"`
	Agency     *MyAgencyProfile     `json:"agency,omitempty"`
	Client     *MyClientProfile     `json:"client,omitempty"`
}

// UserSummary carries the basic user info that's useful for frontend rendering
// before (or without) a full profile existing.
type UserSummary struct {
	ID            string `json:"id" format:"uuid"`
	Email         string `json:"email" format:"email"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	EmailVerified bool   `json:"email_verified"`
	Status        string `json:"status" example:"active"`
	Points        int    `json:"points" example:"30"`
	Rank          string `json:"rank" example:"F"`
}

type MyIndividualProfile struct {
	PublicUrlSlug     string   `json:"public_url_slug" example:"john-doe"`
	Phone             *string  `json:"phone,omitempty" example:"+1234567890"`
	DateOfBirth       string   `json:"date_of_birth,omitempty" example:"1990-01-15"`
	Gender            string   `json:"gender,omitempty" example:"male"`
	AvatarURL         string   `json:"avatar_url,omitempty"`
	Bio               string   `json:"bio,omitempty"`
	Country           string   `json:"country,omitempty"`
	City              string   `json:"city,omitempty"`
	Headline          string   `json:"headline,omitempty"`
	PreferredLanguage string   `json:"preferred_language,omitempty"`
	TimeZone          string   `json:"timezone,omitempty"`
	CountryCode       string   `json:"country_code,omitempty"`
	ExperienceLevel   string   `json:"experience_level,omitempty"`
	Availability      string   `json:"availability,omitempty"`
	Skills            []string `json:"skills,omitempty"`
	ToolsTechnologies []string `json:"tools_technologies,omitempty"`
	ServiceCategories []string `json:"service_categories,omitempty"`
	PortfolioURL      string   `json:"portfolio_url,omitempty"`
	GithubURL         string   `json:"github_url,omitempty"`
	LinkedinURL       string   `json:"linkedin_url,omitempty"`
	ResumeURL         string   `json:"resume_url,omitempty"`
	ProfileVisibility string   `json:"profile_visibility,omitempty" example:"public"`
}

type MyAgencyProfile struct {
	PublicUrlSlug     string   `json:"public_url_slug" example:"acme-digital-agency"`
	AgencyName        string   `json:"agency_name" example:"Acme Digital Agency"`
	LogoURL           string   `json:"logo_url,omitempty"`
	Description       string   `json:"description,omitempty"`
	WebsiteURL        string   `json:"website_url,omitempty"`
	ServicesOffered   []string `json:"services_offered,omitempty"`
	Industries        []string `json:"industries,omitempty"`
	TeamSize          string   `json:"team_size,omitempty"`
	ContactName       string   `json:"contact_name,omitempty"`
	Phone             *string  `json:"phone,omitempty"`
	RegistrationNo    string   `json:"registration_no,omitempty"`
	Country           string   `json:"country,omitempty"`
	City              string   `json:"city,omitempty"`
	TimeZone          string   `json:"timezone,omitempty"`
	CountryCode       string   `json:"country_code,omitempty"`
	ProfileVisibility string   `json:"profile_visibility,omitempty" example:"public"`
}

type MyClientProfile struct {
	PublicUrlSlug     string   `json:"public_url_slug" example:"techstart-inc"`
	CompanyName       string   `json:"company_name" example:"TechStart Inc"`
	LogoURL           string   `json:"logo_url,omitempty"`
	Industry          string   `json:"industry,omitempty"`
	WebsiteURL        string   `json:"website_url,omitempty"`
	ProjectTypes      []string `json:"project_types,omitempty"`
	BudgetRange       string   `json:"budget_range,omitempty" example:"INR 5,00,000 - 20,00,000" doc:"Free-text INR budget band. Prefix the value with INR; amounts are in rupees (major units)."`
	TeamSize          string   `json:"team_size,omitempty"`
	ContactName       string   `json:"contact_name,omitempty"`
	Phone             *string  `json:"phone,omitempty"`
	Country           string   `json:"country,omitempty"`
	City              string   `json:"city,omitempty"`
	TimeZone          string   `json:"timezone,omitempty"`
	CountryCode       string   `json:"country_code,omitempty"`
	ProfileVisibility string   `json:"profile_visibility,omitempty" example:"public"`
}

// ============================================================
// UPLOAD/DELETE DTOs
// ============================================================

type UploadResumeForm struct {
	Resume huma.FormFile `form:"resume" required:"true"`
}
type UploadResumeInput struct {
	RawBody huma.MultipartFormFiles[UploadResumeForm]
}
type UploadResumeResponse struct {
	Message   string `json:"message"`
	ResumeURL string `json:"resume_url"`
}
type UploadResumeOutput struct {
	Body UploadResumeResponse
}

type UploadAvatarForm struct {
	Avatar huma.FormFile `form:"avatar" required:"true"`
}
type UploadAvatarInput struct {
	RawBody huma.MultipartFormFiles[UploadAvatarForm]
}
type UploadAvatarResponse struct {
	Message   string `json:"message"`
	AvatarURL string `json:"avatar_url"`
}
type UploadAvatarOutput struct {
	Body UploadAvatarResponse
}

type UploadLogoForm struct {
	Logo huma.FormFile `form:"logo" required:"true"`
}
type UploadLogoInput struct {
	RawBody huma.MultipartFormFiles[UploadLogoForm]
}
type UploadLogoResponse struct {
	Message string `json:"message"`
	LogoURL string `json:"logo_url"`
}
type UploadLogoOutput struct {
	Body UploadLogoResponse
}

type DeleteAvatarInput struct{}
type DeleteAvatarOutput struct {
	Body MessageResponse
}

type DeleteLogoInput struct{}
type DeleteLogoOutput struct {
	Body MessageResponse
}

type DeleteResumeInput struct{}
type DeleteResumeOutput struct {
	Body MessageResponse
}

// ============================================================
// QUERY INPUTs
// ============================================================

type GetPublicProfileInput struct {
	Slug string `path:"slug" minLength:"3" maxLength:"50" pattern:"^[a-z0-9-]+$" doc:"Profile slug"`
}
type GetPublicProfileOutput struct {
	Body GetPublicProfileResponse
}

type GetUserPointsInput struct{}
type GetUserPointsOutput struct {
	Body UserPointsResponse
}

type ExportProfileInput struct{}
type ExportProfileOutput struct {
	Body ExportResponse
}

type CheckSlugInput struct {
	Slug string `query:"slug" required:"true" minLength:"3" maxLength:"50" pattern:"^[a-z0-9-]+$" doc:"Profile slug"`
}
type CheckSlugOutput struct {
	Body CheckSlugResponse
}
