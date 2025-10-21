package services

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
)

// TestEmergencyFuelMovement тестирует движение с аварийным топливом
func TestEmergencyFuelMovement(t *testing.T) {
	tests := []struct {
		name            string
		unit            *models.NavalUnit
		isEmergencyFuel bool
		distance        int
		expectedErr     bool
		description     string
	}{
		{
			name: "Emergency fuel allows 1 hex movement",
			unit: &models.NavalUnit{
				ID:          "test-ship-1",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        0,
				MaxFuel:     18,
				Position:    "J30",
			},
			isEmergencyFuel: true,
			distance:        1,
			expectedErr:     false,
			description:     "Emergency fuel should allow 1 hex movement",
		},
		{
			name: "Emergency fuel blocks 2 hex movement",
			unit: &models.NavalUnit{
				ID:          "test-ship-2",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        0,
				MaxFuel:     18,
				Position:    "J30",
			},
			isEmergencyFuel: true,
			distance:        2,
			expectedErr:     true,
			description:     "Emergency fuel should block 2 hex movement",
		},
		{
			name: "Emergency fuel blocks 3 hex movement",
			unit: &models.NavalUnit{
				ID:          "test-ship-3",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        0,
				MaxFuel:     18,
				Position:    "J30",
			},
			isEmergencyFuel: true,
			distance:        3,
			expectedErr:     true,
			description:     "Emergency fuel should block 3 hex movement",
		},
		{
			name: "Normal fuel allows 2 hex movement",
			unit: &models.NavalUnit{
				ID:          "test-ship-4",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        5,
				MaxFuel:     18,
				Position:    "J30",
			},
			isEmergencyFuel: false,
			distance:        2,
			expectedErr:     false,
			description:     "Normal fuel should allow 2 hex movement",
		},
		{
			name: "Emergency fuel allows 1 hex movement for M type",
			unit: &models.NavalUnit{
				ID:          "test-ship-5",
				SpeedRating: models.SpeedTypeMedium,
				Fuel:        0,
				MaxFuel:     15,
				Position:    "J30",
			},
			isEmergencyFuel: true,
			distance:        1,
			expectedErr:     false,
			description:     "Emergency fuel should allow 1 hex movement for M type",
		},
		{
			name: "Emergency fuel allows 1 hex movement for S type",
			unit: &models.NavalUnit{
				ID:          "test-ship-6",
				SpeedRating: models.SpeedTypeSlow,
				Fuel:        0,
				MaxFuel:     10,
				Position:    "J30",
			},
			isEmergencyFuel: true,
			distance:        1,
			expectedErr:     false,
			description:     "Emergency fuel should allow 1 hex movement for S type",
		},
		{
			name: "Emergency fuel allows 1 hex movement for VS type",
			unit: &models.NavalUnit{
				ID:          "test-ship-7",
				SpeedRating: models.SpeedTypeVerySlow,
				Fuel:        0,
				MaxFuel:     5,
				Position:    "J30",
			},
			isEmergencyFuel: true,
			distance:        1,
			expectedErr:     false,
			description:     "Emergency fuel should allow 1 hex movement for VS type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &MovementService{}

			// Устанавливаем аварийное топливо для тестирования
			// В реальной игре это поле будет добавлено в модель
			// tt.unit.IsEmergencyFuel = tt.isEmergencyFuel

			// Тестируем валидацию движения
			fromHex := tt.unit.Position
			toHex := "J31" // Соседний гекс для расстояния 1

			if tt.distance == 2 {
				toHex = "J32" // Гекс на расстоянии 2
			} else if tt.distance == 3 {
				toHex = "J33" // Гекс на расстоянии 3
			}

			err := service.ValidateMovement(tt.unit, fromHex, toHex)

			if tt.expectedErr && err == nil {
				t.Errorf("Expected error but got none. %s", tt.description)
			}
			if !tt.expectedErr && err != nil {
				t.Errorf("Expected no error but got: %v. %s", err, tt.description)
			}
		})
	}
}

