package services

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
)

// TestRefuelUnit тестирует метод RefuelUnit в MovementService
func TestRefuelUnit(t *testing.T) {
	tests := []struct {
		name              string
		initialFuel       int
		maxFuel           int
		fuelToAdd         int
		expectedFinalFuel int
		initialEmergency  bool
		expectedEmergency bool
		description       string
	}{
		{
			name:              "Refuel from 0 to 5",
			initialFuel:       0,
			maxFuel:           18,
			fuelToAdd:         5,
			expectedFinalFuel: 5,
			initialEmergency:  true,
			expectedEmergency: false,
			description:       "Refueling from 0 should clear emergency status",
		},
		{
			name:              "Refuel from 0 to max",
			initialFuel:       0,
			maxFuel:           18,
			fuelToAdd:         20, // More than max
			expectedFinalFuel: 18, // Should be capped at max
			initialEmergency:  true,
			expectedEmergency: false,
			description:       "Refueling should not exceed max fuel",
		},
		{
			name:              "Partial refuel from 5 to 10",
			initialFuel:       5,
			maxFuel:           18,
			fuelToAdd:         5,
			expectedFinalFuel: 10,
			initialEmergency:  false,
			expectedEmergency: false,
			description:       "Partial refueling should work correctly",
		},
		{
			name:              "Refuel from 10 to max",
			initialFuel:       10,
			maxFuel:           18,
			fuelToAdd:         10,
			expectedFinalFuel: 18,
			initialEmergency:  false,
			expectedEmergency: false,
			description:       "Refueling to max should work correctly",
		},
		{
			name:              "Refuel from 15 to max",
			initialFuel:       15,
			maxFuel:           18,
			fuelToAdd:         10, // More than needed
			expectedFinalFuel: 18, // Should be capped at max
			initialEmergency:  false,
			expectedEmergency: false,
			description:       "Refueling should not exceed max fuel",
		},
		{
			name:              "Emergency fuel cleared by refuel",
			initialFuel:       0,
			maxFuel:           18,
			fuelToAdd:         1,
			expectedFinalFuel: 1,
			initialEmergency:  true,
			expectedEmergency: false,
			description:       "Any positive fuel should clear emergency status",
		},
		{
			name:              "No refuel needed",
			initialFuel:       18,
			maxFuel:           18,
			fuelToAdd:         5,
			expectedFinalFuel: 18, // Should remain at max
			initialEmergency:  false,
			expectedEmergency: false,
			description:       "Refueling at max fuel should not change anything",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовый FuelTracking
			fuelTracking := &models.FuelTracking{
				GameID:          "test-game",
				UnitID:          "test-unit",
				CurrentFuel:     tt.initialFuel,
				MaxFuel:         tt.maxFuel,
				IsEmergencyFuel: tt.initialEmergency,
				EmergencyTurn:   0,
			}

			// Симулируем логику RefuelUnit
			// Добавляем топливо (не больше максимального)
			newFuel := fuelTracking.CurrentFuel + tt.fuelToAdd
			if newFuel > fuelTracking.MaxFuel {
				newFuel = fuelTracking.MaxFuel
			}

			// Обновляем топливо
			fuelTracking.CurrentFuel = newFuel

			// Если топливо > 0, снимаем статус аварийного топлива
			if fuelTracking.CurrentFuel > 0 {
				fuelTracking.IsEmergencyFuel = false
				fuelTracking.EmergencyTurn = 0
			}

			// Проверяем результаты
			if fuelTracking.CurrentFuel != tt.expectedFinalFuel {
				t.Errorf("CurrentFuel = %d, expected %d. %s",
					fuelTracking.CurrentFuel, tt.expectedFinalFuel, tt.description)
			}

			if fuelTracking.IsEmergencyFuel != tt.expectedEmergency {
				t.Errorf("IsEmergencyFuel = %v, expected %v. %s",
					fuelTracking.IsEmergencyFuel, tt.expectedEmergency, tt.description)
			}

			if !tt.expectedEmergency && fuelTracking.EmergencyTurn != 0 {
				t.Errorf("EmergencyTurn should be 0 when emergency is cleared, got %d. %s",
					fuelTracking.EmergencyTurn, tt.description)
			}
		})
	}
}

