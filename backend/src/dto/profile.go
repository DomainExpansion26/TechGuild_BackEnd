package dto

import (
	"encoding/json"

	"github.com/danielgtaylor/huma/v2"
)

type CreateIndividualProfileRequest struct {
	Phone             *string `json:"phone" huma:"" example:"+1234567890"`
	DateOfBirth       *string `json:"date_of_birth" huma:"" example:"1990-01-15"`
	Gender            *string `json:"gender" huma:"" example:"male"`
	AvatarURL         *string `json:"avatar_url" huma:"" example:"https://storage.example.com/avatars/user.png"`
	Bio               *string `json:"bio" huma:"maxLength=500" example:"Full-stack developer with 5 years of experience"`
	Country           *string `json:"country" huma:"" example:"India"`
	City              *string `json:"city" huma:"" example:"Mumbai"`
	Headline          *string `json:"headline" huma:"maxLength=150" example:"Senior Full-Stack Developer"`
	PreferredLanguage *string `json:"preferred_language" huma:"" example:"en"`
	TimeZone          *string `json:"timezone" huma:"" example:"Asia/Kolkata"`

	ExperienceLevel   *string   `json:"experience_level" huma:"" example:"senior"`
	Availability      *string   `json:"availability" huma:"" example:"full-time"`
	Skills            *[]string `json:"skills" huma:""`
	ToolsTechnologies *[]string `json:"tools_technologies" huma:""`
	ServiceCategories *[]string `json:"service_categories" huma:""`

	PortfolioURL *string `json:"portfolio_url" huma:"" example:"https://portfolio.example.com/johndoe"`
	GithubURL    *string `json:"github_url" huma:"" example:"https://github.com/johndoe"`
	LinkedinURL  *string `json:"linkedin_url" huma:"" example:"https://linkedin.com/in/johndoe"`
	ResumeURL    *string `json:"resume_url" huma:"" example:"https://storage.example.com/resumes/johndoe.pdf"`

	TermsConfirmed *bool `json:"terms_confirmed" huma:"" example:"true"`
}

type UpdateIndividualProfileRequest struct {
	DateOfBirth       *string `json:"date_of_birth" huma:"" example:"1990-01-15"`
	Gender            *string `json:"gender" huma:"" example:"male"`
	AvatarURL         *string `json:"avatar_url" huma:"" example:"https://storage.example.com/avatars/user.png"`
	Bio               *string `json:"bio" huma:"maxLength=500" example:"Full-stack developer with 5 years of experience"`
	Country           *string `json:"country" huma:"" example:"India"`
	City              *string `json:"city" huma:"" example:"Mumbai"`
	Headline          *string `json:"headline" huma:"maxLength=150" example:"Senior Full-Stack Developer"`
	PreferredLanguage *string `json:"preferred_language" huma:"" example:"en"`
	TimeZone          *string `json:"timezone" huma:"" example:"Asia/Kolkata"`

	ExperienceLevel   *string   `json:"experience_level" huma:"" example:"senior"`
	Availability      *string   `json:"availability" huma:"" example:"full-time"`
	Skills            *[]string `json:"skills" huma:""`
	ToolsTechnologies *[]string `json:"tools_technologies" huma:""`
	ServiceCategories *[]string `json:"service_categories" huma:""`

	PortfolioURL *string `json:"portfolio_url" huma:"" example:"https://portfolio.example.com/johndoe"`
	GithubURL    *string `json:"github_url" huma:"" example:"https://github.com/johndoe"`
	LinkedinURL  *string `json:"linkedin_url" huma:"" example:"https://linkedin.com/in/johndoe"`
	ResumeURL    *string `json:"resume_url" huma:"" example:"https://storage.example.com/resumes/johndoe.pdf"`

	TermsConfirmed *bool `json:"terms_confirmed" huma:"" example:"true"`
}

type CreateAgencyProfileRequest struct {
	Phone           *string   `json:"phone" huma:"" example:"+919876543210"`
	AgencyName      string    `json:"agency_name" huma:"required" example:"Acme Digital Agency"`
	LogoURL         *string   `json:"logo_url" huma:"" example:"https://storage.example.com/logos/acme.png"`
	Description     *string   `json:"description" huma:"maxLength=1000" example:"Digital agency specializing in web and mobile development"`
	WebsiteURL      *string   `json:"website_url" huma:"" example:"https://acme.example.com"`
	ServicesOffered *[]string `json:"services_offered" huma:""`
	Industries      *[]string `json:"industries" huma:""`
	TeamSize        *string   `json:"team_size" huma:"" example:"10-50"`

	ContactName *string `json:"contact_name" huma:"" example:"Jane Smith"`
	Country     *string `json:"country" huma:"" example:"India"`
	City        *string `json:"city" huma:"" example:"Mumbai"`
	TimeZone    *string `json:"timezone" huma:"" example:"Asia/Kolkata"`
}

type UpdateAgencyProfileRequest struct {
	AgencyName      *string   `json:"agency_name" huma:"" example:"Acme Digital Agency"`
	LogoURL         *string   `json:"logo_url" huma:"" example:"https://storage.example.com/logos/acme.png"`
	Description     *string   `json:"description" huma:"maxLength=1000" example:"Digital agency specializing in web and mobile development"`
	WebsiteURL      *string   `json:"website_url" huma:"" example:"https://acme.example.com"`
	ServicesOffered *[]string `json:"services_offered" huma:""`
	Industries      *[]string `json:"industries" huma:""`
	TeamSize        *string   `json:"team_size" huma:"" example:"10-50"`

	ContactName *string `json:"contact_name" huma:"" example:"Jane Smith"`
	Country     *string `json:"country" huma:"" example:"India"`
	City        *string `json:"city" huma:"" example:"Mumbai"`
	TimeZone    *string `json:"timezone" huma:"" example:"Asia/Kolkata"`
}

