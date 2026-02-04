#!/bin/bash
# Проверка API сценариев (issue #76)
# Требует: запущенный сервер (make run), jq
#
# Примечание: сценарий применяется только когда в игре уже два игрока.
# Сейчас это происходит в CreateGame только если оба указаны при создании (редкий кейс).
# При обычном потоке (создание → join второго) юниты инициализируются в JoinGame
# через InitializeGameUnits; поддержка scenario_id в JoinGame может быть добавлена отдельно.

set -e
BASE_URL="${BASE_URL:-http://localhost:8080/api}"

echo "=== Проверка сценариев (issue #76) ==="
echo ""

# 1. Логин
USER1="${USER1:-player1}"
PASS1="${PASS1:-test123}"
TOKEN1=$(curl -s -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USER1\",\"password\":\"$PASS1\"}" | jq -r '.data.token // empty')

if [ -z "$TOKEN1" ]; then
  echo "Ошибка: не удалось получить токен."
  echo "Зарегистрируйте пользователей: ./test/scripts/register_users.sh"
  echo "Или задайте: USER1=... PASS1=... $0"
  exit 1
fi

echo "1. GET /api/games/scenarios — список сценариев:"
curl -s -X GET "${BASE_URL}/games/scenarios" \
  -H "Authorization: Bearer $TOKEN1" | jq '.'
echo ""

echo "2. POST /api/games с scenario_id — создание игры (должен вернуть 201 и id):"
CREATE_SC=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/games" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d '{"name":"Test scenario API","side":"german","scenario_id":"default","settings":{}}')
HTTP_BODY=$(echo "$CREATE_SC" | head -n -1)
HTTP_CODE=$(echo "$CREATE_SC" | tail -n 1)
GAME_ID=$(echo "$HTTP_BODY" | jq -r '.data.id // empty')
if [ "$HTTP_CODE" = "201" ] && [ -n "$GAME_ID" ]; then
  echo "   OK: игра создана, id=$GAME_ID"
else
  echo "   Код: $HTTP_CODE, ответ: $HTTP_BODY" | jq '.' 2>/dev/null || echo "$HTTP_BODY"
fi
echo ""

echo "3. POST /api/games с несуществующим scenario_id — ожидаем 400:"
BAD=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/games" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d '{"name":"Bad scenario","side":"german","scenario_id":"no-such-scenario","settings":{}}')
BAD_CODE=$(echo "$BAD" | tail -n 1)
if [ "$BAD_CODE" = "400" ]; then
  echo "   OK: получен 400 (Invalid scenario)"
else
  echo "   Ожидался 400, получен: $BAD_CODE"
fi
echo ""

echo "=== Проверка завершена ==="
echo ""
echo "Ручная проверка:"
echo "  - Список сценариев: curl -s -H 'Authorization: Bearer <TOKEN>' $BASE_URL/games/scenarios | jq"
echo "  - Создать игру со сценарием: curl -s -X POST $BASE_URL/games -H 'Authorization: Bearer <TOKEN>' -H 'Content-Type: application/json' -d '{\"name\":\"...\",\"side\":\"german\",\"scenario_id\":\"default\",\"settings\":{}}' | jq"
