SHELL := /bin/bash

COMPOSE       ?= docker compose
OLLAMA_EXEC   ?= $(COMPOSE) exec -T ollama ollama
DEFAULT_MODEL ?= qwen2.5-coder:14b

.PHONY: help up down restart build logs ps health \
        pull-light pull-default pull-heavy pull-deepseek pull-all \
        models index reset clean \
        tools-build tools-test webui-url

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

up: ## Start all core services in the background
	$(COMPOSE) up -d --build

down: ## Stop and remove containers (keep volumes)
	$(COMPOSE) down

restart: ## Restart core services
	$(COMPOSE) restart

build: ## Rebuild local images
	$(COMPOSE) build

logs: ## Tail logs from all services
	$(COMPOSE) logs -f --tail=100

ps: ## List running services
	$(COMPOSE) ps

health: ## Hit health endpoints
	@curl -fsS http://localhost:$${TOOLS_PORT:-8088}/health || true
	@echo
	@curl -fsS http://localhost:$${OLLAMA_PORT:-11434}/api/tags >/dev/null && echo "ollama: ok" || echo "ollama: down"

pull-light: ## Pull qwen2.5-coder:7b (fastest, low RAM)
	$(OLLAMA_EXEC) pull qwen2.5-coder:7b

pull-default: ## Pull qwen2.5-coder:14b (recommended default)
	$(OLLAMA_EXEC) pull qwen2.5-coder:14b

pull-heavy: ## Pull qwen2.5-coder:32b (best quality, needs RAM)
	$(OLLAMA_EXEC) pull qwen2.5-coder:32b

pull-deepseek: ## Pull deepseek-coder-v2:lite (MoE coder)
	$(OLLAMA_EXEC) pull deepseek-coder-v2:lite

pull-all: pull-light pull-default pull-heavy pull-deepseek ## Pull every recommended model

models: ## List installed models in Ollama
	$(OLLAMA_EXEC) list

index: ## Run the code indexer over ./workspace
	$(COMPOSE) --profile tools run --rm indexer

reset: ## Stop services and remove volumes (DESTRUCTIVE)
	$(COMPOSE) down -v

clean: ## Remove generated index data
	rm -rf data/index

tools-build: ## Build the Go tools-api image only
	$(COMPOSE) build tools-api

tools-test: ## Run go test inside tools-api
	$(COMPOSE) run --rm tools-api go test ./...

webui-url: ## Print the Open WebUI URL
	@echo "http://localhost:$${WEBUI_PORT:-3000}"
