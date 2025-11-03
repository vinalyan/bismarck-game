package services

import (
	"fmt"

	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
)

// EmergencyFuelService предоставляет методы для управления аварийным топливом
type EmergencyFuelService struct {
	db          *database.Database
	logger      *logger.Logger
	phaseManager *PhaseManager
}

// NewEmergencyFuelService создает новый сервис аварийного топлива
func NewEmergencyFuelService(db *database.Database, logger *logger.Logger, phaseManager *PhaseManager) *EmergencyFuelService {
	return &EmergencyFuelService{
		db:          db,
		logger:      logger,
		phaseManager: phaseManager,
	}
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

// updateEmergencyFuelStatus обновляет статус аварийного топлива в базе данных
func (s *EmergencyFuelService) updateEmergencyFuelStatus(unitID, gameID string, isEmergency bool, emergencyTurn int) error {
	query := `
		UPDATE naval_units SET
			is_emergency_fuel = $1,
			emergency_turn = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND game_id = $4`

	_, err := s.db.Exec(query, isEmergency, emergencyTurn, unitID, gameID)
	if err != nil {
		s.logger.Error("Failed to update emergency fuel status", "error", err, "unit_id", unitID)
		return fmt.Errorf("failed to update emergency fuel status: %w", err)
	}

	return nil
}

// ActivateIfNeeded проверяет и активирует аварийное топливо при необходимости
func (s *EmergencyFuelService) ActivateIfNeeded(gameID, unitID string, currentFuel int) error {
	// Проверяем текущий статус аварийного топлива
	query := `SELECT is_emergency_fuel FROM naval_units WHERE id = $1 AND game_id = $2`
	var isEmergencyFuel bool
	err := s.db.QueryRow(query, unitID, gameID).Scan(&isEmergencyFuel)
	if err != nil {
		return fmt.Errorf("failed to get emergency fuel status: %w", err)
	}

	// Проверяем, нужно ли активировать аварийное топливо
	if currentFuel <= 0 && !isEmergencyFuel {
		currentTurn := s.getCurrentTurn(gameID)
		emergencyTurn := currentTurn + 10

		// Обновляем в базе данных
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
	// Получаем текущее топливо и статус аварийного топлива
	query := `SELECT fuel, is_emergency_fuel FROM naval_units WHERE id = $1 AND game_id = $2`
	var currentFuel int
	var isEmergencyFuel bool
	err := s.db.QueryRow(query, unitID, gameID).Scan(&currentFuel, &isEmergencyFuel)
	if err != nil {
		return fmt.Errorf("failed to get fuel status: %w", err)
	}

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

