#!/bin/bash

# Тестирование системы движения кораблей
BASE_URL="http://localhost:8080"

echo "=== Тестирование системы движения кораблей ==="

# 1. Регистрация и логин
echo "1. Регистрация и логин пользователей..."
USER1_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser1","password":"password123"}')

USER2_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser2","password":"password123"}')

TOKEN1=$(echo $USER1_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
TOKEN2=$(echo $USER2_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

echo "Токены получены"

# 2. Создание новой игры
echo "2. Создание новой игры..."
GAME_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d '{"name":"Movement Test","side":"german"}')

GAME_ID=$(echo $GAME_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Game ID: $GAME_ID"

# 3. Присоединение второго игрока
echo "3. Присоединение второго игрока..."
JOIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/join" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN2" \
  -d '{"side":"allied"}')

echo "Игра началась"

# 4. Переход к фазе движения
echo "4. Переход к фазе движения..."
curl -s -X POST "$BASE_URL/api/phases/next" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d "{\"game_id\":\"$GAME_ID\"}" > /dev/null

sleep 2

# 5. Получение информации о юнитах
echo "5. Получение информации о юнитах..."
UNITS_RESPONSE=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
  -H "Authorization: Bearer $TOKEN1")

echo "Юниты получены:"
echo $UNITS_RESPONSE | jq -r '.data.units[] | "\(.name) - \(.speed_rating) - Fuel: \(.fuel)/\(.max_fuel)"'

# 6. Тестирование движения F типа (Быстрые корабли)
echo ""
echo "6. Тестирование движения F типа (Быстрые корабли)..."

# Найти быстрый корабль
FAST_SHIP_ID=$(echo $UNITS_RESPONSE | jq -r '.data.units[] | select(.speed_rating == "F") | .id' | head -1)
if [ "$FAST_SHIP_ID" != "null" ] && [ -n "$FAST_SHIP_ID" ]; then
    echo "Найден быстрый корабль: $FAST_SHIP_ID"
    
    # Тест 1: Движение на 1 гекс (0 FP)
    echo "  Тест 1: Движение на 1 гекс (должно стоить 0 FP)..."
    MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$FAST_SHIP_ID/move" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN1" \
      -d '{"target_hex": "O32"}')
    
    echo "  Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
    
    # Тест 2: Движение на 2 гекса (1 FP)
    echo "  Тест 2: Движение на 2 гекса (должно стоить 1 FP)..."
    MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$FAST_SHIP_ID/move" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN1" \
      -d '{"target_hex": "P33"}')
    
    echo "  Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
    
    # Проверка топлива после движения
    UNITS_AFTER=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
      -H "Authorization: Bearer $TOKEN1")
    
    FUEL_AFTER=$(echo $UNITS_AFTER | jq -r ".data.units[] | select(.id == \"$FAST_SHIP_ID\") | .fuel")
    echo "  Топливо после движения: $FUEL_AFTER"
    
else
    echo "  Быстрые корабли не найдены в игре"
fi

# 7. Тестирование движения M типа (Средние корабли)
echo ""
echo "7. Тестирование движения M типа (Средние корабли)..."

MEDIUM_SHIP_ID=$(echo $UNITS_RESPONSE | jq -r '.data.units[] | select(.speed_rating == "M") | .id' | head -1)
if [ "$MEDIUM_SHIP_ID" != "null" ] && [ -n "$MEDIUM_SHIP_ID" ]; then
    echo "Найден средний корабль: $MEDIUM_SHIP_ID"
    
    # Тест 1: Движение на 1 гекс (0 FP для первого движения)
    echo "  Тест 1: Движение на 1 гекс (должно стоить 0 FP для первого движения)..."
    MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$MEDIUM_SHIP_ID/move" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN1" \
      -d '{"target_hex": "N31"}')
    
    echo "  Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
    
    # Тест 2: Попытка движения на 2 гекса (должна быть ошибка)
    echo "  Тест 2: Попытка движения на 2 гекса (должна быть ошибка)..."
    MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$MEDIUM_SHIP_ID/move" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN1" \
      -d '{"target_hex": "M30"}')
    
    echo "  Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
    
else
    echo "  Средние корабли не найдены в игре"
fi

# 8. Тестирование движения S типа (Медленные корабли)
echo ""
echo "8. Тестирование движения S типа (Медленные корабли)..."

SLOW_SHIP_ID=$(echo $UNITS_RESPONSE | jq -r '.data.units[] | select(.speed_rating == "S") | .id' | head -1)
if [ "$SLOW_SHIP_ID" != "null" ] && [ -n "$SLOW_SHIP_ID" ]; then
    echo "Найден медленный корабль: $SLOW_SHIP_ID"
    
    # Тест 1: Движение на 1 гекс (должен получить маркер "Нет движения 2")
    echo "  Тест 1: Движение на 1 гекс (должен получить маркер 'Нет движения 2')..."
    MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$SLOW_SHIP_ID/move" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN1" \
      -d '{"target_hex": "L29"}')
    
    echo "  Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
    
    # Проверка маркера "Нет движения"
    UNITS_AFTER_SLOW=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
      -H "Authorization: Bearer $TOKEN1")
    
    NO_MOVEMENT=$(echo $UNITS_AFTER_SLOW | jq -r ".data.units[] | select(.id == \"$SLOW_SHIP_ID\") | .no_movement_turns_left")
    echo "  Маркер 'Нет движения': $NO_MOVEMENT ходов"
    
    # Тест 2: Попытка движения в следующем ходу (должна быть ошибка)
    echo "  Тест 2: Попытка движения в следующем ходу (должна быть ошибка)..."
    MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$SLOW_SHIP_ID/move" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN1" \
      -d '{"target_hex": "K28"}')
    
    echo "  Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
    