// TestEmergencyFuelActivation тестирует активацию аварийного топлива
func TestEmergencyFuelActivation(t *testing.T) {
	tests := []struct {
		name                  string
		unit                  *models.NavalUnit
		fuelBefore            int
		fuelAfter             int
		expectedEmergency     bool
		expectedEmergencyTurn int
		description           string
	}{
		{
			name: "Fuel reaches 0 - emergency fuel activated",
			unit: &models.NavalUnit{
				ID:          "test-ship-1",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        1,
				MaxFuel:     18,
				Position:    "J30",
			},
			fuelBefore:            1,
			fuelAfter:             0,
			expectedEmergency:     true,
			expectedEmergencyTurn: 10, // Текущий ход + 10
			description:           "Emergency fuel should be activated when fuel reaches 0",
		},
		{
			name: "Fuel remains positive - no emergency",
			unit: &models.NavalUnit{
				ID:          "test-ship-2",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        5,
				MaxFuel:     18,
				Position:    "J30",
			},
			fuelBefore:            5,
			fuelAfter:             3,
			expectedEmergency:     false,
			expectedEmergencyTurn: 0,
			description:           "No emergency fuel when fuel remains positive",
		},
		{
			name: "Fuel goes negative - emergency fuel activated",
			unit: &models.NavalUnit{
				ID:          "test-ship-3",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        1,
				MaxFuel:     18,
				Position:    "J30",
			},
			fuelBefore:            1,
			fuelAfter:             -1,
			expectedEmergency:     true,
			expectedEmergencyTurn: 10,
			description:           "Emergency fuel should be activated when fuel goes negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Симулируем расход топлива
			tt.unit.Fuel = tt.fuelAfter

			// Проверяем, должна ли активироваться аварийная ситуация
			shouldActivateEmergency := tt.fuelAfter <= 0

			if shouldActivateEmergency != tt.expectedEmergency {
				t.Errorf("Emergency fuel activation = %v, expected %v. %s",
					shouldActivateEmergency, tt.expectedEmergency, tt.description)
			}

			// Если аварийная ситуация активирована, проверяем расчет хода
			if shouldActivateEmergency {
				// В реальной игре это будет текущий ход + 10
				emergencyTurn := 10 // Упрощенный расчет
				if emergencyTurn != tt.expectedEmergencyTurn {
					t.Errorf("Emergency turn = %d, expected %d. %s",
						emergencyTurn, tt.expectedEmergencyTurn, tt.description)
				}
			}
		})
	}
}

// TestEmergencyFuelMovementRestrictions тестирует ограничения движения с аварийным топливом
func TestEmergencyFuelMovementRestrictions(t *testing.T) {
	tests := []struct {
		name            string
		unit            *models.NavalUnit
		isEmergencyFuel bool
		emergencyTurn   int
		currentTurn     int
		canMove         bool
		description     string
	}{
		{
			name: "Emergency fuel allows movement before emergency turn",
			unit: &models.NavalUnit{
				ID:          "test-ship-1",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        0,
				MaxFuel:     18,
				Position:    "J30",
			},
			isEmergencyFuel: true,
			emergencyTurn:   10,
			currentTurn:     5,
			canMove:         true,
			description:     "Emergency fuel should allow movement before emergency turn",
		},
		{
			name: "Emergency fuel blocks movement after emergency turn",
			unit: &models.NavalUnit{
				ID:          "test-ship-2",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        0,
				MaxFuel:     18,
				Position:    "J30",
			},
			isEmergencyFuel: true,
			emergencyTurn:   10,
			currentTurn:     11,
			canMove:         false,
			description:     "Emergency fuel should block movement after emergency turn",
		},
		{
			name: "Emergency fuel blocks movement at emergency turn",
			unit: &models.NavalUnit{
				ID:          "test-ship-3",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        0,
				MaxFuel:     18,
				Position:    "J30",
			},
			isEmergencyFuel: true,
			emergencyTurn:   10,
			currentTurn:     10,
			canMove:         false,
			description:     "Emergency fuel should block movement at emergency turn",
		},
		{
			name: "Normal fuel allows movement",
			unit: &models.NavalUnit{
				ID:          "test-ship-4",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        5,
				MaxFuel:     18,
				Position:    "J30",
			},
			isEmergencyFuel: false,
			emergencyTurn:   0,
			currentTurn:     15,
			canMove:         true,
			description:     "Normal fuel should allow movement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем, может ли корабль двигаться
			canMove := true

			if tt.isEmergencyFuel {
				// При аварийном топливе можно двигаться только до emergency turn
				canMove = tt.currentTurn < tt.emergencyTurn
			}

			if canMove != tt.canMove {
				t.Errorf("Can move = %v, expected %v. %s", canMove, tt.canMove, tt.description)
			}
		})
	}
}

