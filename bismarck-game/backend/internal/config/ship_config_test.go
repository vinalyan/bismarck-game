package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadShipConfigs(t *testing.T) {
	// Создаем временный файл конфигурации кораблей
	shipConfigContent := `{
		"ships": [
			{
				"id": "bismarck",
				"name": "BISMARCK",
				"type": "BB",
				"side": "german",
				"speedType": "fast",
				"maxFuel": 18,
				"baseEvasion": 30,
				"hullBoxes": 15,
				"basePrimaryArmamentBow": 8,
				"basePrimaryArmamentStern": 8,
				"baseSecondaryArmament": 12,
				"maxTorpedos": 0,
				"radarLevel": 2
			},
			{
				"id": "prince_of_wales",
				"name": "P. OF WALES",
				"type": "BB",
				"side": "allied",
				"speedType": "fast",
				"maxFuel": 16,
				"baseEvasion": 25,
				"hullBoxes": 12,
				"basePrimaryArmamentBow": 10,
				"basePrimaryArmamentStern": 10,
				"baseSecondaryArmament": 16,
				"maxTorpedos": 0,
				"radarLevel": 3
			}
		]
	}`

	tmpFile, err := os.CreateTemp("", "test-ships-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(shipConfigContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Создаем менеджер и загружаем конфигурацию
	manager := NewShipConfigManager()
	err = manager.LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load ship configs: %v", err)
	}

	// Получаем все корабли
	configs, err := manager.GetAllShips()
	if err != nil {
		t.Fatalf("Failed to get all ships: %v", err)
	}

	// Проверяем количество кораблей
	if len(configs) != 2 {
		t.Errorf("Expected 2 ships, got %d", len(configs))
	}

	// Проверяем первый корабль
	bismarck := configs[0]
	if bismarck.ID != "bismarck" {
		t.Errorf("Expected ID bismarck, got %s", bismarck.ID)
	}
	if bismarck.Name != "BISMARCK" {
		t.Errorf("Expected name BISMARCK, got %s", bismarck.Name)
	}
	if bismarck.Type != "BB" {
		t.Errorf("Expected type BB, got %s", bismarck.Type)
	}
	if bismarck.Side != "german" {
		t.Errorf("Expected side german, got %s", bismarck.Side)
	}
	if bismarck.SpeedType != "fast" {
		t.Errorf("Expected speed type fast, got %s", bismarck.SpeedType)
	}
	if bismarck.MaxFuel != 18 {
		t.Errorf("Expected max fuel 18, got %d", bismarck.MaxFuel)
	}
	if bismarck.BaseEvasion != 30 {
		t.Errorf("Expected base evasion 30, got %d", bismarck.BaseEvasion)
	}
	if bismarck.HullBoxes != 15 {
		t.Errorf("Expected hull boxes 15, got %d", bismarck.HullBoxes)
	}

	// Проверяем второй корабль
	pow := configs[1]
	if pow.ID != "prince_of_wales" {
		t.Errorf("Expected ID prince_of_wales, got %s", pow.ID)
	}
	if pow.Name != "P. OF WALES" {
		t.Errorf("Expected name P. OF WALES, got %s", pow.Name)
	}
	if pow.Side != "allied" {
		t.Errorf("Expected side allied, got %s", pow.Side)
	}
}

