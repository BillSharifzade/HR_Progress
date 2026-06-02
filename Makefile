.PHONY: up down logs build backend-build backend-test frontend-dev backend-dev psql

ENV_FILE ?= deploy/.env
COMPOSE := docker compose -f deploy/docker-compose.yml --env-file $(ENV_FILE)

up:
	@test -f $(ENV_FILE) || (echo "Missing $(ENV_FILE). Copy deploy/.env.example to deploy/.env and set JWT_SECRET." && exit 1)
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f --tail=200

build:
	$(COMPOSE) build

backend-build:
	cd backend && go build ./...

backend-test:
	cd backend && go test ./...

backend-dev:
	cd backend && \
	  DATABASE_URL=postgres://hrprogress:hrprogress@localhost:5432/hrprogress?sslmode=disable \
	  JWT_SECRET=$$(openssl rand -hex 32) \
	  MIGRATIONS_URL=file://./migrations \
	  go run ./cmd/server

frontend-dev:
	cd frontend && npm run dev

psql:
	$(COMPOSE) exec db psql -U hrprogress hrprogress
