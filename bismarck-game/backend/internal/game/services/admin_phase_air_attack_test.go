package services

import (
	"bismarck-game/backend/internal/game/models"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminPhaseHandler_Start_RemovesAirAttackMarkers(t *testing.T) {
	testServices, cleanup, err := SetupTestServices()
	require.NoError(t, err)
	defer cleanup()

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

	_, err = CreateTestGameModel(testServices.DB, testServices.GameStateService, gameID, 1, models.PhaseAdmin)
	require.NoError(t, err)

	// Создаем AirAttackService для добавления маркеров
	airAttackService := NewAirAttackService(testServices.DB, testServices.Logger, testServices.GameService)
	airAttackService.SetGameStateService(testServices.GameStateService)

	hexID1 := "F26"
	hexID2 := "G27"

	t.Run("removes all german air attack markers", func(t *testing.T) {
		// Очищаем маркеры перед тестом
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureAirAttackInitialized()
			model.AirAttack.German = make(map[string]int)
			model.AirAttack.Allied = make(map[string]int)
			return nil
		}, 3)
		require.NoError(t, err)

		// Добавляем маркеры для немецкого игрока
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID1)
		require.NoError(t, err)
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID1)
		require.NoError(t, err) // 2 маркера в hexID1

		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID2)
		require.NoError(t, err) // 1 маркер в hexID2

		// Проверяем, что маркеры добавлены
		germanMarkers, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Equal(t, 2, germanMarkers[hexID1])
		assert.Equal(t, 1, germanMarkers[hexID2])

		// Создаем AdminPhaseHandler и устанавливаем зависимости
		handler := NewAdminPhaseHandler(testServices.UnitService, testServices.TaskForceService, testServices.SearchService)
		handler.SetGameStateService(testServices.GameStateService)
		handler.SetPhaseManager(testServices.PhaseManager)

		// Запускаем фазу администрирования
		err = handler.Start(gameID, 1)
		require.NoError(t, err)

		// Проверяем, что все маркеры удалены
		germanMarkersAfter, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Empty(t, germanMarkersAfter, "All german air attack markers should be removed in admin phase")
	})

	t.Run("removes all allied air attack markers", func(t *testing.T) {
		// Очищаем маркеры перед тестом
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureAirAttackInitialized()
			model.AirAttack.German = make(map[string]int)
			model.AirAttack.Allied = make(map[string]int)
			return nil
		}, 3)
		require.NoError(t, err)

		// Добавляем маркеры для союзного игрока
		err = airAttackService.AddAirAttackMarker(gameID, player2ID, hexID1)
		require.NoError(t, err)
		err = airAttackService.AddAirAttackMarker(gameID, player2ID, hexID2)
		require.NoError(t, err)
		err = airAttackService.AddAirAttackMarker(gameID, player2ID, hexID2)
		require.NoError(t, err) // 2 маркера в hexID2

		// Проверяем, что маркеры добавлены
		alliedMarkers, err := airAttackService.GetAirAttackMarkers(gameID, player2ID)
		require.NoError(t, err)
		assert.Equal(t, 1, alliedMarkers[hexID1])
		assert.Equal(t, 2, alliedMarkers[hexID2])

		// Создаем AdminPhaseHandler и устанавливаем зависимости
		handler := NewAdminPhaseHandler(testServices.UnitService, testServices.TaskForceService, testServices.SearchService)
		handler.SetGameStateService(testServices.GameStateService)
		handler.SetPhaseManager(testServices.PhaseManager)

		// Запускаем фазу администрирования
		err = handler.Start(gameID, 1)
		require.NoError(t, err)

		// Проверяем, что все маркеры удалены
		alliedMarkersAfter, err := airAttackService.GetAirAttackMarkers(gameID, player2ID)
		require.NoError(t, err)
		assert.Empty(t, alliedMarkersAfter, "All allied air attack markers should be removed in admin phase")
	})

	t.Run("removes all air attack markers from both sides", func(t *testing.T) {
		// Очищаем маркеры перед тестом
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureAirAttackInitialized()
			model.AirAttack.German = make(map[string]int)
			model.AirAttack.Allied = make(map[string]int)
			return nil
		}, 3)
		require.NoError(t, err)

		// Добавляем маркеры для обеих сторон
		err = airAttackService.AddAirAttackMarker(gameID, playerID, hexID1)
		require.NoError(t, err)

		err = airAttackService.AddAirAttackMarker(gameID, player2ID, hexID2)
		require.NoError(t, err)

		// Проверяем, что маркеры добавлены
		germanMarkers, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Equal(t, 1, germanMarkers[hexID1])

		alliedMarkers, err := airAttackService.GetAirAttackMarkers(gameID, player2ID)
		require.NoError(t, err)
		assert.Equal(t, 1, alliedMarkers[hexID2])

		// Создаем AdminPhaseHandler и устанавливаем зависимости
		handler := NewAdminPhaseHandler(testServices.UnitService, testServices.TaskForceService, testServices.SearchService)
		handler.SetGameStateService(testServices.GameStateService)
		handler.SetPhaseManager(testServices.PhaseManager)

		// Запускаем фазу администрирования
		err = handler.Start(gameID, 1)
		require.NoError(t, err)

		// Проверяем, что все маркеры удалены с обеих сторон
		germanMarkersAfter, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Empty(t, germanMarkersAfter, "All german air attack markers should be removed in admin phase")

		alliedMarkersAfter, err := airAttackService.GetAirAttackMarkers(gameID, player2ID)
		require.NoError(t, err)
		assert.Empty(t, alliedMarkersAfter, "All allied air attack markers should be removed in admin phase")
	})

	t.Run("handles empty air attack markers gracefully", func(t *testing.T) {
		// Очищаем маркеры перед тестом
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.EnsureAirAttackInitialized()
			model.AirAttack.German = make(map[string]int)
			model.AirAttack.Allied = make(map[string]int)
			return nil
		}, 3)
		require.NoError(t, err)

		// НЕ добавляем маркеры

		// Создаем AdminPhaseHandler и устанавливаем зависимости
		handler := NewAdminPhaseHandler(testServices.UnitService, testServices.TaskForceService, testServices.SearchService)
		handler.SetGameStateService(testServices.GameStateService)
		handler.SetPhaseManager(testServices.PhaseManager)

		// Запускаем фазу администрирования - не должно быть ошибок даже при отсутствии маркеров
		err = handler.Start(gameID, 1)
		require.NoError(t, err)

		// Проверяем, что маркеры остаются пустыми
		germanMarkers, err := airAttackService.GetAirAttackMarkers(gameID, playerID)
		require.NoError(t, err)
		assert.Empty(t, germanMarkers)

		alliedMarkers, err := airAttackService.GetAirAttackMarkers(gameID, player2ID)
		require.NoError(t, err)
		assert.Empty(t, alliedMarkers)
	})

	t.Run("removes markers even when AirAttack structure is nil", func(t *testing.T) {
		// Устанавливаем AirAttack в nil (не инициализирован)
		err = testServices.GameStateService.UpdateGameModelWithRetry(gameID, func(model *models.GameModel) error {
			model.AirAttack = nil
			return nil
		}, 3)
		require.NoError(t, err)

		// Создаем AdminPhaseHandler и устанавливаем зависимости
		handler := NewAdminPhaseHandler(testServices.UnitService, testServices.TaskForceService, testServices.SearchService)
		handler.SetGameStateService(testServices.GameStateService)
		handler.SetPhaseManager(testServices.PhaseManager)

		// Запускаем фазу администрирования - должно инициализировать AirAttack и очистить его
		err = handler.Start(gameID, 1)
		require.NoError(t, err)

		// Проверяем, что AirAttack инициализирован и пуст
		model, err := testServices.GameStateService.LoadGameModel(gameID)
		require.NoError(t, err)
		model.EnsureAirAttackInitialized()
		assert.NotNil(t, model.AirAttack)
		assert.Empty(t, model.AirAttack.German)
		assert.Empty(t, model.AirAttack.Allied)
	})
}
