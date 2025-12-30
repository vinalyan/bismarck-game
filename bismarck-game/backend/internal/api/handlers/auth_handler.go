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
// @Summary Регистрация пользователя
// @Tags Authentication
// @Accept json
// @Produce json
// @Param body body models.CreateUserRequest true "Данные для регистрации"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteValidationError(w, "Invalid request format", map[string]string{
			"body": "Request body must be valid JSON",
		})
		return
	}

	// Валидация полей
	if !utils.ValidateRequest(w, &req) {
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
// @Summary Вход в систему
// @Tags Authentication
// @Accept json
// @Produce json
// @Param body body models.LoginRequest true "Данные для входа"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteValidationError(w, "Invalid request format", map[string]string{
			"body": "Request body must be valid JSON",
		})
		return
	}

	// Валидация полей
	if !utils.ValidateRequest(w, &req) {
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
// @Summary Выход из системы
// @Tags Authentication
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/logout [post]
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
// @Summary Получение профиля пользователя
// @Tags Authentication
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/profile [get]
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста (устанавливается middleware)
	userIDValue := r.Context().Value("user_id")
	if userIDValue == nil {
		utils.WriteUnauthorized(w, "user not authenticated")
		return
	}
	userID, ok := userIDValue.(string)
	if !ok || userID == "" {
		utils.WriteUnauthorized(w, "user not authenticated")
		return
	}

	// Получаем информацию о пользователе
	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		utils.WriteNotFound(w, "user not found")
		return
	}

	utils.WriteSuccess(w, user.ToResponse())
}

// UpdateProfile обновляет профиль пользователя
// @Summary Обновление профиля пользователя
// @Tags Authentication
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body models.UpdateUserRequest true "Данные для обновления профиля"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/profile [put]
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
// @Summary Смена пароля
// @Tags Authentication
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body models.ChangePasswordRequest true "Данные для смены пароля"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/change-password [post]
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
	if !utils.ValidateRequest(w, &req) {
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
// @Summary Валидация токена
// @Tags Authentication
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/validate [get]
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

	// Добавляем OPTIONS обработчик для всех маршрутов
	authRouter.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Устанавливаем CORS заголовки
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:3000" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.WriteHeader(http.StatusOK)
	})

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