type CreateClientProfileRequest struct {
	Phone        *string   `json:"phone" huma:"" example:"+919876543210"`
	CompanyName  string    `json:"company_name" huma:"required" example:"TechStart Inc"`
	LogoURL      *string   `json:"logo_url" huma:"" example:"https://storage.example.com/logos/techstart.png"`
	Industry     *string   `json:"industry" huma:"" example:"technology"`
	WebsiteURL   *string   `json:"website_url" huma:"" example:"https://techstart.example.com"`
	ProjectTypes *[]string `json:"project_types" huma:""`
	BudgetRange  *string   `json:"budget_range" huma:"" example:"$5,000 - $20,000"`
	TeamSize     *string   `json:"team_size" huma:"" example:"5-10"`

	Country  *string `json:"country" huma:"" example:"United States"`
	City     *string `json:"city" huma:"" example:"San Francisco"`
	TimeZone *string `json:"timezone" huma:"" example:"America/Los_Angeles"`
}

type UpdateClientProfileRequest struct {
	CompanyName  *string   `json:"company_name" huma:"" example:"TechStart Inc"`
	LogoURL      *string   `json:"logo_url" huma:"" example:"https://storage.example.com/logos/techstart.png"`
	Industry     *string   `json:"industry" huma:"" example:"technology"`
	WebsiteURL   *string   `json:"website_url" huma:"" example:"https://techstart.example.com"`
	ProjectTypes *[]string `json:"project_types" huma:""`
	BudgetRange  *string   `json:"budget_range" huma:"" example:"$5,000 - $20,000"`
	TeamSize     *string   `json:"team_size" huma:"" example:"5-10"`

	Country  *string `json:"country" huma:"" example:"United States"`
	City     *string `json:"city" huma:"" example:"San Francisco"`
	TimeZone *string `json:"timezone" huma:"" example:"America/Los_Angeles"`
}

type CreateProfileResponse struct {
	Message       string `json:"message" example:"Profile created successfully"`
	PublicUrlSlug string `json:"public_url_slug" example:"john-doe"`
}

type SetAccountTypeRequest struct {
	Email       string `json:"email" huma:"required,email" example:"test@example.com"`
	Password    string `json:"password" huma:"required" example:"test@123"`
	AccountType string `json:"account_type" huma:"required,enum=individual;agency;client" example:"individual"`
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

// same pattern as GetMyProfileResponse.
type GetPublicProfileResponse struct {
	AccountType string                   `json:"account_type" example:"individual"`
	Individual  *PublicIndividualProfile `json:"individual,omitempty"`
	Agency      *PublicAgencyProfile     `json:"agency,omitempty"`
	Client      *PublicClientProfile     `json:"client,omitempty"`
}

type UserPointsResponse struct {
	Points          int    `json:"points" example:"150"`
	AccountType     string `json:"account_type" example:"individual"`
	ProfileComplete bool   `json:"profile_complete" example:"true"`
}

type ExportResponse struct {
	Message     string `json:"message" example:"Export started"`
	DownloadURL string `json:"download_url" example:"https://storage.example.com/exports/user-data.zip"`
	ExpiresIn   string `json:"expires_in" example:"2026-01-16T00:00:00Z"`
}

type CheckSlugResponse struct {
	Available    bool     `json:"available" example:"true"`
	Alternatives []string `json:"alternatives,omitempty"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" huma:"required" example:"test@123"`
}

// ---------- Huma Input/Output wrapper structs ----------

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
	AccountType string               `json:"account_type" example:"individual"`
	Individual  *MyIndividualProfile `json:"individual,omitempty"`
	Agency      *MyAgencyProfile     `json:"agency,omitempty"`
	Client      *MyClientProfile     `json:"client,omitempty"`
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
	BudgetRange       string   `json:"budget_range,omitempty"`
	TeamSize          string   `json:"team_size,omitempty"`
	ContactName       string   `json:"contact_name,omitempty"`
	Phone             *string  `json:"phone,omitempty"`
	Country           string   `json:"country,omitempty"`
	City              string   `json:"city,omitempty"`
	TimeZone          string   `json:"timezone,omitempty"`
	CountryCode       string   `json:"country_code,omitempty"`
	ProfileVisibility string   `json:"profile_visibility,omitempty" example:"public"`
}

type SetAccountTypeInput struct {
	Body SetAccountTypeRequest
}

// SetAccountTypeResponse mirrors the create-profile message but without an
// implied public slug.
type SetAccountTypeResponse struct {
	Message string `json:"message" example:"account type set successfully"`
}
type SetAccountTypeOutput struct {
	Body SetAccountTypeResponse
}

type UploadResumeForm struct {
	Resume huma.FormFile `form:"resume" contentType:"application/pdf" required:"true"`
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

type GetPublicProfileInput struct {
	Slug string `path:"slug" doc:"Profile slug"`
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
	Slug string `query:"slug" required:"true" doc:"Profile slug"`
}
type CheckSlugOutput struct {
	Body CheckSlugResponse
}

type DeleteProfileAccountInput struct {
	Body DeleteAccountRequest
}
type DeleteProfileAccountOutput struct {
	Body MessageResponse
}

// DeprecatedLegacyProfileRequest is the old unified profile request shape.
type DeprecatedLegacyProfileRequest struct {
	AccountType string          `json:"account_type" huma:"required" example:"individual"`
	ProfileData json.RawMessage `json:"profile_data"`
}
type DeprecatedProfileCreateInput struct {
	Body DeprecatedLegacyProfileRequest
}
type DeprecatedProfileCreateOutput struct {
	Body CreateProfileResponse
}
