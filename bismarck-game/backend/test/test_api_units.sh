#!/bin/bash

BASE_URL="http://localhost:8080"

echo "🚀 ТЕСТИРОВАНИЕ API КОРАБЛЕЙ С ПРАВИЛЬНОЙ ИНИЦИАЛИЗАЦИЕЙ"
echo "======================================================"

# Получаем токены
echo "🔐 Получение токенов авторизации..."
TOKEN1=$(curl -s -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"testuser1","password":"password123"}' | jq -r '.data.token')

TOKEN2=$(curl -s -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"testuser2","password":"password123"}' | jq -r '.data.token')

if [ "$TOKEN1" = "null" ] || [ "$TOKEN2" = "null" ]; then
    echo "❌ Ошибка получения токенов"
    exit 1
fi

echo "✅ Токены получены"
echo "   Token1: ${TOKEN1:0:20}..."
echo "   Token2: ${TOKEN2:0:20}..."

# Создаем игру (игрок за немцев)
echo -e "\n🎮 Создание игры (игрок за немцев)..."
GAME_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games" \
    -H "Authorization: Bearer $TOKEN1" \
    -H "Content-Type: application/json" \
    -d '{"name":"Test Game","side":"german"}')

echo "Ответ создания игры:"
echo "$GAME_RESPONSE" | jq '.'

GAME_ID=$(echo "$GAME_RESPONSE" | jq -r '.data.id')

if [ "$GAME_ID" = "null" ] || [ -z "$GAME_ID" ]; then
    echo "❌ Ошибка создания игры"
    echo "Ответ: $GAME_RESPONSE"
    exit 1
fi

echo "✅ Игра создана: $GAME_ID"

# Подключаем второго игрока
echo -e "\n🔗 Подключение второго игрока..."
JOIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/join" \
    -H "Authorization: Bearer $TOKEN2" \
    -H "Content-Type: application/json" \
    -d '{"faction":"allies"}')

echo "Ответ подключения:"
echo "$JOIN_RESPONSE" | jq '.'

if echo "$JOIN_RESPONSE" | jq -e '.success' > /dev/null; then
    echo "✅ Второй игрок подключен"
else
    echo "❌ Ошибка подключения второго игрока"
    echo "Ответ: $JOIN_RESPONSE"
    exit 1
fi

# Запускаем фазу (важно: это должен делать игрок за немцев)
echo -e "\n🎯 Запуск фазы движения (игрок за немцев)..."
PHASE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/phases/turn/start" \
    -H "Authorization: Bearer $TOKEN1" \
    -H "Content-Type: application/json" \
    -d "{\"game_id\":\"$GAME_ID\"}")

echo "Ответ запуска фазы:"
echo "$PHASE_RESPONSE" | jq '.'

if echo "$PHASE_RESPONSE" | jq -e '.success' > /dev/null; then
    echo "✅ Фаза движения запущена"
else
    echo "❌ Ошибка запуска фазы"
    echo "Ответ: $PHASE_RESPONSE"
    exit 1
fi

# Получаем корабли
echo -e "\n🚢 Получение кораблей игры..."
UNITS_RESPONSE=$(curl -s -X GET "$BASE_URL/api/games/$GAME_ID/units" \
    -H "Authorization: Bearer $TOKEN1")

echo "Ответ получения кораблей:"
echo "$UNITS_RESPONSE" | jq '.'

if echo "$UNITS_RESPONSE" | jq -e '.success' > /dev/null; then
    echo "✅ Корабли получены"
    
    # Показываем информацию о кораблях
    echo -e "\n📋 Информация о кораблях:"
    echo "$UNITS_RESPONSE" | jq -r '.data.units[] | "Корабль: \(.name) (ID: \(.id), Позиция: \(.position), Скорость: \(.speed_rating))"'
    
    # Проверяем, есть ли корабли с позициями
    SHIPS_WITH_POSITIONS=$(echo "$UNITS_RESPONSE" | jq -r '.data.units[] | select(.position != null) | .name' | wc -l)
    echo -e "\n📊 Статистика:"
    echo "   Всего кораблей: $(echo "$UNITS_RESPONSE" | jq -r '.data.units | length')"
    echo "   Кораблей с позициями: $SHIPS_WITH_POSITIONS"
    
    if [ "$SHIPS_WITH_POSITIONS" -gt 0 ]; then
        echo "✅ Найдены корабли с позициями - можно тестировать движение"
    else
        echo "❌ Корабли без позиций - движение невозможно"
    fi
else
    echo "❌ Ошибка получения кораблей"
    echo "Ответ: $UNITS_RESPONSE"
    exit 1
fi

