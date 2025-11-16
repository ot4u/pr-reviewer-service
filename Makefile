run:
	docker compose up --build -d

install: install_tools
	go mod tidy

docker_up:
	docker compose up -d --build

docker_stop:
	docker compose stop

docker_down:
	docker compose down

test:
	chmod +x scripts/run_tests.sh
	./scripts/run_tests.sh

test-unit:
	go test -v ./tests/unit/... -cover
  
test-integration:
	RUN_INTEGRATION_TESTS=1 go test -v ./tests/integration/... -tags=integration

test-e2e:
	docker compose -f tests/docker-compose.e2e.yaml up -d --build
	sleep 20
	API_URL=http://localhost:8081 go test -v ./tests/e2e/... -tags=e2e
	docker compose -f tests/docker-compose.e2e.yaml down

test-load:
	docker compose -f tests/docker-compose.e2e.yaml up -d --build
	sleep 20
	go run tests/load/load_test_testing.go
	docker compose -f tests/docker-compose.e2e.yaml down

test-env:
	docker compose -f tests/docker-compose.e2e.yaml up -d --build

test-clean:
	docker compose -f tests/docker-compose.e2e.yaml down

format:
	go fmt ./...

gen_sqlc:
	sqlc generate

clean:
	rm -rf ./bin

install_tools:
	go install github.com/tsenart/vegeta@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/vektra/mockery/v2@latest

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

gen-mocks:
	mockery --all --dir ./internal/domain --output ./tests/mocks --case underscore