package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/redis"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

// TestConfig represents test configuration
type TestConfig struct {
	JWTSecret string
	JWTExpiry time.Duration
}

// GetTestConfig returns default test configuration
func GetTestConfig() *TestConfig {
	return &TestConfig{
		JWTSecret: "test-secret-key",
		JWTExpiry: 24 * time.Hour,
	}
}

// MockDatabase creates a mock database for testing
func MockDatabase(t *testing.T) (MockDatabaseInterface, sqlmock.Sqlmock) {
	return CreateMockDatabase(t)
}

// MockDatabaseWrapper creates a wrapper that implements database.Database interface
func MockDatabaseWrapper(t *testing.T) (*database.Database, sqlmock.Sqlmock) {
	mockDB, mock := CreateMockDatabase(t)

	// Create a real database.Database instance with the mock connection
	db := database.NewForTesting(mockDB.GetConnection())

	return db, mock
}

// MockRedis creates a mock Redis client for testing
func MockRedis(t *testing.T) *redis.Client {
	// For unit tests, we'll use a simple in-memory mock
	// In integration tests, we'll use testcontainers
	return &redis.Client{}
}

// CreateTestUser creates a test user
func CreateTestUser() *models.User {
	return &models.User{
		ID:        "test-user-id",
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// CreateTestGame creates a test game
func CreateTestGame() *models.Game {
	return &models.Game{
		ID:           "test-game-id",
		Name:         "Test Game",
		Player1ID:    "test-user-id",
		Player2ID:    "",
		CurrentTurn:  1,
		CurrentPhase: models.PhaseWaiting,
		Status:       models.GameStatusWaiting,
		Settings:     models.GetDefaultGameSettings(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// CreateTestRequest creates an HTTP request for testing
func CreateTestRequest(method, url string, body interface{}) *http.Request {
	var reqBody io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req := httptest.NewRequest(method, url, reqBody)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// CreateAuthenticatedRequest creates an authenticated HTTP request
func CreateAuthenticatedRequest(method, url string, body interface{}, userID string, jwtSecret string) *http.Request {
	req := CreateTestRequest(method, url, body)

	// Create JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, _ := token.SignedString([]byte(jwtSecret))
	req.Header.Set("Authorization", "Bearer "+tokenString)

	// Add user_id to context
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	return req
}

// AssertJSONResponse asserts that the response contains expected JSON
func AssertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedBody interface{}) {
	assert.Equal(t, expectedStatus, w.Code)

	if expectedBody != nil {
		var actualBody, expectedJSON interface{}
		json.Unmarshal(w.Body.Bytes(), &actualBody)
		json.Unmarshal([]byte(fmt.Sprintf("%v", expectedBody)), &expectedJSON)
		assert.Equal(t, expectedJSON, actualBody)
	}
}

// AssertErrorResponse asserts that the response contains an error
func AssertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	assert.Equal(t, expectedStatus, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if expectedMessage != "" {
		assert.Contains(t, response["message"], expectedMessage)
	}
}

// SetupTestDatabase sets up a test database using testcontainers
func SetupTestDatabase(t *testing.T) *database.Database {
	// This would be implemented with testcontainers in integration tests
	// For now, return nil as we'll use mocks for unit tests
	return nil
}

// CleanupTestDatabase cleans up test database
func CleanupTestDatabase(t *testing.T, db *database.Database) {
	// Cleanup logic for test database
}

// SetupTestRedis sets up a test Redis instance
func SetupTestRedis(t *testing.T) *redis.Client {
	// This would be implemented with testcontainers in integration tests
	// For now, return mock for unit tests
	return MockRedis(t)
}

// CleanupTestRedis cleans up test Redis
func CleanupTestRedis(t *testing.T, redis *redis.Client) {
	// Cleanup logic for test Redis
}
