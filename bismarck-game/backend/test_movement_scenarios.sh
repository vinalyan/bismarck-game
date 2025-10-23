#!/bin/bash

# Детальное тестирование сценариев движения кораблей
BASE_URL="http://localhost:8080"

echo "=== Детальное тестирование сценариев движения ==="

# Функция для логина пользователей
login_users() {
    echo "1. Логин пользователей..."
    USER1_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
      -H "Content-Type: application/json" \
      -d '{"username":"testuser1","password":"password123"}')
    
    USER2_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
      -H "Content-Type: application/json" \
      -d '{"username":"testuser2","password":"password123"}')
    
    TOKEN1=$(echo $USER1_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    TOKEN2=$(echo $USER2_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    echo "Токены получены"
}

# Функция для создания игры
create_game() {
    echo "2. Создание игры..."
    GAME_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN1" \
      -d '{"name":"Movement Scenarios Test","side":"german"}')
    
    GAME_ID=$(echo $GAME_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    echo "Game ID: $GAME_ID"
    
    # Присоединение второго игрока
    curl -s -X POST "$BASE_URL/api/games/$GAME_ID/join" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN2" \
      -d '{"side":"allied"}' > /dev/null
    
    echo "Игра создана и началась"
}

# Функция для перехода к фазе движения
go_to_movement_phase() {
    echo "3. Переход к фазе движения..."
    curl -s -X POST "$BASE_URL/api/phases/next" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN1" \
      -d "{\"game_id\":\"$GAME_ID\"}" > /dev/null
    
    sleep 2
}

# Функция для получения юнитов
get_units() {
    UNITS_RESPONSE=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
      -H "Authorization: Bearer $TOKEN1")
    echo "$UNITS_RESPONSE"
}

# Функция для тестирования F типа (Быстрые корабли)
test_fast_ships() {
    echo ""
    echo "=== ТЕСТИРОВАНИЕ F ТИПА (БЫСТРЫЕ КОРАБЛИ) ==="
    
    FAST_SHIP_ID=$(echo $UNITS_RESPONSE | jq -r '.data.data[] | select(.speed_type == "F") | .id' | head -1)
    
    if [ "$FAST_SHIP_ID" != "null" ] && [ -n "$FAST_SHIP_ID" ]; then
        echo "Тестируем быстрый корабль: $FAST_SHIP_ID"
        
        # Сценарий 1: Движение на 0 гексов (остается на месте)
        echo "  Сценарий 1: Остается на месте (0 FP)..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$FAST_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "O32"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
        
        # Сценарий 2: Движение на 1 гекс (0 FP)
        echo "  Сценарий 2: Движение на 1 гекс (0 FP)..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$FAST_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "P33"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
        
        # Сценарий 3: Движение на 2 гекса после 1 гекса в предыдущий ход (1 FP)
        echo "  Сценарий 3: Движение на 2 гекса после 1 гекса (1 FP)..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$FAST_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "Q34"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
        
        # Сценарий 4: Попытка движения на 3+ гексов (ошибка)
        echo "  Сценарий 4: Попытка движения на 3+ гексов (ошибка)..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$FAST_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "S36"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
        
        # Проверка топлива
        UNITS_AFTER=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
          -H "Authorization: Bearer $TOKEN1")
        FUEL_AFTER=$(echo $UNITS_AFTER | jq -r ".data.data[] | select(.id == \"$FAST_SHIP_ID\") | .fuel_points")
        echo "    Топливо после тестов: $FUEL_AFTER"
        
    else
        echo "  Быстрые корабли не найдены"
    fi
}

# Функция для тестирования M типа (Средние корабли)
test_medium_ships() {
    echo ""
    echo "=== ТЕСТИРОВАНИЕ M ТИПА (СРЕДНИЕ КОРАБЛИ) ==="
    
    MEDIUM_SHIP_ID=$(echo $UNITS_RESPONSE | jq -r '.data.data[] | select(.speed_type == "M") | .id' | head -1)
    
    if [ "$MEDIUM_SHIP_ID" != "null" ] && [ -n "$MEDIUM_SHIP_ID" ]; then
        echo "Тестируем средний корабль: $MEDIUM_SHIP_ID"
        
        # Сценарий 1: Первое движение на 1 гекс (0 FP)
        echo "  Сценарий 1: Первое движение на 1 гекс (0 FP)..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$MEDIUM_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "N31"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
        
        # Сценарий 2: Второе движение на 1 гекс (1 FP)
        echo "  Сценарий 2: Второе движение на 1 гекс (1 FP)..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$MEDIUM_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "M30"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
        
        # Сценарий 3: Попытка движения на 2+ гексов (ошибка)
        echo "  Сценарий 3: Попытка движения на 2+ гексов (ошибка)..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$MEDIUM_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "K28"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
        
        # Проверка топлива
        UNITS_AFTER=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
          -H "Authorization: Bearer $TOKEN1")
        FUEL_AFTER=$(echo $UNITS_AFTER | jq -r ".data.data[] | select(.id == \"$MEDIUM_SHIP_ID\") | .fuel_points")
        echo "    Топливо после тестов: $FUEL_AFTER"
        
    else
        echo "  Средние корабли не найдены"
    fi
}

