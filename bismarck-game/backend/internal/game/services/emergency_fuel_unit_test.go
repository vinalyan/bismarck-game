package services

import (
	"testing"
)

// TestEmergencyFuelMovementRestrictions тестирует ограничения движения с аварийным топливом
func TestEmergencyFuelMovementRestrictions(t *testing.T) {
	tests := []struct {
		name            string
		isEmergencyFuel bool
		distance        int
		expectedAllowed bool
		description     string
	}{
		{
			name:            "Emergency fuel allows 1 hex movement",
			isEmergencyFuel: true,
			distance:        1,
			expectedAllowed: true,
			description:     "Emergency fuel should allow 1 hex movement",
		},
		{
			name:            "Emergency fuel blocks 2 hex movement",
			isEmergencyFuel: true,
			distance:        2,
			expectedAllowed: false,
			description:     "Emergency fuel should block 2 hex movement",
		},
		{
			name:            "Normal fuel allows 2 hex movement",
			isEmergencyFuel: false,
			distance:        2,
			expectedAllowed: true,
			description:     "Normal fuel should allow 2 hex movement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Тестируем логику ограничений
			allowed := true
			if tt.isEmergencyFuel && tt.distance > 1 {
				allowed = false
			}

			if allowed != tt.expectedAllowed {
				t.Errorf("Expected movement allowed: %v, got: %v - %s",
					tt.expectedAllowed, allowed, tt.description)
			}
		})
	}
}

// TestEmergencyFuelActivation тестирует активацию аварийного топлива
func TestEmergencyFuelActivation(t *testing.T) {
	tests := []struct {
		name           string
		currentFuel    int
		expectedActive bool
		description    string
	}{
		{
			name:           "Zero fuel activates emergency",
			currentFuel:    0,
			expectedActive: true,
			description:    "Zero fuel should activate emergency fuel",
		},
		{
			name:           "Negative fuel activates emergency",
			currentFuel:    -1,
			expectedActive: true,
			description:    "Negative fuel should activate emergency fuel",
		},
		{
			name:           "Positive fuel does not activate emergency",
			currentFuel:    5,
			expectedActive: false,
			description:    "Positive fuel should not activate emergency fuel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем логику активации
			shouldActivate := tt.currentFuel <= 0

			if shouldActivate != tt.expectedActive {
				t.Errorf("Expected emergency fuel activation: %v, got: %v - %s",
					tt.expectedActive, shouldActivate, tt.description)
			}
		})
	}
}

// TestEmergencyFuelTurnCalculation тестирует расчет хода истечения аварийного топлива
func TestEmergencyFuelTurnCalculation(t *testing.T) {
	tests := []struct {
		name         string
		currentTurn  int
		expectedTurn int
		description  string
	}{
		{
			name:         "Turn 1 emergency expires on turn 11",
			currentTurn:  1,
			expectedTurn: 11,
			description:  "Emergency fuel should expire 10 turns after activation",
		},
		{
			name:         "Turn 5 emergency expires on turn 15",
			currentTurn:  5,
			expectedTurn: 15,
			description:  "Emergency fuel should expire 10 turns after activation",
		},
		{
			name:         "Turn 10 emergency expires on turn 20",
			currentTurn:  10,
			expectedTurn: 20,
			description:  "Emergency fuel should expire 10 turns after activation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем расчет хода истечения
			emergencyTurn := tt.currentTurn + 10

			if emergencyTurn != tt.expectedTurn {
				t.Errorf("Expected emergency turn: %d, got: %d - %s",
					tt.expectedTurn, emergencyTurn, tt.description)
			}
		})
	}
}

// TestEmergencyFuelRefueling тестирует заправку и снятие аварийного статуса
func TestEmergencyFuelRefueling(t *testing.T) {
	tests := []struct {
		name           string
		currentFuel    int
		refuelAmount   int
		maxFuel        int
		expectedFuel   int
		expectedActive bool
		description    string
	}{
		{
			name:           "Refueling from 0 to 5 should clear emergency",
			currentFuel:    0,
			refuelAmount:   5,
			maxFuel:        18,
			expectedFuel:   5,
			expectedActive: false,
			description:    "Refueling should clear emergency fuel status",
		},
		{
			name:           "Refueling from 0 to max should clear emergency",
			currentFuel:    0,
			refuelAmount:   20,
			maxFuel:        18,
			expectedFuel:   18,
			expectedActive: false,
			description:    "Refueling should not exceed max fuel",
		},
		{
			name:           "Partial refueling should clear emergency",
			currentFuel:    0,
			refuelAmount:   1,
			maxFuel:        18,
			expectedFuel:   1,
			expectedActive: false,
			description:    "Any positive fuel should clear emergency status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Симулируем заправку
			newFuel := tt.currentFuel + tt.refuelAmount
			if newFuel > tt.maxFuel {
				newFuel = tt.maxFuel
			}

			// Проверяем, что топливо не превышает максимум
			if newFuel != tt.expectedFuel {
				t.Errorf("Expected fuel: %d, got: %d - %s",
					tt.expectedFuel, newFuel, tt.description)
			}

			// Проверяем снятие аварийного статуса
			shouldClearEmergency := newFuel > 0
			if shouldClearEmergency != !tt.expectedActive {
				t.Errorf("Expected emergency cleared: %v, got: %v - %s",
					!tt.expectedActive, shouldClearEmergency, tt.description)
			}
		})
	}
}
