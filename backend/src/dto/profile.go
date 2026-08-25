package dto

import "github.com/danielgtaylor/huma/v2"

type CreateIndividualProfileRequest struct {
	Phone             *string `json:"phone" huma:"" example:"+1234567890"`
	DateOfBirth       *string `json:"date_of_birth" huma:"" example:"1990-01-15"`
	Gender            string  `json:"gender" huma:"" example:"male"`
	AvatarURL         string  `json:"avatar_url" huma:"" example:"https://storage.example.com/avatars/user.png"`
	Bio               string  `json:"bio" huma:"maxLength=500" example:"Full-stack developer with 5 years of experience"`
	Country           string  `json:"country" huma:"" example:"India"`
	State             string  `json:"state" huma:"" example:"Maharashtra"`
	City              string  `json:"city" huma:"" example:"Mumbai"`
	Headline          string  `json:"headline" huma:"maxLength=150" example:"Senior Full-Stack Developer"`
	PreferredLanguage string  `json:"preferred_language" huma:"" example:"en"`
	TimeZone          string  `json:"timezone" huma:"" example:"Asia/Kolkata"`
	CountryCode       string  `json:"country_code" huma:"" example:"+91"`

	ExperienceLevel   string   `json:"experience_level" huma:"" example:"senior"`
	Availability      string   `json:"availability" huma:"" example:"full-time"`
	Skills            []string `json:"skills" huma:""`
	ToolsTechnologies []string `json:"tools_technologies" huma:""`
	ServiceCategories []string `json:"service_categories" huma:""`

	PortfolioURL string `json:"portfolio_url" huma:"" example:"https://portfolio.example.com/johndoe"`
	GithubURL    string `json:"github_url" huma:"" example:"https://github.com/johndoe"`
	LinkedinURL  string `json:"linkedin_url" huma:"" example:"https://linkedin.com/in/johndoe"`
	ResumeURL    string `json:"resume_url" huma:"" example:"https://storage.example.com/resumes/johndoe.pdf"`

	TermsConfirmed    bool   `json:"terms_confirmed" huma:"" example:"true"`
	ProfileVisibility string `json:"profile_visibility" huma:"" example:"public"`
}

type CreateAgencyProfileRequest struct {
	AgencyName      string   `json:"agency_name" huma:"required" example:"Acme Digital Agency"`
	LogoURL         string   `json:"logo_url" huma:"" example:"https://storage.example.com/logos/acme.png"`
	Description     string   `json:"description" huma:"maxLength=1000" example:"Digital agency specializing in web and mobile development"`
	WebsiteURL      string   `json:"website_url" huma:"" example:"https://acme.example.com"`
	ServicesOffered []string `json:"services_offered" huma:""`
	Industries      []string `json:"industries" huma:""`
	TeamSize        string   `json:"team_size" huma:"" example:"10-50"`

	ContactName    string  `json:"contact_name" huma:"" example:"Jane Smith"`
	Phone          *string `json:"phone" huma:"" example:"+1234567890"`
	RegistrationNo string  `json:"registration_no" huma:"" example:"U74110MH2020PTC123456"`
	Country        string  `json:"country" huma:"" example:"India"`
	State          string  `json:"state" huma:"" example:"Maharashtra"`
	City           string  `json:"city" huma:"" example:"Mumbai"`
	TimeZone       string  `json:"timezone" huma:"" example:"Asia/Kolkata"`
	CountryCode    string  `json:"country_code" huma:"" example:"+91"`
}

