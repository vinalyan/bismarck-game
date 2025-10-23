#!/bin/bash

# Главный файл для запуска всех тестов движения
BASE_URL="http://localhost:8080"

echo "🚀 ЗАПУСК ВСЕХ ТЕСТОВ ДВИЖЕНИЯ КОРАБЛЕЙ"
echo "======================================"
echo ""

# Проверка доступности сервера
echo "🔍 Проверка доступности сервера..."
if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
    echo "✅ Сервер доступен"
else
    echo "❌ Сервер недоступен. Убедитесь, что сервер запущен на $BASE_URL"
    exit 1
fi

echo ""
echo "📋 Доступные тесты движения:"
echo "1. test_movement.sh - Основные тесты движения"
echo "2. test_movement_scenarios.sh - Детальные сценарии движения"
echo "3. test_movement_integration.sh - Интеграционные тесты"
echo ""

# Функция для запуска теста с обработкой ошибок
run_test() {
    local test_name="$1"
    local test_file="$2"
    
    echo "🔄 Запуск: $test_name"
    echo "----------------------------------------"
    
    if [ -f "$test_file" ]; then
        if bash "$test_file"; then
            echo "✅ $test_name - УСПЕШНО"
        else
            echo "❌ $test_name - ОШИБКА"
            return 1
        fi
    else
        echo "❌ Файл $test_file не найден"
        return 1
    fi
    
    echo ""
    return 0
}

# Функция для выбора тестов
select_tests() {
    echo "Выберите тесты для запуска:"
    echo "1. Все тесты (рекомендуется)"
    echo "2. Только основные тесты движения"
    echo "3. Только детальные сценарии"
    echo "4. Только интеграционные тесты"
    echo "5. Пользовательский выбор"
    echo ""
    read -p "Введите номер (1-5): " choice
    
    case $choice in
        1)
            echo "🎯 Запуск всех тестов..."
            run_test "Основные тесты движения" "test_movement.sh"
            run_test "Детальные сценарии движения" "test_movement_scenarios.sh"
            run_test "Интеграционные тесты" "test_movement_integration.sh"
            ;;
        2)
            echo "🎯 Запуск основных тестов движения..."
            run_test "Основные тесты движения" "test_movement.sh"
            ;;
        3)
            echo "🎯 Запуск детальных сценариев движения..."
            run_test "Детальные сценарии движения" "test_movement_scenarios.sh"
            ;;
        4)
            echo "🎯 Запуск интеграционных тестов..."
            run_test "Интеграционные тесты" "test_movement_integration.sh"
            ;;
        5)
            echo "🎯 Пользовательский выбор..."
            echo ""
            echo "Выберите тесты для запуска (введите номера через пробел):"
            echo "1. Основные тесты движения"
            echo "2. Детальные сценарии движения"
            echo "3. Интеграционные тесты"
            echo ""
            read -p "Введите номера: " tests
            
            for test_num in $tests; do
                case $test_num in
                    1)
                        run_test "Основные тесты движения" "test_movement.sh"
                        ;;
                    2)
                        run_test "Детальные сценарии движения" "test_movement_scenarios.sh"
                        ;;
                    3)
                        run_test "Интеграционные тесты" "test_movement_integration.sh"
                        ;;
                    *)
                        echo "❌ Неизвестный номер теста: $test_num"
                        ;;
                esac
            done
            ;;
        *)
            echo "❌ Неверный выбор. Запуск всех тестов по умолчанию..."
            run_test "Основные тесты движения" "test_movement.sh"
            run_test "Детальные сценарии движения" "test_movement_scenarios.sh"
            run_test "Интеграционные тесты" "test_movement_integration.sh"
            ;;
    esac
}

# Функция для автоматического запуска всех тестов
run_all_tests() {
    echo "🎯 Автоматический запуск всех тестов..."
    echo ""
    
    local success_count=0
    local total_count=0
    
    # Тест 1: Основные тесты движения
    total_count=$((total_count + 1))
    if run_test "Основные тесты движения" "test_movement.sh"; then
        success_count=$((success_count + 1))
    fi
    
    # Тест 2: Детальные сценарии движения
    total_count=$((total_count + 1))
    if run_test "Детальные сценарии движения" "test_movement_scenarios.sh"; then
        success_count=$((success_count + 1))
    fi
    
    # Тест 3: Интеграционные тесты
    total_count=$((total_count + 1))
    if run_test "Интеграционные тесты" "test_movement_integration.sh"; then
        success_count=$((success_count + 1))
    fi
    
    # Итоговая статистика
    echo "📊 ИТОГОВАЯ СТАТИСТИКА"
    echo "====================="
    echo "✅ Успешных тестов: $success_count"
    echo "❌ Неудачных тестов: $((total_count - success_count))"
    echo "📈 Общий процент успеха: $(( (success_count * 100) / total_count ))%"
    
    if [ $success_count -eq $total_count ]; then
        echo ""
        echo "🎉 ВСЕ ТЕСТЫ ПРОШЛИ УСПЕШНО!"
        echo "🚀 Система движения кораблей работает корректно!"
    else
        echo ""
        echo "⚠️  НЕКОТОРЫЕ ТЕСТЫ НЕ ПРОШЛИ"
        echo "🔧 Рекомендуется проверить логи и исправить ошибки"
    fi
}