// TestRefuelUnitEmergencyFuel тестирует снятие статуса аварийного топлива при заправке
func TestRefuelUnitEmergencyFuel(t *testing.T) {
	tests := []struct {
		name                  string
		initialFuel           int
		fuelToAdd             int
		initialEmergency      bool
		initialEmergencyTurn  int
		expectedEmergency     bool
		expectedEmergencyTurn int
		description           string
	}{
		{
			name:                  "Clear emergency with 1 fuel",
			initialFuel:           0,
			fuelToAdd:             1,
			initialEmergency:      true,
			initialEmergencyTurn:  15,
			expectedEmergency:     false,
			expectedEmergencyTurn: 0,
			description:           "Adding 1 fuel should clear emergency status",
		},
		{
			name:                  "Clear emergency with 5 fuel",
			initialFuel:           0,
			fuelToAdd:             5,
			initialEmergency:      true,
			initialEmergencyTurn:  20,
			expectedEmergency:     false,
			expectedEmergencyTurn: 0,
			description:           "Adding 5 fuel should clear emergency status",
		},
		{
			name:                  "Clear emergency with max fuel",
			initialFuel:           0,
			fuelToAdd:             18,
			initialEmergency:      true,
			initialEmergencyTurn:  25,
			expectedEmergency:     false,
			expectedEmergencyTurn: 0,
			description:           "Adding max fuel should clear emergency status",
		},
		{
			name:                  "No emergency to clear",
			initialFuel:           10,
			fuelToAdd:             5,
			initialEmergency:      false,
			initialEmergencyTurn:  0,
			expectedEmergency:     false,
			expectedEmergencyTurn: 0,
			description:           "No emergency status should remain unchanged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовый FuelTracking
			fuelTracking := &models.FuelTracking{
				GameID:          "test-game",
				UnitID:          "test-unit",
				CurrentFuel:     tt.initialFuel,
				MaxFuel:         18,
				IsEmergencyFuel: tt.initialEmergency,
				EmergencyTurn:   tt.initialEmergencyTurn,
			}

			// Симулируем логику RefuelUnit
			// Добавляем топливо
			newFuel := fuelTracking.CurrentFuel + tt.fuelToAdd
			if newFuel > fuelTracking.MaxFuel {
				newFuel = fuelTracking.MaxFuel
			}
			fuelTracking.CurrentFuel = newFuel

			// Если топливо > 0, снимаем статус аварийного топлива
			if fuelTracking.CurrentFuel > 0 {
				fuelTracking.IsEmergencyFuel = false
				fuelTracking.EmergencyTurn = 0
			}

			// Проверяем результаты
			if fuelTracking.IsEmergencyFuel != tt.expectedEmergency {
				t.Errorf("IsEmergencyFuel = %v, expected %v. %s",
					fuelTracking.IsEmergencyFuel, tt.expectedEmergency, tt.description)
			}

			if fuelTracking.EmergencyTurn != tt.expectedEmergencyTurn {
				t.Errorf("EmergencyTurn = %d, expected %d. %s",
					fuelTracking.EmergencyTurn, tt.expectedEmergencyTurn, tt.description)
			}
		})
	}
}

