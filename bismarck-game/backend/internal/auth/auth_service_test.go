package auth

import (
	"database/sql"
	"testing"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/internal/test"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.CreateUserRequest
		mockSetup     func(sqlmock.Sqlmock)
		expectedError string
		expectedUser  bool
	}{
		{
			name: "Valid registration",
			request: &models.CreateUserRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Mock user creation
				// Mock username check
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users WHERE username = \\$1").
					WithArgs("testuser").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

				// Mock email check
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users WHERE email = \\$1").
					WithArgs("test@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

				// Mock user creation
				mock.ExpectQuery("INSERT INTO users").
					WithArgs("testuser", "test@example.com", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("test-user-id", time.Now(), time.Now()))
			},
			expectedUser: true,
		},
		{
			name: "Duplicate username",
			request: &models.CreateUserRequest{
				Username: "existinguser",
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Mock username check - user exists
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM users WHERE username = \\$1").
					WithArgs("existinguser").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
			expectedError: "username already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock database
			db, mock := test.MockDatabaseWrapper(t)
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			// Create mock Redis
			redisClient := test.MockRedis(t)

			// Create auth service
			config := test.GetTestConfig()
			authService := New(db, redisClient, config.JWTSecret, config.JWTExpiry)

			// Call Register
			user, err := authService.Register(tt.request)

			// Assert results
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				if tt.expectedUser {
					assert.NotNil(t, user)
					assert.Equal(t, tt.request.Username, user.Username)
					assert.Equal(t, tt.request.Email, user.Email)
					assert.NotEmpty(t, user.ID)
				}
			}

			// Verify mock expectations
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.LoginRequest
		mockSetup     func(sqlmock.Sqlmock)
		expectedError string
		expectedUser  bool
		expectedToken bool
	}{
		{
			name: "Valid login",
			request: &models.LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Mock user lookup
				rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "created_at", "updated_at", "last_login"}).
					AddRow("test-user-id", "testuser", "test@example.com", "$2a$10$hashedpassword", time.Now(), time.Now(), nil)

				mock.ExpectQuery("SELECT id, username, email, password_hash, created_at, updated_at, last_login FROM users WHERE username = \\$1 OR email = \\$1").
					WithArgs("testuser").
					WillReturnRows(rows)

				// Mock last login update
				mock.ExpectExec("UPDATE users SET last_login = \\$1, updated_at = \\$2 WHERE id = \\$3").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "test-user-id").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedUser:  true,
			expectedToken: true,
		},
		{
			name: "User not found",
			request: &models.LoginRequest{
				Username: "nonexistent",
				Password: "password123",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, username, email, password_hash, created_at, updated_at, last_login FROM users WHERE username = \\$1 OR email = \\$1").
					WithArgs("nonexistent").
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: "invalid credentials",
		},
		{
			name: "Invalid password",
			request: &models.LoginRequest{
				Username: "testuser",
				Password: "wrongpassword",
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Mock user lookup with different password hash
				rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "created_at", "updated_at", "last_login"}).
					AddRow("test-user-id", "testuser", "test@example.com", "$2a$10$differenthash", time.Now(), time.Now(), nil)

				mock.ExpectQuery("SELECT id, username, email, password_hash, created_at, updated_at, last_login FROM users WHERE username = \\$1 OR email = \\$1").
					WithArgs("testuser").
					WillReturnRows(rows)
			},
			expectedError: "invalid credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock database
			db, mock := test.MockDatabaseWrapper(t)
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			// Create mock Redis
			redisClient := test.MockRedis(t)

			// Create auth service
			config := test.GetTestConfig()
			authService := New(db, redisClient, config.JWTSecret, config.JWTExpiry)

			// Call Login
			user, token, err := authService.Login(tt.request)

			// Assert results
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				if tt.expectedUser {
					assert.NotNil(t, user)
					assert.Equal(t, tt.request.Username, user.Username)
				}
				if tt.expectedToken {
					assert.NotEmpty(t, token)
					// Verify token is valid JWT
					_, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
						return []byte(config.JWTSecret), nil
					})
					assert.NoError(t, err)
				}
			}

			// Verify mock expectations
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		mockSetup     func(sqlmock.Sqlmock)
		expectedError string
		expectedUser  bool
	}{
		{
			name:  "Valid token",
			token: "valid-jwt-token",
			mockSetup: func(mock sqlmock.Sqlmock) {
				// Mock user lookup by ID
				rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "created_at", "updated_at", "last_login"}).
					AddRow("test-user-id", "testuser", "test@example.com", "$2a$10$hashedpassword", time.Now(), time.Now(), nil)

				mock.ExpectQuery("SELECT id, username, email, password_hash, created_at, updated_at, last_login FROM users WHERE id = \\$1").
					WithArgs("test-user-id").
					WillReturnRows(rows)
			},
			expectedUser: true,
		},
		{
			name:          "Invalid token",
			token:         "invalid-token",
			expectedError: "invalid token",
		},
		{
			name:          "Empty token",
			token:         "",
			expectedError: "invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock database
			db, mock := test.MockDatabaseWrapper(t)
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			// Create mock Redis
			redisClient := test.MockRedis(t)

			// Create auth service
			config := test.GetTestConfig()
			authService := New(db, redisClient, config.JWTSecret, config.JWTExpiry)

			// For valid token test, create a real JWT token
			if tt.name == "Valid token" {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id": "test-user-id",
					"exp":     time.Now().Add(24 * time.Hour).Unix(),
				})
				tt.token, _ = token.SignedString([]byte(config.JWTSecret))
			}

			// Call ValidateToken
			user, err := authService.ValidateToken(tt.token)

			// Assert results
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				if tt.expectedUser {
					assert.NotNil(t, user)
					assert.Equal(t, "test-user-id", user.ID)
				}
			}

			// Verify mock expectations
			if tt.mockSetup != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		expectedError string
	}{
		{
			name:   "Valid logout",
			userID: "test-user-id",
		},
		{
			name:          "Empty user ID",
			userID:        "",
			expectedError: "user ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock database
			db, mock := test.MockDatabaseWrapper(t)

			// Create mock Redis
			redisClient := test.MockRedis(t)

			// Create auth service
			config := test.GetTestConfig()
			authService := New(db, redisClient, config.JWTSecret, config.JWTExpiry)

			// Call Logout
			err := authService.Logout(tt.userID)

			// Assert results
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
