package routes

import (
	"techguild-backend/src/controllers"
	"techguild-backend/src/middleware"

	"github.com/danielgtaylor/huma/v2"
)

func RegisterLocationRoutes(api huma.API, c *controllers.LocationController) {
	authMw := huma.Middlewares{middleware.AuthMiddlewareHuma(api)}

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "get-location-countries",
		Method:      "GET",
		Path:        "/v1/locations/countries",
		Tags:        []string{"Locations"},
		Summary:     "List countries for profile location input",
		Middlewares: authMw,
	}, c.GetCountries)

	huma.Register(api, huma.Operation{
		Security:    []map[string][]string{{"bearerAuth": {}}},
		OperationID: "search-location-cities",
		Method:      "GET",
		Path:        "/v1/locations/cities",
		Tags:        []string{"Locations"},
		Summary:     "Search cities within a country",
		Middlewares: authMw,
	}, c.GetCities)
}
