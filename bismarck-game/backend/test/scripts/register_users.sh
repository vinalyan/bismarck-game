#!/bin/bash

echo "Регистрация тестовых пользователей..."
echo "===================================="

# Функция для регистрации пользователя
register_user() {
    local username=$1
    local email=$2
    local password=$3
    
    response=$(curl -s -X POST http://localhost:8080/api/auth/register \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$username\",\"email\":\"$email\",\"password\":\"$password\"}")
    
    if echo "$response" | grep -q '"success":true'; then
        id=$(echo "$response" | jq -r '.data.id')
        echo "✅ $username ($email) - ID: $id"
    else
        echo "❌ $username: $(echo "$response" | jq -r '.error')"
    fi
}

# Регистрация пользователей
register_user "testuser10" "testuser10@example.com" "password123"
register_user "testuser8" "testuser8@example.com" "password123"
register_user "player1" "player1@test.com" "test123"
register_user "player2" "player2@test.com" "test123"
register_user "admin" "admin@test.com" "admin123"

echo ""
echo "===================================="
echo "Регистрация завершена!"
