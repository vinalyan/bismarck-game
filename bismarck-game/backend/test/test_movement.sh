#!/bin/bash

# Основные тесты движения кораблей
BASE_URL="http://localhost:8080"

echo "🚢 ОСНОВНЫЕ ТЕСТЫ ДВИЖЕНИЯ КОРАБЛЕЙ"
echo "==================================="
echo ""

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Счетчики
TESTS_PASSED=0
TESTS_FAILED=0
TOTAL_TESTS=0

# Функция для логирования
log_test() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✅ $test_name${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}❌ $test_name${NC}"
        echo -e "${RED}   $message${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Функция для выполнения HTTP запроса
make_request() {
    local method="$1"
    local url="$2"
    local data="$3"
    local expected_status="$4"
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$url")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" = "$expected_status" ]; then
        echo "$body"
        return 0
    else
        echo "HTTP $http_code: $body" >&2
        return 1
    fi
}

# Функция для получения токена авторизации
get_auth_token() {
    local username="$1"
    local password="$2"
    
    local login_data='{"username":"'$username'","password":"'$password'"}'
    local response=$(make_request "POST" "$BASE_URL/api/auth/login" "$login_data" "200")
    
    if [ $? -eq 0 ]; then
        echo "$response" | jq -r '.data.token' 2>/dev/null
    else
        echo ""
    fi
}

# Функция для создания тестовой игры
create_test_game() {
    local token="$1"
    
    local game_data='{"name":"Movement Test Game","description":"Test game for movement testing"}'
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "$game_data" \
        "$BASE_URL/api/games")
    
    if [ $? -eq 0 ]; then
        echo "$response" | jq -r '.data.id' 2>/dev/null
    else
        echo ""
    fi
}

# Функция для создания корабля
create_ship() {
    local token="$1"
    local game_id="$2"
    local ship_data="$3"
    
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "$ship_data" \
        "$BASE_URL/api/games/$game_id/units")
    
    if [ $? -eq 0 ]; then
        echo "$response" | jq -r '.data.id' 2>/dev/null
    else
        echo ""
    fi
}

# Функция для тестирования движения
test_movement() {
    local token="$1"
    local game_id="$2"
    local unit_id="$3"
    local from_hex="$4"
    local to_hex="$5"
    local expected_result="$6"
    
    local movement_data='{"from_hex":"'$from_hex'","to_hex":"'$to_hex'"}'
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "$movement_data" \
        "$BASE_URL/api/games/$game_id/units/$unit_id/move")
    
    local http_code=$(echo "$response" | jq -r '.success' 2>/dev/null)
    
    if [ "$http_code" = "$expected_result" ]; then
        return 0
    else
        echo "Expected: $expected_result, Got: $http_code"
        return 1
    fi
}

# Функция для получения доступных ходов
get_available_moves() {
    local token="$1"
    local game_id="$2"
    local unit_id="$3"
    
    local response=$(curl -s -X GET \
        -H "Authorization: Bearer $token" \
        "$BASE_URL/api/games/$game_id/units/$unit_id/available-moves")
    
    if [ $? -eq 0 ]; then
        echo "$response" | jq -r '.data.available_hexes[]' 2>/dev/null
    else
        echo ""
    fi
}

echo "🔍 Проверка доступности сервера..."
if ! curl -s "$BASE_URL/health" > /dev/null 2>&1; then
    echo -e "${RED}❌ Сервер недоступен${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Сервер доступен${NC}"
echo ""

echo "🔐 Получение токенов авторизации..."
TOKEN1=$(get_auth_token "testuser1" "password123")
TOKEN2=$(get_auth_token "testuser2" "password123")

if [ -z "$TOKEN1" ] || [ -z "$TOKEN2" ]; then
    echo -e "${RED}❌ Не удалось получить токены авторизации${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Токены получены${NC}"
echo ""

echo "🎮 Создание тестовой игры..."
GAME_ID=$(create_test_game "$TOKEN1")
if [ -z "$GAME_ID" ]; then
    echo -e "${RED}❌ Не удалось создать игру${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Игра создана: $GAME_ID${NC}"
echo ""

echo "🚢 Создание тестовых кораблей..."

# Создание быстрого корабля (F)
FAST_SHIP_DATA='{
    "name": "Fast Destroyer",
    "type": "DD",
    "class": "Fast DD",
    "nationality": "german",
    "position": "J22",
    "speed_rating": "F",
    "fuel": 10,
    "max_fuel": 10,
    "hull_boxes": 3,
    "current_hull": 3,
    "primary_armament_bow": 2,
    "primary_armament_stern": 2,
    "secondary_armament": 4,
    "torpedoes": 8,
    "max_torpedoes": 8,
    "radar_level": 0,
    "evasion": 15,
    "base_evasion": 15
}'

FAST_SHIP_ID=$(create_ship "$TOKEN1" "$GAME_ID" "$FAST_SHIP_DATA")
if [ -z "$FAST_SHIP_ID" ]; then
    echo -e "${RED}❌ Не удалось создать быстрый корабль${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Быстрый корабль создан: $FAST_SHIP_ID${NC}"