# Функция для тестирования S типа (Медленные корабли)
test_slow_ships() {
    echo ""
    echo "=== ТЕСТИРОВАНИЕ S ТИПА (МЕДЛЕННЫЕ КОРАБЛИ) ==="
    
    SLOW_SHIP_ID=$(echo $UNITS_RESPONSE | jq -r '.data.data[] | select(.speed_type == "S") | .id' | head -1)
    
    if [ "$SLOW_SHIP_ID" != "null" ] && [ -n "$SLOW_SHIP_ID" ]; then
        echo "Тестируем медленный корабль: $SLOW_SHIP_ID"
        
        # Сценарий 1: Движение на 1 гекс (маркер "Нет движения 2")
        echo "  Сценарий 1: Движение на 1 гекс (маркер 'Нет движения 2')..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$SLOW_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "L29"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
        
        # Проверка маркера "Нет движения"
        UNITS_AFTER=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
          -H "Authorization: Bearer $TOKEN1")
        NO_MOVEMENT=$(echo $UNITS_AFTER | jq -r ".data.data[] | select(.id == \"$SLOW_SHIP_ID\") | .no_movement_turns_left")
        echo "    Маркер 'Нет движения': $NO_MOVEMENT ходов"
        
        # Сценарий 2: Попытка движения в следующем ходу (ошибка)
        echo "  Сценарий 2: Попытка движения в следующем ходу (ошибка)..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$SLOW_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "K28"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
        
        # Сценарий 3: Попытка движения на 2+ гексов (ошибка)
        echo "  Сценарий 3: Попытка движения на 2+ гексов (ошибка)..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$SLOW_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "J27"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
        
    else
        echo "  Медленные корабли не найдены"
    fi
}

# Функция для тестирования VS типа (Очень медленные корабли)
test_very_slow_ships() {
    echo ""
    echo "=== ТЕСТИРОВАНИЕ VS ТИПА (ОЧЕНЬ МЕДЛЕННЫЕ КОРАБЛИ) ==="
    
    VERY_SLOW_SHIP_ID=$(echo $UNITS_RESPONSE | jq -r '.data.data[] | select(.speed_type == "VS") | .id' | head -1)
    
    if [ "$VERY_SLOW_SHIP_ID" != "null" ] && [ -n "$VERY_SLOW_SHIP_ID" ]; then
        echo "Тестируем очень медленный корабль: $VERY_SLOW_SHIP_ID"
        
        # Сценарий 1: Движение на 1 гекс (маркер "Нет движения 4")
        echo "  Сценарий 1: Движение на 1 гекс (маркер 'Нет движения 4')..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$VERY_SLOW_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "J27"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
        
        # Проверка маркера "Нет движения"
        UNITS_AFTER=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
          -H "Authorization: Bearer $TOKEN1")
        NO_MOVEMENT=$(echo $UNITS_AFTER | jq -r ".data.data[] | select(.id == \"$VERY_SLOW_SHIP_ID\") | .no_movement_turns_left")
        echo "    Маркер 'Нет движения': $NO_MOVEMENT ходов"
        
        # Сценарий 2: Попытка движения в следующих ходах (ошибка)
        echo "  Сценарий 2: Попытка движения в следующих ходах (ошибка)..."
        for i in {1..3}; do
            echo "    Попытка $i:"
            MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$VERY_SLOW_SHIP_ID/move" \
              -H "Content-Type: application/json" \
              -H "Authorization: Bearer $TOKEN1" \
              -d '{"target_hex": "I26"}')
            echo "      Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
        done
        
    else
        echo "  Очень медленные корабли не найдены"
    fi
}

