package validation

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
)

func TestFastShipStrategy_ValidateMovement(t *testing.T) {
	strategy := NewFastShipStrategy()

	tests := []struct {
		name     string
		distance int
		expected bool
	}{
		{
			name:     "valid 1 hex movement",
			distance: 1,
			expected: true,
		},
		{
			name:     "valid 2 hex movement",
			distance: 2,
			expected: true,
		},
		{
			name:     "invalid 0 hex movement",
			distance: 0,
			expected: false,
		},
		{
			name:     "invalid 3 hex movement",
			distance: 3,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createTestContext(models.SpeedTypeFast, tt.distance)
			err := strategy.ValidateMovement(ctx)

			if tt.expected && err != nil {
				t.Errorf("Expected no error for distance %d, but got: %v", tt.distance, err)
			}
			if !tt.expected && err == nil {
				t.Errorf("Expected error for distance %d, but got none", tt.distance)
			}
		})
	}
}

func TestFastShipStrategy_CalculateFuelCost(t *testing.T) {
	strategy := NewFastShipStrategy()

	tests := []struct {
		name              string
		distance          int
		previousTurnMoved int
		expectedFuelCost  int
	}{
		{
			name:              "1 hex movement",
			distance:          1,
			previousTurnMoved: 0,
			expectedFuelCost:  0,
		},
		{
			name:              "2 hex movement after 0 hexes",
			distance:          2,
			previousTurnMoved: 0,
			expectedFuelCost:  1,
		},
		{
			name:              "2 hex movement after 1 hex",
			distance:          2,
			previousTurnMoved: 1,
			expectedFuelCost:  1,
		},
		{
			name:              "2 hex movement after 2 hexes",
			distance:          2,
			previousTurnMoved: 2,
			expectedFuelCost:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.CalculateFuelCost(tt.distance, tt.previousTurnMoved)
			if result != tt.expectedFuelCost {
				t.Errorf("CalculateFuelCost(%d, %d) = %d, expected %d",
					tt.distance, tt.previousTurnMoved, result, tt.expectedFuelCost)
			}
		})
	}
}

func TestMediumShipStrategy_ValidateMovement(t *testing.T) {
	strategy := NewMediumShipStrategy()

	tests := []struct {
		name     string
		distance int
		expected bool
	}{
		{
			name:     "valid 1 hex movement",
			distance: 1,
			expected: true,
		},
		{
			name:     "invalid 2 hex movement",
			distance: 2,
			expected: false,
		},
		{
			name:     "invalid 0 hex movement",
			distance: 0,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createTestContext(models.SpeedTypeMedium, tt.distance)
			err := strategy.ValidateMovement(ctx)

			if tt.expected && err != nil {
				t.Errorf("Expected no error for distance %d, but got: %v", tt.distance, err)
			}
			if !tt.expected && err == nil {
				t.Errorf("Expected error for distance %d, but got none", tt.distance)
			}
		})
	}
}

func TestMediumShipStrategy_CalculateFuelCost(t *testing.T) {
	strategy := NewMediumShipStrategy()

	tests := []struct {
		name              string
		distance          int
		previousTurnMoved int
		expectedFuelCost  int
	}{
		{
			name:              "1 hex movement after no movement",
			distance:          1,
			previousTurnMoved: 0,
			expectedFuelCost:  0,
		},
		{
			name:              "1 hex movement after movement",
			distance:          1,
			previousTurnMoved: 1,
			expectedFuelCost:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.CalculateFuelCost(tt.distance, tt.previousTurnMoved)
			if result != tt.expectedFuelCost {
				t.Errorf("CalculateFuelCost(%d, %d) = %d, expected %d",
					tt.distance, tt.previousTurnMoved, result, tt.expectedFuelCost)
			}
		})
	}
}

func TestSlowShipStrategy_ValidateMovement(t *testing.T) {
	strategy := NewSlowShipStrategy()

	tests := []struct {
		name     string
		distance int
		expected bool
	}{
		{
			name:     "valid 1 hex movement",
			distance: 1,
			expected: true,
		},
		{
			name:     "invalid 2 hex movement",
			distance: 2,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createTestContext(models.SpeedTypeSlow, tt.distance)
			err := strategy.ValidateMovement(ctx)

			if tt.expected && err != nil {
				t.Errorf("Expected no error for distance %d, but got: %v", tt.distance, err)
			}
			if !tt.expected && err == nil {
				t.Errorf("Expected error for distance %d, but got none", tt.distance)
			}
		})
	}
}

func TestSlowShipStrategy_CalculateFuelCost(t *testing.T) {
	strategy := NewSlowShipStrategy()

	// Slow ships never consume fuel
	result := strategy.CalculateFuelCost(1, 0)
	if result != 0 {
		t.Errorf("CalculateFuelCost(1, 0) = %d, expected 0", result)
	}
}

func TestVerySlowShipStrategy_ValidateMovement(t *testing.T) {
	strategy := NewVerySlowShipStrategy()

	tests := []struct {
		name     string
		distance int
		expected bool
	}{
		{
			name:     "valid 1 hex movement",
			distance: 1,
			expected: true,
		},
		{
			name:     "invalid 2 hex movement",
			distance: 2,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createTestContext(models.SpeedTypeVerySlow, tt.distance)
			err := strategy.ValidateMovement(ctx)

			if tt.expected && err != nil {
				t.Errorf("Expected no error for distance %d, but got: %v", tt.distance, err)
			}
			if !tt.expected && err == nil {
				t.Errorf("Expected error for distance %d, but got none", tt.distance)
			}
		})
	}
}

func TestVerySlowShipStrategy_CalculateFuelCost(t *testing.T) {
	strategy := NewVerySlowShipStrategy()

	// Very slow ships never consume fuel
	result := strategy.CalculateFuelCost(1, 0)
	if result != 0 {
		t.Errorf("CalculateFuelCost(1, 0) = %d, expected 0", result)
	}
}

func TestGetStrategyForSpeedType(t *testing.T) {
	tests := []struct {
		name         string
		speedType    models.SpeedType
		expectedType string
	}{
		{
			name:         "fast ship",
			speedType:    models.SpeedTypeFast,
			expectedType: "FastShipStrategy",
		},
		{
			name:         "medium ship",
			speedType:    models.SpeedTypeMedium,
			expectedType: "MediumShipStrategy",
		},
		{
			name:         "slow ship",
			speedType:    models.SpeedTypeSlow,
			expectedType: "SlowShipStrategy",
		},
		{
			name:         "very slow ship",
			speedType:    models.SpeedTypeVerySlow,
			expectedType: "VerySlowShipStrategy",
		},
		{
			name:         "unknown speed type",
			speedType:    "UNKNOWN",
			expectedType: "MediumShipStrategy", // Default fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := GetStrategyForSpeedType(tt.speedType)
			if strategy == nil {
				t.Errorf("GetStrategyForSpeedType(%s) returned nil", tt.speedType)
			}
		})
	}
}
