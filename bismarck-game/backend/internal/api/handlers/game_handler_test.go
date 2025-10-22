package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/test"
	"bismarck-game/backend/pkg/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUnitService is a mock implementation of UnitService
type MockUnitService struct {
	mock.Mock
}

func (m *MockUnitService) GetUnitsByGameID(gameID string) ([]*models.Ship, error) {
	args := m.Called(gameID)
	return args.Get(0).([]*models.Ship), args.Error(1)
}

func (m *MockUnitService) CreateUnit(unit *models.Ship) error {
	args := m.Called(unit)
	return args.Error(0)
}

func (m *MockUnitService) UpdateUnit(unit *models.Ship) error {
	args := m.Called(unit)
	return args.Error(0)
}

func (m *MockUnitService) DeleteUnit(unitID string) error {
	args := m.Called(unitID)
	return args.Error(0)
}

func (m *MockUnitService) MoveUnit(unitID, fromHex, toHex string) error {
	args := m.Called(unitID, fromHex, toHex)
	return args.Error(0)
}

// MockShipConfigService is a mock implementation of ShipConfigService
type MockShipConfigService struct {
	mock.Mock
}

func (m *MockShipConfigService) GetShipConfig(shipID string) (*models.ShipTemplate, error) {
	args := m.Called(shipID)
	return args.Get(0).(*models.ShipTemplate), args.Error(1)
}

func (m *MockShipConfigService) GetAllShipConfigs() ([]*models.ShipTemplate, error) {
	args := m.Called()
	return args.Get(0).([]*models.ShipTemplate), args.Error(1)
}

// MockPhaseManager is a mock implementation of PhaseManager
type MockPhaseManager struct {
	mock.Mock
}

func (m *MockPhaseManager) StartTurn(gameID string) error {
	args := m.Called(gameID)
	return args.Error(0)
}

func (m *MockPhaseManager) EndTurn(gameID string) error {
	args := m.Called(gameID)
	return args.Error(0)
}

func (m *MockPhaseManager) NextPhase(gameID string) error {
	args := m.Called(gameID)
	return args.Error(0)
}

func (m *MockPhaseManager) GetCurrentPhase(gameID string) (models.GamePhase, error) {
	args := m.Called(gameID)
	return args.Get(0).(models.GamePhase), args.Error(1)
}

