#!/bin/bash

# Интеграционные тесты движения кораблей
BASE_URL="http://localhost:8080"

echo "🚢 ИНТЕГРАЦИОННЫЕ ТЕСТЫ ДВИЖЕНИЯ КОРАБЛЕЙ"
echo "======================================="
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
    
    local game_data='{"name":"Integration Test Game","description":"Test game for integration testing"}'
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

# Функция для получения информации об игре
get_game_info() {
    local token="$1"
    local game_id="$2"
    
    local response=$(curl -s -X GET \
        -H "Authorization: Bearer $token" \
        "$BASE_URL/api/games/$game_id")
    
    if [ $? -eq 0 ]; then
        echo "$response" | jq -r '.data' 2>/dev/null
    else
        echo ""
    fi
}

# Функция для получения списка кораблей в игре
get_game_units() {
    local token="$1"
    local game_id="$2"
    
    local response=$(curl -s -X GET \
        -H "Authorization: Bearer $token" \
        "$BASE_URL/api/games/$game_id/units")
    
    if [ $? -eq 0 ]; then
        echo "$response" | jq -r '.data.units[]' 2>/dev/null
    else
        echo ""
    fi
}

# Функция для смены фазы игры
change_phase() {
    local token="$1"
    local game_id="$2"
    local phase="$3"
    
    local phase_data='{"phase":"'$phase'"}'
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "$phase_data" \
        "$BASE_URL/api/games/$game_id/phases")
    
    if [ $? -eq 0 ]; then
        echo "$response" | jq -r '.data.phase' 2>/dev/null
    else
        echo ""
    fi
}

