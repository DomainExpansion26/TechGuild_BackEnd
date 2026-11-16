package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"techguild-backend/src/config"
	"techguild-backend/src/dto"
)

var (
	ErrLocationProviderNotConfigured = errors.New("GeoNames username is not configured")
	ErrInvalidLocationQuery          = errors.New("invalid location query")
)

const (
	geoNamesBaseURL = "https://api.geonames.org"
	maxCityResults  = 10
)

var countryCodePattern = regexp.MustCompile(`^[A-Za-z]{2}$`)

type LocationService struct {
	username string
	baseURL  string
	client   *http.Client
}

func NewLocationService(cfg *config.Config) *LocationService {
	return &LocationService{
		username: strings.TrimSpace(cfg.GeoNamesUsername),
		baseURL:  geoNamesBaseURL,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *LocationService) Countries(ctx context.Context) ([]dto.LocationCountry, error) {
	if err := s.checkConfigured(); err != nil {
		return nil, err
	}

	endpoint, err := url.Parse(s.baseURL + "/countryInfoJSON")
	if err != nil {
		return nil, fmt.Errorf("build GeoNames country request: %w", err)
	}
	query := endpoint.Query()
	query.Set("username", s.username)
	endpoint.RawQuery = query.Encode()

	var response struct {
		Countries []struct {
			Code string `json:"countryCode"`
			Name string `json:"countryName"`
		} `json:"geonames"`
	}
	if err := s.getJSON(ctx, endpoint.String(), &response); err != nil {
		return nil, err
	}

	countries := make([]dto.LocationCountry, 0, len(response.Countries))
	for _, country := range response.Countries {
		if country.Code == "" || country.Name == "" {
			continue
		}
		countries = append(countries, dto.LocationCountry{Code: country.Code, Name: country.Name})
	}
	return countries, nil
}

func (s *LocationService) Cities(ctx context.Context, countryCode, prefix string, limit int) ([]dto.LocationCity, error) {
	if err := s.checkConfigured(); err != nil {
		return nil, err
	}
	countryCode = strings.TrimSpace(countryCode)
	prefix = strings.TrimSpace(prefix)
	if !countryCodePattern.MatchString(countryCode) || utf8.RuneCountInString(prefix) < 2 ||
		utf8.RuneCountInString(prefix) > 100 || limit < 0 || limit > maxCityResults {
		return nil, ErrInvalidLocationQuery
	}
	if limit == 0 {
		limit = maxCityResults
	}

	endpoint, err := url.Parse(s.baseURL + "/searchJSON")
	if err != nil {
		return nil, fmt.Errorf("build GeoNames city request: %w", err)
	}
	query := endpoint.Query()
	query.Set("username", s.username)
	query.Set("country", strings.ToUpper(countryCode))
	query.Set("name_startsWith", prefix)
	query.Set("featureClass", "P")
	query.Set("orderby", "population")
	query.Set("maxRows", fmt.Sprint(limit))
	endpoint.RawQuery = query.Encode()

	var response struct {
		Cities []struct {
			Name        string `json:"name"`
			CountryCode string `json:"countryCode"`
			CountryName string `json:"countryName"`
			Region      string `json:"adminName1"`
		} `json:"geonames"`
	}
	if err := s.getJSON(ctx, endpoint.String(), &response); err != nil {
		return nil, err
	}

	cities := make([]dto.LocationCity, 0, len(response.Cities))
	for _, city := range response.Cities {
		if city.Name == "" {
			continue
		}
		cities = append(cities, dto.LocationCity{
			Name:        city.Name,
			CountryCode: city.CountryCode,
			CountryName: city.CountryName,
			Region:      city.Region,
		})
	}
	return cities, nil
}

func (s *LocationService) checkConfigured() error {
	if s.username == "" {
		return ErrLocationProviderNotConfigured
	}
	return nil
}

func (s *LocationService) getJSON(ctx context.Context, requestURL string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("create GeoNames request: %w", err)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("request GeoNames: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("GeoNames returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return fmt.Errorf("read GeoNames response: %w", err)
	}

	var providerError struct {
		Status *struct {
			Message string `json:"message"`
			Value   int    `json:"value"`
		} `json:"status"`
	}
	if err := json.Unmarshal(body, &providerError); err != nil {
		return fmt.Errorf("decode GeoNames response: %w", err)
	}
	if providerError.Status != nil {
		return fmt.Errorf("GeoNames error %d: %s", providerError.Status.Value, providerError.Status.Message)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode GeoNames response: %w", err)
	}
	return nil
}
