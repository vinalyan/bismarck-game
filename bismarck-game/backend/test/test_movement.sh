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
    body=$(echo "$response" | sed '$d')
    
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
    
    local game_data='{"name":"Movement Test Game","description":"Test game for movement testing","side":"german"}'
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

# Функция для подключения второго игрока к игре
join_game() {
    local token="$1"
    local game_id="$2"
    local side="$3"
    
    local join_data='{"side":"'$side'","password":""}'
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "$join_data" \
        "$BASE_URL/api/games/$game_id/join")
    
    if [ $? -eq 0 ]; then
        echo "$response" | jq -r '.success' 2>/dev/null
    else
        echo ""
    fi
}

# Функция для начала хода
start_turn() {
    local token="$1"
    local game_id="$2"
    
    local turn_data='{"game_id":"'$game_id'"}'
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "$turn_data" \
        "$BASE_URL/api/phases/turn/start")
    
    if [ $? -eq 0 ]; then
        echo "$response" | jq -r '.success' 2>/dev/null
    else
        echo ""
    fi
}

# Функция для создания корабля
create_ship() {
    local token="$1"
    local game_id="$2"
    local ship_id="$3"
    local position="$4"
    local owner="$5"
    
    local ship_data='{"ship_id":"'$ship_id'","game_id":"'$game_id'","owner":"'$owner'","position":"'$position'"}'
    local response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "$ship_data" \
        "$BASE_URL/api/ships/create-unit")
    
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
    
    local movement_data='{"unit_id":"'$unit_id'","to_hex":"'$to_hex'"}'
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

# Функция для получения кораблей игры
get_game_units() {
    local token="$1"
    local game_id="$2"
    
    local response=$(curl -s -X GET \
        -H "Authorization: Bearer $token" \
        "$BASE_URL/api/games/$game_id/units")
    
    if [ $? -eq 0 ]; then
        echo "$response" | jq -r '.data.units[] | .id' 2>/dev/null | tr '\n' ' '
    else
        echo ""
    fi
}

