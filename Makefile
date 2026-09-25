.PHONY: test build saas agent seed demo ci tidy

export QS_JWT_SECRET ?= dev-secret-change-me
export QS_DB_PASSWORD ?= quantsaas

tidy:
	go mod tidy

test:
	go test ./internal/quant/ ./internal/strategies/... ./internal/saas/... ./internal/agent/... -count=1

build: saas agent seed

saas:
	go build -o bin/saas ./cmd/saas

agent:
	go build -o bin/agent ./cmd/agent

seed:
	go build -o bin/seed ./cmd/seed

ci: tidy test build

demo:
	@echo "Ensure SaaS is up (docker compose up --build or go run ./cmd/saas)"
	START_AGENT=1 ./scripts/demo_paper.sh
