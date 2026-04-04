.PHONY: build test lint clean contracts-build contracts-test backend-build backend-test backend-run backend-dev web-build web-dev e2e

# Top-level targets
build: contracts-build backend-build web-build
test: contracts-test backend-test
lint: backend-lint
clean: backend-clean

# Contracts
contracts-build:
	cd contracts && NO_DNA=1 anchor build

contracts-test:
	cd contracts && NO_DNA=1 anchor test

# Backend
backend-build:
	cd backend && go build -o bin/server ./cmd/server

backend-test:
	cd backend && go test -race -count=1 -v ./...

backend-run:
	cd backend && go run ./cmd/server

backend-lint:
	cd backend && golangci-lint run --timeout 5m

backend-clean:
	cd backend && rm -rf bin/

backend-migrate:
	cd backend && go run ./cmd/server migrate

# Frontend
web-build:
	cd web && npm run build

web-dev:
	cd web && npm run dev

# E2E
e2e:
	cd e2e && npx playwright test

# Docker
up:
	docker compose -f deployments/docker-compose.yml up -d

down:
	docker compose -f deployments/docker-compose.yml down

# Backend dev with hot reload (postgres + backend with air)
backend-dev:
	docker compose -f deployments/docker-compose.yml up --build