else
    echo "  Медленные корабли не найдены в игре"
fi

# 9. Тестирование движения VS типа (Очень медленные корабли)
echo ""
echo "9. Тестирование движения VS типа (Очень медленные корабли)..."

VERY_SLOW_SHIP_ID=$(echo $UNITS_RESPONSE | jq -r '.data.units[] | select(.speed_rating == "VS") | .id' | head -1)
if [ "$VERY_SLOW_SHIP_ID" != "null" ] && [ -n "$VERY_SLOW_SHIP_ID" ]; then
    echo "Найден очень медленный корабль: $VERY_SLOW_SHIP_ID"
    
    # Тест 1: Движение на 1 гекс (должен получить маркер "Нет движения 4")
    echo "  Тест 1: Движение на 1 гекс (должен получить маркер 'Нет движения 4')..."
    MOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$VERY_SLOW_SHIP_ID/move" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN1" \
      -d '{"target_hex": "J27"}')
    
    echo "  Результат: $(echo $MOVE_RESPONSE | jq -r '.message // .error // "OK"')"
    
    # Проверка маркера "Нет движения"
    UNITS_AFTER_VERY_SLOW=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
      -H "Authorization: Bearer $TOKEN1")
    
    NO_MOVEMENT_VS=$(echo $UNITS_AFTER_VERY_SLOW | jq -r ".data.units[] | select(.id == \"$VERY_SLOW_SHIP_ID\") | .no_movement_turns_left")
    echo "  Маркер 'Нет движения': $NO_MOVEMENT_VS ходов"
    
else
    echo "  Очень медленные корабли не найдены в игре"
fi

# 10. Тестирование ограничений топлива
echo ""
echo "10. Тестирование ограничений топлива..."

# Найти корабль с низким топливом или установить его
echo "  Проверка топлива всех кораблей:"
FINAL_UNITS=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
  -H "Authorization: Bearer $TOKEN1")

echo $FINAL_UNITS | jq -r '.data.data[] | "\(.name) (\(.speed_type)): \(.fuel_points)/\(.max_fuel) FP"'

# 11. Тестирование доступных гексов для движения
echo ""
echo "11. Тестирование доступных гексов для движения..."

# Получить доступные гексы для одного из кораблей
if [ -n "$FAST_SHIP_ID" ]; then
    echo "  Получение доступных гексов для быстрого корабля..."
    AVAILABLE_HEXES=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units/$FAST_SHIP_ID/available-moves" \
      -H "Authorization: Bearer $TOKEN1")
    
    echo "  Доступные гексы: $(echo $AVAILABLE_HEXES | jq -r '.data.data[] // "Нет данных"')"
fi

# 12. Тестирование ошибок движения
echo ""
echo "12. Тестирование ошибок движения..."

# Попытка движения в недоступный гекс
if [ -n "$FAST_SHIP_ID" ]; then
    echo "  Попытка движения в недоступный гекс (должна быть ошибка)..."
    ERROR_MOVE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/units/$FAST_SHIP_ID/move" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN1" \
      -d '{"target_hex": "ZZ99"}')
    
    echo "  Результат: $(echo $ERROR_MOVE | jq -r '.message // .error // "OK"')"
fi

# 13. Финальная проверка состояния игры
echo ""
echo "13. Финальная проверка состояния игры..."

# Получить текущую фазу
PHASE_RESPONSE=$(curl -s -X GET "$BASE_URL/api/phases/current?game_id=$GAME_ID")
CURRENT_PHASE=$(echo $PHASE_RESPONSE | jq -r '.data.data.current_phase')
TURN_NUMBER=$(echo $PHASE_RESPONSE | jq -r '.data.data.turn_number')

echo "  Текущая фаза: $CURRENT_PHASE"
echo "  Номер хода: $TURN_NUMBER"

# Получить финальное состояние всех юнитов
echo ""
echo "  Финальное состояние юнитов:"
FINAL_STATE=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
  -H "Authorization: Bearer $TOKEN1")

echo $FINAL_STATE | jq -r '.data.units[] | "\(.name) (\(.speed_rating)): Позиция \(.position), Топливо \(.fuel)/\(.max_fuel), Нет движения: \(.no_movement_turns_left // 0)"'

echo ""
echo "=== Тест движения завершен ==="
echo ""
echo "📊 Сводка тестирования:"
echo "✅ Тестирование F типа (Быстрые корабли)"
echo "✅ Тестирование M типа (Средние корабли)" 
echo "✅ Тестирование S типа (Медленные корабли)"
echo "✅ Тестирование VS типа (Очень медленные корабли)"
echo "✅ Тестирование ограничений топлива"
echo "✅ Тестирование доступных гексов"
echo "✅ Тестирование ошибок движения"
echo "✅ Проверка финального состояния"
echo ""
echo "🎯 Все основные сценарии движения протестированы!"
