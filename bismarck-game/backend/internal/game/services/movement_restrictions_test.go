package services

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
)

// TestGermanDestroyerMovementRestrictions тестирует ограничения движения немецких эсминцев
func TestGermanDestroyerMovementRestrictions(t *testing.T) {
	tests := []struct {
		name        string
		unit        *models.NavalUnit
		fromHex     string
		toHex       string
		expectedErr bool
		description string
	}{
		{
			name: "German DD cannot cross boundary line Q29",
			unit: &models.NavalUnit{
				ID:          "test-dd-1",
				Type:        models.UnitTypeDestroyer,
				Owner:       "german",
				SpeedRating: models.SpeedTypeFast,
				Position:    "Q28",
			},
			fromHex:     "Q28",
			toHex:       "Q29",
			expectedErr: true,
			description: "German destroyer cannot cross boundary line at Q29",
		},
		{
			name: "German DD cannot cross boundary line R28",
			unit: &models.NavalUnit{
				ID:          "test-dd-2",
				Type:        models.UnitTypeDestroyer,
				Owner:       "german",
				SpeedRating: models.SpeedTypeFast,
				Position:    "R27",
			},
			fromHex:     "R27",
			toHex:       "R28",
			expectedErr: true,
			description: "German destroyer cannot cross boundary line at R28",
		},
		{
			name: "German DD cannot cross boundary line S27",
			unit: &models.NavalUnit{
				ID:          "test-dd-3",
				Type:        models.UnitTypeDestroyer,
				Owner:       "german",
				SpeedRating: models.SpeedTypeFast,
				Position:    "S26",
			},
			fromHex:     "S26",
			toHex:       "S27",
			expectedErr: true,
			description: "German destroyer cannot cross boundary line at S27",
		},
		{
			name: "German DD cannot cross boundary line T26",
			unit: &models.NavalUnit{
				ID:          "test-dd-4",
				Type:        models.UnitTypeDestroyer,
				Owner:       "german",
				SpeedRating: models.SpeedTypeFast,
				Position:    "T25",
			},
			fromHex:     "T25",
			toHex:       "T26",
			expectedErr: true,
			description: "German destroyer cannot cross boundary line at T26",
		},
		{
			name: "German DD can move within allowed area",
			unit: &models.NavalUnit{
				ID:          "test-dd-5",
				Type:        models.UnitTypeDestroyer,
				Owner:       "german",
				SpeedRating: models.SpeedTypeFast,
				Position:    "Q28",
			},
			fromHex:     "Q28",
			toHex:       "Q27",
			expectedErr: false,
			description: "German destroyer can move within allowed area",
		},
		{
			name: "German DD can move to adjacent hex within boundary",
			unit: &models.NavalUnit{
				ID:          "test-dd-6",
				Type:        models.UnitTypeDestroyer,
				Owner:       "german",
				SpeedRating: models.SpeedTypeFast,
				Position:    "Q27",
			},
			fromHex:     "Q27",
			toHex:       "Q28",
			expectedErr: false,
			description: "German destroyer can move to adjacent hex within boundary",
		},
		{
			name: "Non-German DD has no restrictions",
			unit: &models.NavalUnit{
				ID:          "test-dd-7",
				Type:        models.UnitTypeDestroyer,
				Owner:       "allied",
				SpeedRating: models.SpeedTypeFast,
				Position:    "Q28",
			},
			fromHex:     "Q28",
			toHex:       "Q29",
			expectedErr: false,
			description: "Allied destroyer has no movement restrictions",
		},
		{
			name: "German non-DD has no restrictions",
			unit: &models.NavalUnit{
				ID:          "test-bb-1",
				Type:        models.UnitTypeBattleship,
				Owner:       "german",
				SpeedRating: models.SpeedTypeFast,
				Position:    "Q28",
			},
			fromHex:     "Q28",
			toHex:       "Q29",
			expectedErr: false,
			description: "German battleship has no movement restrictions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &MovementService{}

			// Проверяем только для немецких эсминцев
			if tt.unit.Owner == "german" && tt.unit.Type == models.UnitTypeDestroyer {
				err := service.validateGermanDDMovement(tt.fromHex, tt.toHex)
				if tt.expectedErr && err == nil {
					t.Errorf("Expected error but got none. %s", tt.description)
				}
				if !tt.expectedErr && err != nil {
					t.Errorf("Expected no error but got: %v. %s", err, tt.description)
				}
			} else {
				// Для не-немецких эсминцев или не-эсминцев не должно быть ошибок
				if tt.expectedErr {
					t.Errorf("Expected error for non-German DD or non-DD unit. %s", tt.description)
				}
			}
		})
	}
}