# Создание среднего корабля (M)
MEDIUM_SHIP_DATA='{
    "name": "Medium Cruiser",
    "type": "CA",
    "class": "Medium CA",
    "nationality": "german",
    "position": "K22",
    "speed_rating": "M",
    "fuel": 8,
    "max_fuel": 8,
    "hull_boxes": 5,
    "current_hull": 5,
    "primary_armament_bow": 3,
    "primary_armament_stern": 3,
    "secondary_armament": 5,
    "torpedoes": 0,
    "max_torpedoes": 0,
    "radar_level": 1,
    "evasion": 20,
    "base_evasion": 20
}'

MEDIUM_SHIP_ID=$(create_ship "$TOKEN1" "$GAME_ID" "$MEDIUM_SHIP_DATA")
if [ -z "$MEDIUM_SHIP_ID" ]; then
    echo -e "${RED}❌ Не удалось создать средний корабль${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Средний корабль создан: $MEDIUM_SHIP_ID${NC}"

# Создание медленного корабля (S)
SLOW_SHIP_DATA='{
    "name": "Slow Battleship",
    "type": "BB",
    "class": "Slow BB",
    "nationality": "german",
    "position": "L22",
    "speed_rating": "S",
    "fuel": 15,
    "max_fuel": 15,
    "hull_boxes": 12,
    "current_hull": 12,
    "primary_armament_bow": 8,
    "primary_armament_stern": 2,
    "secondary_armament": 8,
    "torpedoes": 0,
    "max_torpedoes": 0,
    "radar_level": 2,
    "evasion": 10,
    "base_evasion": 10
}'

SLOW_SHIP_ID=$(create_ship "$TOKEN1" "$GAME_ID" "$SLOW_SHIP_DATA")
if [ -z "$SLOW_SHIP_ID" ]; then
    echo -e "${RED}❌ Не удалось создать медленный корабль${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Медленный корабль создан: $SLOW_SHIP_ID${NC}"

# Создание очень медленного корабля (VS)
VERY_SLOW_SHIP_DATA='{
    "name": "Very Slow Tanker",
    "type": "TK",
    "class": "Very Slow TK",
    "nationality": "german",
    "position": "M22",
    "speed_rating": "VS",
    "fuel": 20,
    "max_fuel": 20,
    "hull_boxes": 6,
    "current_hull": 6,
    "primary_armament_bow": 0,
    "primary_armament_stern": 0,
    "secondary_armament": 2,
    "torpedoes": 0,
    "max_torpedoes": 0,
    "radar_level": 0,
    "evasion": 5,
    "base_evasion": 5
}'

VERY_SLOW_SHIP_ID=$(create_ship "$TOKEN1" "$GAME_ID" "$VERY_SLOW_SHIP_DATA")
if [ -z "$VERY_SLOW_SHIP_ID" ]; then
    echo -e "${RED}❌ Не удалось создать очень медленный корабль${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Очень медленный корабль создан: $VERY_SLOW_SHIP_ID${NC}"

echo ""
echo "🧪 НАЧАЛО ТЕСТИРОВАНИЯ ДВИЖЕНИЯ"
echo "==============================="
echo ""

# Тест 1: Движение быстрого корабля
echo "🚀 Тест 1: Движение быстрого корабля (F)"
if test_movement "$TOKEN1" "$GAME_ID" "$FAST_SHIP_ID" "J22" "J23" "true"; then
    log_test "Движение быстрого корабля" "PASS"
else
    log_test "Движение быстрого корабля" "FAIL" "Не удалось переместить быстрый корабль"
fi

# Тест 2: Движение среднего корабля
echo "🚀 Тест 2: Движение среднего корабля (M)"
if test_movement "$TOKEN1" "$GAME_ID" "$MEDIUM_SHIP_ID" "K22" "K23" "true"; then
    log_test "Движение среднего корабля" "PASS"
else
    log_test "Движение среднего корабля" "FAIL" "Не удалось переместить средний корабль"
fi

# Тест 3: Движение медленного корабля
echo "🚀 Тест 3: Движение медленного корабля (S)"
if test_movement "$TOKEN1" "$GAME_ID" "$SLOW_SHIP_ID" "L22" "L23" "true"; then
    log_test "Движение медленного корабля" "PASS"
else
    log_test "Движение медленного корабля" "FAIL" "Не удалось переместить медленный корабль"
fi

# Тест 4: Движение очень медленного корабля
echo "🚀 Тест 4: Движение очень медленного корабля (VS)"
if test_movement "$TOKEN1" "$GAME_ID" "$VERY_SLOW_SHIP_ID" "M22" "M23" "true"; then
    log_test "Движение очень медленного корабля" "PASS"
else
    log_test "Движение очень медленного корабля" "FAIL" "Не удалось переместить очень медленный корабль"
fi