func TestLoadShipConfigsFileNotFound(t *testing.T) {
	manager := NewShipConfigManager()
	err := manager.LoadConfig("nonexistent-file.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestLoadShipConfigsInvalidJSON(t *testing.T) {
	// Создаем файл с невалидным JSON
	tmpFile, err := os.CreateTemp("", "test-invalid-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("invalid json content"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	manager := NewShipConfigManager()
	err = manager.LoadConfig(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestGetShipConfig(t *testing.T) {
	// Создаем менеджер и загружаем тестовые данные
	manager := NewShipConfigManager()
	
	// Создаем временный файл с тестовыми данными
	shipConfigContent := `{
		"ships": [
			{
				"id": "bismarck",
				"name": "BISMARCK",
				"type": "BB",
				"side": "german",
				"speedType": "fast",
				"maxFuel": 18,
				"baseEvasion": 30,
				"hullBoxes": 15,
				"basePrimaryArmamentBow": 8,
				"basePrimaryArmamentStern": 8,
				"baseSecondaryArmament": 12,
				"maxTorpedos": 0,
				"radarLevel": 2
			},
			{
				"id": "prince_of_wales",
				"name": "P. OF WALES",
				"type": "BB",
				"side": "allied",
				"speedType": "fast",
				"maxFuel": 16,
				"baseEvasion": 25,
				"hullBoxes": 12,
				"basePrimaryArmamentBow": 10,
				"basePrimaryArmamentStern": 10,
				"baseSecondaryArmament": 16,
				"maxTorpedos": 0,
				"radarLevel": 3
			}
		]
	}`

	tmpFile, err := os.CreateTemp("", "test-ships-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(shipConfigContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	err = manager.LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Тест существующего корабля
	ship, err := manager.GetShipConfig("bismarck")
	if err != nil {
		t.Errorf("Expected to find bismarck, got error: %v", err)
	}
	if ship.Name != "BISMARCK" {
		t.Errorf("Expected name BISMARCK, got %s", ship.Name)
	}

	// Тест несуществующего корабля
	_, err = manager.GetShipConfig("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent ship")
	}
}

func TestGetShipsBySide(t *testing.T) {
	manager := NewShipConfigManager()
	
	// Создаем временный файл с тестовыми данными
	shipConfigContent := `{
		"ships": [
			{
				"id": "bismarck",
				"name": "BISMARCK",
				"type": "BB",
				"side": "german",
				"speedType": "fast",
				"maxFuel": 18,
				"baseEvasion": 30,
				"hullBoxes": 15,
				"basePrimaryArmamentBow": 8,
				"basePrimaryArmamentStern": 8,
				"baseSecondaryArmament": 12,
				"maxTorpedos": 0,
				"radarLevel": 2
			},
			{
				"id": "prince_of_wales",
				"name": "P. OF WALES",
				"type": "BB",
				"side": "allied",
				"speedType": "fast",
				"maxFuel": 16,
				"baseEvasion": 25,
				"hullBoxes": 12,
				"basePrimaryArmamentBow": 10,
				"basePrimaryArmamentStern": 10,
				"baseSecondaryArmament": 16,
				"maxTorpedos": 0,
				"radarLevel": 3
			}
		]
	}`

	tmpFile, err := os.CreateTemp("", "test-ships-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(shipConfigContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	err = manager.LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Тест получения немецких кораблей
	germanShips, err := manager.GetShipsBySide("german")
	if err != nil {
		t.Fatalf("Failed to get German ships: %v", err)
	}
	if len(germanShips) != 1 {
		t.Errorf("Expected 1 German ship, got %d", len(germanShips))
	}
	if germanShips[0].Name != "BISMARCK" {
		t.Errorf("Expected BISMARCK, got %s", germanShips[0].Name)
	}

	// Тест получения союзных кораблей
	alliedShips, err := manager.GetShipsBySide("allied")
	if err != nil {
		t.Fatalf("Failed to get Allied ships: %v", err)
	}
	if len(alliedShips) != 1 {
		t.Errorf("Expected 1 Allied ship, got %d", len(alliedShips))
	}
	if alliedShips[0].Name != "P. OF WALES" {
		t.Errorf("Expected P. OF WALES, got %s", alliedShips[0].Name)
	}
}

func TestLoadShipConfigsFromDefaultPath(t *testing.T) {
	// Тестируем загрузку из стандартного пути
	// Сначала проверяем, существует ли файл
	defaultPath := filepath.Join("..", "..", "config", "ships.json")
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		t.Skip("Default ships.json not found, skipping test")
	}

	manager := NewShipConfigManager()
	err := manager.LoadConfig(defaultPath)
	if err != nil {
		t.Fatalf("Failed to load ship configs from default path: %v", err)
	}

	// Получаем все корабли
	configs, err := manager.GetAllShips()
	if err != nil {
		t.Fatalf("Failed to get all ships: %v", err)
	}

	// Проверяем, что загружены корабли
	if len(configs) == 0 {
		t.Error("Expected at least one ship in default config")
	}

	// Проверяем, что есть корабли обеих сторон
	hasGerman := false
	hasAllied := false
	for _, ship := range configs {
		if ship.Side == "german" {
			hasGerman = true
		}
		if ship.Side == "allied" {
			hasAllied = true
		}
	}

	if !hasGerman {
		t.Error("Expected at least one German ship")
	}
	if !hasAllied {
		t.Error("Expected at least one Allied ship")
	}
}

func TestShipConfigManagerMethods(t *testing.T) {
	manager := NewShipConfigManager()
	
	// Тест IsConfigLoaded до загрузки
	if manager.IsConfigLoaded() {
		t.Error("Expected config not to be loaded initially")
	}

	// Создаем временный файл с тестовыми данными
	shipConfigContent := `{
		"ships": [
			{
				"id": "bismarck",
				"name": "BISMARCK",
				"type": "BB",
				"side": "german",
				"speedType": "fast",
				"maxFuel": 18,
				"baseEvasion": 30,
				"hullBoxes": 15,
				"basePrimaryArmamentBow": 8,
				"basePrimaryArmamentStern": 8,
				"baseSecondaryArmament": 12,
				"maxTorpedos": 0,
				"radarLevel": 2
			}
		]
	}`

	tmpFile, err := os.CreateTemp("", "test-ships-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(shipConfigContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	err = manager.LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Тест IsConfigLoaded после загрузки
	if !manager.IsConfigLoaded() {
		t.Error("Expected config to be loaded after loading")
	}

	// Тест GetShipNames
	names, err := manager.GetShipNames()
	if err != nil {
		t.Fatalf("Failed to get ship names: %v", err)
	}
	if len(names) != 1 {
		t.Errorf("Expected 1 ship name, got %d", len(names))
	}
	if names[0] != "BISMARCK" {
		t.Errorf("Expected BISMARCK, got %s", names[0])
	}

	// Тест GetShipsByType
	bbShips, err := manager.GetShipsByType("BB")
	if err != nil {
		t.Fatalf("Failed to get BB ships: %v", err)
	}
	if len(bbShips) != 1 {
		t.Errorf("Expected 1 BB ship, got %d", len(bbShips))
	}

	// Тест GetConfigStats
	stats, err := manager.GetConfigStats()
	if err != nil {
		t.Fatalf("Failed to get config stats: %v", err)
	}
	if stats.TotalShips != 1 {
		t.Errorf("Expected 1 total ship, got %d", stats.TotalShips)
	}
}
