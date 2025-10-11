package middleware

import (
	"context"
	"net/http"
	"strings"

	"bismarck-game/backend/pkg/logger"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
)

// Claims представляет JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// AuthMiddleware создает middleware для аутентификации
func AuthMiddleware(jwtSecret string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем токен из заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Проверяем формат "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Парсим и валидируем токен
			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				// Проверяем метод подписи
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil {
				logger.Warn("JWT validation failed", "error", err.Error())
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			if !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Извлекаем данные из claims
			userID, _ := claims["user_id"].(string)
			username, _ := claims["username"].(string)

			// Добавляем информацию о пользователе в контекст
			ctx := context.WithValue(r.Context(), "user_id", userID)
			ctx = context.WithValue(ctx, "username", username)
			ctx = context.WithValue(ctx, "claims", claims)

			// Передаем управление следующему обработчику
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware создает middleware для опциональной аутентификации
func OptionalAuthMiddleware(jwtSecret string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем токен из заголовка Authorization
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// Если токена нет, просто продолжаем без аутентификации
				next.ServeHTTP(w, r)
				return
			}

			// Проверяем формат "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				// Если формат неправильный, продолжаем без аутентификации
				next.ServeHTTP(w, r)
				return
			}

			tokenString := parts[1]

			// Парсим и валидируем токен
			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				// Если токен невалидный, продолжаем без аутентификации
				next.ServeHTTP(w, r)
				return
			}

			// Извлекаем данные из claims
			userID, _ := claims["user_id"].(string)
			username, _ := claims["username"].(string)

			// Добавляем информацию о пользователе в контекст
			ctx := context.WithValue(r.Context(), "user_id", userID)
			ctx = context.WithValue(ctx, "username", username)
			ctx = context.WithValue(ctx, "claims", claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext извлекает ID пользователя из контекста
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value("user_id").(string)
	return userID, ok
}

// GetUsernameFromContext извлекает имя пользователя из контекста
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value("username").(string)
	return username, ok
}

// GetClaimsFromContext извлекает claims из контекста
func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value("claims").(*Claims)
	return claims, ok
}

// RequireAuth проверяет, что пользователь аутентифицирован
func RequireAuth(w http.ResponseWriter, r *http.Request) bool {
	_, ok := GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return false
	}
	return true
}
