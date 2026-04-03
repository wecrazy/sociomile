SHELL := /bin/bash
.DEFAULT_GOAL := help

COMPOSE ?= $(shell if command -v podman >/dev/null 2>&1 && podman compose version >/dev/null 2>&1; then echo "podman compose"; elif command -v podman-compose >/dev/null 2>&1; then echo "podman-compose"; elif command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then echo "docker compose"; elif command -v docker-compose >/dev/null 2>&1; then echo "docker-compose"; else echo "podman compose"; fi)
REVIVE_VERSION ?= v1.7.0
ROOT_DIR := $(CURDIR)
BACKEND_DIR := $(ROOT_DIR)/backend
FRONTEND_DIR := $(ROOT_DIR)/frontend
COVERAGE_DIR := $(ROOT_DIR)/coverage
BACKEND_COVERAGE_FILE := $(COVERAGE_DIR)/backend.out
BACKEND_COVERAGE_HTML := $(COVERAGE_DIR)/backend.html
BACKEND_COVERAGE_TXT := $(COVERAGE_DIR)/backend.txt
ROOT_ENV_FILE := $(ROOT_DIR)/.env
COMPOSE_ENV_FILE := $(ROOT_DIR)/.env.compose
BACKEND_ENV_FILE := $(BACKEND_DIR)/.env
FRONTEND_ENV_FILE := $(FRONTEND_DIR)/.env

define LOAD_ROOT_ENV
if [ -f "$(ROOT_ENV_FILE)" ]; then set -a; . "$(ROOT_ENV_FILE)"; set +a; fi
endef

define LOAD_BACKEND_ENV
if [ -f "$(BACKEND_ENV_FILE)" ]; then set -a; . "$(BACKEND_ENV_FILE)"; set +a; fi
endef

define RUN_STEP
	@printf "\n\033[1;36m==>\033[0m %s\n" "$(1)"; \
	if { $(2); }; then \
		printf "\033[1;32m[OK]\033[0m %s\n" "$(3)"; \
	else \
		status=$$?; \
		printf "\033[1;31m[FAIL]\033[0m %s\n" "$(4)"; \
		exit $$status; \
	fi
endef

.PHONY: help env setup dev dev-attached dev-down dev-logs config backend-run worker-run frontend-run \
	migrate seed fmt backend-fmt frontend-fmt frontend-fmt-check backend-lint backend-test \
	frontend-test backend-coverage frontend-coverage coverage lint build swagger clean

##@ Help
help: ## Show all available targets with a short description.
	@printf "\n\033[1;35m"
	@printf "███████╗ ██████╗  ██████╗██╗ ██████╗ ███╗   ███╗██╗██╗     ███████╗\n"
	@printf "██╔════╝██╔═══██╗██╔════╝██║██╔═══██╗████╗ ████║██║██║     ██╔════╝\n"
	@printf "███████╗██║   ██║██║     ██║██║   ██║██╔████╔██║██║██║     █████╗  \n"
	@printf "╚════██║██║   ██║██║     ██║██║   ██║██║╚██╔╝██║██║██║     ██╔══╝  \n"
	@printf "███████║╚██████╔╝╚██████╗██║╚██████╔╝██║ ╚═╝ ██║██║███████╗███████╗\n"
	@printf "╚══════╝ ╚═════╝  ╚═════╝╚═╝ ╚═════╝ ╚═╝     ╚═╝╚═╝╚══════╝╚══════╝\n"
	@printf "\033[0m"

	@printf "\n\033[1mMake Commands\033[0m\n"
	@printf "\033[2mUse make <target> to run a local workflow.\033[0m\n"

	@awk 'BEGIN {FS = ":.*## "} \
		/^##@/ {printf "\n\033[1;37m%s\033[0m\n", substr($$0, 5); next} \
		/^[a-zA-Z0-9_.-]+:.*## / {printf "  \033[1;36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

