package auth

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/database"
	"bismarck-game/backend/pkg/logger"
	"bismarck-game/backend/pkg/redis"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// AuthService представляет сервис аутентификации
type AuthService struct {
	db        *database.Database
	redis     *redis.Client
	jwtSecret string
	jwtExpiry time.Duration
}

// New создает новый сервис аутентификации
func New(db *database.Database, redis *redis.Client, jwtSecret string, jwtExpiry time.Duration) *AuthService {
	return &AuthService{
		db:        db,
		redis:     redis,
		jwtSecret: jwtSecret,
		jwtExpiry: jwtExpiry,
	}
}

// Register регистрирует нового пользователя
func (s *AuthService) Register(req *models.CreateUserRequest) (*models.User, error) {
	ctx := context.Background()
	// Проверяем, существует ли пользователь с таким именем
	var count int
	err := s.db.GetConnection().QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE username = $1", req.Username).Scan(&count)
	if err != nil {
		logger.Error("Failed to check username existence", "error", err)
		return nil, fmt.Errorf("failed to check username: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("username already exists")
	}

	// Проверяем, существует ли пользователь с таким email
	err = s.db.GetConnection().QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE email = $1", req.Email).Scan(&count)
	if err != nil {
		logger.Error("Failed to check email existence", "error", err)
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("email already exists")
	}

	// Хешируем пароль
	hashedPassword, err := s.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Сохраняем в базу данных
	query := `
		INSERT INTO users (username, email, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	var user models.User
	err = s.db.GetConnection().QueryRowContext(ctx, query,
		req.Username,
		req.Email,
		hashedPassword,
		time.Now(),
		time.Now(),
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	user.Username = req.Username
	user.Email = req.Email
	user.PasswordHash = hashedPassword
	user.Role = models.RolePlayer
	user.Stats = models.GetDefaultUserStats()
	user.IsActive = true

	if err != nil {
		logger.Error("Failed to create user", "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	logger.Info("User registered successfully", "user_id", user.ID, "username", user.Username)
	return &user, nil
}

// Login выполняет вход пользователя
func (s *AuthService) Login(req *models.LoginRequest) (*models.User, string, error) {
	ctx := context.Background()
	// Находим пользователя по имени пользователя или email
	var user models.User
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at, last_login
		FROM users 
		WHERE username = $1 OR email = $1
	`

	err := s.db.GetConnection().QueryRowContext(ctx, query, req.Username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", fmt.Errorf("invalid credentials")
		}
		logger.Error("Failed to find user", "error", err)
		return nil, "", fmt.Errorf("failed to find user: %w", err)
	}

	// Проверяем пароль
	if !s.CheckPassword(req.Password, user.PasswordHash) {
		return nil, "", fmt.Errorf("invalid credentials")
	}

	// Генерируем JWT токен
	token, err := s.GenerateToken(&user)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Обновляем время последнего входа
	now := time.Now()
	_, err = s.db.Exec("UPDATE users SET last_login = $1, updated_at = $2 WHERE id = $3",
		now, now, user.ID)
	if err != nil {
		logger.Warn("Failed to update last login", "error", err)
	}

	// Сохраняем сессию в Redis
	err = s.redis.SetSession(user.ID, token, s.jwtExpiry)
	if err != nil {
		logger.Warn("Failed to save session to Redis", "error", err)
	}

	// Сохраняем сессию в базу данных
	session := &models.UserSession{
		UserID:    user.ID,
		TokenHash: s.hashToken(token),
		ExpiresAt: now.Add(s.jwtExpiry),
		CreatedAt: now,
		IsActive:  true,
	}

	_, err = s.db.Exec(`
		INSERT INTO user_sessions (user_id, token_hash, expires_at, created_at, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`, session.UserID, session.TokenHash, session.ExpiresAt, session.CreatedAt, session.IsActive)

	if err != nil {
		logger.Warn("Failed to save session to database", "error", err)
	}

	logger.Info("User logged in successfully", "user_id", user.ID, "username", user.Username)
	return &user, token, nil
}

