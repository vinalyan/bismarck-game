#!/bin/bash

# Скрипт для проверки последовательности фаз
# Запуск: ./check_phases.sh

BASE_URL="http://localhost:8080"
GAME_ID="manual-test-game"

echo "🧪 Проверка последовательности фаз"
echo "============================================================"

# Проверяем доступность сервера
echo "1️⃣ Проверка доступности сервера..."
if curl -s "$BASE_URL/health" > /dev/null; then
    echo "✅ Сервер доступен"
else
    echo "❌ Сервер недоступен. Запустите сервер командой: go run cmd/server/main.go"
    exit 1
fi

# Создаем игру
echo -e "\n2️⃣ Создание игры..."
GAME_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games" \
    -H "Content-Type: application/json" \
    -d '{"name": "Manual Phase Test Game"}')

echo "Ответ создания игры: $GAME_RESPONSE"

# Начинаем ход
echo -e "\n3️⃣ Начало хода..."
START_RESPONSE=$(curl -s -X POST "$BASE_URL/api/phases/start-turn" \
    -H "Content-Type: application/json" \
    -d "{\"game_id\": \"$GAME_ID\"}")

echo "Ответ начала хода: $START_RESPONSE"

# Проверяем текущую фазу
echo -e "\n4️⃣ Проверка текущей фазы..."
CURRENT_RESPONSE=$(curl -s "$BASE_URL/api/phases/current?game_id=$GAME_ID")
echo "Текущая фаза: $CURRENT_RESPONSE"

# Последовательно переходим по фазам
echo -e "\n5️⃣ Последовательный переход по фазам..."

PHASES=("movement" "search" "air_attack" "naval_combat" "chance" "admin")

for i in "${!PHASES[@]}"; do
    PHASE_NUM=$((i + 1))
    EXPECTED_PHASE="${PHASES[i]}"
    
    echo -e "\n--- Фаза $PHASE_NUM/6: $EXPECTED_PHASE ---"
    
    # Переходим к следующей фазе
    NEXT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/phases/next" \
        -H "Content-Type: application/json" \
        -d "{\"game_id\": \"$GAME_ID\"}")
    
    echo "Ответ перехода: $NEXT_RESPONSE"
    
    # Проверяем текущую фазу
    sleep 0.1
    CURRENT_RESPONSE=$(curl -s "$BASE_URL/api/phases/current?game_id=$GAME_ID")
    echo "Текущая фаза: $CURRENT_RESPONSE"
    
    # Проверяем, что фаза корректна
    if echo "$CURRENT_RESPONSE" | grep -q "$EXPECTED_PHASE"; then
        echo "✅ Фаза корректна: $EXPECTED_PHASE"
    else
        echo "❌ Неожиданная фаза: ожидалось $EXPECTED_PHASE"
    fi
done

# Проверяем записи фаз
echo -e "\n6️⃣ Проверка записей фаз..."
RECORDS_RESPONSE=$(curl -s "$BASE_URL/api/phases/records?game_id=$GAME_ID")
echo "Записи фаз: $RECORDS_RESPONSE"

echo -e "\n🎉 Проверка завершена!"