##@ Bootstrap
env: ## Create local .env files from templates without overwriting existing ones.
	@report_env_file() { \
		label="$$1"; \
		src="$$2"; \
		dest="$$3"; \
		if [ -f "$$dest" ]; then \
			printf "  \033[2mexisting\033[0m %s\n" "$$label"; \
		else \
			cp "$$src" "$$dest"; \
			printf "  \033[1;32mcreated \033[0m %s\n" "$$label"; \
			created=$$((created + 1)); \
		fi; \
	}; \
	created=0; \
	printf "\033[1mEnv bootstrap\033[0m\n"; \
	report_env_file ".env" "$(ROOT_DIR)/.env.example" "$(ROOT_ENV_FILE)"; \
	report_env_file ".env.compose" "$(ROOT_DIR)/.env.compose.example" "$(COMPOSE_ENV_FILE)"; \
	report_env_file "backend/.env" "$(BACKEND_DIR)/.env.example" "$(BACKEND_ENV_FILE)"; \
	report_env_file "frontend/.env" "$(FRONTEND_DIR)/.env.example" "$(FRONTEND_ENV_FILE)"; \
	if [ "$$created" -eq 0 ]; then \
		printf "\033[1;33mNo new env files were created.\033[0m\n"; \
	else \
		printf "\033[1;32mCreated %s env file(s).\033[0m\n" "$$created"; \
	fi

setup: env ## Install backend and frontend dependencies.
	cd $(BACKEND_DIR) && go mod tidy
	cd $(FRONTEND_DIR) && npm install

##@ Local Stack
dev: env ## Build and run the full local stack in detached mode.
	$(LOAD_ROOT_ENV) && $(COMPOSE) --env-file "$(COMPOSE_ENV_FILE)" up --build -d && \
	printf "\033[1;32mLocal stack is running in the background.\033[0m\n" && \
	printf "  logs: make dev-logs\n" && \
	printf "  stop: make dev-down\n"

dev-attached: env ## Build and run the full local stack attached to container logs.
	$(LOAD_ROOT_ENV) && $(COMPOSE) --env-file "$(COMPOSE_ENV_FILE)" up --build

dev-down: ## Stop and remove the local stack containers.
	$(LOAD_ROOT_ENV) && $(COMPOSE) --env-file "$(COMPOSE_ENV_FILE)" down

dev-logs: ## Stream the local stack logs in real time.
	$(LOAD_ROOT_ENV) && $(COMPOSE) --env-file "$(COMPOSE_ENV_FILE)" logs -f

config: env ## Print the resolved compose configuration using .env and .env.compose.
	$(LOAD_ROOT_ENV) && $(COMPOSE) --env-file "$(COMPOSE_ENV_FILE)" config

##@ Service
backend-run: env ## Run the Fiber API directly from the host using backend/.env.
	cd $(BACKEND_DIR) && $(LOAD_BACKEND_ENV) && go run ./cmd/api

worker-run: env ## Run the outbox worker directly from the host using backend/.env.
	cd $(BACKEND_DIR) && $(LOAD_BACKEND_ENV) && go run ./cmd/worker

frontend-run: env ## Run the Vite dev server from the host using frontend/.env.
	cd $(FRONTEND_DIR) && npm run dev -- --host

##@ Database
migrate: env ## Apply backend SQL migrations using backend/.env.
	cd $(BACKEND_DIR) && $(LOAD_BACKEND_ENV) && go run ./cmd/api migrate

seed: env ## Load demo tenants, users, and channels.
	cd $(BACKEND_DIR) && $(LOAD_BACKEND_ENV) && go run ./cmd/api seed

##@ Quality
fmt: ## Format backend Go sources and frontend source or config files.
	$(MAKE) backend-fmt
	$(MAKE) frontend-fmt

backend-fmt: ## Format backend Go sources with gofmt.
	cd $(BACKEND_DIR) && find . -name '*.go' -not -path './vendor/*' -print0 | xargs -0 gofmt -w

frontend-fmt: ## Format frontend source and config files with Prettier.
	cd $(FRONTEND_DIR) && npm run format

frontend-fmt-check: ## Check frontend source and config formatting with Prettier.
	cd $(FRONTEND_DIR) && npm run format:check

