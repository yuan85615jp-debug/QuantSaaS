.PHONY: test build saas agent seed tvimport demo ci tidy vet

export QS_JWT_SECRET ?= dev-secret-change-me
export QS_DB_PASSWORD ?= quantsaas

tidy:
	go mod tidy

vet:
	go vet ./cmd/... ./internal/...

test:
	go test ./internal/quant/ ./internal/strategies/... ./internal/saas/... ./internal/agent/... ./cmd/tvimport/ -count=1 -timeout 120s

build: saas agent seed tvimport

saas:
	go build -o bin/saas ./cmd/saas

agent:
	go build -o bin/agent ./cmd/agent

seed:
	go build -o bin/seed ./cmd/seed

tvimport:
	go build -o bin/tvimport ./cmd/tvimport

ci: tidy vet test build

demo:
	@echo "Ensure SaaS is up (docker compose up --build or go run ./cmd/saas)"
	START_AGENT=1 ./scripts/demo_paper.sh
