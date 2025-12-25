package services

import (
	"testing"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGameModelValidator(t *testing.T) *GameModelValidator {
	logger, err := logger.New(logger.INFO, "test", "stdout")
	require.NoError(t, err)
	return NewGameModelValidator(logger)
}

// TestGameModelValidator_ValidateModel_Success тестирует успешную валидацию
func TestGameModelValidator_ValidateModel_Success(t *testing.T) {
	validator := setupGameModelValidator(t)

	validModel := &models.GameModel{
		GameID:      "test-game",
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{
			Turn:  1,
			Phase: models.PhaseMovement,
		},
		Units:         make(map[string]*models.UnitModel),
		TaskForces:    make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Events:        []*models.GameEventModel{},
		Search: &models.SearchData{
			German: make(map[string]models.SearchHexData),
			Allied: make(map[string]models.SearchHexData),
		},
	}

	err := validator.ValidateModel(validModel)
	assert.NoError(t, err)
}

// TestGameModelValidator_ValidateModel_InvalidVersion тестирует невалидную версию
func TestGameModelValidator_ValidateModel_InvalidVersion(t *testing.T) {
	validator := setupGameModelValidator(t)

	invalidModel := &models.GameModel{
		GameID:      "test-game",
		Version:     0, // Invalid: version < 1
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{
			Turn:  1,
			Phase: models.PhaseMovement,
		},
		Units:         make(map[string]*models.UnitModel),
		TaskForces:    make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Events:        []*models.GameEventModel{},
	}

	err := validator.ValidateModel(invalidModel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version must be >= 1")
}

// TestGameModelValidator_ValidateModel_InvalidGameID тестирует пустой GameID
func TestGameModelValidator_ValidateModel_InvalidGameID(t *testing.T) {
	validator := setupGameModelValidator(t)

	invalidModel := &models.GameModel{
		GameID:      "", // Invalid: empty GameID
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: &models.GameTurnModel{
			Turn:  1,
			Phase: models.PhaseMovement,
		},
		Units:         make(map[string]*models.UnitModel),
		TaskForces:    make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Events:        []*models.GameEventModel{},
	}

	err := validator.ValidateModel(invalidModel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "game_id is required")
}

// TestGameModelValidator_ValidateModel_MissingCurrentTurn тестирует отсутствие CurrentTurn
func TestGameModelValidator_ValidateModel_MissingCurrentTurn(t *testing.T) {
	validator := setupGameModelValidator(t)

	invalidModel := &models.GameModel{
		GameID:      "test-game",
		Version:     1,
		LastUpdated: time.Now(),
		CurrentTurn: nil, // Invalid: nil CurrentTurn
		Units:         make(map[string]*models.UnitModel),
		TaskForces:    make(map[string]*models.TaskForceModel),
		EnemyContacts: []*models.EnemyContactModel{},
		Events:        []*models.GameEventModel{},
	}

	err := validator.ValidateModel(invalidModel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "current_turn is required")
}

// TestGameModelValidator_ValidateModel_InvalidUnits тестирует невалидные Units
func TestGameModelValidator_ValidateModel_InvalidUnits(t *testing.T) {
	validator := setupGameModelValidator(t)

	t.Run("nil unit", func(t *testing.T) {
		invalidModel := &models.GameModel{
			GameID:      "test-game",
			Version:     1,
			LastUpdated: time.Now(),
			CurrentTurn: &models.GameTurnModel{
				Turn:  1,
				Phase: models.PhaseMovement,
			},
			Units: map[string]*models.UnitModel{
				"unit1": nil, // Invalid: nil unit
			},
			TaskForces:    make(map[string]*models.TaskForceModel),
			EnemyContacts: []*models.EnemyContactModel{},
			Events:        []*models.GameEventModel{},
		}

		err := validator.ValidateModel(invalidModel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unit is nil")
	})

	t.Run("unit id mismatch", func(t *testing.T) {
		invalidModel := &models.GameModel{
			GameID:      "test-game",
			Version:     1,
			LastUpdated: time.Now(),
			CurrentTurn: &models.GameTurnModel{
				Turn:  1,
				Phase: models.PhaseMovement,
			},
			Units: map[string]*models.UnitModel{
				"unit1": {
					ID:     "unit2", // Invalid: ID mismatch
					GameID: "test-game",
					Name:   "Test Unit",
					Category: models.UnitCategoryNaval,
					NavalData: &models.NavalUnitData{
						Fuel:       80,
						MaxFuel:    80,
						CurrentHull: 6,
						HullBoxes:  6,
					},
				},
			},
			TaskForces:    make(map[string]*models.TaskForceModel),
			EnemyContacts: []*models.EnemyContactModel{},
			Events:        []*models.GameEventModel{},
		}

		err := validator.ValidateModel(invalidModel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unit id mismatch")
	})

	t.Run("unit game_id mismatch", func(t *testing.T) {
		invalidModel := &models.GameModel{
			GameID:      "test-game",
			Version:     1,
			LastUpdated: time.Now(),
			CurrentTurn: &models.GameTurnModel{
				Turn:  1,
				Phase: models.PhaseMovement,
			},
			Units: map[string]*models.UnitModel{
				"unit1": {
					ID:     "unit1",
					GameID: "other-game", // Invalid: GameID mismatch
					Name:   "Test Unit",
					Category: models.UnitCategoryNaval,
					NavalData: &models.NavalUnitData{
						Fuel:       80,
						MaxFuel:    80,
						CurrentHull: 6,
						HullBoxes:  6,
					},
				},
			},
			TaskForces:    make(map[string]*models.TaskForceModel),
			EnemyContacts: []*models.EnemyContactModel{},
			Events:        []*models.GameEventModel{},
		}

		err := validator.ValidateModel(invalidModel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unit game_id mismatch")
	})
}

// TestGameModelValidator_ValidateModel_InvalidTaskForces тестирует невалидные TaskForces
func TestGameModelValidator_ValidateModel_InvalidTaskForces(t *testing.T) {
	validator := setupGameModelValidator(t)

	t.Run("nil task force", func(t *testing.T) {
		invalidModel := &models.GameModel{
			GameID:      "test-game",
			Version:     1,
			LastUpdated: time.Now(),
			CurrentTurn: &models.GameTurnModel{
				Turn:  1,
				Phase: models.PhaseMovement,
			},
			Units: make(map[string]*models.UnitModel),
			TaskForces: map[string]*models.TaskForceModel{
				"tf1": nil, // Invalid: nil task force
			},
			EnemyContacts: []*models.EnemyContactModel{},
			Events:        []*models.GameEventModel{},
		}

		err := validator.ValidateModel(invalidModel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task_force is nil")
	})

	t.Run("task force id mismatch", func(t *testing.T) {
		invalidModel := &models.GameModel{
			GameID:      "test-game",
			Version:     1,
			LastUpdated: time.Now(),
			CurrentTurn: &models.GameTurnModel{
				Turn:  1,
				Phase: models.PhaseMovement,
			},
			Units: make(map[string]*models.UnitModel),
			TaskForces: map[string]*models.TaskForceModel{
				"tf1": {
					ID:       "tf2", // Invalid: ID mismatch
					GameID:   "test-game",
					Name:     "Test TF",
					Position: "A1",
					Nationality: "german",
				},
			},
			EnemyContacts: []*models.EnemyContactModel{},
			Events:        []*models.GameEventModel{},
		}

		err := validator.ValidateModel(invalidModel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task_force id mismatch")
	})

	t.Run("task force with non-existent unit", func(t *testing.T) {
		invalidModel := &models.GameModel{
			GameID:      "test-game",
			Version:     1,
			LastUpdated: time.Now(),
			CurrentTurn: &models.GameTurnModel{
				Turn:  1,
				Phase: models.PhaseMovement,
			},
			Units: make(map[string]*models.UnitModel),
			TaskForces: map[string]*models.TaskForceModel{
				"tf1": {
					ID:       "tf1",
					GameID:   "test-game",
					Name:     "Test TF",
					Position: "A1",
					Nationality: "german",
					Units:    []string{"non-existent-unit"}, // Invalid: unit doesn't exist
				},
			},
			EnemyContacts: []*models.EnemyContactModel{},
			Events:        []*models.GameEventModel{},
		}

		err := validator.ValidateModel(invalidModel)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "in task force does not exist")
	})
}

// TestGameModelValidator_ValidateModel_NilModel тестирует nil модель
func TestGameModelValidator_ValidateModel_NilModel(t *testing.T) {
	validator := setupGameModelValidator(t)

	err := validator.ValidateModel(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model is nil")
}

