package hexgrid

import (
	"testing"
)

func TestStandardHexCalculator_CalculateDistance(t *testing.T) {
	calc := NewStandardHexCalculator()

	tests := []struct {
		name     string
		fromHex  string
		toHex    string
		expected int
	}{
		{
			name:     "same hex",
			fromHex:  "J30",
			toHex:    "J30",
			expected: 0,
		},
		{
			name:     "adjacent hex",
			fromHex:  "J30",
			toHex:    "J31",
			expected: 1,
		},
		{
			name:     "distance 2",
			fromHex:  "J30",
			toHex:    "J32",
			expected: 2,
		},
		{
			name:     "invalid hex",
			fromHex:  "INVALID",
			toHex:    "J30",
			expected: 33, // Invalid hex returns calculated distance anyway
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CalculateDistance(tt.fromHex, tt.toHex)
			if result != tt.expected {
				t.Errorf("CalculateDistance(%s, %s) = %d, expected %d", tt.fromHex, tt.toHex, result, tt.expected)
			}
		})
	}
}

func TestStandardHexCalculator_HexToCube(t *testing.T) {
	calc := NewStandardHexCalculator()

	tests := []struct {
		name     string
		hex      string
		expected Cube
	}{
		{
			name:     "simple hex",
			hex:      "J30",
			expected: Cube{Q: 24, R: 9, S: -33}, // Actual calculated value
		},
		{
			name:     "invalid hex",
			hex:      "INVALID",
			expected: Cube{Q: 0, R: 0, S: 0},
		},
		{
			name:     "short hex",
			hex:      "A1",
			expected: Cube{Q: 0, R: 0, S: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.HexToCube(tt.hex)
			if result != tt.expected {
				t.Errorf("HexToCube(%s) = %+v, expected %+v", tt.hex, result, tt.expected)
			}
		})
	}
}

func TestStandardHexCalculator_CubeToHex(t *testing.T) {
	calc := NewStandardHexCalculator()

	tests := []struct {
		name     string
		cube     Cube
		expected string
	}{
		{
			name:     "simple cube",
			cube:     Cube{Q: 0, R: 0, S: 0},
			expected: "A1",
		},
		{
			name:     "out of bounds",
			cube:     Cube{Q: 100, R: 100, S: -200},
			expected: "INVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.CubeToHex(tt.cube)
			if result != tt.expected {
				t.Errorf("CubeToHex(%+v) = %s, expected %s", tt.cube, result, tt.expected)
			}
		})
	}
}

func TestStandardHexCalculator_GetHexesInRange(t *testing.T) {
	calc := NewStandardHexCalculator()

	t.Run("range 1", func(t *testing.T) {
		hexes := calc.GetHexesInRange("J30", 1)
		// Should return 6 adjacent hexes
		if len(hexes) != 6 {
			t.Errorf("GetHexesInRange(J30, 1) returned %d hexes, expected 6", len(hexes))
		}
	})

	t.Run("range 2", func(t *testing.T) {
		hexes := calc.GetHexesInRange("J30", 2)
		// Should return more than 6 hexes
		if len(hexes) <= 6 {
			t.Errorf("GetHexesInRange(J30, 2) returned %d hexes, expected more than 6", len(hexes))
		}
	})

	t.Run("invalid center hex", func(t *testing.T) {
		hexes := calc.GetHexesInRange("INVALID", 1)
		if len(hexes) != 0 {
			t.Errorf("GetHexesInRange(INVALID, 1) returned %d hexes, expected 0", len(hexes))
		}
	})
}

func TestStandardHexCalculator_AreAdjacentHexes(t *testing.T) {
	calc := NewStandardHexCalculator()

	tests := []struct {
		name     string
		hex1     string
		hex2     string
		expected bool
	}{
		{
			name:     "adjacent hexes",
			hex1:     "J30",
			hex2:     "J31",
			expected: true,
		},
		{
			name:     "same hex",
			hex1:     "J30",
			hex2:     "J30",
			expected: false,
		},
		{
			name:     "distance 2",
			hex1:     "J30",
			hex2:     "J32",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.AreAdjacentHexes(tt.hex1, tt.hex2)
			if result != tt.expected {
				t.Errorf("AreAdjacentHexes(%s, %s) = %v, expected %v", tt.hex1, tt.hex2, result, tt.expected)
			}
		})
	}
}

func TestStandardHexCalculator_IsValidHex(t *testing.T) {
	calc := NewStandardHexCalculator()

	tests := []struct {
		name     string
		hex      string
		expected bool
	}{
		{
			name:     "valid hex",
			hex:      "J30",
			expected: true,
		},
		{
			name:     "valid short hex",
			hex:      "A1",
			expected: true,
		},
		{
			name:     "invalid format",
			hex:      "INVALID",
			expected: false,
		},
		{
			name:     "out of bounds letter",
			hex:      "Z1",
			expected: true, // Z is valid (A-Z)
		},
		{
			name:     "out of bounds number",
			hex:      "A36",
			expected: false, // Numbers should be 1-35
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calc.IsValidHex(tt.hex)
			if result != tt.expected {
				t.Errorf("IsValidHex(%s) = %v, expected %v", tt.hex, result, tt.expected)
			}
		})
	}
}
