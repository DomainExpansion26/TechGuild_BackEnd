package controllers

import (
	"context"
	"errors"

	"techguild-backend/src/config"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"

	"github.com/danielgtaylor/huma/v2"
)

type LocationController struct {
	locationService *services.LocationService
}

func NewLocationController(cfg *config.Config) *LocationController {
	return &LocationController{locationService: services.NewLocationService(cfg)}
}

func (c *LocationController) GetCountries(ctx context.Context, _ *dto.LocationCountriesInput) (*dto.LocationCountriesOutput, error) {
	countries, err := c.locationService.Countries(ctx)
	if err != nil {
		if errors.Is(err, services.ErrLocationProviderNotConfigured) {
			return nil, huma.Error503ServiceUnavailable(err.Error())
		}
		return nil, huma.Error502BadGateway("Location provider request failed")
	}
	return &dto.LocationCountriesOutput{
		Body: dto.LocationCountriesResponse{Countries: countries},
	}, nil
}

func (c *LocationController) GetCities(ctx context.Context, input *dto.LocationCitiesInput) (*dto.LocationCitiesOutput, error) {
	cities, err := c.locationService.Cities(ctx, input.CountryCode, input.Query, input.Limit)
	if err != nil {
		if errors.Is(err, services.ErrInvalidLocationQuery) {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if errors.Is(err, services.ErrLocationProviderNotConfigured) {
			return nil, huma.Error503ServiceUnavailable(err.Error())
		}
		return nil, huma.Error502BadGateway("Location provider request failed")
	}
	return &dto.LocationCitiesOutput{
		Body: dto.LocationCitiesResponse{Cities: cities},
	}, nil
}