# Функция для показа справки
show_help() {
    echo "📖 СПРАВКА ПО ТЕСТАМ ДВИЖЕНИЯ"
    echo "============================="
    echo ""
    echo "Этот скрипт запускает комплексные тесты системы движения кораблей."
    echo ""
    echo "Доступные тесты:"
    echo "1. Основные тесты движения (test_movement.sh)"
    echo "   - Тестирование F, M, S, VS типов кораблей"
    echo "   - Проверка ограничений топлива"
    echo "   - Тестирование доступных гексов"
    echo "   - Проверка ошибок движения"
    echo ""
    echo "2. Детальные сценарии движения (test_movement_scenarios.sh)"
    echo "   - 13+ детальных сценариев движения"
    echo "   - Тестирование всех типов кораблей"
    echo "   - Проверка доступных гексов"
    echo "   - Тестирование ошибок"
    echo ""
    echo "3. Интеграционные тесты (test_movement_integration.sh)"
    echo "   - Смешанные типы кораблей"
    echo "   - Переходы между ходами"
    echo "   - Фаза администрирования"
    echo "   - Синхронизация фронтенд-бэкенд"
    echo "   - Тестирование производительности"
    echo ""
    echo "Использование:"
    echo "  ./run_all_movement_tests.sh          # Интерактивный режим"
    echo "  ./run_all_movement_tests.sh --all    # Запуск всех тестов"
    echo "  ./run_all_movement_tests.sh --help   # Показать справку"
    echo ""
    echo "Требования:"
    echo "  - Сервер должен быть запущен на $BASE_URL"
    echo "  - Пользователи testuser1 и testuser2 должны существовать"
    echo "  - Все тестовые файлы должны быть исполняемыми"
    echo ""
}

# Функция для проверки зависимостей
check_dependencies() {
    echo "🔍 Проверка зависимостей..."
    
    local missing_deps=()
    
    # Проверка curl
    if ! command -v curl &> /dev/null; then
        missing_deps+=("curl")
    fi
    
    # Проверка jq
    if ! command -v jq &> /dev/null; then
        missing_deps+=("jq")
    fi
    
    # Проверка bash
    if ! command -v bash &> /dev/null; then
        missing_deps+=("bash")
    fi
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        echo "❌ Отсутствуют зависимости: ${missing_deps[*]}"
        echo "Установите их перед запуском тестов"
        exit 1
    fi
    
    echo "✅ Все зависимости установлены"
}

# Функция для проверки тестовых файлов
check_test_files() {
    echo "🔍 Проверка тестовых файлов..."
    
    local missing_files=()
    
    if [ ! -f "test_movement.sh" ]; then
        missing_files+=("test_movement.sh")
    fi
    
    if [ ! -f "test_movement_scenarios.sh" ]; then
        missing_files+=("test_movement_scenarios.sh")
    fi
    
    if [ ! -f "test_movement_integration.sh" ]; then
        missing_files+=("test_movement_integration.sh")
    fi
    
    if [ ${#missing_files[@]} -gt 0 ]; then
        echo "❌ Отсутствуют тестовые файлы: ${missing_files[*]}"
        echo "Убедитесь, что все файлы находятся в текущей директории"
        exit 1
    fi
    
    echo "✅ Все тестовые файлы найдены"
}

# Основная функция
main() {
    echo "🚀 СИСТЕМА ТЕСТИРОВАНИЯ ДВИЖЕНИЯ КОРАБЛЕЙ"
    echo "======================================="
    echo ""
    
    # Проверка аргументов командной строки
    case "${1:-}" in
        --help|-h)
            show_help
            exit 0
            ;;
        --all|-a)
            check_dependencies
            check_test_files
            run_all_tests
            exit 0
            ;;
        "")
            check_dependencies
            check_test_files
            select_tests
            ;;
        *)
            echo "❌ Неизвестный аргумент: $1"
            echo "Используйте --help для получения справки"
            exit 1
            ;;
    esac
}

# Запуск основной функции
main "$@"