# Функция для получения информации о корабле
get_unit_info() {
    local token="$1"
    local game_id="$2"
    local unit_id="$3"
    
    local response=$(curl -s -X GET \
        -H "Authorization: Bearer $token" \
        "$BASE_URL/api/games/$game_id/units/$unit_id")
    
    if [ $? -eq 0 ]; then
        echo "$response"
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
TOKEN1="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjEzMjcwMTIsImlhdCI6MTc2MTI0MDYxMiwibmJmIjoxNzYxMjQwNjEyLCJ1c2VyX2lkIjoiODUyYjY4MGUtNDNiYi00NThjLWFlODEtOWIxZDdlNjU0OWU2IiwidXNlcm5hbWUiOiJ0ZXN0dXNlcjE3NjEyNDA1OTIifQ.gJtwLCEwNMnDuh80h6mDfzJvCeCOHast9L1AK6D6nOc"
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

echo "🔗 Подключение второго игрока к игре..."
JOIN_RESULT=$(join_game "$TOKEN2" "$GAME_ID" "allied")
if [ "$JOIN_RESULT" = "true" ]; then
    echo -e "${GREEN}✅ Второй игрок подключен к игре${NC}"
else
    echo -e "${RED}❌ Не удалось подключить второго игрока${NC}"
    exit 1
fi

echo "🎯 Начало первого хода (фаза движения)..."
TURN_RESULT=$(start_turn "$TOKEN1" "$GAME_ID")
if [ "$TURN_RESULT" = "true" ]; then
    echo -e "${GREEN}✅ Первый ход начат, игра переведена в фазу движения${NC}"
else
    echo -e "${RED}❌ Не удалось начать первый ход${NC}"
    exit 1
fi
echo ""

echo "🚢 Получение кораблей игры..."

# Получаем корабли, созданные игрой автоматически
GAME_UNITS=$(get_game_units "$TOKEN1" "$GAME_ID")
if [ -z "$GAME_UNITS" ]; then
    echo -e "${RED}❌ Не удалось получить корабли игры${NC}"
    exit 1
fi

# Преобразуем в массив
UNIT_IDS=($GAME_UNITS)
echo -e "${GREEN}✅ Найдено кораблей: ${#UNIT_IDS[@]}${NC}"

# Выводим информацию о кораблях
for unit_id in "${UNIT_IDS[@]}"; do
    UNIT_INFO=$(get_unit_info "$TOKEN1" "$GAME_ID" "$unit_id")
    UNIT_NAME=$(echo "$UNIT_INFO" | jq -r '.unit.name' 2>/dev/null)
    UNIT_POSITION=$(echo "$UNIT_INFO" | jq -r '.unit.position' 2>/dev/null)
    UNIT_SPEED=$(echo "$UNIT_INFO" | jq -r '.unit.speed_rating' 2>/dev/null)
    echo -e "${BLUE}   Корабль: $UNIT_NAME (ID: $unit_id, Позиция: $UNIT_POSITION, Скорость: $UNIT_SPEED)${NC}"
done

# Выбираем корабли для тестирования по типу скорости
FAST_SHIP_ID=""
MEDIUM_SHIP_ID=""
SLOW_SHIP_ID=""
VERY_SLOW_SHIP_ID=""

for unit_id in "${UNIT_IDS[@]}"; do
    UNIT_INFO=$(get_unit_info "$TOKEN1" "$GAME_ID" "$unit_id")
    UNIT_SPEED=$(echo "$UNIT_INFO" | jq -r '.unit.speed_rating' 2>/dev/null)
    UNIT_NAME=$(echo "$UNIT_INFO" | jq -r '.unit.name' 2>/dev/null)
    
    case "$UNIT_SPEED" in
        "F")
            if [ -z "$FAST_SHIP_ID" ]; then
                FAST_SHIP_ID="$unit_id"
                echo -e "${GREEN}✅ Быстрый корабль выбран: $UNIT_NAME (ID: $unit_id)${NC}"
            fi
            ;;
        "M")
            if [ -z "$MEDIUM_SHIP_ID" ]; then
                MEDIUM_SHIP_ID="$unit_id"
                echo -e "${GREEN}✅ Средний корабль выбран: $UNIT_NAME (ID: $unit_id)${NC}"
            fi
            ;;
        "S")
            if [ -z "$SLOW_SHIP_ID" ]; then
                SLOW_SHIP_ID="$unit_id"
                echo -e "${GREEN}✅ Медленный корабль выбран: $UNIT_NAME (ID: $unit_id)${NC}"
            fi
            ;;
        "VS")
            if [ -z "$VERY_SLOW_SHIP_ID" ]; then
                VERY_SLOW_SHIP_ID="$unit_id"
                echo -e "${GREEN}✅ Очень медленный корабль выбран: $UNIT_NAME (ID: $unit_id)${NC}"
            fi
            ;;
    esac
done

# Проверяем, что у нас есть хотя бы один корабль для тестирования
if [ -z "$FAST_SHIP_ID" ] && [ -z "$MEDIUM_SHIP_ID" ] && [ -z "$SLOW_SHIP_ID" ] && [ -z "$VERY_SLOW_SHIP_ID" ]; then
    echo -e "${RED}❌ Не найдено кораблей для тестирования${NC}"
    exit 1
fi

echo ""
echo "🧪 НАЧАЛО ТЕСТИРОВАНИЯ ДВИЖЕНИЯ"
echo "==============================="
echo ""