backend-lint: ## Run backend Go formatting, vet, and revive checks.
	$(call RUN_STEP,Checking Go formatting,cd $(BACKEND_DIR) && files="$$(find . -name '*.go' -not -path './vendor/*' -print0 | xargs -0 gofmt -l)" && if [ -n "$$files" ]; then printf "Unformatted Go files:\n%s\n" "$$files"; exit 1; fi,Go formatting is clean,Go formatting check failed)
	$(call RUN_STEP,Running go vet,cd $(BACKEND_DIR) && go vet ./...,go vet passed,go vet failed)
	$(call RUN_STEP,Running revive,cd $(BACKEND_DIR) && go run github.com/mgechev/revive@$(REVIVE_VERSION) -config revive.toml -formatter friendly ./...,revive passed,revive failed)

backend-test: ## Run backend unit tests.
	cd $(BACKEND_DIR) && go test ./...

frontend-test: ## Run frontend tests.
	cd $(FRONTEND_DIR) && npm run test

backend-coverage: ## Run backend tests with repository-wide coverage output.
	mkdir -p $(COVERAGE_DIR)
	cd $(BACKEND_DIR) && go test -covermode=atomic -coverpkg=./... ./... -coverprofile=$(BACKEND_COVERAGE_FILE)
	cd $(BACKEND_DIR) && go tool cover -func=$(BACKEND_COVERAGE_FILE) | tee $(BACKEND_COVERAGE_TXT)
	cd $(BACKEND_DIR) && go tool cover -html=$(BACKEND_COVERAGE_FILE) -o $(BACKEND_COVERAGE_HTML)

frontend-coverage: ## Run frontend tests with V8 coverage reports.
	mkdir -p $(COVERAGE_DIR)
	cd $(FRONTEND_DIR) && npm run test:coverage

coverage: ## Run backend and frontend coverage workflows.
	$(MAKE) backend-coverage
	$(MAKE) frontend-coverage

lint: ## Run backend Go linting, frontend formatting checks, and frontend TypeScript linting.
	@printf "\n\033[1mLint Suite\033[0m\n"
	$(call RUN_STEP,Checking Go formatting,cd $(BACKEND_DIR) && files="$$(find . -name '*.go' -not -path './vendor/*' -print0 | xargs -0 gofmt -l)" && if [ -n "$$files" ]; then printf "Unformatted Go files:\n%s\n" "$$files"; exit 1; fi,Go formatting is clean,Go formatting check failed)
	$(call RUN_STEP,Running go vet,cd $(BACKEND_DIR) && go vet ./...,go vet passed,go vet failed)
	$(call RUN_STEP,Running revive,cd $(BACKEND_DIR) && go run github.com/mgechev/revive@$(REVIVE_VERSION) -config revive.toml -formatter friendly ./...,revive passed,revive failed)
	$(call RUN_STEP,Running frontend format check,cd $(FRONTEND_DIR) && npm run format:check,Frontend formatting is clean,Frontend format check failed)
	$(call RUN_STEP,Running frontend TypeScript lint,cd $(FRONTEND_DIR) && npm run lint,Frontend TypeScript lint passed,Frontend TypeScript lint failed)
	@printf "\n\033[1;32mLint finished with no errors.\033[0m\n"

build: env ## Build container images without starting the stack.
	$(LOAD_ROOT_ENV) && $(COMPOSE) --env-file "$(COMPOSE_ENV_FILE)" build

##@ Documentation
swagger: ## Show the local Swagger UI location.
	@echo "Swagger UI will serve backend/docs/openapi.yaml at /swagger"

##@ Utilities
clean: ## Remove local frontend and backend build artifacts.
	rm -rf $(FRONTEND_DIR)/dist $(FRONTEND_DIR)/node_modules
	rm -rf $(BACKEND_DIR)/bin
	rm -rf $(COVERAGE_DIR)
	rm -f $(FRONTEND_DIR)/*.tsbuildinfo $(FRONTEND_DIR)/vite.config.js $(FRONTEND_DIR)/vite.config.d.ts $(FRONTEND_DIR)/vite.config.d.ts.map