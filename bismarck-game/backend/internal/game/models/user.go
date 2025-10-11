package models

import (
	"time"
)

// UserRole представляет роль пользователя
type UserRole string

const (
	RolePlayer    UserRole = "player"
	RoleAdmin     UserRole = "admin"
	RoleModerator UserRole = "moderator"
)

// User представляет пользователя системы
type User struct {
	ID           string     `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Role         UserRole   `json:"role" db:"role"`
	Stats        UserStats  `json:"stats" db:"stats"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	LastLogin    *time.Time `json:"last_login" db:"last_login"`
	IsActive     bool       `json:"is_active" db:"is_active"`
}

// UserStats представляет статистику пользователя
type UserStats struct {
	GamesPlayed  int     `json:"games_played"`
	GamesWon     int     `json:"games_won"`
	WinRate      float64 `json:"win_rate"`
	FavoriteSide string  `json:"favorite_side"`
	TotalVP      int     `json:"total_vp"`
	TimePlayed   int     `json:"time_played"` // в минутах
	Rank         int     `json:"rank"`
	Experience   int     `json:"experience"`
	Level        int     `json:"level"`
}

// UserPreferences представляет настройки пользователя
type UserPreferences struct {
	UserID          string    `json:"user_id" db:"user_id"`
	Theme           string    `json:"theme" db:"theme"`
	Language        string    `json:"language" db:"language"`
	Notifications   bool      `json:"notifications" db:"notifications"`
	SoundEnabled    bool      `json:"sound_enabled" db:"sound_enabled"`
	AutoSave        bool      `json:"auto_save" db:"auto_save"`
	ShowTutorials   bool      `json:"show_tutorials" db:"show_tutorials"`
	DefaultGameMode string    `json:"default_game_mode" db:"default_game_mode"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// UserSession представляет сессию пользователя
type UserSession struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	TokenHash string    `json:"-" db:"token_hash"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	IPAddress string    `json:"ip_address" db:"ip_address"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
	IsActive  bool      `json:"is_active" db:"is_active"`
}

// UserAchievement представляет достижение пользователя
type UserAchievement struct {
	ID          string    `json:"id" db:"id"`
	UserID      string    `json:"user_id" db:"user_id"`
	Achievement string    `json:"achievement" db:"achievement"`
	UnlockedAt  time.Time `json:"unlocked_at" db:"unlocked_at"`
	Progress    int       `json:"progress" db:"progress"`
	MaxProgress int       `json:"max_progress" db:"max_progress"`
}

// CreateUserRequest представляет запрос на создание пользователя
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// LoginRequest представляет запрос на вход
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// UpdateUserRequest представляет запрос на обновление пользователя
type UpdateUserRequest struct {
	Username *string `json:"username,omitempty" validate:"omitempty,min=3,max=50"`
	Email    *string `json:"email,omitempty" validate:"omitempty,email"`
}

// ChangePasswordRequest представляет запрос на смену пароля
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=6"`
}

// UserResponse представляет ответ с информацией о пользователе
type UserResponse struct {
	ID        string     `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Role      UserRole   `json:"role"`
	Stats     UserStats  `json:"stats"`
	CreatedAt time.Time  `json:"created_at"`
	LastLogin *time.Time `json:"last_login"`
	IsActive  bool       `json:"is_active"`
}

// ToResponse преобразует User в UserResponse
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		Stats:     u.Stats,
		CreatedAt: u.CreatedAt,
		LastLogin: u.LastLogin,
		IsActive:  u.IsActive,
	}
}

// CalculateWinRate вычисляет процент побед
func (s *UserStats) CalculateWinRate() {
	if s.GamesPlayed > 0 {
		s.WinRate = float64(s.GamesWon) / float64(s.GamesPlayed) * 100
	} else {
		s.WinRate = 0
	}
}

// CalculateLevel вычисляет уровень пользователя на основе опыта
func (s *UserStats) CalculateLevel() {
	// Простая формула: каждые 1000 опыта = 1 уровень
	s.Level = s.Experience/1000 + 1
}

// AddExperience добавляет опыт пользователю
func (s *UserStats) AddExperience(amount int) {
	s.Experience += amount
	s.CalculateLevel()
}

// IsValidRole проверяет, является ли роль валидной
func IsValidRole(role string) bool {
	switch UserRole(role) {
	case RolePlayer, RoleAdmin, RoleModerator:
		return true
	default:
		return false
	}
}

// GetDefaultUserStats возвращает статистику по умолчанию
func GetDefaultUserStats() UserStats {
	return UserStats{
		GamesPlayed:  0,
		GamesWon:     0,
		WinRate:      0,
		FavoriteSide: "",
		TotalVP:      0,
		TimePlayed:   0,
		Rank:         0,
		Experience:   0,
		Level:        1,
	}
}

// GetDefaultUserPreferences возвращает настройки по умолчанию
func GetDefaultUserPreferences() UserPreferences {
	return UserPreferences{
		Theme:           "dark",
		Language:        "en",
		Notifications:   true,
		SoundEnabled:    true,
		AutoSave:        true,
		ShowTutorials:   true,
		DefaultGameMode: "standard",
	}
}
