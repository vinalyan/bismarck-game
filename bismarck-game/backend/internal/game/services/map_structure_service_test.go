package services

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
)

func TestMapStructureService_LoadConfig(t *testing.T) {
	service := NewMapStructureService()

	// Тест загрузки конфигурации
	err := service.LoadConfig("../../../config/map-structures.json")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	structures := service.GetMapStructures()
	if structures == nil {
		t.Fatal("Map structures should not be nil")
	}
}

func TestMapStructureService_GetHexType(t *testing.T) {
	service := NewMapStructureService()
	err := service.LoadConfig("../../../config/map-structures.json")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	tests := []struct {
		hexId    string
		expected models.HexType
	}{
		{"A1", models.HexTypeLand},    // Сухопутный гекс
		{"Q1", models.HexTypeNonGame}, // Неигровой гекс
		{"J30", models.HexTypeWater},  // Морской гекс (по умолчанию)
		{"K30", models.HexTypeWater},  // Морской гекс
	}

	for _, test := range tests {
		result := service.GetHexType(test.hexId)
		if result != test.expected {
			t.Errorf("GetHexType(%s) = %v, expected %v", test.hexId, result, test.expected)
		}
	}
}

func TestMapStructureService_IsLandHex(t *testing.T) {
	service := NewMapStructureService()
	err := service.LoadConfig("../../../config/map-structures.json")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	tests := []struct {
		hexId    string
		expected bool
	}{
		{"A1", true},   // Сухопутный гекс
		{"J30", false}, // Морской гекс
		{"Q1", false},  // Неигровой гекс
	}

	for _, test := range tests {
		result := service.IsLandHex(test.hexId)
		if result != test.expected {
			t.Errorf("IsLandHex(%s) = %v, expected %v", test.hexId, result, test.expected)
		}
	}
}

func TestMapStructureService_IsNonGameHex(t *testing.T) {
	service := NewMapStructureService()
	err := service.LoadConfig("../../../config/map-structures.json")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	tests := []struct {
		hexId    string
		expected bool
	}{
		{"Q1", true},   // Неигровой гекс
		{"J30", false}, // Морской гекс
		{"A1", false},  // Сухопутный гекс
	}

	for _, test := range tests {
		result := service.IsNonGameHex(test.hexId)
		if result != test.expected {
			t.Errorf("IsNonGameHex(%s) = %v, expected %v", test.hexId, result, test.expected)
		}
	}
}

func TestMapStructureService_IsRestrictedDDHex(t *testing.T) {
	service := NewMapStructureService()
	err := service.LoadConfig("../../../config/map-structures.json")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	tests := []struct {
		hexId    string
		expected bool
	}{
		{"Y24", true}, // Разрешенный для немецких DD
		{"J30", true}, // Разрешенный для немецких DD
		{"A1", false}, // Не разрешенный для немецких DD
	}

	for _, test := range tests {
		result := service.IsRestrictedDDHex(test.hexId)
		if result != test.expected {
			t.Errorf("IsRestrictedDDHex(%s) = %v, expected %v", test.hexId, result, test.expected)
		}
	}
}

func TestMapStructureService_CanUnitMoveTo(t *testing.T) {
	service := NewMapStructureService()
	err := service.LoadConfig("../../../config/map-structures.json")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Создаем тестовые юниты
	germanDD := &models.NavalUnit{
		Nationality: "german",
		Type:        "DD",
	}

	alliedBB := &models.NavalUnit{
		Nationality: "allied",
		Type:        "BB",
	}

	tests := []struct {
		unit     *models.NavalUnit
		hexId    string
		expected bool
		desc     string
	}{
		{germanDD, "Y24", true, "Немецкий DD в разрешенном гексе"},
		{germanDD, "J30", true, "Немецкий DD в разрешенном гексе"},
		{germanDD, "A1", false, "Немецкий DD в запрещенном гексе"},
		{germanDD, "Q1", false, "Немецкий DD в неигровом гексе"},
		{alliedBB, "Y24", true, "Союзный BB в любом морском гексе"},
		{alliedBB, "J30", true, "Союзный BB в любом морском гексе"},
		{alliedBB, "A1", false, "Союзный BB в сухопутном гексе"},
		{alliedBB, "Q1", false, "Союзный BB в неигровом гексе"},
	}

	for _, test := range tests {
		result := service.CanUnitMoveTo(test.unit, test.hexId)
		if result != test.expected {
			t.Errorf("CanUnitMoveTo(%s, %s) = %v, expected %v (%s)",
				test.unit.Type, test.hexId, result, test.expected, test.desc)
		}
	}
}

func TestMapStructureService_ClassifyHex(t *testing.T) {
	service := NewMapStructureService()
	err := service.LoadConfig("../../../config/map-structures.json")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	classification := service.ClassifyHex("Y24")

	if classification.BaseType != models.HexTypeWater {
		t.Errorf("Expected base type water, got %v", classification.BaseType)
	}

	// Проверяем, что гекс имеет специальную зону restricted_dd
	hasRestrictedDD := false
	for _, zone := range classification.SpecialZones {
		if zone == "restricted_dd" {
			hasRestrictedDD = true
			break
		}
	}

	if !hasRestrictedDD {
		t.Error("Expected restricted_dd special zone")
	}
}