# Тест 5: Проверка доступных ходов для быстрого корабля
echo "🚀 Тест 5: Проверка доступных ходов для быстрого корабля"
AVAILABLE_MOVES=$(get_available_moves "$TOKEN1" "$GAME_ID" "$FAST_SHIP_ID")
if [ -n "$AVAILABLE_MOVES" ]; then
    log_test "Доступные ходы для быстрого корабля" "PASS"
else
    log_test "Доступные ходы для быстрого корабля" "FAIL" "Не удалось получить доступные ходы"
fi

# Тест 6: Проверка доступных ходов для среднего корабля
echo "🚀 Тест 6: Проверка доступных ходов для среднего корабля"
AVAILABLE_MOVES=$(get_available_moves "$TOKEN1" "$GAME_ID" "$MEDIUM_SHIP_ID")
if [ -n "$AVAILABLE_MOVES" ]; then
    log_test "Доступные ходы для среднего корабля" "PASS"
else
    log_test "Доступные ходы для среднего корабля" "FAIL" "Не удалось получить доступные ходы"
fi

# Тест 7: Проверка доступных ходов для медленного корабля
echo "🚀 Тест 7: Проверка доступных ходов для медленного корабля"
AVAILABLE_MOVES=$(get_available_moves "$TOKEN1" "$GAME_ID" "$SLOW_SHIP_ID")
if [ -n "$AVAILABLE_MOVES" ]; then
    log_test "Доступные ходы для медленного корабля" "PASS"
else
    log_test "Доступные ходы для медленного корабля" "FAIL" "Не удалось получить доступные ходы"
fi

# Тест 8: Проверка доступных ходов для очень медленного корабля
echo "🚀 Тест 8: Проверка доступных ходов для очень медленного корабля"
AVAILABLE_MOVES=$(get_available_moves "$TOKEN1" "$GAME_ID" "$VERY_SLOW_SHIP_ID")
if [ -n "$AVAILABLE_MOVES" ]; then
    log_test "Доступные ходы для очень медленного корабля" "PASS"
else
    log_test "Доступные ходы для очень медленного корабля" "FAIL" "Не удалось получить доступные ходы"
fi

# Тест 9: Попытка движения на слишком большое расстояние
echo "🚀 Тест 9: Попытка движения на слишком большое расстояние"
if test_movement "$TOKEN1" "$GAME_ID" "$FAST_SHIP_ID" "J23" "J30" "false"; then
    log_test "Ограничение расстояния движения" "PASS"
else
    log_test "Ограничение расстояния движения" "FAIL" "Движение на большое расстояние должно быть заблокировано"
fi

# Тест 10: Попытка движения без топлива
echo "🚀 Тест 10: Попытка движения без топлива"
# Создаем корабль с 0 топливом
NO_FUEL_SHIP_DATA='{
    "name": "No Fuel Ship",
    "type": "DD",
    "class": "No Fuel DD",
    "nationality": "german",
    "position": "N22",
    "speed_rating": "F",
    "fuel": 0,
    "max_fuel": 10,
    "hull_boxes": 3,
    "current_hull": 3,
    "primary_armament_bow": 2,
    "primary_armament_stern": 2,
    "secondary_armament": 4,
    "torpedoes": 8,
    "max_torpedoes": 8,
    "radar_level": 0,
    "evasion": 15,
    "base_evasion": 15
}'

NO_FUEL_SHIP_ID=$(create_ship "$TOKEN1" "$GAME_ID" "$NO_FUEL_SHIP_DATA")
if [ -n "$NO_FUEL_SHIP_ID" ]; then
    if test_movement "$TOKEN1" "$GAME_ID" "$NO_FUEL_SHIP_ID" "N22" "N23" "false"; then
        log_test "Движение без топлива" "PASS"
    else
        log_test "Движение без топлива" "FAIL" "Движение без топлива должно быть заблокировано"
    fi
else
    log_test "Движение без топлива" "FAIL" "Не удалось создать корабль без топлива"
fi

echo ""
echo "📊 ИТОГОВАЯ СТАТИСТИКА"
echo "====================="
echo -e "${GREEN}✅ Успешных тестов: $TESTS_PASSED${NC}"
echo -e "${RED}❌ Неудачных тестов: $TESTS_FAILED${NC}"
echo -e "${BLUE}📈 Общий процент успеха: $(( (TESTS_PASSED * 100) / TOTAL_TESTS ))%${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo ""
    echo -e "${GREEN}🎉 ВСЕ ТЕСТЫ ПРОШЛИ УСПЕШНО!${NC}"
    echo -e "${GREEN}🚀 Система движения кораблей работает корректно!${NC}"
    exit 0
else
    echo ""
    echo -e "${YELLOW}⚠️  НЕКОТОРЫЕ ТЕСТЫ НЕ ПРОШЛИ${NC}"
    echo -e "${YELLOW}🔧 Рекомендуется проверить логи и исправить ошибки${NC}"
    exit 1
fi
