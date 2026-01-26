package services

import (
	"bismarck-game/backend/internal/game/models"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAirAttackPhaseHandler_Start_NoMarkers(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	// Создаем тестовую игру
	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseAirAttack)
	require.NoError(t, err)

	// Устанавливаем текущую фазу в air_attack
	err = testServices.PhaseManager.StartPhase(gameID, 1, models.PhaseAirAttack)
	require.NoError(t, err)

	// Очищаем все маркеры воздушной атаки
	err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		model.EnsureAirAttackInitialized()
		model.AirAttack.German = make(map[string]int)
		model.AirAttack.Allied = make(map[string]int)
		return nil
	}, 3)
	require.NoError(t, err)

	// Создаем обработчик
	handler := &AirAttackPhaseHandler{}
	handler.SetPhaseManager(testServices.PhaseManager)

	// Запускаем фазу
	err = handler.Start(gameID, 1)
	require.NoError(t, err)

	// Проверяем, что фаза автоматически завершилась (перешла к следующей)
	// Подождем немного, чтобы горутина успела выполниться
	time.Sleep(100 * time.Millisecond)

	currentPhase, err := testServices.PhaseManager.GetCurrentPhase(gameID)
	require.NoError(t, err)
	// Фаза должна перейти к следующей после air_attack
	assert.NotEqual(t, models.PhaseAirAttack, currentPhase.CurrentPhase)
}

func TestAirAttackPhaseHandler_Start_WithMarkersButNoShips(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	// Создаем тестовую игру
	gameID := uuid.New().String()
	playerID := uuid.New().String()
	player2ID := uuid.New().String()

	// Создаем тестовых пользователей
	_, err = testServices.DB.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, $2, $3, 'hash1'), ($4, $5, $6, 'hash2')",
		playerID, fmt.Sprintf("player1_%s", playerID), fmt.Sprintf("p1_%s@test.com", playerID),
		player2ID, fmt.Sprintf("player2_%s", player2ID), fmt.Sprintf("p2_%s@test.com", player2ID),
	)
	require.NoError(t, err)

	// Создаем тестовую игру
	_, err = testServices.DB.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseAirAttack)
	require.NoError(t, err)

	// Устанавливаем текущую фазу в air_attack
	err = testServices.PhaseManager.StartPhase(gameID, 1, models.PhaseAirAttack)
	require.NoError(t, err)

	// Добавляем маркеры воздушной атаки в гекс БЕЗ кораблей
	hexID := "A1" // Пустой гекс без кораблей
	err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		model.EnsureAirAttackInitialized()
		model.AirAttack.German = make(map[string]int)
		model.AirAttack.German[hexID] = 2 // 2 маркера в пустом гексе
		return nil
	}, 3)
	require.NoError(t, err)

	// Создаем обработчик
	handler := &AirAttackPhaseHandler{}
	handler.SetPhaseManager(testServices.PhaseManager)

	// Запускаем фазу
	err = handler.Start(gameID, 1)
	require.NoError(t, err)

	// Проверяем, что фаза автоматически завершилась (т.к. маркеры в гексе без кораблей игнорируются)
	time.Sleep(100 * time.Millisecond)

	currentPhase, err := testServices.PhaseManager.GetCurrentPhase(gameID)
	require.NoError(t, err)
	// Фаза должна перейти к следующей после air_attack
	assert.NotEqual(t, models.PhaseAirAttack, currentPhase.CurrentPhase)
}

