package dto

type CreateProfileRequest struct {
	Headline          string `json:"headline" binding:"required,max=150"`
	Bio               string `json:"bio" binding:"max=500"`
	PreferredLanguage string `json:"preferred_language" binding:"required"`
	TimeZone          string `json:"timezone" binding:"required"`
	CountryCode       string `json:"country_code" binding:"required,len=2"`
}

type CreateProfileResponse struct {
	Message       string `json:"message"`
	PublicUrlSlug string `json:"public_url_slug"`
}
