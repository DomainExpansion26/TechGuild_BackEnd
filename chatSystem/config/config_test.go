package config

import (
	"testing"
)

func TestLoadConfig_EnvDiscovery(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	t.Logf("Loaded .env file from path: %s", cfg.EnvFilePath)
	t.Logf("App Name: %s", cfg.AppName)
	t.Logf("Port: %s", cfg.Port)
	t.Logf("Database URL set: %t", cfg.DatabaseURL != "")
	t.Logf("Redis Host: %s, Port: %s", cfg.RedisHost, cfg.RedisPort)
	t.Logf("JWT Access Secret set: %t", cfg.JWTAccessSecret != "")

	// Verify that the root .env file was discovered and loaded
	if cfg.EnvFilePath == "" {
		t.Error("expected .env file to be discovered and loaded, but EnvFilePath is empty")
	}

	if cfg.AppName == "" {
		t.Error("expected AppName to be populated from .env or defaults")
	}
}

