# Bismarck Game API Documentation

## Swagger UI

Документация API доступна через Swagger UI по следующим адресам:

- **Главная страница сервера**: http://localhost:8080/
- **Swagger UI**: http://localhost:8080/docs
- **Swagger JSON**: http://localhost:8080/docs/swagger.json

## Как использовать

1. **Запустите сервер**:
   ```bash
   cd backend
   go run cmd/server/main.go
   ```

2. **Откройте браузер** и перейдите по адресу http://localhost:8080/

3. **Нажмите на кнопку "📚 API Документация"** или перейдите напрямую на http://localhost:8080/docs

## Возможности Swagger UI

- **Интерактивное тестирование API** - можно выполнять запросы прямо из браузера
- **Авторизация** - поддержка JWT токенов через кнопку "Authorize"
- **Документация всех эндпоинтов** с примерами запросов и ответов
- **Схемы данных** - полное описание всех моделей данных

## Аутентификация

Для тестирования защищенных эндпоинтов:

1. Нажмите кнопку **"Authorize"** в Swagger UI
2. Введите JWT токен в формате: `Bearer your_jwt_token_here`
3. Нажмите **"Authorize"**

## Получение JWT токена

1. Выполните POST запрос на `/api/auth/login` с вашими учетными данными
2. Скопируйте токен из ответа
3. Используйте его для авторизации в Swagger UI

## Структура файлов

```
docs/
├── README.md          # Этот файл
├── swagger.json       # OpenAPI спецификация
└── swagger.html       # HTML страница с Swagger UI
```

## Обновление документации

Для обновления документации:

1. Отредактируйте файл `docs/swagger.json`
2. Перезапустите сервер
3. Обновите страницу в браузере

## Поддерживаемые эндпоинты

### Система
- `GET /health` - Проверка состояния сервера

### Аутентификация
- `POST /api/auth/register` - Регистрация пользователя
- `POST /api/auth/login` - Вход в систему

### Игры
- `GET /api/games` - Список игр
- `POST /api/games` - Создание игры
- `GET /api/games/{id}` - Информация об игре
- `POST /api/games/{id}/join` - Присоединение к игре

### WebSocket
- `GET /ws` - WebSocket соединение для real-time обновлений

## Примеры использования

### Регистрация пользователя
```bash
curl -X POST "http://localhost:8080/api/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }'
```

### Вход в систему
```bash
curl -X POST "http://localhost:8080/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123"
  }'
```

### Создание игры
```bash
curl -X POST "http://localhost:8080/api/games" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Моя игра",
    "settings": {
      "private_lobby": false,
      "time_limit_minutes": 180
    }
  }'
```

### Присоединение к игре
```bash
curl -X POST "http://localhost:8080/api/games/{game_id}/join" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "password": ""
  }'
```