type CreateClientProfileRequest struct {
	CompanyName  string   `json:"company_name" huma:"required" example:"TechStart Inc"`
	LogoURL      string   `json:"logo_url" huma:"" example:"https://storage.example.com/logos/techstart.png"`
	Industry     string   `json:"industry" huma:"" example:"technology"`
	WebsiteURL   string   `json:"website_url" huma:"" example:"https://techstart.example.com"`
	ProjectTypes []string `json:"project_types" huma:""`
	BudgetRange  string   `json:"budget_range" huma:"" example:"$5,000 - $20,000"`
	TeamSize     string   `json:"team_size" huma:"" example:"5-10"`

	ContactName string  `json:"contact_name" huma:"" example:"Alice Johnson"`
	Phone       *string `json:"phone" huma:"" example:"+1234567890"`
	Country     string  `json:"country" huma:"" example:"United States"`
	State       string  `json:"state" huma:"" example:"California"`
	City        string  `json:"city" huma:"" example:"San Francisco"`
	TimeZone    string  `json:"timezone" huma:"" example:"America/Los_Angeles"`
	CountryCode string  `json:"country_code" huma:"" example:"+1"`
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

type PublicProfileResponse struct {
	FirstName   string `json:"first_name" example:"John"`
	LastName    string `json:"last_name" example:"Doe"`
	AccountType string `json:"account_type" example:"individual"`
	Points      int    `json:"points" example:"150"`

	Phone             string   `json:"phone,omitempty" example:"+1234567890"`
	DateOfBirth       string   `json:"date_of_birth,omitempty" example:"1990-01-15"`
	Gender            string   `json:"gender,omitempty" example:"male"`
	AvatarURL         string   `json:"avatar_url,omitempty" example:"https://storage.example.com/avatars/user.png"`
	Bio               string   `json:"bio,omitempty" example:"Full-stack developer with 5 years of experience"`
	Country           string   `json:"country,omitempty" example:"India"`
	State             string   `json:"state,omitempty" example:"Maharashtra"`
	City              string   `json:"city,omitempty" example:"Mumbai"`
	Headline          string   `json:"headline,omitempty" example:"Senior Full-Stack Developer"`
	PreferredLanguage string   `json:"preferred_language,omitempty" example:"en"`
	TimeZone          string   `json:"timezone,omitempty" example:"Asia/Kolkata"`
	CountryCode       string   `json:"country_code,omitempty" example:"+91"`
	PublicUrlSlug     string   `json:"public_url_slug" example:"john-doe"`
	ExperienceLevel   string   `json:"experience_level,omitempty" example:"senior"`
	Availability      string   `json:"availability,omitempty" example:"full-time"`
	Skills            []string `json:"skills,omitempty"`
	ToolsTechnologies []string `json:"tools_technologies,omitempty"`
	ServiceCategories []string `json:"service_categories,omitempty"`
	PortfolioURL      string   `json:"portfolio_url,omitempty" example:"https://portfolio.example.com/johndoe"`
	GithubURL         string   `json:"github_url,omitempty" example:"https://github.com/johndoe"`
	LinkedinURL       string   `json:"linkedin_url,omitempty" example:"https://linkedin.com/in/johndoe"`
	ResumeURL         string   `json:"resume_url,omitempty" example:"https://storage.example.com/resumes/johndoe.pdf"`
	ProfileVisibility string   `json:"profile_visibility,omitempty" example:"public"`
	MemberSince       string   `json:"member_since" example:"2026-01-15T00:00:00Z"`
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

type CreateIndividualProfileInput struct {
	Body CreateIndividualProfileRequest
}
type CreateIndividualProfileOutput struct {
	Body CreateProfileResponse
}

type CreateAgencyProfileInput struct {
	Body CreateAgencyProfileRequest
}
type CreateAgencyProfileOutput struct {
	Body CreateProfileResponse
}

type CreateClientProfileInput struct {
	Body CreateClientProfileRequest
}
type CreateClientProfileOutput struct {
	Body CreateProfileResponse
}

type GetMyProfileInput struct{}
type GetMyProfileOutput struct {
	Body any
}

type SetAccountTypeInput struct {
	Body SetAccountTypeRequest
}
type SetAccountTypeOutput struct {
	Body CreateProfileResponse
}

type UploadResumeForm struct {
	Resume huma.FormFile `form:"resume" contentType:"application/pdf" required:"true"`
}
type UploadResumeInput struct {
	RawBody huma.MultipartFormFiles[UploadResumeForm]
}
type UploadResumeOutput struct {
	Body struct {
		Message   string `json:"message"`
		ResumeURL string `json:"resume_url"`
	}
}

type UploadAvatarForm struct {
	Avatar huma.FormFile `form:"avatar" required:"true"`
}
type UploadAvatarInput struct {
	RawBody huma.MultipartFormFiles[UploadAvatarForm]
}
type UploadAvatarOutput struct {
	Body struct {
		Message   string `json:"message"`
		AvatarURL string `json:"avatar_url"`
	}
}

type UploadLogoForm struct {
	Logo huma.FormFile `form:"logo" required:"true"`
}
type UploadLogoInput struct {
	RawBody huma.MultipartFormFiles[UploadLogoForm]
}
type UploadLogoOutput struct {
	Body struct {
		Message string `json:"message"`
		LogoURL string `json:"logo_url"`
	}
}

type DeleteAvatarInput struct{}
type DeleteAvatarOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

type DeleteLogoInput struct{}
type DeleteLogoOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

type DeleteResumeInput struct{}
type DeleteResumeOutput struct {
	Body struct {
		Message string `json:"message"`
	}
}

type GetPublicProfileInput struct {
	Slug string `path:"slug" doc:"Profile slug"`
}
type GetPublicProfileOutput struct {
	Body PublicProfileResponse
}

type GetUserPointsInput struct{}
type GetUserPointsOutput struct {
	Body UserPointsResponse
}

type ExportProfileInput struct{}
type ExportProfileOutput struct {
	Body ExportResponse
}

type CreateOrUpdateProfileForm struct {
	ProfileData string `form:"profile_data"`
}
type CreateOrUpdateProfileInput struct {
	RawBody huma.MultipartFormFiles[CreateOrUpdateProfileForm]
}
type CreateOrUpdateProfileOutput struct {
	Body CreateProfileResponse
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
	Body struct {
		Message string `json:"message"`
	}
}