func TestAirAttackPhaseHandler_Start_WithMarkersAndShips(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	// Создаем тестовую игру
	gameID := uuid.New().String()
	playerID := uuid.New().String()
	player2ID := uuid.New().String()

	// Создаем тестовых пользователей
	_, err = testServices.DB.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, $2, $3, 'hash1'), ($4, $5, $6, 'hash2')",
		playerID, fmt.Sprintf("player1_%s", playerID), fmt.Sprintf("p1_%s@test.com", playerID),
		player2ID, fmt.Sprintf("player2_%s", player2ID), fmt.Sprintf("p2_%s@test.com", player2ID),
	)
	require.NoError(t, err)

	// Создаем тестовую игру
	_, err = testServices.DB.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseAirAttack)
	require.NoError(t, err)

	// Создаем вражеский корабль союзников в гексе
	enemyUnitID := uuid.New().String()
	enemyUnit := &models.UnitModel{
		ID:          enemyUnitID,
		GameID:      gameID,
		Name:        "Enemy Allied Ship",
		Type:        models.UnitTypeBattleship,
		Category:    models.UnitCategoryNaval,
		Owner:       player2ID,
		Nationality: "allied",
		Position:    "F26",
		Status:      string(models.UnitStatusActive),
		Visibility:  models.VisibilityShadowed,
		NavalData: &models.NavalUnitData{
			Fuel:        10,
			MaxFuel:     18,
			IsActivated: false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = AddTestUnitToGameModel(testServices.GameStateService, gameID, enemyUnit)
	require.NoError(t, err)

	// Устанавливаем текущую фазу в air_attack
	err = testServices.PhaseManager.StartPhase(gameID, 1, models.PhaseAirAttack)
	require.NoError(t, err)

	// Добавляем маркеры воздушной атаки немцев в гекс С кораблями союзников
	hexID := "F26"
	err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
		model.EnsureAirAttackInitialized()
		model.AirAttack.German = make(map[string]int)
		model.AirAttack.German[hexID] = 2 // 2 маркера в гексе с кораблями
		return nil
	}, 3)
	require.NoError(t, err)

	// Создаем обработчик
	handler := &AirAttackPhaseHandler{}
	handler.SetPhaseManager(testServices.PhaseManager)

	// Запускаем фазу
	err = handler.Start(gameID, 1)
	require.NoError(t, err)

	// Проверяем, что фаза НЕ завершилась автоматически (есть маркеры в гексе с кораблями)
	time.Sleep(100 * time.Millisecond)

	currentPhase, err := testServices.PhaseManager.GetCurrentPhase(gameID)
	require.NoError(t, err)
	// Фаза должна остаться в air_attack, так как есть маркеры
	assert.Equal(t, models.PhaseAirAttack, currentPhase.CurrentPhase)
}