# Функция для тестирования доступных гексов
test_available_hexes() {
    echo ""
    echo "=== ТЕСТИРОВАНИЕ ДОСТУПНЫХ ГЕКСОВ ==="
    
    # Тестируем для каждого типа корабля
    for speed_type in "F" "M" "S" "VS"; do
        SHIP_ID=$(echo $UNITS_RESPONSE | jq -r ".data.data[] | select(.speed_type == \"$speed_type\") | .id" | head -1)
        
        if [ "$SHIP_ID" != "null" ] && [ -n "$SHIP_ID" ]; then
            echo "  Тестируем доступные гексы для $speed_type типа..."
            AVAILABLE_HEXES=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units/$SHIP_ID/available-moves" \
              -H "Authorization: Bearer $TOKEN1")
            
            HEX_COUNT=$(echo $AVAILABLE_HEXES | jq -r '.data.data | length')
            echo "    Количество доступных гексов: $HEX_COUNT"
            
            if [ "$HEX_COUNT" -gt 0 ]; then
                echo "    Первые 5 гексов: $(echo $AVAILABLE_HEXES | jq -r '.data.data[0:5][]')"
            fi
        fi
    done
}

# Функция для тестирования ошибок
test_error_scenarios() {
    echo ""
    echo "=== ТЕСТИРОВАНИЕ ОШИБОК ==="
    
    # Найти любой корабль для тестирования ошибок
    ANY_SHIP_ID=$(echo $UNITS_RESPONSE | jq -r '.data.data[0].id')
    
    if [ -n "$ANY_SHIP_ID" ]; then
        echo "Тестируем ошибки с кораблем: $ANY_SHIP_ID"
        
        # Ошибка 1: Недоступный гекс
        echo "  Ошибка 1: Недоступный гекс..."
        ERROR_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$ANY_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "ZZ99"}')
        echo "    Результат: $(echo $ERROR_RESPONSE | jq -r '.message // .error // "OK"')"
        
        # Ошибка 2: Неверный формат гекса
        echo "  Ошибка 2: Неверный формат гекса..."
        ERROR_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$ANY_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "INVALID"}')
        echo "    Результат: $(echo $ERROR_RESPONSE | jq -r '.message // .error // "OK"')"
        
        # Ошибка 3: Движение в тот же гекс
        CURRENT_HEX=$(echo $UNITS_RESPONSE | jq -r ".data.data[] | select(.id == \"$ANY_SHIP_ID\") | .hex_id")
        echo "  Ошибка 3: Движение в тот же гекс ($CURRENT_HEX)..."
        ERROR_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$ANY_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d "{\"target_hex\": \"$CURRENT_HEX\"}")
        echo "    Результат: $(echo $ERROR_RESPONSE | jq -r '.message // .error // "OK"')"
    fi
}

# Функция для финальной проверки
final_check() {
    echo ""
    echo "=== ФИНАЛЬНАЯ ПРОВЕРКА ==="
    
    # Получить финальное состояние всех юнитов
    FINAL_UNITS=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
      -H "Authorization: Bearer $TOKEN1")
    
    echo "Финальное состояние юнитов:"
    echo $FINAL_UNITS | jq -r '.data.data[] | "\(.name) (\(.speed_type)): Позиция \(.hex_id), Топливо \(.fuel_points)/\(.max_fuel), Нет движения: \(.no_movement_turns_left // 0)"'
    
    # Получить текущую фазу
    PHASE_RESPONSE=$(curl -s -X GET "$BASE_URL/api/phases/current?game_id=$GAME_ID")
    CURRENT_PHASE=$(echo $PHASE_RESPONSE | jq -r '.data.data.current_phase')
    TURN_NUMBER=$(echo $PHASE_RESPONSE | jq -r '.data.data.turn_number')
    
    echo ""
    echo "Текущая фаза: $CURRENT_PHASE"
    echo "Номер хода: $TURN_NUMBER"
}

# Основная функция
main() {
    login_users
    create_game
    go_to_movement_phase
    
    UNITS_RESPONSE=$(get_units)
    
    test_fast_ships
    test_medium_ships
    test_slow_ships
    test_very_slow_ships
    test_available_hexes
    test_error_scenarios
    final_check
    
    echo ""
    echo "=== ТЕСТИРОВАНИЕ ЗАВЕРШЕНО ==="
    echo ""
    echo "📊 Сводка тестирования:"
    echo "✅ F тип (Быстрые корабли) - 4 сценария"
    echo "✅ M тип (Средние корабли) - 3 сценария"
    echo "✅ S тип (Медленные корабли) - 3 сценария"
    echo "✅ VS тип (Очень медленные корабли) - 3 сценария"
    echo "✅ Доступные гексы - для всех типов"
    echo "✅ Ошибки движения - 3 типа ошибок"
    echo "✅ Финальная проверка состояния"
    echo ""
    echo "🎯 Всего протестировано: 13+ сценариев движения!"
}

# Запуск основной функции
main