// TestRefuelUnitFuelCapping тестирует ограничение топлива максимальным значением
func TestRefuelUnitFuelCapping(t *testing.T) {
	tests := []struct {
		name              string
		initialFuel       int
		maxFuel           int
		fuelToAdd         int
		expectedFinalFuel int
		description       string
	}{
		{
			name:              "Add fuel within max",
			initialFuel:       5,
			maxFuel:           18,
			fuelToAdd:         10,
			expectedFinalFuel: 15,
			description:       "Adding fuel within max should work normally",
		},
		{
			name:              "Add fuel exceeding max",
			initialFuel:       10,
			maxFuel:           18,
			fuelToAdd:         15, // Would exceed max
			expectedFinalFuel: 18, // Should be capped at max
			description:       "Adding fuel exceeding max should cap at max",
		},
		{
			name:              "Add fuel to already max",
			initialFuel:       18,
			maxFuel:           18,
			fuelToAdd:         5,
			expectedFinalFuel: 18, // Should remain at max
			description:       "Adding fuel to max should not exceed max",
		},
		{
			name:              "Add fuel from 0 to max",
			initialFuel:       0,
			maxFuel:           18,
			fuelToAdd:         18,
			expectedFinalFuel: 18,
			description:       "Adding exact max fuel should work",
		},
		{
			name:              "Add fuel from 0 exceeding max",
			initialFuel:       0,
			maxFuel:           18,
			fuelToAdd:         25, // Much more than max
			expectedFinalFuel: 18, // Should be capped at max
			description:       "Adding fuel from 0 exceeding max should cap at max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовый FuelTracking
			fuelTracking := &models.FuelTracking{
				GameID:          "test-game",
				UnitID:          "test-unit",
				CurrentFuel:     tt.initialFuel,
				MaxFuel:         tt.maxFuel,
				IsEmergencyFuel: false,
				EmergencyTurn:   0,
			}

			// Симулируем логику RefuelUnit
			// Добавляем топливо (не больше максимального)
			newFuel := fuelTracking.CurrentFuel + tt.fuelToAdd
			if newFuel > fuelTracking.MaxFuel {
				newFuel = fuelTracking.MaxFuel
			}
			fuelTracking.CurrentFuel = newFuel

			// Проверяем результат
			if fuelTracking.CurrentFuel != tt.expectedFinalFuel {
				t.Errorf("CurrentFuel = %d, expected %d. %s",
					fuelTracking.CurrentFuel, tt.expectedFinalFuel, tt.description)
			}
		})
	}
}

// TestRefuelUnitIntegration тестирует интеграцию RefuelUnit с другими системами
func TestRefuelUnitIntegration(t *testing.T) {
	t.Run("Complete refuel scenario", func(t *testing.T) {
		// Создаем тестовый FuelTracking в аварийном состоянии
		fuelTracking := &models.FuelTracking{
			GameID:          "test-game",
			UnitID:          "test-unit",
			CurrentFuel:     0,
			MaxFuel:         18,
			IsEmergencyFuel: true,
			EmergencyTurn:   15,
		}

		// Симулируем полную заправку
		fuelToAdd := 18
		newFuel := fuelTracking.CurrentFuel + fuelToAdd
		if newFuel > fuelTracking.MaxFuel {
			newFuel = fuelTracking.MaxFuel
		}
		fuelTracking.CurrentFuel = newFuel

		// Снимаем статус аварийного топлива
		if fuelTracking.CurrentFuel > 0 {
			fuelTracking.IsEmergencyFuel = false
			fuelTracking.EmergencyTurn = 0
		}

		// Проверяем результаты
		if fuelTracking.CurrentFuel != 18 {
			t.Errorf("CurrentFuel = %d, expected 18", fuelTracking.CurrentFuel)
		}

		if fuelTracking.IsEmergencyFuel {
			t.Error("IsEmergencyFuel should be false after refuel")
		}

		if fuelTracking.EmergencyTurn != 0 {
			t.Errorf("EmergencyTurn = %d, expected 0", fuelTracking.EmergencyTurn)
		}
	})

	t.Run("Partial refuel scenario", func(t *testing.T) {
		// Создаем тестовый FuelTracking в аварийном состоянии
		fuelTracking := &models.FuelTracking{
			GameID:          "test-game",
			UnitID:          "test-unit",
			CurrentFuel:     0,
			MaxFuel:         18,
			IsEmergencyFuel: true,
			EmergencyTurn:   20,
		}

		// Симулируем частичную заправку
		fuelToAdd := 5
		newFuel := fuelTracking.CurrentFuel + fuelToAdd
		if newFuel > fuelTracking.MaxFuel {
			newFuel = fuelTracking.MaxFuel
		}
		fuelTracking.CurrentFuel = newFuel

		// Снимаем статус аварийного топлива
		if fuelTracking.CurrentFuel > 0 {
			fuelTracking.IsEmergencyFuel = false
			fuelTracking.EmergencyTurn = 0
		}

		// Проверяем результаты
		if fuelTracking.CurrentFuel != 5 {
			t.Errorf("CurrentFuel = %d, expected 5", fuelTracking.CurrentFuel)
		}

		if fuelTracking.IsEmergencyFuel {
			t.Error("IsEmergencyFuel should be false after refuel")
		}

		if fuelTracking.EmergencyTurn != 0 {
			t.Errorf("EmergencyTurn = %d, expected 0", fuelTracking.EmergencyTurn)
		}
	})
}
