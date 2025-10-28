#!/bin/bash

# Скрипт для сброса и настройки тестовой среды
# Автоматизирует процесс перезапуска Docker контейнеров и создания тестовых пользователей

set -e  # Прекратить выполнение при первой ошибке

echo "🔄 Сброс тестовой среды..."

# Удаляем контейнеры и volume
echo "📦 Удаление Docker контейнеров и volumes..."
docker-compose down -v

# Перезапускаем
echo "🚀 Запуск Docker контейнеров..."
docker-compose up -d

# Ждем, пока контейнеры полностью запустятся
echo "⏳ Ожидание запуска контейнеров (10 секунд)..."
sleep 10

# Запускаем миграции
echo "🗄️  Запуск миграций базы данных..."
make migrate

# Создаем тестовых пользователей
echo "👥 Создание тестовых пользователей..."

# Проверяем, есть ли файл create_test_users.go
if [ -f "create_test_users.go" ]; then
    echo "   Используется create_test_users.go..."
    go run create_test_users.go
elif [ -f "register_users.sh" ]; then
    echo "   Используется register_users.sh..."
    bash register_users.sh
else
    echo "⚠️  Не найден ни create_test_users.go, ни register_users.sh"
    echo "   Создание пользователей пропущено"
fi

echo "✅ Тестовая среда готова!"
echo ""
echo "🎮 Теперь вы можете:"
echo "   • Войти на http://localhost:3000"
echo "   • Использовать testuser10/password123 (немцы)"
echo "   • Использовать testuser8/password123 (союзники)"
echo "   • Создать игру и протестировать систему фаз"
echo ""
