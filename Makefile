.PHONY: test test-python test-go test-ts lint up down build clean

## Run all tests
test: test-python test-go test-ts

## Run Python tests
test-python:
	cd services/aggregator && pip install -q -r requirements.txt && pytest -v

## Run Go tests
test-go:
	cd services/collector && go test -v ./...

## Run TypeScript tests
test-ts:
	cd services/dashboard && npm install --silent && npm test

## Run all linters
lint:
	cd services/aggregator && flake8 --max-line-length=120 app.py tests/
	cd services/collector && go vet ./...
	cd services/dashboard && npm run lint

## Start all services with Docker Compose
up:
	docker compose up -d --build

## Stop all services
down:
	docker compose down

## Build all Docker images
build:
	docker compose build

## Clean build artifacts
clean:
	rm -rf services/dashboard/dist services/dashboard/node_modules
	rm -rf services/aggregator/__pycache__ services/aggregator/tests/__pycache__
	rm -f services/collector/collector