func TestGameHandler_CreateGame(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    models.CreateGameRequest
		userID         string
		mockSetup      func(*database.Database, sqlmock.Sqlmock)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid game creation - German side",
			requestBody: models.CreateGameRequest{
				Name: "Test Game",
				Side: models.PlayerSideGerman,
			},
			userID: "test-user-id",
			mockSetup: func(db *database.Database, mock sqlmock.Sqlmock) {
				mock.ExpectQuery("INSERT INTO games").
					WithArgs("Test Game", "test-user-id", nil, 1, models.PhaseWaiting, models.GameStatusWaiting, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("test-game-id"))
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Valid game creation - Allied side",
			requestBody: models.CreateGameRequest{
				Name: "Test Game",
				Side: models.PlayerSideAllied,
			},
			userID: "test-user-id",
			mockSetup: func(db *database.Database, mock sqlmock.Sqlmock) {
				mock.ExpectQuery("INSERT INTO games").
					WithArgs("Test Game", nil, "test-user-id", 1, models.PhaseWaiting, models.GameStatusWaiting, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("test-game-id"))
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Missing game name",
			requestBody: models.CreateGameRequest{
				Side: models.PlayerSideGerman,
			},
			userID:         "test-user-id",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Game name is required",
		},
		{
			name: "Game name too short",
			requestBody: models.CreateGameRequest{
				Name: "AB",
				Side: models.PlayerSideGerman,
			},
			userID:         "test-user-id",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid game name length",
		},
		{
			name: "Game name too long",
			requestBody: models.CreateGameRequest{
				Name: "This is a very long game name that exceeds the maximum allowed length of 100 characters and should be rejected by the validation",
				Side: models.PlayerSideGerman,
			},
			userID:         "test-user-id",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid game name length",
		},
		{
			name: "Missing side",
			requestBody: models.CreateGameRequest{
				Name: "Test Game",
			},
			userID:         "test-user-id",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Player side is required",
		},
		{
			name: "Invalid side",
			requestBody: models.CreateGameRequest{
				Name: "Test Game",
				Side: "invalid",
			},
			userID:         "test-user-id",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid player side",
		},
		{
			name: "Unauthorized - no user ID",
			requestBody: models.CreateGameRequest{
				Name: "Test Game",
				Side: models.PlayerSideGerman,
			},
			userID:         "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authentication required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock database
			db, mock := test.MockDatabase(t)
			if tt.mockSetup != nil {
				tt.mockSetup(db, mock)
			}

			// Create mock services
			mockUnitService := &MockUnitService{}
			mockShipConfigService := &MockShipConfigService{}
			mockPhaseManager := &MockPhaseManager{}

			// Create handler
			handler := NewGameHandler(db, mockUnitService, mockShipConfigService, mockPhaseManager)

			// Create request
			jsonBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// Add user context if userID is provided
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), "user_id", tt.userID)
				req = req.WithContext(ctx)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handler.CreateGame(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				assert.Contains(t, response["message"], tt.expectedError)
			}

			// Verify mock expectations
			if tt.mockSetup != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

func TestGameHandler_JoinGame(t *testing.T) {
	tests := []struct {
		name           string
		gameID         string
		userID         string
		requestBody    models.JoinGameRequest
		mockSetup      func(*database.Database, sqlmock.Sqlmock)
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "Valid join game",
			gameID: "test-game-id",
			userID: "test-user-id",
			requestBody: models.JoinGameRequest{
				Side: models.PlayerSideAllied,
			},
			mockSetup: func(db *database.Database, mock sqlmock.Sqlmock) {
				// Mock game query
				rows := sqlmock.NewRows([]string{"id", "name", "player1_id", "player2_id", "current_turn", "current_phase", "status", "settings", "created_at", "updated_at", "completed_at", "player1_username"}).
					AddRow("test-game-id", "Test Game", "existing-player-id", nil, 1, models.PhaseWaiting, models.GameStatusWaiting, `{}`, "2023-01-01T00:00:00Z", "2023-01-01T00:00:00Z", nil, "existing-player")

				mock.ExpectQuery("SELECT g.id, g.name, g.player1_id, g.player2_id").
					WithArgs("test-game-id").
					WillReturnRows(rows)

				// Mock update query
				mock.ExpectExec("UPDATE games SET player2_id").
					WithArgs("test-user-id", "test-game-id").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Game not found",
			gameID: "non-existent-game",
			userID: "test-user-id",
			requestBody: models.JoinGameRequest{
				Side: models.PlayerSideAllied,
			},
			mockSetup: func(db *database.Database, mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT g.id, g.name, g.player1_id, g.player2_id").
					WithArgs("non-existent-game").
					WillReturnError(sqlmock.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "Game not found",
		},
		{
			name:   "Game already full",
			gameID: "test-game-id",
			userID: "test-user-id",
			requestBody: models.JoinGameRequest{
				Side: models.PlayerSideAllied,
			},
			mockSetup: func(db *database.Database, mock sqlmock.Sqlmock) {
				// Mock game query with both players already set
				rows := sqlmock.NewRows([]string{"id", "name", "player1_id", "player2_id", "current_turn", "current_phase", "status", "settings", "created_at", "updated_at", "completed_at", "player1_username"}).
					AddRow("test-game-id", "Test Game", "player1-id", "player2-id", 1, models.PhaseWaiting, models.GameStatusWaiting, `{}`, "2023-01-01T00:00:00Z", "2023-01-01T00:00:00Z", nil, "player1")

				mock.ExpectQuery("SELECT g.id, g.name, g.player1_id, g.player2_id").
					WithArgs("test-game-id").
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Cannot join this game",
		},
		{
			name:   "Unauthorized - no user ID",
			gameID: "test-game-id",
			userID: "",
			requestBody: models.JoinGameRequest{
				Side: models.PlayerSideAllied,
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authentication required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock database
			db, mock := test.MockDatabase(t)
			if tt.mockSetup != nil {
				tt.mockSetup(db, mock)
			}

			// Create mock services
			mockUnitService := &MockUnitService{}
			mockShipConfigService := &MockShipConfigService{}
			mockPhaseManager := &MockPhaseManager{}

			// Create handler
			handler := NewGameHandler(db, mockUnitService, mockShipConfigService, mockPhaseManager)

			// Create request
			jsonBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/games/"+tt.gameID+"/join", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// Add user context if userID is provided
			if tt.userID != "" {
				ctx := context.WithValue(req.Context(), "user_id", tt.userID)
				req = req.WithContext(ctx)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handler.JoinGame(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				assert.Contains(t, response["message"], tt.expectedError)
			}

			// Verify mock expectations
			if tt.mockSetup != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}
