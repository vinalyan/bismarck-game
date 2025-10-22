#!/bin/bash

# Тестирование последовательности фаз для Turn 2+
BASE_URL="http://localhost:8080"

echo "=== Тестирование последовательности фаз Turn 2+ ==="

# 1. Логин пользователей
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

# 2. Создание новой игры
echo "2. Создание новой игры..."
GAME_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d '{"name":"Turn 2+ Phase Test","side":"german"}')

GAME_ID=$(echo $GAME_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Game ID: $GAME_ID"

# 3. Присоединение второго игрока
echo "3. Присоединение второго игрока..."
JOIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/join" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN2" \
  -d '{"side":"allied"}')

echo "Игра началась"

# 4. Завершение setup фазы
echo "4. Завершение setup фазы..."
curl -s -X POST "$BASE_URL/api/phases/next" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d "{\"game_id\":\"$GAME_ID\"}" > /dev/null

sleep 2

# 5. Завершение первого хода (turn 1)
echo "5. Завершение первого хода..."
# Завершаем movement фазу
curl -s -X POST "$BASE_URL/api/phases/next" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d "{\"game_id\":\"$GAME_ID\"}" > /dev/null

sleep 2

# Ждем автоматического завершения остальных фаз turn 1
echo "Ожидание завершения turn 1..."
sleep 10

# 6. Проверка начала turn 2
echo "6. Проверка начала turn 2..."
PHASE_RESPONSE=$(curl -s -X GET "$BASE_URL/api/phases/current?game_id=$GAME_ID")
CURRENT_PHASE=$(echo $PHASE_RESPONSE | jq -r '.data.data.current_phase')
TURN_NUMBER=$(echo $PHASE_RESPONSE | jq -r '.data.data.turn_number')

echo "Turn $TURN_NUMBER, Phase: $CURRENT_PHASE"

# 7. Тестирование последовательности Turn 2+
echo "7. Тестирование последовательности Turn 2+..."
echo "Ожидаемая последовательность: visibility → shadow → movement → search → air_attack → naval_combat → chance → admin"

# Проверяем visibility фазу
if [ "$CURRENT_PHASE" = "visibility" ]; then
    echo "✅ Visibility фаза корректна"
else
    echo "❌ Ожидалась visibility фаза, получена: $CURRENT_PHASE"
fi

# Переходим к shadow фазе
echo "Переход к shadow фазе..."
curl -s -X POST "$BASE_URL/api/phases/next" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d "{\"game_id\":\"$GAME_ID\"}" > /dev/null

sleep 2

PHASE_RESPONSE=$(curl -s -X GET "$BASE_URL/api/phases/current?game_id=$GAME_ID")
CURRENT_PHASE=$(echo $PHASE_RESPONSE | jq -r '.data.data.current_phase')
echo "Фаза после visibility: $CURRENT_PHASE"

if [ "$CURRENT_PHASE" = "shadow" ]; then
    echo "✅ Shadow фаза корректна"
else
    echo "❌ Ожидалась shadow фаза, получена: $CURRENT_PHASE"
fi

# Переходим к movement фазе
echo "Переход к movement фазе..."
curl -s -X POST "$BASE_URL/api/phases/next" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d "{\"game_id\":\"$GAME_ID\"}" > /dev/null

sleep 2

PHASE_RESPONSE=$(curl -s -X GET "$BASE_URL/api/phases/current?game_id=$GAME_ID")
CURRENT_PHASE=$(echo $PHASE_RESPONSE | jq -r '.data.data.current_phase')
echo "Фаза после shadow: $CURRENT_PHASE"

if [ "$CURRENT_PHASE" = "movement" ]; then
    echo "✅ Movement фаза корректна"
else
    echo "❌ Ожидалась movement фаза, получена: $CURRENT_PHASE"
fi

# Завершаем movement фазу и проверяем автоматические переходы
echo "Завершение movement фазы и проверка автоматических переходов..."
curl -s -X POST "$BASE_URL/api/phases/next" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d "{\"game_id\":\"$GAME_ID\"}" > /dev/null

sleep 5

# Проверяем, что автоматически перешли к search
PHASE_RESPONSE=$(curl -s -X GET "$BASE_URL/api/phases/current?game_id=$GAME_ID")
CURRENT_PHASE=$(echo $PHASE_RESPONSE | jq -r '.data.data.current_phase')
echo "Фаза после movement: $CURRENT_PHASE"

if [ "$CURRENT_PHASE" = "search" ]; then
    echo "✅ Search фаза корректна"
else
    echo "❌ Ожидалась search фаза, получена: $CURRENT_PHASE"
fi

echo "=== Тест завершен ==="