// Logout выполняет выход пользователя
func (s *AuthService) Logout(token string) error {
	// Удаляем сессию из Redis
	err := s.redis.DeleteSession(token)
	if err != nil {
		logger.Warn("Failed to delete session from Redis", "error", err)
	}

	// Деактивируем сессию в базе данных
	tokenHash := s.hashToken(token)
	_, err = s.db.Exec("UPDATE user_sessions SET is_active = false WHERE token_hash = $1", tokenHash)
	if err != nil {
		logger.Warn("Failed to deactivate session in database", "error", err)
	}

	return nil
}

// ValidateToken валидирует JWT токен
func (s *AuthService) ValidateToken(token string) (*models.User, error) {
	ctx := context.Background()
	// Парсим токен
	claims := jwt.MapClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !parsedToken.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Проверяем, что токен не истек
	if exp, ok := claims["exp"].(float64); ok {
		if time.Unix(int64(exp), 0).Before(time.Now()) {
			return nil, fmt.Errorf("token expired")
		}
	}

	// Получаем пользователя из базы данных
	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		return nil, fmt.Errorf("invalid token: missing user_id")
	}

	var user models.User
	query := `
		SELECT id, username, email, created_at, updated_at, last_login
		FROM users 
		WHERE id = $1
	`

	err = s.db.GetConnection().QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.Username, &user.Email,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Проверяем сессию в Redis
	_, err = s.redis.GetSession(token)
	if err != nil {
		return nil, fmt.Errorf("session not found or expired")
	}

	return &user, nil
}

// GenerateToken генерирует JWT токен для пользователя
func (s *AuthService) GenerateToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(s.jwtExpiry).Unix(),
		"nbf":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// HashPassword хеширует пароль
func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword проверяет пароль
func (s *AuthService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// hashToken хеширует токен для хранения в базе данных
func (s *AuthService) hashToken(token string) string {
	hashed, _ := s.HashPassword(token)
	return hashed
}

// GetUserByID получает пользователя по ID
func (s *AuthService) GetUserByID(userID string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, username, email, created_at, updated_at, last_login
		FROM users 
		WHERE id = $1
	`

	err := s.db.GetConnection().QueryRowContext(context.Background(), query, userID).Scan(
		&user.ID, &user.Username, &user.Email,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLogin,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// UpdateUser обновляет информацию о пользователе
func (s *AuthService) UpdateUser(userID string, req *models.UpdateUserRequest) (*models.User, error) {
	// Строим запрос динамически в зависимости от переданных полей
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Username != nil {
		setParts = append(setParts, fmt.Sprintf("username = $%d", argIndex))
		args = append(args, *req.Username)
		argIndex++
	}

	if req.Email != nil {
		setParts = append(setParts, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, *req.Email)
		argIndex++
	}

	if len(setParts) == 0 {
		return s.GetUserByID(userID)
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	args = append(args, userID)

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d",
		strings.Join(setParts, ", "), argIndex)

	_, err := s.db.GetConnection().ExecContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return s.GetUserByID(userID)
}

// ChangePassword меняет пароль пользователя
func (s *AuthService) ChangePassword(userID string, req *models.ChangePasswordRequest) error {
	// Получаем текущий хеш пароля
	var currentHash string
	err := s.db.GetConnection().QueryRowContext(context.Background(), "SELECT password_hash FROM users WHERE id = $1", userID).Scan(&currentHash)
	if err != nil {
		return fmt.Errorf("failed to get current password: %w", err)
	}

	// Проверяем текущий пароль
	if !s.CheckPassword(req.CurrentPassword, currentHash) {
		return fmt.Errorf("current password is incorrect")
	}

	// Хешируем новый пароль
	newHash, err := s.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Обновляем пароль
	_, err = s.db.GetConnection().ExecContext(context.Background(), "UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3",
		newHash, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	logger.Info("Password changed successfully", "user_id", userID)
	return nil
}

// CleanupExpiredSessions удаляет истекшие сессии
func (s *AuthService) CleanupExpiredSessions() error {
	// Удаляем истекшие сессии из базы данных
	_, err := s.db.Exec("DELETE FROM user_sessions WHERE expires_at < NOW() OR is_active = false")
	if err != nil {
		logger.Warn("Failed to cleanup expired sessions from database", "error", err)
	}

	logger.Info("Expired sessions cleaned up")
	return nil
}
