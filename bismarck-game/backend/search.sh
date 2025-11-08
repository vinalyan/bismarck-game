#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080/api}"
GAME_NAME="${GAME_NAME:-Тестовая игра Bismarck}"
PLAYER1_USER="${PLAYER1_USER:-player1}"
PLAYER1_PASS="${PLAYER1_PASS:-test123}"
PLAYER2_USER="${PLAYER2_USER:-player2}"
PLAYER2_PASS="${PLAYER2_PASS:-test123}"

login() {
  local user=$1 pass=$2
  curl -sS -X POST "$BASE_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$user\",\"password\":\"$pass\"}" |
    jq -e '.data'
}

create_game() {
  local token=$1
  curl -sS -X POST "$BASE_URL/games" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $token" \
    -d "{\"name\":\"$GAME_NAME\",\"side\":\"german\"}" |
    jq -er '.data.id'
}

join_game_allied() {
  local token=$1 game_id=$2
  curl -sS -X POST "$BASE_URL/games/$game_id/join" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $token" \
    -d '{"side":"allied"}' > /dev/null
}

start_turn() {
  local token=$1 game_id=$2
  curl -sS -X POST "$BASE_URL/phases/turn/start" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $token" \
    -d "{\"game_id\":\"$game_id\"}" > /dev/null
}

add_hex_marker() {
  local token=$1 game_id=$2 hex=$3
  curl -sS -X POST "$BASE_URL/games/$game_id/hex-markers" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $token" \
    -d "{\"hex_id\":\"$hex\",\"marker_type\":\"flight_path_search\"}" > /dev/null
}

add_air_marker() {
  local token=$1 game_id=$2 hex=$3
  curl -sS -X POST "$BASE_URL/games/$game_id/flight-path-search/markers" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $token" \
    -d "{\"hex_id\":\"$hex\"}" > /dev/null
}

advance_phase() {
  local game_id=$1
  curl -sS -X POST "$BASE_URL/phases/next" \
    -H "Content-Type: application/json" \
    -d "{\"game_id\":\"$game_id\"}" > /dev/null
}

get_units() {
  local token=$1 game_id=$2
  curl -sS -X GET "$BASE_URL/games/$game_id/units" \
    -H "Authorization: Bearer $token"
}

main() {
  command -v jq >/dev/null || { echo "jq не найден"; exit 1; }

  echo "🔐 Логин..."
  P1_LOGIN_JSON=$(login "$PLAYER1_USER" "$PLAYER1_PASS")
  P2_LOGIN_JSON=$(login "$PLAYER2_USER" "$PLAYER2_PASS")

  P1_TOKEN=$(echo "$P1_LOGIN_JSON" | jq -r '.token')
  P2_TOKEN=$(echo "$P2_LOGIN_JSON" | jq -r '.token')
  P1_ID=$(echo "$P1_LOGIN_JSON" | jq -r '.user.id')
  P2_ID=$(echo "$P2_LOGIN_JSON" | jq -r '.user.id')

  echo "🎮 Создание игры..."
  GAME_ID=$(create_game "$P1_TOKEN")
  echo "    Game ID: $GAME_ID"

  echo "🤝 Присоединение союзника..."
  join_game_allied "$P2_TOKEN" "$GAME_ID"

  echo "▶️ Старт хода..."
  start_turn "$P1_TOKEN" "$GAME_ID"

  echo "📍 Добавление маркеров поиска A9/A10..."
  add_hex_marker "$P1_TOKEN" "$GAME_ID" "A9"
  add_hex_marker "$P1_TOKEN" "$GAME_ID" "A9"
  add_hex_marker "$P1_TOKEN" "$GAME_ID" "A10"
  add_hex_marker "$P1_TOKEN" "$GAME_ID" "A10"

  echo "✈️  Добавление маркеров воздушной разведки J30 (союзники)..."
  add_air_marker "$P2_TOKEN" "$GAME_ID" "J30"
  add_air_marker "$P2_TOKEN" "$GAME_ID" "J30"

  echo "⏭️  Завершение фазы движения..."
  advance_phase "$GAME_ID"

  echo "⏳ Ожидание обработки фаз..."
  sleep 3

  echo "📊 Получение статуса юнитов..."
  UNITS_GERMAN_JSON=$(get_units "$P1_TOKEN" "$GAME_ID")
  UNITS_ALLIED_JSON=$(get_units "$P2_TOKEN" "$GAME_ID")

  ALLIED_SHADOWED=$(echo "$UNITS_ALLIED_JSON" | jq '[.data.units[] | select(.detection_level == "shadowed")] | length')
  ALLIED_TOTAL=$(echo "$UNITS_ALLIED_JSON" | jq '.data.units | length')
  TF_STATUS=$(echo "$UNITS_GERMAN_JSON" | jq -r '.data.task_forces[] | .detection_level' | head -n 1)

  echo "✅ Обнаружено союзников (shadowed): $ALLIED_SHADOWED / $ALLIED_TOTAL"
  echo "✅ Состояние немецкой TF: ${TF_STATUS:-none}"

  echo "🔁 Game URL: http://localhost:3000/game/$GAME_ID"
}

main "$@"