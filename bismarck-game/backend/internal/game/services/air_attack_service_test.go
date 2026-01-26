package services

import (
	"bismarck-game/backend/internal/game/models"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAirAttackService_AddAirAttackMarker(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	// Создаем AirAttackService
	airAttackService := NewAirAttackService(testServices.DB, testServices.Logger, testServices.GameService)
	airAttackService.SetGameStateService(testServices.GameStateService)

	// Создаем тестовую игру
	gameID := uuid.New().String()
	playerID := uuid.New().String()
	player2ID := uuid.New().String()

	// Clean up before test - удаляем существующих пользователей, чтобы избежать конфликтов
	_, err = testServices.DB.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)

	// Delete games that reference users with these usernames first to avoid foreign key constraint violation
	_, err = testServices.DB.GetConnection().Exec(`
		DELETE FROM games 
		WHERE player1_id IN (SELECT id FROM users WHERE username IN ('player1', 'player2'))
		   OR player2_id IN (SELECT id FROM users WHERE username IN ('player1', 'player2'))
	`)
	require.NoError(t, err)
	// Delete users by username to ensure they can be created with the specified IDs
	_, err = testServices.DB.GetConnection().Exec("DELETE FROM users WHERE username IN ('player1', 'player2')")
	require.NoError(t, err)

	// Создаем тестовых пользователей
	_, err = testServices.DB.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2')",
		playerID, player2ID,
	)
	require.NoError(t, err)

	// Создаем тестовую игру
	_, err = testServices.DB.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	// Создаем GameModel
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	hexID := "F26"

	t.Run("add single marker for german player", func(t *testing.T) {
		// Очищаем маркеры перед тестом
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureAirAttackInitialized()
			if model.AirAttack.German != nil {
				delete(model.AirAttack.German, hexID)
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Добавляем маркер
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID)
		assert.NoError(t, err)

		// Проверяем, что маркер добавлен
		markers, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Equal(t, 1, markers[hexID])
	})

	t.Run("add multiple markers in same hex", func(t *testing.T) {
		// Очищаем маркеры перед тестом
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureAirAttackInitialized()
			if model.AirAttack.German != nil {
				delete(model.AirAttack.German, hexID)
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Добавляем первый маркер
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID)
		require.NoError(t, err)

		// Добавляем второй маркер в тот же гекс
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID)
		assert.NoError(t, err)

		// Проверяем, что оба маркера добавлены
		markers, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Equal(t, 2, markers[hexID])
	})

	t.Run("add markers in different hexes", func(t *testing.T) {
		hexID1 := "F26"
		hexID2 := "G27"

		// Очищаем маркеры перед тестом
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureAirAttackInitialized()
			if model.AirAttack.German != nil {
				delete(model.AirAttack.German, hexID1)
				delete(model.AirAttack.German, hexID2)
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Добавляем маркеры в разные гексы
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID1)
		require.NoError(t, err)

		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID2)
		require.NoError(t, err)

		// Проверяем, что маркеры добавлены в оба гекса
		markers, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Equal(t, 1, markers[hexID1])
		assert.Equal(t, 1, markers[hexID2])
	})

	t.Run("add marker for allied player", func(t *testing.T) {
		hexID := "K27"

		// Очищаем маркеры перед тестом
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureAirAttackInitialized()
			if model.AirAttack.Allied != nil {
				delete(model.AirAttack.Allied, hexID)
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Добавляем маркер для союзного игрока (player2ID)
		err = airAttackService.AddAirAttackMarker(gameID, player2ID, hexID)
		assert.NoError(t, err)

		// Проверяем, что маркер добавлен для союзной стороны
		markers, err := airAttackService.GetAirAttackMarkers(gameID, player2ID)
		require.NoError(t, err)
		assert.Equal(t, 1, markers[hexID])

		// Проверяем, что немецкий игрок не видит маркеры союзников
		germanMarkers, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Equal(t, 0, germanMarkers[hexID])
	})

	t.Run("error when gameStateService is nil", func(t *testing.T) {
		serviceWithoutState := NewAirAttackService(testServices.DB, testServices.Logger, testServices.GameService)
		// Не устанавливаем gameStateService

		err := serviceWithoutState.AddAirAttackMarker(gameID, playerID, hexID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gameStateService is required")
	})

	t.Run("error when player is not in game", func(t *testing.T) {
		unknownPlayerID := uuid.New().String()

		err := airAttackService.AddAirAttackMarker(gameID, unknownPlayerID, hexID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "player is not part of this game")
	})
}

func TestAirAttackService_RemoveAirAttackMarker(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	// Создаем AirAttackService
	airAttackService := NewAirAttackService(testServices.DB, testServices.Logger, testServices.GameService)
	airAttackService.SetGameStateService(testServices.GameStateService)

	// Создаем тестовую игру
	gameID := uuid.New().String()
	playerID := uuid.New().String()
	player2ID := uuid.New().String()

	// Clean up before test - удаляем существующих пользователей, чтобы избежать конфликтов
	_, err = testServices.DB.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)

	// Delete games that reference users with these usernames first to avoid foreign key constraint violation
	_, err = testServices.DB.GetConnection().Exec(`
		DELETE FROM games 
		WHERE player1_id IN (SELECT id FROM users WHERE username IN ('player1', 'player2'))
		   OR player2_id IN (SELECT id FROM users WHERE username IN ('player1', 'player2'))
	`)
	require.NoError(t, err)
	// Delete users by username to ensure they can be created with the specified IDs
	_, err = testServices.DB.GetConnection().Exec("DELETE FROM users WHERE username IN ('player1', 'player2')")
	require.NoError(t, err)

	// Создаем тестовых пользователей
	_, err = testServices.DB.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2')",
		playerID, player2ID,
	)
	require.NoError(t, err)

	// Создаем тестовую игру
	_, err = testServices.DB.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	// Создаем GameModel
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	hexID := "F26"

	t.Run("remove single marker", func(t *testing.T) {
		// Добавляем маркер
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID)
		require.NoError(t, err)

		// Удаляем маркер
		err = airAttackService.RemoveAirAttackMarker(gameID, playerID, hexID)
		assert.NoError(t, err)

		// Проверяем, что маркер удален
		markers, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Equal(t, 0, markers[hexID])
	})

	t.Run("remove one marker when multiple exist", func(t *testing.T) {
		// Добавляем два маркера
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID)
		require.NoError(t, err)
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID)
		require.NoError(t, err)

		// Удаляем один маркер
		err = airAttackService.RemoveAirAttackMarker(gameID, playerID, hexID)
		assert.NoError(t, err)

		// Проверяем, что остался один маркер
		markers, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Equal(t, 1, markers[hexID])
	})

	t.Run("remove marker when none exists", func(t *testing.T) {
		hexIDEmpty := "A1"

		// Очищаем маркеры перед тестом
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureAirAttackInitialized()
			if model.AirAttack.German != nil {
				delete(model.AirAttack.German, hexIDEmpty)
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Пытаемся удалить несуществующий маркер
		err = airAttackService.RemoveAirAttackMarker(gameID, playerID, hexIDEmpty)
		assert.NoError(t, err) // Должно вернуть nil, а не ошибку

		// Проверяем, что маркеров нет
		markers, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Equal(t, 0, markers[hexIDEmpty])
	})

	t.Run("error when gameStateService is nil", func(t *testing.T) {
		serviceWithoutState := NewAirAttackService(testServices.DB, testServices.Logger, testServices.GameService)
		// Не устанавливаем gameStateService

		err := serviceWithoutState.RemoveAirAttackMarker(gameID, playerID, hexID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gameStateService is required")
	})
}

func TestAirAttackService_GetAirAttackMarkers(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

	// Создаем AirAttackService
	airAttackService := NewAirAttackService(testServices.DB, testServices.Logger, testServices.GameService)
	airAttackService.SetGameStateService(testServices.GameStateService)

	// Создаем тестовую игру
	gameID := uuid.New().String()
	playerID := uuid.New().String()
	player2ID := uuid.New().String()

	// Clean up before test - удаляем существующих пользователей, чтобы избежать конфликтов
	_, err = testServices.DB.GetConnection().Exec("DELETE FROM games WHERE id = $1", gameID)
	require.NoError(t, err)

	// Delete games that reference users with these usernames first to avoid foreign key constraint violation
	_, err = testServices.DB.GetConnection().Exec(`
		DELETE FROM games 
		WHERE player1_id IN (SELECT id FROM users WHERE username IN ('player1', 'player2'))
		   OR player2_id IN (SELECT id FROM users WHERE username IN ('player1', 'player2'))
	`)
	require.NoError(t, err)
	// Delete users by username to ensure they can be created with the specified IDs
	_, err = testServices.DB.GetConnection().Exec("DELETE FROM users WHERE username IN ('player1', 'player2')")
	require.NoError(t, err)

	// Создаем тестовых пользователей
	_, err = testServices.DB.GetConnection().Exec(
		"INSERT INTO users (id, username, email, password_hash) VALUES ($1, 'player1', 'p1@test.com', 'hash1'), ($2, 'player2', 'p2@test.com', 'hash2')",
		playerID, player2ID,
	)
	require.NoError(t, err)

	// Создаем тестовую игру
	_, err = testServices.DB.GetConnection().Exec(
		"INSERT INTO games (id, name, status, player1_id, player2_id) VALUES ($1, 'Test Game', 'active', $2, $3)",
		gameID, playerID, player2ID,
	)
	require.NoError(t, err)

	// Создаем GameModel
	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseMovement)
	require.NoError(t, err)

	t.Run("get empty markers", func(t *testing.T) {
		// Очищаем маркеры перед тестом
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureAirAttackInitialized()
			if model.AirAttack.German != nil {
				model.AirAttack.German = make(map[string]int)
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Получаем маркеры
		markers, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Empty(t, markers)
	})

	t.Run("get markers with multiple hexes", func(t *testing.T) {
		hexID1 := "F26"
		hexID2 := "G27"
		hexID3 := "H28"

		// Очищаем маркеры перед тестом
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureAirAttackInitialized()
			if model.AirAttack.German != nil {
				model.AirAttack.German = make(map[string]int)
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Добавляем маркеры в разные гексы
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID1)
		require.NoError(t, err)
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID1)
		require.NoError(t, err) // 2 маркера в hexID1

		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID2)
		require.NoError(t, err) // 1 маркер в hexID2

		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID3)
		require.NoError(t, err)
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID3)
		require.NoError(t, err)
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID3)
		require.NoError(t, err) // 3 маркера в hexID3

		// Получаем маркеры
		markers, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Equal(t, 2, markers[hexID1])
		assert.Equal(t, 1, markers[hexID2])
		assert.Equal(t, 3, markers[hexID3])
		assert.Equal(t, 3, len(markers))
	})

	t.Run("get markers returns copy", func(t *testing.T) {
		hexID := "F26"

		// Очищаем маркеры перед тестом
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureAirAttackInitialized()
			if model.AirAttack.German != nil {
				model.AirAttack.German = make(map[string]int)
			}
			return nil
		}, 3)
		require.NoError(t, err)

		// Добавляем маркер
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID)
		require.NoError(t, err)

		// Получаем маркеры
		markers, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)

		// Модифицируем копию
		markers["new_hex"] = 5

		// Проверяем, что оригинал не изменился
		markers2, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Equal(t, 1, markers2[hexID])
		assert.Equal(t, 0, markers2["new_hex"])
	})

	t.Run("error when gameStateService is nil", func(t *testing.T) {
		serviceWithoutState := NewAirAttackService(testServices.DB, testServices.Logger, testServices.GameService)
		// Не устанавливаем gameStateService

		markers, err := serviceWithoutState.GetAirAttackMarkers(gameID, playerID)
		assert.Error(t, err)
		assert.Nil(t, markers)
		assert.Contains(t, err.Error(), "gameStateService is required")
	})

	t.Run("error when player is not in game", func(t *testing.T) {
		unknownPlayerID := uuid.New().String()

		markers, err := airAttackService.GetAirAttackMarkers(gameID, unknownPlayerID)
		assert.Error(t, err)
		assert.Nil(t, markers)
		assert.Contains(t, err.Error(), "player is not part of this game")
	})
}