# Тест 1: Движение быстрого корабля
if [ -n "$FAST_SHIP_ID" ]; then
    echo "🚀 Тест 1: Движение быстрого корабля (F)"
    # Получаем текущую позицию корабля
    FAST_SHIP_INFO=$(get_unit_info "$TOKEN1" "$GAME_ID" "$FAST_SHIP_ID")
    FAST_SHIP_POSITION=$(echo "$FAST_SHIP_INFO" | jq -r '.unit.position' 2>/dev/null)
    
    if test_movement "$TOKEN1" "$GAME_ID" "$FAST_SHIP_ID" "$FAST_SHIP_POSITION" "J23" "true"; then
        log_test "Движение быстрого корабля" "PASS"
    else
        log_test "Движение быстрого корабля" "FAIL" "Не удалось переместить быстрый корабль"
    fi
else
    log_test "Движение быстрого корабля" "SKIP" "Быстрый корабль не найден"
fi

# Тест 2: Движение среднего корабля
if [ -n "$MEDIUM_SHIP_ID" ]; then
    echo "🚀 Тест 2: Движение среднего корабля (M)"
    MEDIUM_SHIP_INFO=$(get_unit_info "$TOKEN1" "$GAME_ID" "$MEDIUM_SHIP_ID")
    MEDIUM_SHIP_POSITION=$(echo "$MEDIUM_SHIP_INFO" | jq -r '.unit.position' 2>/dev/null)
    
    if test_movement "$TOKEN1" "$GAME_ID" "$MEDIUM_SHIP_ID" "$MEDIUM_SHIP_POSITION" "K23" "true"; then
        log_test "Движение среднего корабля" "PASS"
    else
        log_test "Движение среднего корабля" "FAIL" "Не удалось переместить средний корабль"
    fi
else
    log_test "Движение среднего корабля" "SKIP" "Средний корабль не найден"
fi

# Тест 3: Движение медленного корабля
if [ -n "$SLOW_SHIP_ID" ]; then
    echo "🚀 Тест 3: Движение медленного корабля (S)"
    SLOW_SHIP_INFO=$(get_unit_info "$TOKEN1" "$GAME_ID" "$SLOW_SHIP_ID")
    SLOW_SHIP_POSITION=$(echo "$SLOW_SHIP_INFO" | jq -r '.unit.position' 2>/dev/null)
    
    if test_movement "$TOKEN1" "$GAME_ID" "$SLOW_SHIP_ID" "$SLOW_SHIP_POSITION" "L23" "true"; then
        log_test "Движение медленного корабля" "PASS"
    else
        log_test "Движение медленного корабля" "FAIL" "Не удалось переместить медленный корабль"
    fi
else
    log_test "Движение медленного корабля" "SKIP" "Медленный корабль не найден"
fi

# Тест 4: Движение очень медленного корабля
if [ -n "$VERY_SLOW_SHIP_ID" ]; then
    echo "🚀 Тест 4: Движение очень медленного корабля (VS)"
    VERY_SLOW_SHIP_INFO=$(get_unit_info "$TOKEN1" "$GAME_ID" "$VERY_SLOW_SHIP_ID")
    VERY_SLOW_SHIP_POSITION=$(echo "$VERY_SLOW_SHIP_INFO" | jq -r '.unit.position' 2>/dev/null)
    
    if test_movement "$TOKEN1" "$GAME_ID" "$VERY_SLOW_SHIP_ID" "$VERY_SLOW_SHIP_POSITION" "M23" "true"; then
        log_test "Движение очень медленного корабля" "PASS"
    else
        log_test "Движение очень медленного корабля" "FAIL" "Не удалось переместить очень медленный корабль"
    fi
else
    log_test "Движение очень медленного корабля" "SKIP" "Очень медленный корабль не найден"
fi

# Тест 5: Проверка доступных ходов для быстрого корабля
if [ -n "$FAST_SHIP_ID" ]; then
    echo "🚀 Тест 5: Проверка доступных ходов для быстрого корабля"
    AVAILABLE_MOVES=$(get_available_moves "$TOKEN1" "$GAME_ID" "$FAST_SHIP_ID")
    if [ -n "$AVAILABLE_MOVES" ]; then
        log_test "Доступные ходы для быстрого корабля" "PASS"
    else
        log_test "Доступные ходы для быстрого корабля" "FAIL" "Не удалось получить доступные ходы"
    fi
