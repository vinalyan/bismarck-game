#!/bin/bash

# Тестирование смены фаз
BASE_URL="http://localhost:8080"

echo "=== Тестирование смены фаз ==="

# 1. Регистрация пользователей
echo "1. Регистрация пользователей..."
USER1_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser1","password":"password123","email":"test1@example.com"}')

USER2_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser2","password":"password123","email":"test2@example.com"}')

echo "User1 response: $USER1_RESPONSE"
echo "User2 response: $USER2_RESPONSE"

# 2. Логин пользователей
echo "2. Логин пользователей..."
LOGIN1_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser1","password":"password123"}')

LOGIN2_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser2","password":"password123"}')

echo "Login1 response: $LOGIN1_RESPONSE"
echo "Login2 response: $LOGIN2_RESPONSE"

# Извлекаем токены
TOKEN1=$(echo $LOGIN1_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
TOKEN2=$(echo $LOGIN2_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

echo "Token1: $TOKEN1"
echo "Token2: $TOKEN2"

# 3. Создание игры
echo "3. Создание игры..."
GAME_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d '{"name":"Phase Test Game","side":"german"}')

echo "Game response: $GAME_RESPONSE"

# Извлекаем ID игры
GAME_ID=$(echo $GAME_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "Game ID: $GAME_ID"

# 4. Присоединение второго игрока
echo "4. Присоединение второго игрока..."
JOIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/games/$GAME_ID/join" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN2" \
  -d '{"side":"allied"}')

echo "Join response: $JOIN_RESPONSE"

# 5. Проверка текущей фазы
echo "5. Проверка текущей фазы..."
PHASE_RESPONSE=$(curl -s -X GET "$BASE_URL/api/phases/current?game_id=$GAME_ID")
echo "Current phase: $PHASE_RESPONSE"

# 6. Тестирование переходов между фазами
echo "6. Тестирование переходов между фазами..."

# Переход к следующей фазе
NEXT_PHASE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/phases/next" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN1" \
  -d "{\"game_id\":\"$GAME_ID\"}")

echo "Next phase response: $NEXT_PHASE_RESPONSE"

# Проверяем новую фазу
sleep 2
NEW_PHASE_RESPONSE=$(curl -s -X GET "$BASE_URL/api/phases/current?game_id=$GAME_ID")
echo "New phase: $NEW_PHASE_RESPONSE"

echo "=== Тест завершен ==="
