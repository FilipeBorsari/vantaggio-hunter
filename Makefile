.PHONY: dev-up dev-down api web migrate migrate-down lint build

dev-up:
	docker compose -f infra/docker-compose.yml --env-file .env up -d

dev-down:
	docker compose -f infra/docker-compose.yml --env-file .env down

api:
	cd api && go run ./cmd/server

web:
	cd web && npm run dev

migrate:
	cd api && go run ./cmd/migrate up

migrate-down:
	cd api && go run ./cmd/migrate down

lint:
	cd api && golangci-lint run
	cd web && npm run lint

build:
	cd api && go build -o bin/server ./cmd/server
	cd web && npm run build
