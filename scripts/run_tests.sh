#!/bin/bash

set -e

echo "Запуск всех тестов PR Reviewer Service"
echo "========================================"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

FAILED_TESTS=0
TOTAL_TESTS=0
TEST_ENV_STARTED=false

run_test() {
    local test_name=$1
    local test_command=$2
    local test_type=$3
    
    echo -e "\n${BLUE}Запуск $test_type: $test_name${NC}"
    
    local start_time=$(date +%s)
    
    if eval "$test_command"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${GREEN}УСПЕХ: $test_name ($duration сек)${NC}"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        echo -e "${RED}ОШИБКА: $test_name ($duration сек)${NC}"
        return 1
    fi
}

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

if [ ! -f "go.mod" ]; then
    echo -e "${RED}Ошибка: Запускайте скрипт из корня проекта${NC}"
    exit 1
fi

echo -e "\n${YELLOW}1. Запуск тестового окружения...${NC}"
if docker compose -f tests/docker-compose.e2e.yaml up -d --build; then
    TEST_ENV_STARTED=true
    echo -e "${GREEN}Контейнеры запущены${NC}"
    
    echo -e "\n${YELLOW}Ожидание запуска сервисов...${NC}"
    if wait_for_service "http://localhost:8081/health" 60; then
        echo -e "${GREEN}Сервис готов к тестированию${NC}"
    else
        echo -e "${RED}Сервис не запустился за отведенное время${NC}"
        docker compose -f tests/docker-compose.e2e.yaml logs pr-reviewer-service_e2e
        exit 1
    fi
else
    echo -e "${RED}Не удалось запустить тестовое окружение${NC}"
    exit 1
fi

run_tests_in_dir() {
    local dir=$1
    local test_type=$2
    local tags=$3
    local env_vars=$4
    
    if [ -d "$dir" ]; then
        echo -e "\n${YELLOW}Поиск тестов в $dir...${NC}"
        
        local test_files=$(find "$dir" -name "*_test.go" -type f)
        
        if [ -z "$test_files" ]; then
            echo -e "${YELLOW}Тесты не найдены в $dir${NC}"
            return 0
        fi
        
        for test_file in $test_files; do
            TOTAL_TESTS=$((TOTAL_TESTS + 1))
            local test_cmd="go test -v $test_file"
            
            if [ -n "$tags" ]; then
                test_cmd="$test_cmd -tags=$tags"
            fi
            
            if [ -n "$env_vars" ]; then
                test_cmd="$env_vars $test_cmd"
            fi
            
            if ! run_test "$(basename $test_file)" "$test_cmd" "$test_type"; then
                FAILED_TESTS=$((FAILED_TESTS + 1))
            fi
        done
    else
        echo -e "${YELLOW}Директория $dir не найдена${NC}"
    fi
}

echo -e "\n${YELLOW}2. Запуск модульных тестов...${NC}"
run_tests_in_dir "tests/unit" "unit тест" ""

echo -e "\n${YELLOW}3. Запуск интеграционных тестов...${NC}"
run_tests_in_dir "tests/integration" "integration тест" "integration" "RUN_INTEGRATION_TESTS=1"

echo -e "\n${YELLOW}4. Запуск E2E тестов...${NC}"
run_tests_in_dir "tests/e2e" "E2E тест" "e2e" "API_URL=http://localhost:8081"

echo -e "\n${YELLOW}5. Запуск нагрузочных тестов...${NC}"

if [ -f "tests/load/load_testing.go" ]; then
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if ! command -v vegeta &> /dev/null; then
        echo -e "${YELLOW}Установка Vegeta...${NC}"
        go install github.com/tsenart/vegeta@latest
        export PATH=$PATH:$(go env GOPATH)/bin
    fi

    echo -e "${YELLOW}Настройка нагрузочного теста...${NC}"
    echo -e "${YELLOW}Длительность: 2 минуты, RPS: 5${NC}"
    
    if ! run_test "load_testing.go" "go run tests/load/load_testing.go" "load тест"; then
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
else
    echo -e "${YELLOW}Файл load_testing.go не найден${NC}"
    echo -e "${YELLOW}Ищем другие файлы для нагрузочного тестирования...${NC}"
    
    # Ищем любые .go файлы в load директории кроме _test.go
    local load_files=$(find tests/load -name "*.go" -type f ! -name "*_test.go")
    
    if [ -n "$load_files" ]; then
        for load_file in $load_files; do
            TOTAL_TESTS=$((TOTAL_TESTS + 1))
            if ! run_test "$(basename $load_file)" "go run $load_file" "load тест"; then
                FAILED_TESTS=$((FAILED_TESTS + 1))
            fi
        done
    else
        echo -e "${YELLOW}Файлы для нагрузочного тестирования не найдены${NC}"
    fi
fi

echo -e "\n${YELLOW}6. Остановка тестового окружения...${NC}"
if [ "$TEST_ENV_STARTED" = true ]; then
    docker compose -f tests/docker-compose.e2e.yaml down
    echo -e "${GREEN}Тестовое окружение остановлено${NC}"
fi

echo -e "\n${YELLOW}==========================================${NC}"
echo -e "${YELLOW}ИТОГИ ТЕСТИРОВАНИЯ:${NC}"

SUCCESS_TESTS=$((TOTAL_TESTS - FAILED_TESTS))

if [ $TOTAL_TESTS -eq 0 ]; then
    echo -e "${YELLOW}Тесты не найдены${NC}"
    exit 0
fi

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}ВСЕ ТЕСТЫ ПРОЙДЕНЫ!${NC}"
    echo -e "${GREEN}Успешных тестов: $SUCCESS_TESTS/$TOTAL_TESTS${NC}"
    echo -e "${GREEN}Общий результат: 100%${NC}"
    exit 0
else
    echo -e "${RED}НЕКОТОРЫЕ ТЕСТЫ ПРОВАЛИЛИСЬ${NC}"
    echo -e "${GREEN}Успешных тестов: $SUCCESS_TESTS${NC}"
    echo -e "${RED}Проваленных тестов: $FAILED_TESTS${NC}"
    echo -e "${RED}Общий результат: $((SUCCESS_TESTS * 100 / TOTAL_TESTS))%${NC}"
    exit 1
fi