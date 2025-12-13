#!/bin/bash

BASE_URL="http://localhost:8080/api"

echo "🧪 ТЕСТИРОВАНИЕ ОБНОВЛЕНИЯ ФАЗЫ В GAMEMODEL"
echo "=========================================="

# Получаем токен для player1
echo "🔐 Получение токена для player1..."
TOKEN1=$(curl -s -X POST "$BASE_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"player1","password":"test123"}' | jq -r '.data.token')

if [ "$TOKEN1" = "null" ] || [ -z "$TOKEN1" ]; then
    echo "❌ Ошибка получения токена для player1"
    exit 1
fi

echo "✅ Токен получен: ${TOKEN1:0:30}..."

# Получаем список игр
echo -e "\n🎮 Поиск активной игры..."
GAMES=$(curl -s -X GET "$BASE_URL/games" \
    -H "Authorization: Bearer $TOKEN1" | jq -r '.data[] | select(.status == "active") | .id' | head -1)

if [ -z "$GAMES" ]; then
    echo "❌ Нет активных игр. Создайте игру сначала."
    exit 1
fi

GAME_ID=$(echo "$GAMES" | head -1)
echo "✅ Найдена игра: $GAME_ID"

# Проверяем текущую фазу ДО вызова nextPhase
echo -e "\n📊 Текущая фаза ДО вызова nextPhase:"
BEFORE_RESPONSE=$(curl -s -X GET "http://localhost:8080/api/games/$GAME_ID/model" \
    -H "Authorization: Bearer $TOKEN1")
BEFORE_TURN=$(echo "$BEFORE_RESPONSE" | jq -r '.data.current_turn.turn // "null"')
BEFORE_PHASE=$(echo "$BEFORE_RESPONSE" | jq -r '.data.current_turn.phase // "null"')
echo "   Turn: $BEFORE_TURN, Phase: $BEFORE_PHASE"

# Вызываем nextPhase
echo -e "\n🔄 Вызов /api/phases/next..."
NEXT_RESPONSE=$(curl -s -X POST "$BASE_URL/phases/next" \
    -H "Authorization: Bearer $TOKEN1" \
    -H "Content-Type: application/json" \
    -d "{\"game_id\":\"$GAME_ID\"}")

echo "Ответ nextPhase:"
echo "$NEXT_RESPONSE" | jq '.'

# Ждем немного для обработки
sleep 1

# Проверяем текущую фазу ПОСЛЕ вызова nextPhase
echo -e "\n📊 Текущая фаза ПОСЛЕ вызова nextPhase:"
AFTER_RESPONSE=$(curl -s -X GET "http://localhost:8080/api/games/$GAME_ID/model" \
    -H "Authorization: Bearer $TOKEN1")
AFTER_TURN=$(echo "$AFTER_RESPONSE" | jq -r '.data.current_turn.turn // "null"')
AFTER_PHASE=$(echo "$AFTER_RESPONSE" | jq -r '.data.current_turn.phase // "null"')
echo "   Turn: $AFTER_TURN, Phase: $AFTER_PHASE"

# Проверяем, изменилась ли фаза
if [ "$BEFORE_PHASE" != "$AFTER_PHASE" ] || [ "$BEFORE_TURN" != "$AFTER_TURN" ]; then
    echo -e "\n✅ ФАЗА ОБНОВИЛАСЬ!"
    echo "   Было: Turn=$BEFORE_TURN, Phase=$BEFORE_PHASE"
    echo "   Стало: Turn=$AFTER_TURN, Phase=$AFTER_PHASE"
else
    echo -e "\n❌ ФАЗА НЕ ОБНОВИЛАСЬ!"
    echo "   Turn: $BEFORE_TURN -> $AFTER_TURN"
    echo "   Phase: $BEFORE_PHASE -> $AFTER_PHASE"
fi

# Проверяем логи сервера
echo -e "\n📋 Проверка логов сервера..."
if [ -f "server.log" ]; then
    echo "Последние сообщения о GameModel:"
    tail -50 server.log | grep -E "GameModel|invalidation|Current turn loaded|NextPhase" | tail -10
else
    echo "⚠️  Файл server.log не найден"
fi

echo -e "\n✅ Тест завершен"