func TestAirAttackPhaseHandler_hasEnemyShipsInHex(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	// Создаем тестовую игру
	gameID := uuid.New().String()
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseAirAttack)
	require.NoError(t, err)

	handler := &AirAttackPhaseHandler{}

	t.Run("returns true when enemy naval unit exists", func(t *testing.T) {
		hexID := "F26"

		// Создаем вражеский корабль
		enemyUnitID := uuid.New().String()
		enemyUnit := &models.UnitModel{
			ID:          enemyUnitID,
			GameID:      gameID,
			Name:        "Enemy Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "enemy_player",
			Nationality: "allied",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        10,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, enemyUnit)
		require.NoError(t, err)

		// Загружаем модель
		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем наличие вражеских кораблей (allied)
		result := handler.hasEnemyShipsInHex(model, hexID, "allied")
		assert.True(t, result)
	})

	t.Run("returns false when no enemy ships", func(t *testing.T) {
		hexID := "G27"

		// Загружаем модель
		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем наличие вражеских кораблей в пустом гексе
		result := handler.hasEnemyShipsInHex(model, hexID, "allied")
		assert.False(t, result)
	})

	t.Run("returns false when ship is sunk", func(t *testing.T) {
		hexID := "H28"

		// Создаем потопленный корабль
		sunkUnitID := uuid.New().String()
		sunkUnit := &models.UnitModel{
			ID:          sunkUnitID,
			GameID:      gameID,
			Name:        "Sunk Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "enemy_player",
			Nationality: "allied",
			Position:    hexID,
			Status:      string(models.UnitStatusSunk),
			NavalData: &models.NavalUnitData{
				Fuel:        0,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, sunkUnit)
		require.NoError(t, err)

		// Загружаем модель
		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем наличие вражеских кораблей - потопленные не считаются
		result := handler.hasEnemyShipsInHex(model, hexID, "allied")
		assert.False(t, result)
	})

	t.Run("returns false when unit is not naval", func(t *testing.T) {
		hexID := "I29"

		// Создаем воздушный юнит (не морской)
		airUnitID := uuid.New().String()
		airUnit := &models.UnitModel{
			ID:          airUnitID,
			GameID:      gameID,
			Name:        "Air Unit",
			Type:        models.UnitTypeCombatAircraft,
			Category:    models.UnitCategoryAir,
			Owner:       "enemy_player",
			Nationality: "allied",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			AirData: &models.AirUnitData{
				BasePosition: hexID, // Базовая позиция (авиабаза/авианосец)
				IsActivated:  false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, airUnit)
		require.NoError(t, err)

		// Загружаем модель
		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем наличие вражеских кораблей - воздушные юниты не считаются
		result := handler.hasEnemyShipsInHex(model, hexID, "allied")
		assert.False(t, result)
	})

	t.Run("returns true when enemy task force exists", func(t *testing.T) {
		hexID := "J30"

		// Создаем вражеский корабль для Task Force
		tfUnitID := uuid.New().String()
		tfUnit := &models.UnitModel{
			ID:          tfUnitID,
			GameID:      gameID,
			Name:        "TF Unit",
			Type:        models.UnitTypeHeavyCruiser,
			Category:    models.UnitCategoryNaval,
			Owner:       "enemy_player",
			Nationality: "allied",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        10,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, tfUnit)
		require.NoError(t, err)

		// Создаем Task Force с этим кораблем
		tfID := uuid.New().String()
		tf := &models.TaskForceModel{
			ID:          tfID,
			GameID:      gameID,
			Name:        "Enemy TF",
			Owner:       "enemy_player",
			Nationality: "allied",
			Position:    hexID,
			Visibility:  models.VisibilityShadowed,
			Units:       []string{tfUnitID},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err = AddTestTaskForceToGameModel(testServices.GameStateService, gameID, tf)
		require.NoError(t, err)

		// Загружаем модель
		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем наличие вражеских кораблей в Task Force
		result := handler.hasEnemyShipsInHex(model, hexID, "allied")
		assert.True(t, result)
	})

	t.Run("returns false when task force has only sunk ships", func(t *testing.T) {
		hexID := "K31"

		// Создаем потопленный корабль для Task Force
		tfSunkUnitID := uuid.New().String()
		tfSunkUnit := &models.UnitModel{
			ID:          tfSunkUnitID,
			GameID:      gameID,
			Name:        "TF Sunk Unit",
			Type:        models.UnitTypeHeavyCruiser,
			Category:    models.UnitCategoryNaval,
			Owner:       "enemy_player",
			Nationality: "allied",
			Position:    hexID,
			Status:      string(models.UnitStatusSunk),
			NavalData: &models.NavalUnitData{
				Fuel:        0,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, tfSunkUnit)
		require.NoError(t, err)

		// Создаем Task Force только с потопленным кораблем
		tfID2 := uuid.New().String()
		tf2 := &models.TaskForceModel{
			ID:          tfID2,
			GameID:      gameID,
			Name:        "TF With Sunk Ships",
			Owner:       "enemy_player",
			Nationality: "allied",
			Position:    hexID,
			Visibility:  models.VisibilityShadowed,
			Units:       []string{tfSunkUnitID},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err = AddTestTaskForceToGameModel(testServices.GameStateService, gameID, tf2)
		require.NoError(t, err)

		// Загружаем модель
		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем наличие вражеских кораблей - Task Force с потопленными кораблями не считается
		result := handler.hasEnemyShipsInHex(model, hexID, "allied")
		assert.False(t, result)
	})

	t.Run("returns false when nationality does not match", func(t *testing.T) {
		hexID := "L32"

		// Создаем немецкий корабль (не союзный)
		germanUnitID := uuid.New().String()
		germanUnit := &models.UnitModel{
			ID:          germanUnitID,
			GameID:      gameID,
			Name:        "German Ship",
			Type:        models.UnitTypeBattleship,
			Category:    models.UnitCategoryNaval,
			Owner:       "german_player",
			Nationality: "german",
			Position:    hexID,
			Status:      string(models.UnitStatusActive),
			NavalData: &models.NavalUnitData{
				Fuel:        10,
				MaxFuel:     18,
				IsActivated: false,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err = AddTestUnitToGameModel(testServices.GameStateService, gameID, germanUnit)
		require.NoError(t, err)

		// Загружаем модель
		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)

		// Проверяем наличие вражеских кораблей (allied) - немецкий корабль не считается
		result := handler.hasEnemyShipsInHex(model, hexID, "allied")
		assert.False(t, result)
	})
}

func TestAirAttackPhaseHandler_GetName(t *testing.T) {
	handler := &AirAttackPhaseHandler{}
	assert.Equal(t, "Воздушная атака", handler.GetName())
}

func TestAirAttackPhaseHandler_GetDescription(t *testing.T) {
	handler := &AirAttackPhaseHandler{}
	assert.Equal(t, "Атаки с воздуха", handler.GetDescription())
}

func TestAirAttackPhaseHandler_CanStart(t *testing.T) {
	handler := &AirAttackPhaseHandler{}
	canStart, err := handler.CanStart("game_id", 1)
	assert.NoError(t, err)
	assert.True(t, canStart)
}

func TestAirAttackPhaseHandler_CanComplete(t *testing.T) {
	handler := &AirAttackPhaseHandler{}
	canComplete, err := handler.CanComplete("game_id", 1)
	assert.NoError(t, err)
	assert.True(t, canComplete)
}