# Функция для получения текущей фазы
get_current_phase() {
    local token="$1"
    local game_id="$2"
    
    local response=$(curl -s -X GET \
        -H "Authorization: Bearer $token" \
        "$BASE_URL/api/games/$game_id/phases/current")
    
    if [ $? -eq 0 ]; then
        echo "$response" | jq -r '.data.phase' 2>/dev/null
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

# Создание флота для интеграционного тестирования
echo "📋 Создание флота для интеграционного тестирования..."

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
echo "🧪 НАЧАЛО ИНТЕГРАЦИОННОГО ТЕСТИРОВАНИЯ"
echo "===================================="
echo ""

# Интеграционный тест 1: Смешанные типы кораблей
echo -e "${BLUE}🚀 Интеграционный тест 1: Смешанные типы кораблей${NC}"
echo "Тестирование одновременного движения кораблей разных типов"

# Движение всех кораблей одновременно
if test_movement "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" "J22" "J23" "true"; then
    log_test "F корабль: одновременное движение" "PASS"
else
    log_test "F корабль: одновременное движение" "FAIL" "Не удалось переместить F корабль"
fi

if test_movement "$TOKEN1" "$GAME_ID" "$MEDIUM_CA_ID" "K22" "K23" "true"; then
    log_test "M корабль: одновременное движение" "PASS"
else
    log_test "M корабль: одновременное движение" "FAIL" "Не удалось переместить M корабль"
fi

if test_movement "$TOKEN1" "$GAME_ID" "$SLOW_BB_ID" "L22" "L23" "true"; then
    log_test "S корабль: одновременное движение" "PASS"
else
    log_test "S корабль: одновременное движение" "FAIL" "Не удалось переместить S корабль"
fi

if test_movement "$TOKEN1" "$GAME_ID" "$VERY_SLOW_TK_ID" "M22" "M23" "true"; then
    log_test "VS корабль: одновременное движение" "PASS"
else
    log_test "VS корабль: одновременное движение" "FAIL" "Не удалось переместить VS корабль"
fi

echo ""

# Интеграционный тест 2: Переходы между ходами
echo -e "${BLUE}🚀 Интеграционный тест 2: Переходы между ходами${NC}"
echo "Тестирование сохранения состояния кораблей между ходами"

# Получение информации о кораблях после движения
FAST_DD_INFO=$(get_ship_info "$TOKEN1" "$GAME_ID" "$FAST_DD_ID")
if [ -n "$FAST_DD_INFO" ]; then
    log_test "F корабль: получение информации после движения" "PASS"
else
    log_test "F корабль: получение информации после движения" "FAIL" "Не удалось получить информацию о F корабле"
fi

MEDIUM_CA_INFO=$(get_ship_info "$TOKEN1" "$GAME_ID" "$MEDIUM_CA_ID")
if [ -n "$MEDIUM_CA_INFO" ]; then
    log_test "M корабль: получение информации после движения" "PASS"
else
    log_test "M корабль: получение информации после движения" "FAIL" "Не удалось получить информацию о M корабле"
fi

SLOW_BB_INFO=$(get_ship_info "$TOKEN1" "$GAME_ID" "$SLOW_BB_ID")
if [ -n "$SLOW_BB_INFO" ]; then
    log_test "S корабль: получение информации после движения" "PASS"
else
    log_test "S корабль: получение информации после движения" "FAIL" "Не удалось получить информацию о S корабле"
fi

VERY_SLOW_TK_INFO=$(get_ship_info "$TOKEN1" "$GAME_ID" "$VERY_SLOW_TK_ID")
if [ -n "$VERY_SLOW_TK_INFO" ]; then
    log_test "VS корабль: получение информации после движения" "PASS"
else
    log_test "VS корабль: получение информации после движения" "FAIL" "Не удалось получить информацию о VS корабле"
fi

echo ""

# Интеграционный тест 3: Фаза администрирования
echo -e "${BLUE}🚀 Интеграционный тест 3: Фаза администрирования${NC}"
echo "Тестирование смены фаз игры и их влияния на движение"

# Получение текущей фазы
CURRENT_PHASE=$(get_current_phase "$TOKEN1" "$GAME_ID")
if [ -n "$CURRENT_PHASE" ]; then
    log_test "Получение текущей фазы" "PASS"
    echo -e "${CYAN}   Текущая фаза: $CURRENT_PHASE${NC}"
else
    log_test "Получение текущей фазы" "FAIL" "Не удалось получить текущую фазу"
fi

# Попытка смены фазы
NEW_PHASE=$(change_phase "$TOKEN1" "$GAME_ID" "admin")
if [ -n "$NEW_PHASE" ]; then
    log_test "Смена фазы на админ" "PASS"
    echo -e "${CYAN}   Новая фаза: $NEW_PHASE${NC}"
else
    log_test "Смена фазы на админ" "FAIL" "Не удалось сменить фазу на админ"
fi

# Попытка движения в админ фазе
if test_movement "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" "J23" "J24" "false"; then
    log_test "Движение в админ фазе" "PASS"
else
    log_test "Движение в админ фазе" "FAIL" "Движение в админ фазе должно быть заблокировано"
fi

echo ""

# Интеграционный тест 4: Синхронизация фронтенд-бэкенд
echo -e "${BLUE}🚀 Интеграционный тест 4: Синхронизация фронтенд-бэкенд${NC}"
echo "Тестирование синхронизации данных между фронтендом и бэкендом"

# Получение списка всех кораблей в игре
GAME_UNITS=$(get_game_units "$TOKEN1" "$GAME_ID")
if [ -n "$GAME_UNITS" ]; then
    log_test "Получение списка кораблей в игре" "PASS"
    echo -e "${CYAN}   Количество кораблей: $(echo "$GAME_UNITS" | wc -l)${NC}"
else
    log_test "Получение списка кораблей в игре" "FAIL" "Не удалось получить список кораблей"
fi

# Проверка синхронизации позиций
FAST_DD_POSITION=$(echo "$FAST_DD_INFO" | jq -r '.position' 2>/dev/null)
if [ "$FAST_DD_POSITION" = "J23" ]; then
    log_test "Синхронизация позиции F корабля" "PASS"
else
    log_test "Синхронизация позиции F корабля" "FAIL" "Позиция F корабля не синхронизирована: $FAST_DD_POSITION"
fi

MEDIUM_CA_POSITION=$(echo "$MEDIUM_CA_INFO" | jq -r '.position' 2>/dev/null)
if [ "$MEDIUM_CA_POSITION" = "K23" ]; then
    log_test "Синхронизация позиции M корабля" "PASS"
else
    log_test "Синхронизация позиции M корабля" "FAIL" "Позиция M корабля не синхронизирована: $MEDIUM_CA_POSITION"
fi

SLOW_BB_POSITION=$(echo "$SLOW_BB_INFO" | jq -r '.position' 2>/dev/null)
if [ "$SLOW_BB_POSITION" = "L23" ]; then
    log_test "Синхронизация позиции S корабля" "PASS"
else
    log_test "Синхронизация позиции S корабля" "FAIL" "Позиция S корабля не синхронизирована: $SLOW_BB_POSITION"
fi

VERY_SLOW_TK_POSITION=$(echo "$VERY_SLOW_TK_INFO" | jq -r '.position' 2>/dev/null)
if [ "$VERY_SLOW_TK_POSITION" = "M23" ]; then
    log_test "Синхронизация позиции VS корабля" "PASS"
else
    log_test "Синхронизация позиции VS корабля" "FAIL" "Позиция VS корабля не синхронизирована: $VERY_SLOW_TK_POSITION"
fi

echo ""

# Интеграционный тест 5: Тестирование производительности
echo -e "${BLUE}🚀 Интеграционный тест 5: Тестирование производительности${NC}"
echo "Проверка производительности системы при множественных операциях"

# Измерение времени выполнения множественных запросов
start_time=$(date +%s%N)
for i in {1..10}; do
    get_available_moves "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" > /dev/null
done
end_time=$(date +%s%N)
execution_time=$(( (end_time - start_time) / 1000000 ))

if [ $execution_time -lt 5000 ]; then
    log_test "Производительность: 10 запросов доступных ходов" "PASS"
    echo -e "${CYAN}   Время выполнения: ${execution_time}ms${NC}"
else
    log_test "Производительность: 10 запросов доступных ходов" "FAIL" "Время выполнения превышает 5 секунд: ${execution_time}ms"
fi

# Измерение времени выполнения множественных движений
start_time=$(date +%s%N)
for i in {1..5}; do
    test_movement "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" "J23" "J24" "true" > /dev/null
done
end_time=$(date +%s%N)
execution_time=$(( (end_time - start_time) / 1000000 ))

if [ $execution_time -lt 5000 ]; then
    log_test "Производительность: 5 операций движения" "PASS"
    echo -e "${CYAN}   Время выполнения: ${execution_time}ms${NC}"
else
    log_test "Производительность: 5 операций движения" "FAIL" "Время выполнения превышает 5 секунд: ${execution_time}ms"
fi

echo ""

# Интеграционный тест 6: Тестирование ошибок и восстановления
echo -e "${BLUE}🚀 Интеграционный тест 6: Тестирование ошибок и восстановления${NC}"
echo "Проверка корректности обработки ошибок и восстановления системы"

# Попытка движения несуществующего корабля
if test_movement "$TOKEN1" "$GAME_ID" "nonexistent-id" "J24" "J25" "false"; then
    log_test "Движение несуществующего корабля" "PASS"
else
    log_test "Движение несуществующего корабля" "FAIL" "Движение несуществующего корабля должно быть заблокировано"
fi

# Попытка движения с неверными координатами
if test_movement "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" "invalid" "J25" "false"; then
    log_test "Движение с неверными координатами" "PASS"
else
    log_test "Движение с неверными координатами" "FAIL" "Движение с неверными координатами должно быть заблокировано"
fi

# Попытка движения на занятый гекс
if test_movement "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" "J24" "J25" "false"; then
    log_test "Движение на занятый гекс" "PASS"
else
    log_test "Движение на занятый гекс" "FAIL" "Движение на занятый гекс должно быть заблокировано"
fi

echo ""

# Интеграционный тест 7: Тестирование масштабируемости
echo -e "${BLUE}🚀 Интеграционный тест 7: Тестирование масштабируемости${NC}"
echo "Проверка работы системы с большим количеством кораблей"

# Создание дополнительных кораблей для тестирования масштабируемости
echo "📋 Создание дополнительных кораблей для тестирования масштабируемости..."

ADDITIONAL_SHIPS=()
for i in {1..5}; do
    ADDITIONAL_SHIP_DATA='{
        "name": "Additional Ship '$i'",
        "type": "DD",
        "class": "Additional DD '$i'",
        "nationality": "german",
        "position": "N'$((22 + i))'",
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
    
    ADDITIONAL_SHIP_ID=$(create_ship "$TOKEN1" "$GAME_ID" "$ADDITIONAL_SHIP_DATA")
    if [ -n "$ADDITIONAL_SHIP_ID" ]; then
        ADDITIONAL_SHIPS+=("$ADDITIONAL_SHIP_ID")
        echo -e "${GREEN}✅ Дополнительный корабль $i создан: $ADDITIONAL_SHIP_ID${NC}"
    else
        echo -e "${RED}❌ Не удалось создать дополнительный корабль $i${NC}"
    fi
done

# Тестирование движения всех дополнительных кораблей
for i in "${!ADDITIONAL_SHIPS[@]}"; do
    SHIP_ID="${ADDITIONAL_SHIPS[$i]}"
    NEW_POSITION="N$((23 + i))"
    
    if test_movement "$TOKEN1" "$GAME_ID" "$SHIP_ID" "N$((22 + i + 1))" "$NEW_POSITION" "true"; then
        log_test "Дополнительный корабль $((i + 1)): движение" "PASS"
    else
        log_test "Дополнительный корабль $((i + 1)): движение" "FAIL" "Не удалось переместить дополнительный корабль $((i + 1))"
    fi
done

echo ""

# Интеграционный тест 8: Тестирование конкурентности
echo -e "${BLUE}🚀 Интеграционный тест 8: Тестирование конкурентности${NC}"
echo "Проверка работы системы при одновременных запросах от разных пользователей"

# Создание второй игры для второго пользователя
GAME2_ID=$(create_test_game "$TOKEN2")
if [ -n "$GAME2_ID" ]; then
    echo -e "${GREEN}✅ Вторая игра создана: $GAME2_ID${NC}"
    
    # Создание корабля для второго пользователя
    USER2_SHIP_DATA='{
        "name": "User2 Ship",
        "type": "DD",
        "class": "User2 DD",
        "nationality": "allied",
        "position": "P22",
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
    
    USER2_SHIP_ID=$(create_ship "$TOKEN2" "$GAME2_ID" "$USER2_SHIP_DATA")
    if [ -n "$USER2_SHIP_ID" ]; then
        echo -e "${GREEN}✅ Корабль второго пользователя создан: $USER2_SHIP_ID${NC}"
        
        # Одновременное движение кораблей разных пользователей
        if test_movement "$TOKEN1" "$GAME_ID" "$FAST_DD_ID" "J24" "J25" "true" && \
           test_movement "$TOKEN2" "$GAME2_ID" "$USER2_SHIP_ID" "P22" "P23" "true"; then
            log_test "Конкурентное движение кораблей" "PASS"
        else
            log_test "Конкурентное движение кораблей" "FAIL" "Не удалось выполнить конкурентное движение"
        fi
    else
        log_test "Корабль второго пользователя: создание" "FAIL" "Не удалось создать корабль для второго пользователя"
    fi
else
    log_test "Вторая игра: создание" "FAIL" "Не удалось создать вторую игру"
fi

echo ""
echo "📊 ИТОГОВАЯ СТАТИСТИКА"
echo "====================="
echo -e "${GREEN}✅ Успешных тестов: $TESTS_PASSED${NC}"
echo -e "${RED}❌ Неудачных тестов: $TESTS_FAILED${NC}"
echo -e "${BLUE}📈 Общий процент успеха: $(( (TESTS_PASSED * 100) / TOTAL_TESTS ))%${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo ""
    echo -e "${GREEN}🎉 ВСЕ ИНТЕГРАЦИОННЫЕ ТЕСТЫ ПРОШЛИ УСПЕШНО!${NC}"
    echo -e "${GREEN}🚀 Система движения кораблей работает корректно в интеграционном режиме!${NC}"
    exit 0
else
    echo ""
    echo -e "${YELLOW}⚠️  НЕКОТОРЫЕ ИНТЕГРАЦИОННЫЕ ТЕСТЫ НЕ ПРОШЛИ${NC}"
    echo -e "${YELLOW}🔧 Рекомендуется проверить логи и исправить ошибки${NC}"
    exit 1
fi
