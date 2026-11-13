package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"techguild-backend/src/config"
)

func TestLocationServiceCountries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/countryInfoJSON" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("username") != "test-user" {
			t.Fatalf("unexpected username query: %q", r.URL.Query().Get("username"))
		}
		_, _ = w.Write([]byte(`{"geonames":[{"countryCode":"IN","countryName":"India"},{"countryCode":"","countryName":"invalid"}]}`))
	}))
	defer server.Close()

	service := newTestLocationService(server.URL)
	countries, err := service.Countries(context.Background())
	if err != nil {
		t.Fatalf("Countries() error = %v", err)
	}
	if len(countries) != 1 || countries[0].Code != "IN" || countries[0].Name != "India" {
		t.Fatalf("Countries() = %#v", countries)
	}
}

func TestLocationServiceCities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if r.URL.Path != "/searchJSON" || query.Get("country") != "IN" ||
			query.Get("name_startsWith") != "Mum" || query.Get("maxRows") != "3" ||
			query.Get("featureClass") != "P" {
			t.Fatalf("unexpected GeoNames request: %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"geonames":[{"name":"Mumbai","countryCode":"IN","countryName":"India","adminName1":"Maharashtra"}]}`))
	}))
	defer server.Close()

	service := newTestLocationService(server.URL)
	cities, err := service.Cities(context.Background(), "in", " Mum ", 3)
	if err != nil {
		t.Fatalf("Cities() error = %v", err)
	}
	if len(cities) != 1 || cities[0].Name != "Mumbai" || cities[0].CountryCode != "IN" ||
		cities[0].CountryName != "India" || cities[0].Region != "Maharashtra" {
		t.Fatalf("Cities() = %#v", cities)
	}
}

func TestLocationServiceRejectsInvalidCityQuery(t *testing.T) {
	service := newTestLocationService("http://unused")
	_, err := service.Cities(context.Background(), "IND", "M", 11)
	if err != ErrInvalidLocationQuery {
		t.Fatalf("Cities() error = %v, want %v", err, ErrInvalidLocationQuery)
	}
}

func TestLocationServiceReportsGeoNamesErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":{"message":"user name not found","value":10}}`))
	}))
	defer server.Close()

	service := newTestLocationService(server.URL)
	_, err := service.Countries(context.Background())
	if err == nil {
		t.Fatal("Countries() error = nil, want GeoNames error")
	}
}

func TestLocationServiceRequiresUsername(t *testing.T) {
	service := NewLocationService(&config.Config{})
	_, err := service.Countries(context.Background())
	if err != ErrLocationProviderNotConfigured {
		t.Fatalf("Countries() error = %v, want %v", err, ErrLocationProviderNotConfigured)
	}
}

func newTestLocationService(baseURL string) *LocationService {
	return &LocationService{
		username: "test-user",
		baseURL:  baseURL,
		client:   http.DefaultClient,
	}
}
