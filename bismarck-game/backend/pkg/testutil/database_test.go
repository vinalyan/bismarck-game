package testutil

import (
	"testing"
)

// TestLoadTestConfig тестирует загрузку конфигурации для тестов
func TestLoadTestConfig(t *testing.T) {
	cfg, err := loadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	if cfg.Database.Host == "" {
		t.Error("Database host should not be empty")
	}

	if cfg.Database.Port == 0 {
		t.Error("Database port should not be zero")
	}

	t.Logf("Loaded config: DB Host=%s, Port=%d, Name=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
}

// TestFindConfigFile тестирует поиск файла конфигурации
func TestFindConfigFile(t *testing.T) {
	configPath := findConfigFile()
	if configPath == "" {
		t.Log("No config file found, using default test config")
	} else {
		t.Logf("Found config file: %s", configPath)
	}
}
