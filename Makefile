.PHONY: dev-up dev-down api web migrate migrate-down lint build test

dev-up:
	docker compose -f infra/docker-compose.yml --env-file .env up -d

dev-down:
	docker compose -f infra/docker-compose.yml --env-file .env down

api:
	cd api && env $(grep -v '^#' ../.env | grep '=' | xargs) go run ./cmd/server

web:
	cd web && npm run dev

migrate:
	cd api && env $(grep -v '^#' ../.env | grep '=' | xargs) go run ./cmd/migrate up

migrate-down:
	cd api && env $(grep -v '^#' ../.env | grep '=' | xargs) go run ./cmd/migrate down

test:
	cd api && /home/filipeborsari/go/bin/gotestsum --format testdox

lint:
	cd api && /home/filipeborsari/go/bin/go vet ./...
	cd web && npm run lint

build:
	cd api && go build -o bin/server ./cmd/server
	cd web && npm run build
