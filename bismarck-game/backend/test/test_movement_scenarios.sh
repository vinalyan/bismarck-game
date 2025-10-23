#!/bin/bash

# Детальные сценарии тестирования движения кораблей
BASE_URL="http://localhost:8080"

echo "🚢 ДЕТАЛЬНЫЕ СЦЕНАРИИ ТЕСТИРОВАНИЯ ДВИЖЕНИЯ"
echo "=========================================="
echo ""

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
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
    
    local game_data='{"name":"Movement Scenarios Test Game","description":"Test game for movement scenarios testing"}'
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

# Функция для получения информации о корабле
get_ship_info() {
    local token="$1"
    local game_id="$2"
    local unit_id="$3"
    
    local response=$(curl -s -X GET \
        -H "Authorization: Bearer $token" \
        "$BASE_URL/api/games/$game_id/units/$unit_id")
    
    if [ $? -eq 0 ]; then
        echo "$response" | jq -r '.data' 2>/dev/null
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

# Создание различных типов кораблей для тестирования
echo "📋 Создание кораблей разных типов..."

# Быстрый эсминец (F)
FAST_DD_DATA='{
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

FAST_DD_ID=$(create_ship "$TOKEN1" "$GAME_ID" "$FAST_DD_DATA")
if [ -n "$FAST_DD_ID" ]; then
    echo -e "${GREEN}✅ Быстрый эсминец создан: $FAST_DD_ID${NC}"
else
    echo -e "${RED}❌ Не удалось создать быстрый эсминец${NC}"
    exit 1
fi

# Средний крейсер (M)
MEDIUM_CA_DATA='{
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

MEDIUM_CA_ID=$(create_ship "$TOKEN1" "$GAME_ID" "$MEDIUM_CA_DATA")
if [ -n "$MEDIUM_CA_ID" ]; then
    echo -e "${GREEN}✅ Средний крейсер создан: $MEDIUM_CA_ID${NC}"
else
    echo -e "${RED}❌ Не удалось создать средний крейсер${NC}"
    exit 1
fi

# Медленный линкор (S)
SLOW_BB_DATA='{
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

SLOW_BB_ID=$(create_ship "$TOKEN1" "$GAME_ID" "$SLOW_BB_DATA")
if [ -n "$SLOW_BB_ID" ]; then
    echo -e "${GREEN}✅ Медленный линкор создан: $SLOW_BB_ID${NC}"
else
    echo -e "${RED}❌ Не удалось создать медленный линкор${NC}"
    exit 1
fi

# Очень медленный танкер (VS)
VERY_SLOW_TK_DATA='{
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

VERY_SLOW_TK_ID=$(create_ship "$TOKEN1" "$GAME_ID" "$VERY_SLOW_TK_DATA")
if [ -n "$VERY_SLOW_TK_ID" ]; then
    echo -e "${GREEN}✅ Очень медленный танкер создан: $VERY_SLOW_TK_ID${NC}"
else
    echo -e "${RED}❌ Не удалось создать очень медленный танкер${NC}"
    exit 1
fi

echo ""
echo "🧪 НАЧАЛО ДЕТАЛЬНОГО ТЕСТИРОВАНИЯ СЦЕНАРИЕВ"
echo "=========================================="
echo ""

# Сценарий 1: Тестирование движения быстрого эсминца
echo -e "${BLUE}🚀 Сценарий 1: Движение быстрого эсминца (F)${NC}"
echo "Тестирование максимального расстояния движения для F корабля"

# Движение на 1 гекс
if test_movement "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" "J22" "J23" "true"; then
    log_test "F корабль: движение на 1 гекс" "PASS"
else
    log_test "F корабль: движение на 1 гекс" "FAIL" "Не удалось переместить F корабль на 1 гекс"
fi

# Движение на 2 гекса
if test_movement "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" "J23" "J25" "true"; then
    log_test "F корабль: движение на 2 гекса" "PASS"
else
    log_test "F корабль: движение на 2 гекса" "FAIL" "Не удалось переместить F корабль на 2 гекса"
fi

# Движение на 3 гекса
if test_movement "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" "J25" "J28" "true"; then
    log_test "F корабль: движение на 3 гекса" "PASS"
else
    log_test "F корабль: движение на 3 гекса" "FAIL" "Не удалось переместить F корабль на 3 гекса"
fi

# Попытка движения на 4+ гексов (должно быть заблокировано)
if test_movement "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" "J28" "J32" "false"; then
    log_test "F корабль: ограничение на 4+ гексов" "PASS"
else
    log_test "F корабль: ограничение на 4+ гексов" "FAIL" "Движение на 4+ гексов должно быть заблокировано"
fi

echo ""

# Сценарий 2: Тестирование движения среднего крейсера
echo -e "${BLUE}🚀 Сценарий 2: Движение среднего крейсера (M)${NC}"
echo "Тестирование максимального расстояния движения для M корабля"

# Движение на 1 гекс
if test_movement "$TOKEN1" "$GAME_ID" "$MEDIUM_CA_ID" "K22" "K23" "true"; then
    log_test "M корабль: движение на 1 гекс" "PASS"
else
    log_test "M корабль: движение на 1 гекс" "FAIL" "Не удалось переместить M корабль на 1 гекс"
fi

# Движение на 2 гекса
if test_movement "$TOKEN1" "$GAME_ID" "$MEDIUM_CA_ID" "K23" "K25" "true"; then
    log_test "M корабль: движение на 2 гекса" "PASS"
else
    log_test "M корабль: движение на 2 гекса" "FAIL" "Не удалось переместить M корабль на 2 гекса"
fi

# Попытка движения на 3+ гексов (должно быть заблокировано)
if test_movement "$TOKEN1" "$GAME_ID" "$MEDIUM_CA_ID" "K25" "K28" "false"; then
    log_test "M корабль: ограничение на 3+ гексов" "PASS"
else
    log_test "M корабль: ограничение на 3+ гексов" "FAIL" "Движение на 3+ гексов должно быть заблокировано"
fi

echo ""

# Сценарий 3: Тестирование движения медленного линкора
echo -e "${BLUE}🚀 Сценарий 3: Движение медленного линкора (S)${NC}"
echo "Тестирование максимального расстояния движения для S корабля"

# Движение на 1 гекс
if test_movement "$TOKEN1" "$GAME_ID" "$SLOW_BB_ID" "L22" "L23" "true"; then
    log_test "S корабль: движение на 1 гекс" "PASS"
else
    log_test "S корабль: движение на 1 гекс" "FAIL" "Не удалось переместить S корабль на 1 гекс"
fi

# Попытка движения на 2+ гексов (должно быть заблокировано)
if test_movement "$TOKEN1" "$GAME_ID" "$SLOW_BB_ID" "L23" "L25" "false"; then
    log_test "S корабль: ограничение на 2+ гексов" "PASS"
else
    log_test "S корабль: ограничение на 2+ гексов" "FAIL" "Движение на 2+ гексов должно быть заблокировано"
fi

echo ""

# Сценарий 4: Тестирование движения очень медленного танкера
echo -e "${BLUE}🚀 Сценарий 4: Движение очень медленного танкера (VS)${NC}"
echo "Тестирование максимального расстояния движения для VS корабля"

# Движение на 1 гекс
if test_movement "$TOKEN1" "$GAME_ID" "$VERY_SLOW_TK_ID" "M22" "M23" "true"; then
    log_test "VS корабль: движение на 1 гекс" "PASS"
else
    log_test "VS корабль: движение на 1 гекс" "FAIL" "Не удалось переместить VS корабль на 1 гекс"
fi

# Попытка движения на 2+ гексов (должно быть заблокировано)
if test_movement "$TOKEN1" "$GAME_ID" "$VERY_SLOW_TK_ID" "M23" "M25" "false"; then
    log_test "VS корабль: ограничение на 2+ гексов" "PASS"
else
    log_test "VS корабль: ограничение на 2+ гексов" "FAIL" "Движение на 2+ гексов должно быть заблокировано"
fi

echo ""

# Сценарий 5: Тестирование доступных ходов
echo -e "${BLUE}🚀 Сценарий 5: Тестирование доступных ходов${NC}"
echo "Проверка корректности расчета доступных ходов для разных типов кораблей"

# Проверка доступных ходов для быстрого эсминца
AVAILABLE_MOVES=$(get_available_moves "$TOKEN1" "$GAME_ID" "$FAST_DD_ID")
if [ -n "$AVAILABLE_MOVES" ]; then
    log_test "F корабль: получение доступных ходов" "PASS"
else
    log_test "F корабль: получение доступных ходов" "FAIL" "Не удалось получить доступные ходы для F корабля"
fi

# Проверка доступных ходов для среднего крейсера
AVAILABLE_MOVES=$(get_available_moves "$TOKEN1" "$GAME_ID" "$MEDIUM_CA_ID")
if [ -n "$AVAILABLE_MOVES" ]; then
    log_test "M корабль: получение доступных ходов" "PASS"
else
    log_test "M корабль: получение доступных ходов" "FAIL" "Не удалось получить доступные ходы для M корабля"
fi

# Проверка доступных ходов для медленного линкора
AVAILABLE_MOVES=$(get_available_moves "$TOKEN1" "$GAME_ID" "$SLOW_BB_ID")
if [ -n "$AVAILABLE_MOVES" ]; then
    log_test "S корабль: получение доступных ходов" "PASS"
else
    log_test "S корабль: получение доступных ходов" "FAIL" "Не удалось получить доступные ходы для S корабля"
fi

# Проверка доступных ходов для очень медленного танкера
AVAILABLE_MOVES=$(get_available_moves "$TOKEN1" "$GAME_ID" "$VERY_SLOW_TK_ID")
if [ -n "$AVAILABLE_MOVES" ]; then
    log_test "VS корабль: получение доступных ходов" "PASS"
else
    log_test "VS корабль: получение доступных ходов" "FAIL" "Не удалось получить доступные ходы для VS корабля"
fi

echo ""

# Сценарий 6: Тестирование ограничений топлива
echo -e "${BLUE}🚀 Сценарий 6: Тестирование ограничений топлива${NC}"
echo "Проверка блокировки движения при отсутствии топлива"

# Создание корабля без топлива
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
    echo -e "${GREEN}✅ Корабль без топлива создан: $NO_FUEL_SHIP_ID${NC}"
    
    # Попытка движения без топлива
    if test_movement "$TOKEN1" "$GAME_ID" "$NO_FUEL_SHIP_ID" "N22" "N23" "false"; then
        log_test "Корабль без топлива: блокировка движения" "PASS"
    else
        log_test "Корабль без топлива: блокировка движения" "FAIL" "Движение без топлива должно быть заблокировано"
    fi
else
    log_test "Корабль без топлива: создание" "FAIL" "Не удалось создать корабль без топлива"
fi

echo ""

# Сценарий 7: Тестирование движения в разных направлениях
echo -e "${BLUE}🚀 Сценарий 7: Тестирование движения в разных направлениях${NC}"
echo "Проверка корректности движения в различных направлениях"

# Создание корабля для тестирования направлений
DIRECTION_TEST_SHIP_DATA='{
    "name": "Direction Test Ship",
    "type": "DD",
    "class": "Direction Test DD",
    "nationality": "german",
    "position": "O22",
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

DIRECTION_TEST_SHIP_ID=$(create_ship "$TOKEN1" "$GAME_ID" "$DIRECTION_TEST_SHIP_DATA")
if [ -n "$DIRECTION_TEST_SHIP_ID" ]; then
    echo -e "${GREEN}✅ Корабль для тестирования направлений создан: $DIRECTION_TEST_SHIP_ID${NC}"
    
    # Движение вправо
    if test_movement "$TOKEN1" "$GAME_ID" "$DIRECTION_TEST_SHIP_ID" "O22" "O23" "true"; then
        log_test "Движение вправо" "PASS"
    else
        log_test "Движение вправо" "FAIL" "Не удалось переместить корабль вправо"
    fi
    
    # Движение влево
    if test_movement "$TOKEN1" "$GAME_ID" "$DIRECTION_TEST_SHIP_ID" "O23" "O22" "true"; then
        log_test "Движение влево" "PASS"
    else
        log_test "Движение влево" "FAIL" "Не удалось переместить корабль влево"
    fi
    
    # Движение вверх
    if test_movement "$TOKEN1" "$GAME_ID" "$DIRECTION_TEST_SHIP_ID" "O22" "N22" "true"; then
        log_test "Движение вверх" "PASS"
    else
        log_test "Движение вверх" "FAIL" "Не удалось переместить корабль вверх"
    fi
    
    # Движение вниз
    if test_movement "$TOKEN1" "$GAME_ID" "$DIRECTION_TEST_SHIP_ID" "N22" "O22" "true"; then
        log_test "Движение вниз" "PASS"
    else
        log_test "Движение вниз" "FAIL" "Не удалось переместить корабль вниз"
    fi
else
    log_test "Корабль для тестирования направлений: создание" "FAIL" "Не удалось создать корабль для тестирования направлений"
fi

echo ""

# Сценарий 8: Тестирование ошибок движения
echo -e "${BLUE}🚀 Сценарий 8: Тестирование ошибок движения${NC}"
echo "Проверка корректности обработки ошибок при некорректных запросах"

# Попытка движения на несуществующий гекс
if test_movement "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" "J28" "ZZ99" "false"; then
    log_test "Движение на несуществующий гекс" "PASS"
else
    log_test "Движение на несуществующий гекс" "FAIL" "Движение на несуществующий гекс должно быть заблокировано"
fi

# Попытка движения с неверного гекса
if test_movement "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" "ZZ99" "J29" "false"; then
    log_test "Движение с неверного гекса" "PASS"
else
    log_test "Движение с неверного гекса" "FAIL" "Движение с неверного гекса должно быть заблокировано"
fi

# Попытка движения на тот же гекс
if test_movement "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" "J28" "J28" "false"; then
    log_test "Движение на тот же гекс" "PASS"
else
    log_test "Движение на тот же гекс" "FAIL" "Движение на тот же гекс должно быть заблокировано"
fi

echo ""

# Сценарий 9: Тестирование производительности
echo -e "${BLUE}🚀 Сценарий 9: Тестирование производительности${NC}"
echo "Проверка времени выполнения операций движения"

# Измерение времени выполнения запроса доступных ходов
start_time=$(date +%s%N)
AVAILABLE_MOVES=$(get_available_moves "$TOKEN1" "$GAME_ID" "$FAST_DD_ID")
end_time=$(date +%s%N)
execution_time=$(( (end_time - start_time) / 1000000 ))

if [ $execution_time -lt 1000 ]; then
    log_test "Производительность: получение доступных ходов" "PASS"
else
    log_test "Производительность: получение доступных ходов" "FAIL" "Время выполнения превышает 1 секунду: ${execution_time}ms"
fi

# Измерение времени выполнения запроса движения
start_time=$(date +%s%N)
test_movement "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" "J28" "J29" "true"
end_time=$(date +%s%N)
execution_time=$(( (end_time - start_time) / 1000000 ))

if [ $execution_time -lt 1000 ]; then
    log_test "Производительность: выполнение движения" "PASS"
else
    log_test "Производительность: выполнение движения" "FAIL" "Время выполнения превышает 1 секунду: ${execution_time}ms"
fi

echo ""

# Сценарий 10: Тестирование граничных условий
echo -e "${BLUE}🚀 Сценарий 10: Тестирование граничных условий${NC}"
echo "Проверка корректности обработки граничных условий"

# Создание корабля на границе карты
BOUNDARY_SHIP_DATA='{
    "name": "Boundary Test Ship",
    "type": "DD",
    "class": "Boundary Test DD",
    "nationality": "german",
    "position": "A1",
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

BOUNDARY_SHIP_ID=$(create_ship "$TOKEN1" "$GAME_ID" "$BOUNDARY_SHIP_DATA")
if [ -n "$BOUNDARY_SHIP_ID" ]; then
    echo -e "${GREEN}✅ Корабль на границе создан: $BOUNDARY_SHIP_ID${NC}"
    
    # Попытка движения за границы карты
    if test_movement "$TOKEN1" "$GAME_ID" "$BOUNDARY_SHIP_ID" "A1" "A0" "false"; then
        log_test "Движение за границы карты" "PASS"
    else
        log_test "Движение за границы карты" "FAIL" "Движение за границы карты должно быть заблокировано"
    fi
    
    # Попытка движения в пределах карты
    if test_movement "$TOKEN1" "$GAME_ID" "$BOUNDARY_SHIP_ID" "A1" "A2" "true"; then
        log_test "Движение в пределах карты" "PASS"
    else
        log_test "Движение в пределах карты" "FAIL" "Движение в пределах карты должно быть разрешено"
    fi
else
    log_test "Корабль на границе: создание" "FAIL" "Не удалось создать корабль на границе"
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