else
    log_test "Доступные ходы для быстрого корабля" "SKIP" "Быстрый корабль не найден"
fi

# Тест 6: Проверка доступных ходов для среднего корабля
if [ -n "$MEDIUM_SHIP_ID" ]; then
    echo "🚀 Тест 6: Проверка доступных ходов для среднего корабля"
    AVAILABLE_MOVES=$(get_available_moves "$TOKEN1" "$GAME_ID" "$MEDIUM_SHIP_ID")
    if [ -n "$AVAILABLE_MOVES" ]; then
        log_test "Доступные ходы для среднего корабля" "PASS"
    else
        log_test "Доступные ходы для среднего корабля" "FAIL" "Не удалось получить доступные ходы"
    fi
else
    log_test "Доступные ходы для среднего корабля" "SKIP" "Средний корабль не найден"
fi

# Тест 7: Проверка доступных ходов для медленного корабля
if [ -n "$SLOW_SHIP_ID" ]; then
    echo "🚀 Тест 7: Проверка доступных ходов для медленного корабля"
    AVAILABLE_MOVES=$(get_available_moves "$TOKEN1" "$GAME_ID" "$SLOW_SHIP_ID")
    if [ -n "$AVAILABLE_MOVES" ]; then
        log_test "Доступные ходы для медленного корабля" "PASS"
    else
        log_test "Доступные ходы для медленного корабля" "FAIL" "Не удалось получить доступные ходы"
    fi
else
    log_test "Доступные ходы для медленного корабля" "SKIP" "Медленный корабль не найден"
fi

# Тест 8: Проверка доступных ходов для очень медленного корабля
if [ -n "$VERY_SLOW_SHIP_ID" ]; then
    echo "🚀 Тест 8: Проверка доступных ходов для очень медленного корабля"
    AVAILABLE_MOVES=$(get_available_moves "$TOKEN1" "$GAME_ID" "$VERY_SLOW_SHIP_ID")
    if [ -n "$AVAILABLE_MOVES" ]; then
        log_test "Доступные ходы для очень медленного корабля" "PASS"
    else
        log_test "Доступные ходы для очень медленного корабля" "FAIL" "Не удалось получить доступные ходы"
    fi
else
    log_test "Доступные ходы для очень медленного корабля" "SKIP" "Очень медленный корабль не найден"
fi

# Тест 9: Попытка движения на слишком большое расстояние
if [ -n "$FAST_SHIP_ID" ]; then
    echo "🚀 Тест 9: Попытка движения на слишком большое расстояние"
    FAST_SHIP_INFO=$(get_unit_info "$TOKEN1" "$GAME_ID" "$FAST_SHIP_ID")
    FAST_SHIP_POSITION=$(echo "$FAST_SHIP_INFO" | jq -r '.unit.position' 2>/dev/null)
    
    if test_movement "$TOKEN1" "$GAME_ID" "$FAST_SHIP_ID" "$FAST_SHIP_POSITION" "J30" "false"; then
        log_test "Ограничение расстояния движения" "PASS"
    else
        log_test "Ограничение расстояния движения" "FAIL" "Движение на большое расстояние должно быть заблокировано"
    fi
else
    log_test "Ограничение расстояния движения" "SKIP" "Быстрый корабль не найден"
fi

# Тест 10: Попытка движения без топлива
echo "🚀 Тест 10: Попытка движения без топлива"
# Создаем корабль с 0 топливом (используем эсминец)
NO_FUEL_SHIP_ID=$(create_ship "$TOKEN1" "$GAME_ID" "6_zerstorerfl" "N22" "e3384df7-7ea8-40ee-97ba-c616908d59dd")
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
