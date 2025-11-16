#!/bin/bash

set -e  # Выход при первой ошибке

echo "🚀 Запуск всех тестов PR Reviewer Service"
echo "=========================================="

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функция для вывода с таймингом
run_test() {
    local test_name=$1
    local test_command=$2
    local test_type=$3
    
    echo -e "\n${BLUE}▶ Запуск $test_type: $test_name${NC}"
    echo "Команда: $test_command"
    
    local start_time=$(date +%s)
    
    if eval "$test_command"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${GREEN}✅ УСПЕХ: $test_name ($duration сек)${NC}"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${RED}❌ ОШИБКА: $test_name ($duration сек)${NC}"
        return 1
    fi
}

# Функция проверки доступности сервиса
wait_for_service() {
    local url=$1
    local timeout=$2
    local interval=5
    local total_time=0
    
    echo -n "Ожидание доступности $url "
    
    while [ $total_time -lt $timeout ]; do
        if curl -f -s "$url" > /dev/null 2>&1; then
            echo -e " ${GREEN}✓${NC}"
            return 0
        fi
        echo -n "."
        sleep $interval
        total_time=$((total_time + interval))
    done
    
    echo -e " ${RED}✗${NC}"
    return 1
}

# Проверяем что находимся в корне проекта
if [ ! -f "go.mod" ]; then
    echo -e "${RED}❌ Ошибка: Запускайте скрипт из корня проекта${NC}"
    echo "Текущая директория: $(pwd)"
    exit 1
fi

# Переменные
FAILED_TESTS=0
TOTAL_TESTS=0
TEST_ENV_STARTED=false

echo -e "\n${YELLOW}📋 Проверка окружения...${NC}"

# Проверяем установку зависимостей
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go не установлен${NC}"
    exit 1
fi

if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker не установлен${NC}"
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! command -v docker compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose не установлен${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Все зависимости установлены${NC}"

# Останавливаем предыдущие контейнеры если есть
echo -e "\n${YELLOW}🧹 Очистка предыдущих контейнеров...${NC}"
docker compose -f tests/docker-compose.e2e.yaml down > /dev/null 2>&1 || true

echo -e "\n${YELLOW}1. Запуск тестового окружения...${NC}"
if docker compose -f tests/docker-compose.e2e.yaml up -d --build; then
    TEST_ENV_STARTED=true
    echo -e "${GREEN}✓ Контейнеры запущены${NC}"
    
    # Ждем запуск сервисов
    echo -e "\n${YELLOW}⏳ Ожидание запуска сервисов...${NC}"
    
    if wait_for_service "http://localhost:8081/health" 60; then
        echo -e "${GREEN}✓ Сервис готов к тестированию${NC}"
    else
        echo -e "${RED}❌ Сервис не запустился за отведенное время${NC}"
        docker compose -f tests/docker-compose.e2e.yaml logs pr-reviewer-service_e2e
        exit 1
    fi
else
    echo -e "${RED}❌ Не удалось запустить тестовое окружение${NC}"
    exit 1
fi

# Функция для запуска тестов в директории
run_tests_in_dir() {
    local dir=$1
    local test_type=$2
    local tags=$3
    local env_vars=$4
    
    if [ -d "$dir" ]; then
        echo -e "\n${YELLOW}🔍 Поиск тестов в $dir...${NC}"
        
        # Ищем все _test.go файлы
        local test_files=$(find "$dir" -name "*_test.go" -type f)
        
        if [ -z "$test_files" ]; then
            echo -e "${YELLOW}⚠ Тесты не найдены в $dir${NC}"
            return 0
        fi
        
        for test_file in $test_files; do
            TOTAL_TESTS=$((TOTAL_TESTS + 1))
            local test_cmd="go test -v $test_file"
            
            # Добавляем теги если указаны
            if [ -n "$tags" ]; then
                test_cmd="$test_cmd -tags=$tags"
            fi
            
            # Добавляем переменные окружения если указаны
            if [ -n "$env_vars" ]; then
                test_cmd="$env_vars $test_cmd"
            fi
            
            if ! run_test "$(basename $test_file)" "$test_cmd" "$test_type"; then
                FAILED_TESTS=$((FAILED_TESTS + 1))
            fi
        done
    else
        echo -e "${YELLOW}⚠ Директория $dir не найдена${NC}"
    fi
}

# Запускаем тесты в правильном порядке

echo -e "\n${YELLOW}2. Запуск модульных тестов...${NC}"
run_tests_in_dir "tests/unit" "unit тест" ""

echo -e "\n${YELLOW}3. Запуск интеграционных тестов...${NC}"
run_tests_in_dir "tests/integration" "integration тест" "integration" "RUN_INTEGRATION_TESTS=1"

echo -e "\n${YELLOW}4. Запуск E2E тестов...${NC}"
run_tests_in_dir "tests/e2e" "E2E тест" "e2e" "API_URL=http://localhost:8081"

echo -e "\n${YELLOW}5. Запуск нагрузочных тестов...${NC}"

# Для load тестов используем специальную обработку
if [ -d "tests/load" ]; then
    echo -e "${BLUE}▶ Запуск нагрузочных тестов...${NC}"
    
    # Проверяем установлен ли vegeta
    if ! command -v vegeta &> /dev/null; then
        echo -e "${YELLOW}⚠ Установка Vegeta для нагрузочного тестирования...${NC}"
        go install github.com/tsenart/vegeta@latest
        export PATH=$PATH:$(go env GOPATH)/bin
    fi
    
    # Запускаем load тесты как обычные go тесты или через специальный скрипт
    if [ -f "tests/load/load_test_vegeta.go" ]; then
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        if run_test "load_test_vegeta.go" "go run tests/load/load_test_vegeta.go" "load тест"; then
            echo -e "${GREEN}✓ Нагрузочное тестирование завершено${NC}"
        else
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    else
        # Или запускаем как обычные тесты
        run_tests_in_dir "tests/load" "load тест" "load"
    fi
else
    echo -e "${YELLOW}⚠ Директория load тестов не найдена${NC}"
fi

echo -e "\n${YELLOW}6. Остановка тестового окружения...${NC}"
if [ "$TEST_ENV_STARTED" = true ]; then
    docker compose -f tests/docker-compose.e2e.yaml down
    echo -e "${GREEN}✓ Тестовое окружение остановлено${NC}"
fi

# Итоги
echo -e "\n${YELLOW}==========================================${NC}"
echo -e "${YELLOW}📊 ИТОГИ ТЕСТИРОВАНИЯ:${NC}"
echo -e "${YELLOW}==========================================${NC}"

SUCCESS_TESTS=$((TOTAL_TESTS - FAILED_TESTS))

if [ $TOTAL_TESTS -eq 0 ]; then
    echo -e "${YELLOW}⚠ Тесты не найдены${NC}"
    exit 0
fi

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}🎉 ВСЕ ТЕСТЫ ПРОЙДЕНЫ!${NC}"
    echo -e "${GREEN}✅ Успешных тестов: $SUCCESS_TESTS/$TOTAL_TESTS${NC}"
    echo -e "${GREEN}📈 Общий результат: 100%${NC}"
    exit 0
else
    echo -e "${RED}💥 НЕКОТОРЫЕ ТЕСТЫ ПРОВАЛИЛИСЬ${NC}"
    echo -e "${GREEN}✅ Успешных тестов: $SUCCESS_TESTS${NC}"
    echo -e "${RED}❌ Проваленных тестов: $FAILED_TESTS${NC}"
    echo -e "${RED}📉 Общий результат: $((SUCCESS_TESTS * 100 / TOTAL_TESTS))%${NC}"
    exit 1
fi