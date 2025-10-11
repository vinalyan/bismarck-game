package handlers

import (
	"net/http"

	"bismarck-game/backend/internal/api/middleware"
	"bismarck-game/backend/internal/auth"
	"bismarck-game/backend/internal/game/models"
	"bismarck-game/backend/pkg/utils"

	"github.com/gorilla/mux"
)

// AuthHandler представляет обработчик аутентификации
type AuthHandler struct {
	authService *auth.AuthService
}

// NewAuthHandler создает новый обработчик аутентификации
func NewAuthHandler(authService *auth.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register регистрирует нового пользователя
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteValidationError(w, "Invalid request format", map[string]string{
			"body": "Request body must be valid JSON",
		})
		return
	}

	// Валидация полей
	if req.Username == "" {
		utils.WriteValidationError(w, "Username is required", map[string]string{
			"username": "Username cannot be empty",
		})
		return
	}

	if req.Email == "" {
		utils.WriteValidationError(w, "Email is required", map[string]string{
			"email": "Email cannot be empty",
		})
		return
	}

	if req.Password == "" {
		utils.WriteValidationError(w, "Password is required", map[string]string{
			"password": "Password cannot be empty",
		})
		return
	}

	if len(req.Password) < 6 {
		utils.WriteValidationError(w, "Password is too short", map[string]string{
			"password": "Password must be at least 6 characters long",
		})
		return
	}

	// Создаем пользователя
	user, err := h.authService.Register(&req)
	if err != nil {
		if err.Error() == "username already exists" {
			utils.WriteValidationError(w, "Username already exists", map[string]string{
				"username": "This username is already taken",
			})
			return
		}
		if err.Error() == "email already exists" {
			utils.WriteValidationError(w, "Email already exists", map[string]string{
				"email": "This email is already registered",
			})
			return
		}
		utils.WriteInternalError(w, "Failed to create user")
		return
	}

	// Возвращаем информацию о пользователе (без пароля)
	response := user.ToResponse()
	utils.WriteCreated(w, response)
}

// Login выполняет вход пользователя
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteValidationError(w, "Invalid request format", map[string]string{
			"body": "Request body must be valid JSON",
		})
		return
	}

	// Валидация полей
	if req.Username == "" {
		utils.WriteValidationError(w, "Username is required", map[string]string{
			"username": "Username cannot be empty",
		})
		return
	}

	if req.Password == "" {
		utils.WriteValidationError(w, "Password is required", map[string]string{
			"password": "Password cannot be empty",
		})
		return
	}

	// Выполняем вход
	user, token, err := h.authService.Login(&req)
	if err != nil {
		if err.Error() == "invalid credentials" {
			utils.WriteUnauthorized(w, "Invalid username or password")
			return
		}
		utils.WriteInternalError(w, "Login failed")
		return
	}

	// Возвращаем токен и информацию о пользователе
	response := map[string]interface{}{
		"user":  user.ToResponse(),
		"token": token,
	}

	utils.WriteSuccess(w, response)
}

// Logout выполняет выход пользователя
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Получаем токен из заголовка Authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		utils.WriteUnauthorized(w, "Authorization header required")
		return
	}

	// Извлекаем токен
	token := extractTokenFromHeader(authHeader)
	if token == "" {
		utils.WriteUnauthorized(w, "Invalid authorization header format")
		return
	}

	// Выполняем выход
	err := h.authService.Logout(token)
	if err != nil {
		utils.WriteInternalError(w, "Logout failed")
		return
	}

	utils.WriteSuccess(w, map[string]string{"message": "Logged out successfully"})
}

// GetProfile возвращает профиль текущего пользователя
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста (устанавливается middleware)
	userID := r.Context().Value("user_id").(string)

	// Получаем информацию о пользователе
	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		utils.WriteNotFound(w, "User not found")
		return
	}

	utils.WriteSuccess(w, user.ToResponse())
}

// UpdateProfile обновляет профиль пользователя
func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID := r.Context().Value("user_id").(string)

	var req models.UpdateUserRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteValidationError(w, "Invalid request format", map[string]string{
			"body": "Request body must be valid JSON",
		})
		return
	}

	// Обновляем пользователя
	user, err := h.authService.UpdateUser(userID, &req)
	if err != nil {
		utils.WriteInternalError(w, "Failed to update profile")
		return
	}

	utils.WriteSuccess(w, user.ToResponse())
}

// ChangePassword меняет пароль пользователя
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста
	userID := r.Context().Value("user_id").(string)

	var req models.ChangePasswordRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteValidationError(w, "Invalid request format", map[string]string{
			"body": "Request body must be valid JSON",
		})
		return
	}

	// Валидация полей
	if req.CurrentPassword == "" {
		utils.WriteValidationError(w, "Current password is required", map[string]string{
			"current_password": "Current password cannot be empty",
		})
		return
	}

	if req.NewPassword == "" {
		utils.WriteValidationError(w, "New password is required", map[string]string{
			"new_password": "New password cannot be empty",
		})
		return
	}

	if len(req.NewPassword) < 6 {
		utils.WriteValidationError(w, "New password is too short", map[string]string{
			"new_password": "New password must be at least 6 characters long",
		})
		return
	}

	// Меняем пароль
	err := h.authService.ChangePassword(userID, &req)
	if err != nil {
		if err.Error() == "current password is incorrect" {
			utils.WriteValidationError(w, "Current password is incorrect", map[string]string{
				"current_password": "The current password you entered is incorrect",
			})
			return
		}
		utils.WriteInternalError(w, "Failed to change password")
		return
	}

	utils.WriteSuccess(w, map[string]string{"message": "Password changed successfully"})
}

// ValidateToken валидирует токен
func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	// Получаем токен из заголовка Authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		utils.WriteUnauthorized(w, "Authorization header required")
		return
	}

	// Извлекаем токен
	token := extractTokenFromHeader(authHeader)
	if token == "" {
		utils.WriteUnauthorized(w, "Invalid authorization header format")
		return
	}

	// Валидируем токен
	user, err := h.authService.ValidateToken(token)
	if err != nil {
		utils.WriteUnauthorized(w, "Invalid or expired token")
		return
	}

	utils.WriteSuccess(w, user.ToResponse())
}

// extractTokenFromHeader извлекает токен из заголовка Authorization
func extractTokenFromHeader(authHeader string) string {
	// Проверяем формат "Bearer <token>"
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

// RegisterRoutes регистрирует маршруты аутентификации
func (h *AuthHandler) RegisterRoutes(router *mux.Router, jwtSecret string) {
	authRouter := router.PathPrefix("/api/auth").Subrouter()

	// Публичные маршруты
	authRouter.HandleFunc("/register", h.Register).Methods("POST")
	authRouter.HandleFunc("/login", h.Login).Methods("POST")
	authRouter.HandleFunc("/validate", h.ValidateToken).Methods("GET")

	// Защищенные маршруты (требуют аутентификации)
	protectedRouter := authRouter.PathPrefix("").Subrouter()
	protectedRouter.Use(middleware.AuthMiddleware(jwtSecret))

	protectedRouter.HandleFunc("/logout", h.Logout).Methods("POST")
	protectedRouter.HandleFunc("/profile", h.GetProfile).Methods("GET")
	protectedRouter.HandleFunc("/profile", h.UpdateProfile).Methods("PUT")
	protectedRouter.HandleFunc("/change-password", h.ChangePassword).Methods("POST")
}
