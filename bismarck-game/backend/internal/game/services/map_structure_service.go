package services

import (
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"
	"encoding/json"
	"fmt"
	"os"
)

// MapStructureService предоставляет методы для работы со структурами карты
type MapStructureService struct {
	mapStructures *models.MapStructure
	logger        *logger.Logger
}

// NewMapStructureService создает новый сервис структур карты
func NewMapStructureService() *MapStructureService {
	log, _ := logger.New(logger.INFO, "map-structure-service", "stdout")
	return &MapStructureService{
		logger: log,
	}
}

// LoadConfig загружает конфигурацию структур карты
func (s *MapStructureService) LoadConfig(path string) error {
	s.logger.Info("Загрузка конфигурации структур карты", "path", path)

	data, err := os.ReadFile(path)
	if err != nil {
		s.logger.Error("Ошибка чтения файла конфигурации", "error", err)
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var mapStructures models.MapStructure
	if err := json.Unmarshal(data, &mapStructures); err != nil {
		s.logger.Error("Ошибка парсинга JSON конфигурации", "error", err)
		return fmt.Errorf("failed to parse config JSON: %w", err)
	}

	s.mapStructures = &mapStructures
	s.logger.Info("Конфигурация структур карты успешно загружена",
		"landAreas", len(mapStructures.LandAreas),
		"nonGameHexes", len(mapStructures.NonGameHexes),
		"hasRestrictedDD", mapStructures.RestrictedDD != nil,
		"fogAreas", len(mapStructures.FogAreas))

	return nil
}

// GetMapStructures возвращает структуры карты
func (s *MapStructureService) GetMapStructures() *models.MapStructure {
	return s.mapStructures
}

// GetHexType определяет тип гекса
func (s *MapStructureService) GetHexType(hexId string) models.HexType {
	if s.mapStructures == nil {
		return models.HexTypeWater // По умолчанию морской
	}

	// Проверяем неигровые гексы
	for _, nonGame := range s.mapStructures.NonGameHexes {
		for _, id := range nonGame.HexIds {
			if id == hexId {
				return models.HexTypeNonGame
			}
		}
	}

	// Проверяем сухопутные гексы
	for _, landArea := range s.mapStructures.LandAreas {
		for _, id := range landArea.HexIds {
			if id == hexId {
				return models.HexTypeLand
			}
		}
	}

	// По умолчанию морской гекс
	return models.HexTypeWater
}

// IsLandHex проверяет, является ли гекс сухопутным
func (s *MapStructureService) IsLandHex(hexId string) bool {
	return s.GetHexType(hexId) == models.HexTypeLand
}

// IsNonGameHex проверяет, является ли гекс неигровым
func (s *MapStructureService) IsNonGameHex(hexId string) bool {
	return s.GetHexType(hexId) == models.HexTypeNonGame
}

// IsRestrictedDDHex проверяет, является ли гекс разрешенным для немецких DD
func (s *MapStructureService) IsRestrictedDDHex(hexId string) bool {
	if s.mapStructures == nil || s.mapStructures.RestrictedDD == nil {
		return false
	}

	for _, id := range s.mapStructures.RestrictedDD.HexIds {
		if id == hexId {
			return true
		}
	}
	return false
}

// IsFogHex проверяет, является ли гекс туманным
func (s *MapStructureService) IsFogHex(hexId string) bool {
	if s.mapStructures == nil {
		return false
	}

	for _, fogArea := range s.mapStructures.FogAreas {
		for _, id := range fogArea.HexIds {
			if id == hexId {
				return true
			}
		}
	}
	return false
}

// CanUnitMoveTo проверяет, может ли юнит двигаться в указанный гекс
func (s *MapStructureService) CanUnitMoveTo(unit *models.NavalUnit, hexId string) bool {
	if unit == nil {
		return false
	}

	// Неигровые гексы запрещены для всех
	if s.IsNonGameHex(hexId) {
		return false
	}

	// Сухопутные гексы запрещены для морских юнитов
	if s.IsLandHex(hexId) {
		return false
	}

	// Специальные правила для немецких DD
	if unit.Nationality == "german" && unit.Type == "DD" {
		// Немецкие DD могут двигаться ТОЛЬКО в разрешенные гексы
		return s.IsRestrictedDDHex(hexId)
	}

	// Остальные корабли могут двигаться в любые морские гексы
	return true
}

// ClassifyHex возвращает полную классификацию гекса
func (s *MapStructureService) ClassifyHex(hexId string) models.HexClassification {
	classification := models.HexClassification{
		BaseType:     s.GetHexType(hexId),
		SpecialZones: []string{},
		Restrictions: []string{},
	}

	// Добавляем специальные зоны
	if s.IsRestrictedDDHex(hexId) {
		classification.SpecialZones = append(classification.SpecialZones, "restricted_dd")
	}

	// Добавляем ограничения
	if classification.BaseType == models.HexTypeLand {
		classification.Restrictions = append(classification.Restrictions, "no_naval_movement")
	}

	return classification
}
