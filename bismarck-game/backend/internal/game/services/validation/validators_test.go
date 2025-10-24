package validation

import (
	"bismarck-game/backend/internal/game/models"
	"testing"
)

func TestDamagedShipValidator(t *testing.T) {
	tests := []struct {
		name        string
		evasion     int
		speedRating models.SpeedType
		distance    int
		shouldFail  bool
	}{
		{
			name:        "damaged fast ship 1 hex - allowed",
			evasion:     25,
			speedRating: models.SpeedTypeFast,
			distance:    1,
			shouldFail:  false,
		},
		{
			name:        "damaged fast ship 2 hexes - blocked",
			evasion:     25,
			speedRating: models.SpeedTypeFast,
			distance:    2,
			shouldFail:  true,
		},
		{
			name:        "heavily damaged fast ship 1 hex - allowed",
			evasion:     20,
			speedRating: models.SpeedTypeFast,
			distance:    1,
			shouldFail:  false,
		},
		{
			name:        "heavily damaged fast ship 2 hexes - blocked",
			evasion:     15,
			speedRating: models.SpeedTypeFast,
			distance:    2,
			shouldFail:  true,
		},
		{
			name:        "undamaged fast ship 2 hexes - allowed",
			evasion:     26,
			speedRating: models.SpeedTypeFast,
			distance:    2,
			shouldFail:  false,
		},
		{
			name:        "damaged medium ship - not affected",
			evasion:     20,
			speedRating: models.SpeedTypeMedium,
			distance:    1,
			shouldFail:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewDamagedShipValidator()
			ctx := &ValidationContext{
				Unit: &models.NavalUnit{
					Evasion:     tt.evasion,
					SpeedRating: tt.speedRating,
				},
				Distance: tt.distance,
			}

			err := validator.Validate(ctx)
			if tt.shouldFail && err == nil {
				t.Errorf("Expected validation to fail, but it passed")
			}
			if !tt.shouldFail && err != nil {
				t.Errorf("Expected validation to pass, but got error: %v", err)
			}
		})
	}
}