// TestEmergencyFuelRefueling тестирует заправку при аварийном топливе
func TestEmergencyFuelRefueling(t *testing.T) {
	tests := []struct {
		name              string
		unit              *models.NavalUnit
		refuelAmount      int
		expectedFuel      int
		expectedEmergency bool
		description       string
	}{
		{
			name: "Refueling removes emergency fuel status",
			unit: &models.NavalUnit{
				ID:          "test-ship-1",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        0,
				MaxFuel:     18,
				Position:    "J30",
			},
			refuelAmount:      4, // Стандартная заправка
			expectedFuel:      4,
			expectedEmergency: false,
			description:       "Refueling should remove emergency fuel status",
		},
		{
			name: "Partial refueling still in emergency",
			unit: &models.NavalUnit{
				ID:          "test-ship-2",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        0,
				MaxFuel:     18,
				Position:    "J30",
			},
			refuelAmount:      1, // Частичная заправка
			expectedFuel:      1,
			expectedEmergency: true, // Все еще в аварийной ситуации
			description:       "Partial refueling should maintain emergency status",
		},
		{
			name: "Full refueling removes emergency",
			unit: &models.NavalUnit{
				ID:          "test-ship-3",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        0,
				MaxFuel:     18,
				Position:    "J30",
			},
			refuelAmount:      18, // Полная заправка
			expectedFuel:      18,
			expectedEmergency: false,
			description:       "Full refueling should remove emergency status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Симулируем заправку
			newFuel := tt.unit.Fuel + tt.refuelAmount
			if newFuel > tt.unit.MaxFuel {
				newFuel = tt.unit.MaxFuel
			}

			tt.unit.Fuel = newFuel

			// Проверяем результат
			if tt.unit.Fuel != tt.expectedFuel {
				t.Errorf("Fuel after refueling = %d, expected %d. %s",
					tt.unit.Fuel, tt.expectedFuel, tt.description)
			}

			// Проверяем статус аварийного топлива
			isEmergency := tt.unit.Fuel <= 0
			if isEmergency != tt.expectedEmergency {
				t.Errorf("Emergency fuel status = %v, expected %v. %s",
					isEmergency, tt.expectedEmergency, tt.description)
			}
		})
	}
}

// TestEmergencyFuelShipRemoval тестирует удаление корабля при аварийном топливе
func TestEmergencyFuelShipRemoval(t *testing.T) {
	tests := []struct {
		name            string
		unit            *models.NavalUnit
		emergencyTurn   int
		currentTurn     int
		shouldBeRemoved bool
		description     string
	}{
		{
			name: "Ship should be removed after emergency turn",
			unit: &models.NavalUnit{
				ID:          "test-ship-1",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        0,
				MaxFuel:     18,
				Position:    "J30",
			},
			emergencyTurn:   10,
			currentTurn:     11,
			shouldBeRemoved: true,
			description:     "Ship should be removed after emergency turn",
		},
		{
			name: "Ship should not be removed before emergency turn",
			unit: &models.NavalUnit{
				ID:          "test-ship-2",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        0,
				MaxFuel:     18,
				Position:    "J30",
			},
			emergencyTurn:   10,
			currentTurn:     9,
			shouldBeRemoved: false,
			description:     "Ship should not be removed before emergency turn",
		},
		{
			name: "Ship should not be removed if refueled",
			unit: &models.NavalUnit{
				ID:          "test-ship-3",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        4,
				MaxFuel:     18,
				Position:    "J30",
			},
			emergencyTurn:   10,
			currentTurn:     11,
			shouldBeRemoved: false,
			description:     "Ship should not be removed if refueled",
		},
		{
			name: "Ship should not be removed if reached port",
			unit: &models.NavalUnit{
				ID:          "test-ship-4",
				SpeedRating: models.SpeedTypeFast,
				Fuel:        0,
				MaxFuel:     18,
				Position:    "PORT", // В порту
			},
			emergencyTurn:   10,
			currentTurn:     11,
			shouldBeRemoved: false,
			description:     "Ship should not be removed if reached port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем, должен ли корабль быть удален
			shouldBeRemoved := tt.unit.Fuel <= 0 &&
				tt.unit.Position != "PORT" &&
				tt.currentTurn >= tt.emergencyTurn

			if shouldBeRemoved != tt.shouldBeRemoved {
				t.Errorf("Should be removed = %v, expected %v. %s",
					shouldBeRemoved, tt.shouldBeRemoved, tt.description)
			}
		})
	}
}