// TestTankerMovementRestrictions тестирует ограничения движения танкеров
func TestTankerMovementRestrictions(t *testing.T) {
	tests := []struct {
		name        string
		unit        *models.NavalUnit
		toHex       string
		expectedErr bool
		description string
	}{
		{
			name: "Tanker cannot enter convoy hex H15",
			unit: &models.NavalUnit{
				ID:          "test-tanker-1",
				Type:        models.UnitTypeTanker,
				Owner:       "german",
				SpeedRating: models.SpeedTypeVerySlow,
				Position:    "H14",
			},
			toHex:       "H15",
			expectedErr: true,
			description: "German tanker cannot enter convoy hex H15",
		},
		{
			name: "Tanker cannot enter convoy hex I16",
			unit: &models.NavalUnit{
				ID:          "test-tanker-2",
				Type:        models.UnitTypeTanker,
				Owner:       "german",
				SpeedRating: models.SpeedTypeVerySlow,
				Position:    "I15",
			},
			toHex:       "I16",
			expectedErr: true,
			description: "German tanker cannot enter convoy hex I16",
		},
		{
			name: "Tanker cannot enter convoy hex J17",
			unit: &models.NavalUnit{
				ID:          "test-tanker-3",
				Type:        models.UnitTypeTanker,
				Owner:       "german",
				SpeedRating: models.SpeedTypeVerySlow,
				Position:    "J16",
			},
			toHex:       "J17",
			expectedErr: true,
			description: "German tanker cannot enter convoy hex J17",
		},
		{
			name: "Tanker can move to regular hex",
			unit: &models.NavalUnit{
				ID:          "test-tanker-4",
				Type:        models.UnitTypeTanker,
				Owner:       "german",
				SpeedRating: models.SpeedTypeVerySlow,
				Position:    "J30",
			},
			toHex:       "J31",
			expectedErr: false,
			description: "German tanker can move to regular hex",
		},
		{
			name: "Tanker can move to port hex",
			unit: &models.NavalUnit{
				ID:          "test-tanker-5",
				Type:        models.UnitTypeTanker,
				Owner:       "german",
				SpeedRating: models.SpeedTypeVerySlow,
				Position:    "K30",
			},
			toHex:       "K31",
			expectedErr: false,
			description: "German tanker can move to port hex",
		},
		{
			name: "Non-tanker has no convoy restrictions",
			unit: &models.NavalUnit{
				ID:          "test-bb-1",
				Type:        models.UnitTypeBattleship,
				Owner:       "german",
				SpeedRating: models.SpeedTypeFast,
				Position:    "H14",
			},
			toHex:       "H15",
			expectedErr: false,
			description: "German battleship has no convoy restrictions",
		},
		{
			name: "Allied tanker has no convoy restrictions",
			unit: &models.NavalUnit{
				ID:          "test-tanker-6",
				Type:        models.UnitTypeTanker,
				Owner:       "allied",
				SpeedRating: models.SpeedTypeVerySlow,
				Position:    "H14",
			},
			toHex:       "H15",
			expectedErr: false,
			description: "Allied tanker has no convoy restrictions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &MovementService{}

			// Проверяем только для немецких танкеров
			if tt.unit.Owner == "german" && tt.unit.Type == models.UnitTypeTanker {
				err := service.validateTankerMovement(tt.toHex)
				if tt.expectedErr && err == nil {
					t.Errorf("Expected error but got none. %s", tt.description)
				}
				if !tt.expectedErr && err != nil {
					t.Errorf("Expected no error but got: %v. %s", err, tt.description)
				}
			} else {
				// Для не-немецких танкеров или не-танкеров не должно быть ошибок
				if tt.expectedErr {
					t.Errorf("Expected error for non-German tanker or non-tanker unit. %s", tt.description)
				}
			}
		})
	}
}

