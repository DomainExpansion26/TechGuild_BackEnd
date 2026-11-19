package dto

type LocationCountriesResponse struct {
	Countries []LocationCountry `json:"countries"`
}

type LocationCountry struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type LocationCitiesResponse struct {
	Cities []LocationCity `json:"cities"`
}

type LocationCity struct {
	Name        string `json:"name"`
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	Region      string `json:"region,omitempty"`
}

type LocationCountriesInput struct{}

type LocationCountriesOutput struct {
	Body LocationCountriesResponse
}

type LocationCitiesInput struct {
	CountryCode string `query:"country" required:"true" doc:"Two-letter ISO country code"`
	Query       string `query:"q" required:"true" doc:"City name prefix (at least 2 characters)"`
	Limit       int    `query:"limit" doc:"Maximum results (1-10, defaults to 10)"`
}

type LocationCitiesOutput struct {
	Body LocationCitiesResponse
}
