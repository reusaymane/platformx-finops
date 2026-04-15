.PHONY: dev-up dev-down seed-data run-all test lint build help

## ── Local dev ────────────────────────────────────────────────────────────────

dev-up: ## Start local stack (TimescaleDB, Redis, Prometheus, Grafana)
	docker compose up -d timescaledb redis prometheus grafana
	@echo "Waiting for TimescaleDB to be ready..."
	@sleep 5
	@echo "Stack ready. TimescaleDB: localhost:5432 | Grafana: http://localhost:3001"

dev-down: ## Stop all local services
	docker compose down -v

dev-full: ## Start full stack including all services
	docker compose up -d
	@echo "All services started. Check logs with: make logs"

logs: ## Stream logs from all services
	docker compose logs -f

seed-data: ## Generate 1 year of simulated cost data
	docker compose run --rm data-collector sh -c "FAKE_MODE=true COLLECT_INTERVAL=1s /collector"
	@echo "Data seeded successfully"

## ── Build ────────────────────────────────────────────────────────────────────

build: ## Build all services
	@for svc in data-collector cost-analyzer recommendation-engine anomaly-detection budget-alert; do \
		echo "Building $$svc..."; \
		cd services/$$svc && go build ./... && cd ../..; \
	done
	@echo "Building ml-forecasting..."
	cd services/ml-forecasting && pip install -r requirements.txt -q

build-images: ## Build all Docker images
	docker compose build

## ── Test ─────────────────────────────────────────────────────────────────────

test: ## Run all tests
	@for svc in data-collector cost-analyzer recommendation-engine anomaly-detection budget-alert; do \
		echo "Testing $$svc..."; \
		cd services/$$svc && go test ./... -v -race && cd ../..; \
	done

test-operator: ## Run operator tests
	cd operator && go test ./... -v

lint: ## Run linters
	@for svc in data-collector cost-analyzer recommendation-engine anomaly-detection budget-alert operator; do \
		echo "Linting $$svc..."; \
		cd services/$$svc 2>/dev/null || cd $$svc; \
		golangci-lint run ./...; \
		cd $(shell pwd); \
	done

## ── Operator ─────────────────────────────────────────────────────────────────

operator-install: ## Install CRDs into local cluster (requires kubectl)
	kubectl apply -f operator/config/crd/

operator-run: ## Run operator locally against cluster
	cd operator && go run main.go

## ── WASM Policies ────────────────────────────────────────────────────────────

wasm-build: ## Compile OPA policies to WebAssembly
	@for policy in policies/rego/*.rego; do \
		name=$$(basename $$policy .rego); \
		echo "Compiling $$name to WASM..."; \
		opa build -t wasm -e $$name/deny $$policy -o policies/wasm/$$name.tar.gz; \
	done

wasm-test: ## Test OPA policies
	opa test policies/rego/ -v

## ── Infra ────────────────────────────────────────────────────────────────────

tf-init: ## Init Terraform (dev env)
	cd infra/terragrunt/dev && terragrunt init

tf-plan: ## Plan Terraform (dev env)
	cd infra/terragrunt/dev && terragrunt plan

tf-apply: ## Apply Terraform (dev env)
	cd infra/terragrunt/dev && terragrunt apply

tf-destroy: ## Destroy Terraform (dev env) — saves AWS costs
	cd infra/terragrunt/dev && terragrunt destroy

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