// TestMovementRestrictionsIntegration тестирует интеграцию ограничений движения
func TestMovementRestrictionsIntegration(t *testing.T) {
	tests := []struct {
		name        string
		unit        *models.NavalUnit
		fromHex     string
		toHex       string
		expectedErr bool
		description string
	}{
		{
			name: "German DD with multiple restrictions",
			unit: &models.NavalUnit{
				ID:          "test-dd-1",
				Type:        models.UnitTypeDestroyer,
				Owner:       "german",
				SpeedRating: models.SpeedTypeFast,
				Position:    "Q28",
			},
			fromHex:     "Q28",
			toHex:       "Q29", // Boundary restriction
			expectedErr: true,
			description: "German DD should be blocked by boundary restriction",
		},
		{
			name: "German tanker with convoy restriction",
			unit: &models.NavalUnit{
				ID:          "test-tanker-1",
				Type:        models.UnitTypeTanker,
				Owner:       "german",
				SpeedRating: models.SpeedTypeVerySlow,
				Position:    "H14",
			},
			fromHex:     "H14",
			toHex:       "H15", // Convoy hex
			expectedErr: true,
			description: "German tanker should be blocked by convoy restriction",
		},
		{
			name: "Valid movement for German BB",
			unit: &models.NavalUnit{
				ID:          "test-bb-1",
				Type:        models.UnitTypeBattleship,
				Owner:       "german",
				SpeedRating: models.SpeedTypeFast,
				Position:    "Q28",
			},
			fromHex:     "Q28",
			toHex:       "Q29", // No restrictions for BB
			expectedErr: false,
			description: "German BB should have no movement restrictions",
		},
		{
			name: "Valid movement for Allied DD",
			unit: &models.NavalUnit{
				ID:          "test-dd-2",
				Type:        models.UnitTypeDestroyer,
				Owner:       "allied",
				SpeedRating: models.SpeedTypeFast,
				Position:    "Q28",
			},
			fromHex:     "Q28",
			toHex:       "Q29", // No restrictions for Allied DD
			expectedErr: false,
			description: "Allied DD should have no movement restrictions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &MovementService{}

			err := service.validateMovementRestrictions(tt.unit, tt.fromHex, tt.toHex)

			if tt.expectedErr && err == nil {
				t.Errorf("Expected error but got none. %s", tt.description)
			}
			if !tt.expectedErr && err != nil {
				t.Errorf("Expected no error but got: %v. %s", err, tt.description)
			}
		})
	}
}

// TestBoundaryLineCoordinates тестирует координаты граничной линии
func TestBoundaryLineCoordinates(t *testing.T) {
	restrictedHexes := []string{"Q29", "R28", "S27", "T26"}

	for _, hex := range restrictedHexes {
		t.Run("Restricted hex "+hex, func(t *testing.T) {
			service := &MovementService{}

			// Создаем немецкий эсминец
			unit := &models.NavalUnit{
				ID:          "test-dd",
				Type:        models.UnitTypeDestroyer,
				Owner:       "german",
				SpeedRating: models.SpeedTypeFast,
				Position:    "Q28",
			}

			// Пытаемся переместиться в ограниченный гекс
			err := service.validateMovementRestrictions(unit, "Q28", hex)

			if err == nil {
				t.Errorf("Expected error for restricted hex %s, but got none", hex)
			}
		})
	}
}

// TestConvoyHexCoordinates тестирует координаты гексов конвоев
func TestConvoyHexCoordinates(t *testing.T) {
	convoyHexes := []string{"H15", "I16", "J17"}

	for _, hex := range convoyHexes {
		t.Run("Convoy hex "+hex, func(t *testing.T) {
			service := &MovementService{}

			// Создаем немецкий танкер
			unit := &models.NavalUnit{
				ID:          "test-tanker",
				Type:        models.UnitTypeTanker,
				Owner:       "german",
				SpeedRating: models.SpeedTypeVerySlow,
				Position:    "H14",
			}

			// Пытаемся переместиться в гекс конвоя
			err := service.validateMovementRestrictions(unit, "H14", hex)

			if err == nil {
				t.Errorf("Expected error for convoy hex %s, but got none", hex)
			}
		})
	}
}
