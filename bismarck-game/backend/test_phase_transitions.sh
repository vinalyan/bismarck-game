#!/bin/bash

# Тестирование переходов между фазами в первом ходе
BASE_URL="http://localhost:8080"
GAME_ID="857f3312-937a-447f-a4a8-8edb19fb223f"
TOKEN1="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjEyNjE2NjYsImlhdCI6MTc2MTE3NTI2NiwibmJmIjoxNzYxMTc1MjY2LCJ1c2VyX2lkIjoiZTMzODRkZjctN2VhOC00MGVlLTk3YmEtYzYxNjkwOGQ1OWRkIiwidXNlcm5hbWUiOiJ0ZXN0dXNlcjEifQ.fP1iHllm_uwe-Q8sL5BaNxeT3kbDXs9wgMpa-jcsZho"

echo "=== Тестирование переходов между фазами ==="

# Проверяем текущую фазу
echo "Текущая фаза:"
curl -s -X GET "$BASE_URL/api/phases/current?game_id=$GAME_ID" | jq '.'

echo -e "\nПереходим к следующей фазе..."
NEXT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/phases/next" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d "{\"game_id\":\"$GAME_ID\"}")

echo "Ответ: $NEXT_RESPONSE"

sleep 2

echo -e "\nНовая фаза:"
curl -s -X GET "$BASE_URL/api/phases/current?game_id=$GAME_ID" | jq '.'

echo -e "\nПереходим к следующей фазе..."
NEXT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/phases/next" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d "{\"game_id\":\"$GAME_ID\"}")

echo "Ответ: $NEXT_RESPONSE"

sleep 2

echo -e "\nНовая фаза:"
curl -s -X GET "$BASE_URL/api/phases/current?game_id=$GAME_ID" | jq '.'

echo -e "\nПереходим к следующей фазе..."
NEXT_RESPONSE=$(curl -s -X POST "$BASE_URL/api/phases/next" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d "{\"game_id\":\"$GAME_ID\"}")

echo "Ответ: $NEXT_RESPONSE"

sleep 2

echo -e "\nНовая фаза:"
curl -s -X GET "$BASE_URL/api/phases/current?game_id=$GAME_ID" | jq '.'

echo "=== Тест завершен ==="
