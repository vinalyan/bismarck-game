#!/bin/bash

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🧪 Тестирование системы фаз${NC}"
echo "===================================="

# Получаем токен для testuser10
echo "🔑 Авторизация testuser10..."
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser10","password":"password123"}' | jq -r '.data.token')

if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
  echo -e "${RED}❌ Ошибка авторизации${NC}"
  exit 1
fi
echo -e "${GREEN}✅ Авторизация успешна${NC}"

# Получаем состояние игры
echo ""
echo "📊 Получение информации об игре..."
GAME_INFO=$(curl -s "http://localhost:8080/api/games/29a0b39a-fb90-41a1-85ef-a3b0f204ff2e" \
  -H "Authorization: Bearer $TOKEN")

CURRENT_TURN=$(echo $GAME_INFO | jq -r '.data.current_turn')
CURRENT_PHASE=$(echo $GAME_INFO | jq -r '.data.current_phase')
STATUS=$(echo $GAME_INFO | jq -r '.data.status')

echo "  Текущий ход: $CURRENT_TURN"
echo "  Текущая фаза: $CURRENT_PHASE"
echo "  Статус: $STATUS"

# Проверяем текущую фазу
echo ""
echo "🔄 Переход к следующей фазе..."
NEXT_PHASE=$(curl -s -X POST http://localhost:8080/api/phases/next \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"game_id":"29a0b39a-fb90-41a1-85ef-a3b0f204ff2e"}')

echo $NEXT_PHASE | jq '.'

# Получаем обновленную информацию об игре
echo ""
echo "📊 Обновленное состояние игры..."
GAME_INFO=$(curl -s "http://localhost:8080/api/games/29a0b39a-fb90-41a1-85ef-a3b0f204ff2e" \
  -H "Authorization: Bearer $TOKEN")

CURRENT_TURN=$(echo $GAME_INFO | jq -r '.data.current_turn')
CURRENT_PHASE=$(echo $GAME_INFO | jq -r '.data.current_phase')

echo "  Текущий ход: $CURRENT_TURN"
echo "  Текущая фаза: $CURRENT_PHASE"

echo ""
echo -e "${GREEN}✅ Тест завершен${NC}"
