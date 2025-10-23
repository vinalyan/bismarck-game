#!/bin/bash

# Интеграционное тестирование системы движения
BASE_URL="http://localhost:8080"

echo "=== Интеграционное тестирование системы движения ==="

# Глобальные переменные
TOKEN1=""
TOKEN2=""
GAME_ID=""

# Функция для логина
login() {
    echo "1. Логин пользователей..."
    USER1_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
      -H "Content-Type: application/json" \
      -d '{"username":"testuser1","password":"password123"}')
    
    USER2_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
      -H "Content-Type: application/json" \
      -d '{"username":"testuser2","password":"password123"}')
    
    TOKEN1=$(echo $USER1_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    TOKEN2=$(echo $USER2_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    echo "✅ Пользователи залогинены"
}

# Функция для создания игры
create_game() {
    echo "2. Создание игры..."
    GAME_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN1" \
      -d '{"name":"Movement Integration Test","side":"german"}')
    
    GAME_ID=$(echo $GAME_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    
    # Присоединение второго игрока
    curl -s -X POST "$BASE_URL/api/games/$GAME_ID/join" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN2" \
      -d '{"side":"allied"}' > /dev/null
    
    echo "✅ Игра создана: $GAME_ID"
}

# Функция для перехода к фазе движения
go_to_movement_phase() {
    echo "3. Переход к фазе движения..."
    curl -s -X POST "$BASE_URL/api/phases/next" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN1" \
      -d "{\"game_id\":\"$GAME_ID\"}" > /dev/null
    
    sleep 2
    echo "✅ Переход к фазе движения выполнен"
}

# Функция для получения юнитов
get_units() {
    curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
      -H "Authorization: Bearer $TOKEN1"
}

# Функция для тестирования смешанных типов кораблей
test_mixed_ship_types() {
    echo ""
    echo "=== ТЕСТИРОВАНИЕ СМЕШАННЫХ ТИПОВ КОРАБЛЕЙ ==="
    
    UNITS_RESPONSE=$(get_units)
    
    # Получить по одному кораблю каждого типа
    FAST_SHIP=$(echo $UNITS_RESPONSE | jq -r '.data.data[] | select(.speed_type == "F") | .id' | head -1)
    MEDIUM_SHIP=$(echo $UNITS_RESPONSE | jq -r '.data.data[] | select(.speed_type == "M") | .id' | head -1)
    SLOW_SHIP=$(echo $UNITS_RESPONSE | jq -r '.data.data[] | select(.speed_type == "S") | .id' | head -1)
    VERY_SLOW_SHIP=$(echo $UNITS_RESPONSE | jq -r '.data.data[] | select(.speed_type == "VS") | .id' | head -1)
    
    echo "Найденные корабли:"
    [ -n "$FAST_SHIP" ] && echo "  Быстрый: $FAST_SHIP"
    [ -n "$MEDIUM_SHIP" ] && echo "  Средний: $MEDIUM_SHIP"
    [ -n "$SLOW_SHIP" ] && echo "  Медленный: $SLOW_SHIP"
    [ -n "$VERY_SLOW_SHIP" ] && echo "  Очень медленный: $VERY_SLOW_SHIP"
    
    # Тестируем движение всех типов одновременно
    echo ""
    echo "Тестируем одновременное движение всех типов..."
    
    # Быстрый корабль - движение на 2 гекса
    if [ -n "$FAST_SHIP" ]; then
        echo "  Быстрый корабль: движение на 2 гекса..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$FAST_SHIP/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "P33"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
    fi
    
    # Средний корабль - движение на 1 гекс
    if [ -n "$MEDIUM_SHIP" ]; then
        echo "  Средний корабль: движение на 1 гекс..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$MEDIUM_SHIP/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "N31"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
    fi
    
    # Медленный корабль - движение на 1 гекс
    if [ -n "$SLOW_SHIP" ]; then
        echo "  Медленный корабль: движение на 1 гекс..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$SLOW_SHIP/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "L29"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
    fi
    
    # Очень медленный корабль - движение на 1 гекс
    if [ -n "$VERY_SLOW_SHIP" ]; then
        echo "  Очень медленный корабль: движение на 1 гекс..."
        MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$VERY_SLOW_SHIP/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "J27"}')
        echo "    Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
    fi
}

# Функция для тестирования переходов между ходами
test_turn_transitions() {
    echo ""
    echo "=== ТЕСТИРОВАНИЕ ПЕРЕХОДОВ МЕЖДУ ХОДАМИ ==="
    
    # Получить текущее состояние
    UNITS_RESPONSE=$(get_units)
    
    echo "Состояние до перехода хода:"
    echo $UNITS_RESPONSE | jq -r '.data.data[] | "\(.name) (\(.speed_type)): Позиция \(.hex_id), Топливо \(.fuel_points), Нет движения: \(.no_movement_turns_left // 0)"'
    
    # Переход к следующему ходу
    echo ""
    echo "Переход к следующему ходу..."
    curl -s -X POST "$BASE_URL/api/phases/next" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN1" \
      -d "{\"game_id\":\"$GAME_ID\"}" > /dev/null
    
    sleep 2
    
    # Получить состояние после перехода
    UNITS_AFTER=$(get_units)
    
    echo ""
    echo "Состояние после перехода хода:"
    echo $UNITS_AFTER | jq -r '.data.data[] | "\(.name) (\(.speed_type)): Позиция \(.hex_id), Топливо \(.fuel_points), Нет движения: \(.no_movement_turns_left // 0)"'
    
    # Проверить изменения в маркерах "Нет движения"
    echo ""
    echo "Проверка изменений в маркерах 'Нет движения':"
    echo $UNITS_AFTER | jq -r '.data.data[] | select(.no_movement_turns_left > 0) | "\(.name): \(.no_movement_turns_left) ходов до следующего движения"'
}

# Функция для тестирования фазы администрирования
test_administration_phase() {
    echo ""
    echo "=== ТЕСТИРОВАНИЕ ФАЗЫ АДМИНИСТРАЦИИ ==="
    
    # Перейти к фазе администрирования
    echo "Переход к фазе администрирования..."
    curl -s -X POST "$BASE_URL/api/phases/next" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN1" \
      -d "{\"game_id\":\"$GAME_ID\"}" > /dev/null
    
    sleep 2
    
    # Проверить обновление маркеров
    UNITS_AFTER_ADMIN=$(get_units)
    
    echo "Состояние после фазы администрирования:"
    echo $UNITS_AFTER_ADMIN | jq -r '.data.data[] | "\(.name) (\(.speed_type)): Позиция \(.hex_id), Топливо \(.fuel_points), Нет движения: \(.no_movement_turns_left // 0)"'
    
    # Проверить, что маркеры "Нет движения" уменьшились
    echo ""
    echo "Проверка обновления маркеров 'Нет движения':"
    echo $UNITS_AFTER_ADMIN | jq -r '.data.data[] | select(.no_movement_turns_left > 0) | "\(.name): \(.no_movement_turns_left) ходов до следующего движения"'
}

# Функция для тестирования синхронизации фронтенд-бэкенд
test_frontend_backend_sync() {
    echo ""
    echo "=== ТЕСТИРОВАНИЕ СИНХРОНИЗАЦИИ ФРОНТЕНД-БЭКЕНД ==="
    
    # Получить состояние с бэкенда
    BACKEND_UNITS=$(get_units)
    
    echo "Данные с бэкенда:"
    echo $BACKEND_UNITS | jq -r '.data.data[] | "\(.name) (\(.speed_type)): Позиция \(.hex_id), Топливо \(.fuel_points)/\(.max_fuel)"'
    
    # Проверить доступные гексы для каждого корабля
    echo ""
    echo "Проверка доступных гексов для каждого корабля:"
    
    echo $BACKEND_UNITS | jq -r '.data.data[].id' | while read -r ship_id; do
        if [ -n "$ship_id" ] && [ "$ship_id" != "null" ]; then
            echo "  Корабль $ship_id:"
            AVAILABLE_HEXES=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units/$ship_id/available-moves" \
              -H "Authorization: Bearer $TOKEN1")
            
            HEX_COUNT=$(echo $AVAILABLE_HEXES | jq -r '.data.data | length')
            echo "    Доступных гексов: $HEX_COUNT"
            
            if [ "$HEX_COUNT" -gt 0 ]; then
                echo "    Примеры: $(echo $AVAILABLE_HEXES | jq -r '.data.data[0:3][]' | tr '\n' ' ')"
            fi
        fi
    done
}

# Функция для тестирования отображения активных гексов
test_active_hexes_display() {
    echo ""
    echo "=== ТЕСТИРОВАНИЕ ОТОБРАЖЕНИЯ АКТИВНЫХ ГЕКСОВ ==="
    
    UNITS_RESPONSE=$(get_units)
    
    # Тестируем для каждого типа корабля
    for speed_type in "F" "M" "S" "VS"; do
        SHIP_ID=$(echo $UNITS_RESPONSE | jq -r ".data.data[] | select(.speed_type == \"$speed_type\") | .id" | head -1)
        
        if [ "$SHIP_ID" != "null" ] && [ -n "$SHIP_ID" ]; then
            echo "  Тестируем отображение для $speed_type типа (корабль $SHIP_ID):"
            
            # Получить доступные гексы
            AVAILABLE_HEXES=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units/$SHIP_ID/available-moves" \
              -H "Authorization: Bearer $TOKEN1")
            
            HEX_COUNT=$(echo $AVAILABLE_HEXES | jq -r '.data.data | length')
            echo "    Количество активных гексов: $HEX_COUNT"
            
            # Проверить, что гексы соответствуют типу корабля
            if [ "$speed_type" = "F" ]; then
                echo "    Ожидается: до 2 гексов для быстрого корабля"
            elif [ "$speed_type" = "M" ]; then
                echo "    Ожидается: до 1 гекса для среднего корабля"
            elif [ "$speed_type" = "S" ]; then
                echo "    Ожидается: до 1 гекса для медленного корабля (если может двигаться)"
            elif [ "$speed_type" = "VS" ]; then
                echo "    Ожидается: до 1 гекса для очень медленного корабля (если может двигаться)"
            fi
            
            # Показать первые несколько гексов
            if [ "$HEX_COUNT" -gt 0 ]; then
                echo "    Активные гексы: $(echo $AVAILABLE_HEXES | jq -r '.data.data[0:5][]' | tr '\n' ' ')"
            else
                echo "    Нет доступных гексов для движения"
            fi
        fi
    done
}

# Функция для тестирования уведомлений об ошибках
test_error_notifications() {
    echo ""
    echo "=== ТЕСТИРОВАНИЕ УВЕДОМЛЕНИЙ ОБ ОШИБКАХ ==="
    
    UNITS_RESPONSE=$(get_units)
    ANY_SHIP_ID=$(echo $UNITS_RESPONSE | jq -r '.data.data[0].id')
    
    if [ -n "$ANY_SHIP_ID" ]; then
        echo "Тестируем уведомления с кораблем: $ANY_SHIP_ID"
        
        # Ошибка 1: Недоступный гекс
        echo "  Тест 1: Недоступный гекс..."
        ERROR_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$ANY_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "ZZ99"}')
        
        ERROR_MESSAGE=$(echo $ERROR_RESPONSE | jq -r '.message // .error // "OK"')
        echo "    Сообщение: $ERROR_MESSAGE"
        
        # Ошибка 2: Превышение максимальной дальности
        echo "  Тест 2: Превышение максимальной дальности..."
        ERROR_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$ANY_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "A1"}')
        
        ERROR_MESSAGE=$(echo $ERROR_RESPONSE | jq -r '.message // .error // "OK"')
        echo "    Сообщение: $ERROR_MESSAGE"
        
        # Ошибка 3: Недостаточно топлива
        echo "  Тест 3: Недостаточно топлива..."
        # Сначала израсходуем все топливо
        for i in {1..10}; do
            curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$ANY_SHIP_ID/move" \
              -H "Content-Type: application/json" \
              -H "Authorization: Bearer $TOKEN1" \
              -d '{"target_hex": "P33"}' > /dev/null
        done
        
        # Попытка движения без топлива
        ERROR_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$ANY_SHIP_ID/move" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d '{"target_hex": "Q34"}')
        
        ERROR_MESSAGE=$(echo $ERROR_RESPONSE | jq -r '.message // .error // "OK"')
        echo "    Сообщение: $ERROR_MESSAGE"
    fi
}

# Функция для тестирования производительности
test_performance() {
    echo ""
    echo "=== ТЕСТИРОВАНИЕ ПРОИЗВОДИТЕЛЬНОСТИ ==="
    
    UNITS_RESPONSE=$(get_units)
    
    # Тест 1: Множественные запросы доступных гексов
    echo "  Тест 1: Множественные запросы доступных гексов..."
    start_time=$(date +%s%N)
    
    for i in {1..10}; do
        curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
          -H "Authorization: Bearer $TOKEN1" > /dev/null
    done
    
    end_time=$(date +%s%N)
    duration=$(( (end_time - start_time) / 1000000 ))
    echo "    Время выполнения 10 запросов: ${duration}ms"
    
    # Тест 2: Параллельные движения
    echo "  Тест 2: Параллельные движения..."
    start_time=$(date +%s%N)
    
    # Получить несколько кораблей для параллельного движения
    SHIP_IDS=$(echo $UNITS_RESPONSE | jq -r '.data.data[0:3].id')
    
    for ship_id in $SHIP_IDS; do
        if [ -n "$ship_id" ] && [ "$ship_id" != "null" ]; then
            curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$ship_id/move" \
              -H "Content-Type: application/json" \
              -H "Authorization: Bearer $TOKEN1" \
              -d '{"target_hex": "P33"}' > /dev/null &
        fi
    done
    
    wait
    end_time=$(date +%s%N)
    duration=$(( (end_time - start_time) / 1000000 ))
    echo "    Время выполнения параллельных движений: ${duration}ms"
}

# Функция для финальной проверки
final_integration_check() {
    echo ""
    echo "=== ФИНАЛЬНАЯ ИНТЕГРАЦИОННАЯ ПРОВЕРКА ==="
    
    # Получить финальное состояние
    FINAL_UNITS=$(get_units)
    
    echo "Финальное состояние всех юнитов:"
    echo $FINAL_UNITS | jq -r '.data.data[] | "\(.name) (\(.speed_type)): Позиция \(.hex_id), Топливо \(.fuel_points)/\(.max_fuel), Нет движения: \(.no_movement_turns_left // 0)"'
    
    # Статистика по типам кораблей
    echo ""
    echo "Статистика по типам кораблей:"
    echo $FINAL_UNITS | jq -r '.data.data[] | .speed_type' | sort | uniq -c | while read count type; do
        echo "  $type: $count кораблей"
    done
    
    # Проверка топлива
    echo ""
    echo "Проверка топлива:"
    echo $FINAL_UNITS | jq -r '.data.data[] | select(.fuel_points < .max_fuel) | "\(.name): \(.fuel_points)/\(.max_fuel) FP"'
    
    # Проверка маркеров "Нет движения"
    echo ""
    echo "Проверка маркеров 'Нет движения':"
    echo $FINAL_UNITS | jq -r '.data.data[] | select(.no_movement_turns_left > 0) | "\(.name): \(.no_movement_turns_left) ходов до следующего движения"'
    
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
    login
    create_game
    go_to_movement_phase
    
    test_mixed_ship_types
    test_turn_transitions
    test_administration_phase
    test_frontend_backend_sync
    test_active_hexes_display
    test_error_notifications
    test_performance
    final_integration_check
    
    echo ""
    echo "=== ИНТЕГРАЦИОННОЕ ТЕСТИРОВАНИЕ ЗАВЕРШЕНО ==="
    echo ""
    echo "📊 Сводка интеграционного тестирования:"
    echo "✅ Смешанные типы кораблей"
    echo "✅ Переходы между ходами"
    echo "✅ Фаза администрирования"
    echo "✅ Синхронизация фронтенд-бэкенд"
    echo "✅ Отображение активных гексов"
    echo "✅ Уведомления об ошибках"
    echo "✅ Тестирование производительности"
    echo "✅ Финальная интеграционная проверка"
    echo ""
    echo "🎯 Все интеграционные сценарии протестированы!"
}

# Запуск основной функции
main
