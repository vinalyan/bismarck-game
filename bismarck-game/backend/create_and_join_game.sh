#!/bin/bash

echo "Создание игры и подключение к ней..."
echo "==================================="

# Базовые настройки
BASE_URL="http://localhost:8080/api"
GAME_NAME="Тестовая игра Bismarck"

# Функция для авторизации пользователя
login_user() {
    local username=$1
    local password=$2
    
    response=$(curl -s -X POST ${BASE_URL}/auth/login \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$username\",\"password\":\"$password\"}")
    
    if echo "$response" | grep -q '"success":true'; then
        token=$(echo "$response" | jq -r '.data.token')
        echo "$token"
    else
        echo ""
    fi
}

# Функция для создания игры
create_game() {
    local token=$1
    local game_name=$2
    
    # Создаем JSON с настройками игры по умолчанию
    local settings='{
        "use_optional_units": false,
        "enable_crew_exhaustion": false,
        "victory_conditions": {
            "bismarck_sunk_vp": -10,
            "bismarck_france_vp": -5,
            "bismarck_norway_vp": -7,
            "bismarck_end_game_vp": -10,
            "bismarck_no_fuel_vp": -15,
            "ship_vp_values": {},
            "convoy_vp": {}
        },
        "time_limit_minutes": 0,
        "private_lobby": false,
        "max_turn_time": 30,
        "allow_spectators": false,
        "auto_save": true,
        "difficulty": "normal"
    }'
    
    response=$(curl -s -X POST ${BASE_URL}/games \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "{\"name\":\"$game_name\",\"side\":\"german\",\"settings\":$settings}")
    
    if echo "$response" | grep -q '"success":true'; then
        game_id=$(echo "$response" | jq -r '.data.id')
        echo "$game_id"
    else
        echo ""
    fi
}

# Функция для присоединения к игре
join_game() {
    local token=$1
    local game_id=$2
    
    response=$(curl -s -X POST ${BASE_URL}/games/${game_id}/join \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d '{"side":"allied"}')
    
    if echo "$response" | grep -q '"success":true'; then
        echo "success"
    else
        error=$(echo "$response" | jq -r '.error // .message // "Unknown error"')
        echo "error: $error"
    fi
}

# Функция для получения информации о игре
get_game_info() {
    local token=$1
    local game_id=$2
    
    response=$(curl -s -X GET ${BASE_URL}/games/${game_id} \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token")
    
    if echo "$response" | grep -q '"success":true'; then
        echo "$response" | jq '.data'
    else
        echo "Ошибка получения информации о игре"
    fi
}

echo ""
echo "🔐 Авторизация player1 (немцы)..."
player1_token=$(login_user "player1" "test123")
if [ -z "$player1_token" ]; then
    echo "❌ Не удалось авторизовать player1"
    echo "Убедитесь, что пользователь player1 зарегистрирован и сервер запущен"
    exit 1
fi
echo "✅ player1 авторизован"

echo ""
echo "🎮 Создание игры за немцев..."
game_id=$(create_game "$player1_token" "$GAME_NAME")
if [ -z "$game_id" ]; then
    echo "❌ Не удалось создать игру"
    exit 1
fi
echo "✅ Игра создана: $game_id"

echo ""
echo "🔐 Авторизация player2 (союзники)..."
player2_token=$(login_user "player2" "test123")
if [ -z "$player2_token" ]; then
    echo "❌ Не удалось авторизовать player2"
    echo "Убедитесь, что пользователь player2 зарегистрирован"
    exit 1
fi
echo "✅ player2 авторизован"

echo ""
echo "🤝 Присоединение к игре за союзников..."
join_result=$(join_game "$player2_token" "$game_id")
if [[ "$join_result" == "success" ]]; then
    echo "✅ player2 присоединился к игре"
else
    echo "❌ Ошибка присоединения: $join_result"
    exit 1
fi

echo ""
echo "📊 Информация об игре:"
echo "======================"
echo "ID игры: $game_id"
echo "Название: $GAME_NAME"
echo "Player1 (немцы): player1"
echo "Player2 (союзники): player2"

echo ""
echo "🎯 Получение подробной информации об игре..."
game_info=$(get_game_info "$player1_token" "$game_id")
echo "$game_info"

echo ""
echo "================================="
echo "✅ Игра успешно создана и настроена!"
echo "🔗 Можете открыть игру по ссылке: http://localhost:3000/game/$game_id"
echo "================================="
