#!/bin/bash

# Полное тестирование последовательности фаз
BASE_URL="http://localhost:8080"

echo "=== Полное тестирование последовательности фаз ==="

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
  -d '{"name":"Complete Phase Test","side":"german"}')

GAME_ID=$(echo $GAME_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Game ID: $GAME_ID"

# 3. Присоединение второго игрока
echo "3. Присоединение второго игрока..."
JOIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/join" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN2" \
  -d '{"side":"allied"}')

echo "Игра началась"

# 4. Проверка setup фазы
echo "4. Проверка setup фазы..."
PHASE_RESPONSE=$(curl -s -X GET "$BASE_URL/api/phases/current?game_id=$GAME_ID")
echo "Setup фаза: $(echo $PHASE_RESPONSE | jq -r '.data.data.current_phase')"

# 5. Переход к первому ходу
echo "5. Переход к первому ходу..."
curl -s -X POST "$BASE_URL/api/phases/next" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d "{\"game_id\":\"$GAME_ID\"}" > /dev/null

sleep 2

# 6. Проверка фаз первого хода
echo "6. Проверка фаз первого хода..."
echo "Ожидаемая последовательность: setup → movement → search → air_attack → naval_combat → chance → admin"

for i in {1..6}; do
    echo "Переход $i:"
    PHASE_RESPONSE=$(curl -s -X GET "$BASE_URL/api/phases/current?game_id=$GAME_ID")
    CURRENT_PHASE=$(echo $PHASE_RESPONSE | jq -r '.data.data.current_phase')
    TURN_NUMBER=$(echo $PHASE_RESPONSE | jq -r '.data.data.turn_number')
    echo "  Turn $TURN_NUMBER, Phase: $CURRENT_PHASE"
    
    if [ "$CURRENT_PHASE" != "movement" ]; then
        echo "  Переходим к следующей фазе..."
        curl -s -X POST "$BASE_URL/api/phases/next" \
          -H "Content-Type: application/json" \
          -H "Authorization: Bearer $TOKEN1" \
          -d "{\"game_id\":\"$GAME_ID\"}" > /dev/null
        sleep 2
    else
        echo "  Фаза movement требует ручного завершения"
        break
    fi
done

echo "=== Тест завершен ==="
