#!/bin/bash

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 Тестирование всех фаз хода${NC}"
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

GAME_ID="29a0b39a-fb90-41a1-85ef-a3b0f204ff2e"

# Функция для получения текущего состояния игры
get_game_state() {
  curl -s "http://localhost:8080/api/games/$GAME_ID" \
    -H "Authorization: Bearer $TOKEN" | jq -r '.data | "\(.current_turn)|\(.current_phase)"'
}

# Функция для перехода к следующей фазе
next_phase() {
  curl -s -X POST http://localhost:8080/api/phases/next \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"game_id\":\"$GAME_ID\"}" > /dev/null
}

echo ""
echo "📊 Начальное состояние игры..."

IFS='|' read -r TURN PHASE <<< "$(get_game_state)"
echo "  Ход: $TURN"
echo "  Фаза: $PHASE"

echo ""
echo "🔄 Прохождение фаз..."
echo "--------------------"

PHASES=0
MAX_ITERATIONS=20  # Защита от бесконечного цикла
ITERATION=0

while [ $ITERATION -lt $MAX_ITERATIONS ]; do
  # Получаем текущее состояние
  IFS='|' read -r CURRENT_TURN CURRENT_PHASE <<< "$(get_game_state)"
  
  echo -e "${YELLOW}Итерация $((ITERATION + 1)): Ход $CURRENT_TURN, Фаза: $CURRENT_PHASE${NC}"
  
  # Переходим к следующей фазе
  next_phase
  sleep 0.5
  
  # Получаем новое состояние
  IFS='|' read -r NEW_TURN NEW_PHASE <<< "$(get_game_state)"
  
  # Проверяем, изменилась ли фаза
  if [ "$NEW_PHASE" = "$CURRENT_PHASE" ]; then
    echo -e "${GREEN}✅ Достигли конца цикла фаз или ход завершен${NC}"
    echo "  Ход: $NEW_TURN"
    echo "  Фаза: $NEW_PHASE"
    break
  fi
  
  PHASES=$((PHASES + 1))
  ITERATION=$((ITERATION + 1))
  
  echo "  → Переход к: $NEW_PHASE"
done

if [ $ITERATION -eq $MAX_ITERATIONS ]; then
  echo -e "${RED}⚠️  Достигнуто максимальное количество итераций${NC}"
fi

echo ""
echo -e "${GREEN}✅ Тест завершен. Пройдено фаз: $PHASES${NC}"
