#!/bin/bash

echo "Тестирование создания игры..."
echo "============================="

# Базовые настройки
BASE_URL="http://localhost:8080/api"

# Функция для проверки сервера
check_server() {
    echo "🔍 Проверка состояния сервера..."
    response=$(curl -s -o /dev/null -w "%{http_code}" ${BASE_URL}/health)
    
    if [ "$response" = "200" ]; then
        echo "✅ Сервер работает"
        return 0
    else
        echo "❌ Сервер не отвечает (код: $response)"
        return 1
    fi
}

# Функция для проверки авторизации
check_user_auth() {
    local username=$1
    local password=$2
    
    response=$(curl -s -X POST ${BASE_URL}/auth/login \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$username\",\"password\":\"$password\"}")
    
    if echo "$response" | grep -q '"success":true'; then
        echo "✅ $username авторизуется успешно"
        return 0
    else
        error=$(echo "$response" | jq -r '.error // .message // "Unknown error"')
        echo "❌ $username: $error"
        return 1
    fi
}

# Проверяем сервер
if ! check_server; then
    echo ""
    echo "Запустите сервер командой:"
    echo "  cd /Users/vikozhemyakin/Documents/Projects/bismarck-game/bismarck-game/backend"
    echo "  ./main"
    exit 1
fi

echo ""
echo "👥 Проверка пользователей..."
if ! check_user_auth "player1" "test123"; then
    echo "Зарегистрируйте пользователей командой: ./register_users.sh"
    exit 1
fi

if ! check_user_auth "player2" "test123"; then
    echo "Зарегистрируйте пользователей командой: ./register_users.sh"
    exit 1
fi

echo ""
echo "✅ Все готово для создания игры!"
echo ""
echo "Запустите создание игры командой:"
echo "  ./create_and_join_game.sh"
