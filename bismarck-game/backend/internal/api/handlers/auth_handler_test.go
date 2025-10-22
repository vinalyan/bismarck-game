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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthService is a mock implementation of AuthService
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(req *models.CreateUserRequest) (*models.User, error) {
	args := m.Called(req)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) Login(req *models.LoginRequest) (*models.User, string, error) {
	args := m.Called(req)
	return args.Get(0).(*models.User), args.String(1), args.Error(2)
}

func (m *MockAuthService) ValidateToken(token string) (*models.User, error) {
	args := m.Called(token)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) Logout(userID string) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockAuthService) GetProfile(userID string) (*models.User, error) {
	args := m.Called(userID)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) UpdateProfile(userID string, req *models.UpdateUserRequest) (*models.User, error) {
	args := m.Called(userID, req)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthService) ChangePassword(userID string, req *models.ChangePasswordRequest) error {
	args := m.Called(userID, req)
	return args.Error(0)
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    models.CreateUserRequest
		mockSetup      func(*MockAuthService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid registration",
			requestBody: models.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func(m *MockAuthService) {
				user := test.CreateTestUser()
				m.On("Register", mock.AnythingOfType("*models.CreateUserRequest")).Return(user, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Missing username",
			requestBody: models.CreateUserRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Username is required",
		},
		{
			name: "Missing email",
			requestBody: models.CreateUserRequest{
				Username: "testuser",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Email is required",
		},
		{
			name: "Missing password",
			requestBody: models.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Password is required",
		},
		{
			name: "Password too short",
			requestBody: models.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Password is too short",
		},
		{
			name: "Duplicate username",
			requestBody: models.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Register", mock.AnythingOfType("*models.CreateUserRequest")).Return((*models.User)(nil), assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to create user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockAuthService := &MockAuthService{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockAuthService)
			}

			// Create handler
			handler := NewAuthHandler(mockAuthService)

			// Create request
			jsonBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handler.Register(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				assert.Contains(t, response["message"], tt.expectedError)
			}

			// Verify mock expectations
			mockAuthService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    models.LoginRequest
		mockSetup      func(*MockAuthService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid login",
			requestBody: models.LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			mockSetup: func(m *MockAuthService) {
				user := test.CreateTestUser()
				m.On("Login", mock.AnythingOfType("*models.LoginRequest")).Return(user, "test-token", nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Missing username",
			requestBody: models.LoginRequest{
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Username is required",
		},
		{
			name: "Missing password",
			requestBody: models.LoginRequest{
				Username: "testuser",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Password is required",
		},
		{
			name: "Invalid credentials",
			requestBody: models.LoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Login", mock.AnythingOfType("*models.LoginRequest")).Return((*models.User)(nil), "", assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Login failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockAuthService := &MockAuthService{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockAuthService)
			}

			// Create handler
			handler := NewAuthHandler(mockAuthService)

			// Create request
			jsonBody, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handler.Login(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				assert.Contains(t, response["message"], tt.expectedError)
			}

			// Verify mock expectations
			mockAuthService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		mockSetup      func(*MockAuthService)
		expectedStatus int
	}{
		{
			name:   "Valid logout",
			userID: "test-user-id",
			mockSetup: func(m *MockAuthService) {
				m.On("Logout", "test-user-id").Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Logout error",
			userID: "test-user-id",
			mockSetup: func(m *MockAuthService) {
				m.On("Logout", "test-user-id").Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockAuthService := &MockAuthService{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockAuthService)
			}

			// Create handler
			handler := NewAuthHandler(mockAuthService)

			// Create request with user context
			req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
			ctx := context.WithValue(req.Context(), "user_id", tt.userID)
			req = req.WithContext(ctx)

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handler.Logout(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify mock expectations
			mockAuthService.AssertExpectations(t)
		})
	}
}
