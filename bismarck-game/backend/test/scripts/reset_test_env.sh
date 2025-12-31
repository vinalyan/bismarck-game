#!/bin/bash

# Скрипт для сброса и настройки тестовой среды
# Автоматизирует процесс перезапуска Docker контейнеров и создания тестовых пользователей

set -e  # Прекратить выполнение при первой ошибке

# Определяем директорию скрипта и переходим в backend директорию
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
SCRIPTS_DIR="$SCRIPT_DIR"  # Сохраняем путь к директории со скриптами
cd "$BACKEND_DIR"

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

# Временно отключаем set -e для этой секции, чтобы не прерывать скрипт при ошибках
set +e

# Проверяем, есть ли файл register_users.sh
REGISTER_SCRIPT="$SCRIPTS_DIR/register_users.sh"
if [ -f "$REGISTER_SCRIPT" ]; then
    # Проверяем права на выполнение
    if [ ! -x "$REGISTER_SCRIPT" ]; then
        echo "   Установка прав на выполнение для register_users.sh..."
        chmod +x "$REGISTER_SCRIPT"
    fi
    
    # Функция для проверки доступности сервера
    check_server() {
        curl -s -f http://localhost:8080/api/health > /dev/null 2>&1 || \
        curl -s -f http://localhost:8080/health > /dev/null 2>&1 || \
        curl -s -f http://localhost:8080/ > /dev/null 2>&1
    }
    
    # Проверяем, запущен ли сервер
    echo "   Проверка доступности сервера..."
    if check_server; then
        echo "   ✅ Сервер доступен, создаем пользователей..."
        "$REGISTER_SCRIPT"
        REGISTER_STATUS=$?
    else
        echo "   ⚠️  Сервер не доступен на http://localhost:8080"
        echo "   Запускаем сервер в фоновом режиме..."
        
        # Проверяем, запущен ли сервер уже
        SERVER_RUNNING=false
        if [ -f "server.pid" ]; then
            SERVER_PID=$(cat server.pid 2>/dev/null)
            if [ -n "$SERVER_PID" ] && ps -p "$SERVER_PID" > /dev/null 2>&1; then
                echo "   Сервер уже запущен (PID: $SERVER_PID)"
                SERVER_RUNNING=true
            fi
        fi
        
        if [ "$SERVER_RUNNING" = false ]; then
            # Проверяем наличие бинарника сервера
            if [ -f "./bin/server" ]; then
                SERVER_CMD="./bin/server"
            elif [ -f "./server" ]; then
                SERVER_CMD="./server"
            elif command -v go > /dev/null 2>&1; then
                echo "   Компилируем сервер..."
                go build -o bin/server ./cmd/server 2>&1 | grep -v "go: downloading" || true
                if [ -f "./bin/server" ]; then
                    SERVER_CMD="./bin/server"
                else
                    echo "   ⚠️  Не удалось скомпилировать сервер"
                    SERVER_CMD=""
                fi
            else
                SERVER_CMD=""
            fi
            
            if [ -n "$SERVER_CMD" ]; then
                # Запускаем сервер в фоне
                nohup $SERVER_CMD > server.log 2>&1 &
                SERVER_PID=$!
                echo $SERVER_PID > server.pid
                echo "   Сервер запущен (PID: $SERVER_PID)"
                
                # Ждем, пока сервер запустится (до 15 секунд)
                echo "   Ожидание запуска сервера..."
                for i in {1..15}; do
                    sleep 1
                    if check_server; then
                        echo "   ✅ Сервер запущен (через ${i} секунд), создаем пользователей..."
                        "$REGISTER_SCRIPT"
                        REGISTER_STATUS=$?
                        break
                    fi
                    if [ $i -eq 15 ]; then
                        echo "   ⚠️  Не удалось дождаться запуска сервера (таймаут 15 секунд)"
                        echo "   Создание пользователей пропущено"
                        echo "   Вы можете создать пользователей вручную позже:"
                        echo "   ./register_users.sh"
                        REGISTER_STATUS=1
                    fi
                done
            else
                echo "   ⚠️  Не найден исполняемый файл сервера"
                echo "   Создание пользователей пропущено"
                echo "   Запустите сервер вручную и выполните:"
                echo "   ./register_users.sh"
                REGISTER_STATUS=1
            fi
        else
            # Сервер уже запущен, но недоступен - возможно еще не готов
            echo "   Ожидание готовности сервера..."
            for i in {1..10}; do
                sleep 1
                if check_server; then
                    echo "   ✅ Сервер готов, создаем пользователей..."
                    "$REGISTER_SCRIPT"
                    REGISTER_STATUS=$?
                    break
                fi
            done
            if [ $i -eq 10 ] && ! check_server; then
                echo "   ⚠️  Сервер не отвечает"
                REGISTER_STATUS=1
            fi
        fi
    fi
    
    if [ ${REGISTER_STATUS:-0} -eq 0 ]; then
        echo "   ✅ Пользователи успешно созданы"
    fi
elif [ -f "create_test_users.go" ]; then
    echo "   Используется create_test_users.go..."
    go run create_test_users.go
    REGISTER_STATUS=$?
else
    echo "⚠️  Не найден файл register_users.sh или create_test_users.go"
    echo "   Создание пользователей пропущено"
    echo "   Вы можете создать пользователей вручную позже:"
    echo "   ./register_users.sh"
    REGISTER_STATUS=1
fi

# Возвращаем set -e
set -e

echo "✅ Тестовая среда готова!"
echo ""
echo "🎮 Теперь вы можете:"
echo "   • Войти на http://localhost:3000"
echo "   • Использовать player1/test123 (немцы)"
echo "   • Использовать player2/test123 (союзники)"
echo "   • Создать игру и протестировать систему фаз"
echo ""
