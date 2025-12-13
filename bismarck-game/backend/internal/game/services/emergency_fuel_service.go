package services

import (
	"fmt"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
)

// EmergencyFuelService предоставляет методы для управления аварийным топливом
type EmergencyFuelService struct {
	db               *database.Database
	logger           *logger.Logger
	phaseManager     *PhaseManager
	gameStateService *GameStateService
	unitService      *UnitService
}

// NewEmergencyFuelService создает новый сервис аварийного топлива
func NewEmergencyFuelService(db *database.Database, logger *logger.Logger, phaseManager *PhaseManager) *EmergencyFuelService {
	return &EmergencyFuelService{
		db:           db,
		logger:       logger,
		phaseManager: phaseManager,
	}
}

// SetGameStateService устанавливает GameStateService для работы с GameModel
func (s *EmergencyFuelService) SetGameStateService(gameStateService *GameStateService) {
	s.gameStateService = gameStateService
}

// SetUnitService устанавливает UnitService для работы с юнитами
func (s *EmergencyFuelService) SetUnitService(unitService *UnitService) {
	s.unitService = unitService
}

// getCurrentTurn получает текущий ход игры
func (s *EmergencyFuelService) getCurrentTurn(gameID string) int {
	// Если PhaseManager не инициализирован (для тестов), возвращаем 1
	if s.phaseManager == nil {
		return 1
	}

	// Получаем текущий ход из PhaseManager
	turn, err := s.phaseManager.GetCurrentPhase(gameID)
	if err != nil || turn == nil {
		s.logger.Warn("Failed to get current turn, using fallback", "game_id", gameID, "error", err)
		return 1 // fallback
	}
	return turn.TurnNumber
}

// updateEmergencyFuelStatus обновляет статус аварийного топлива в GameModel
func (s *EmergencyFuelService) updateEmergencyFuelStatus(unitID, gameID string, isEmergency bool, emergencyTurn int) error {
	if s.gameStateService == nil {
		return fmt.Errorf("gameStateService is required for updateEmergencyFuelStatus")
	}

	// Обновляем статус аварийного топлива в GameModel
	if err := s.gameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		unit, exists := model.Units[unitID]
		if !exists {
			return fmt.Errorf("unit not found: %s", unitID)
		}

		if unit.NavalData == nil {
			return fmt.Errorf("naval data is missing for unit: %s", unitID)
		}

		unit.NavalData.IsEmergencyFuel = isEmergency
		unit.NavalData.EmergencyTurn = emergencyTurn

		return nil
	}, 3); err != nil {
		s.logger.Error("Failed to update emergency fuel status in GameModel", "error", err, "unit_id", unitID)
		return fmt.Errorf("failed to update emergency fuel status: %w", err)
	}

	return nil
}

// ActivateIfNeeded проверяет и активирует аварийное топливо при необходимости
func (s *EmergencyFuelService) ActivateIfNeeded(gameID, unitID string, currentFuel int) error {
	if s.unitService == nil {
		return fmt.Errorf("unitService is required for ActivateIfNeeded")
	}

	// Получаем текущий статус аварийного топлива из GameModel
	unit, err := s.unitService.GetNavalUnitByIDFromGameModel(gameID, unitID)
	if err != nil {
		return fmt.Errorf("failed to get unit from GameModel: %w", err)
	}

	isEmergencyFuel := unit.IsEmergencyFuel

	// Проверяем, нужно ли активировать аварийное топливо
	if currentFuel <= 0 && !isEmergencyFuel {
		currentTurn := s.getCurrentTurn(gameID)
		emergencyTurn := currentTurn + 10

		// Обновляем в GameModel
		if err := s.updateEmergencyFuelStatus(unitID, gameID, true, emergencyTurn); err != nil {
			return fmt.Errorf("failed to update emergency fuel status: %w", err)
		}

		s.logger.Warn("Emergency fuel activated - ship must reach port or refuel within 10 turns",
			"unit_id", unitID,
			"current_fuel", currentFuel,
			"current_turn", currentTurn,
			"emergency_turn", emergencyTurn,
			"turns_remaining", 10)
	}

	return nil
}

// ClearIfRefueled проверяет и очищает статус аварийного топлива при заправке
func (s *EmergencyFuelService) ClearIfRefueled(gameID, unitID string) error {
	if s.unitService == nil {
		return fmt.Errorf("unitService is required for ClearIfRefueled")
	}

	// Получаем текущее топливо и статус аварийного топлива из GameModel
	unit, err := s.unitService.GetNavalUnitByIDFromGameModel(gameID, unitID)
	if err != nil {
		return fmt.Errorf("failed to get unit from GameModel: %w", err)
	}

	currentFuel := unit.Fuel
	isEmergencyFuel := unit.IsEmergencyFuel

	// Если топливо > 0 и аварийное топливо активно, снимаем статус
	if currentFuel > 0 && isEmergencyFuel {
		if err := s.updateEmergencyFuelStatus(unitID, gameID, false, 0); err != nil {
			return fmt.Errorf("failed to clear emergency fuel status: %w", err)
		}

		s.logger.Info("Emergency fuel status cleared due to refueling",
			"unit_id", unitID,
			"new_fuel", currentFuel)
	}

	return nil
}
