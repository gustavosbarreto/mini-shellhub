KEY_DIR := keys
SERVER  ?= http://127.0.0.1:8080
DEVICE_ID ?= DEVICE123
SINGLE_PASS ?=

.PHONY: all build ssh agent keys run-server run-agent up down clean tidy fmt help

all: build ## Build ssh-server and agent

# Build SSH server binary
ssh: ## Build only ssh-server
	cd ssh && go build -o ssh-server

# Build Agent binary
agent: ## Build only agent
	cd agent && go build -o agent

# Build both
build: ssh agent ## Build both binaries

# Generate RSA keys for server and agent
keys: keys/server_hostkey keys/agent_hostkey ## Generate host keys (server/agent)

$(KEY_DIR)/server_hostkey:
	mkdir -p $(KEY_DIR)
	ssh-keygen -t rsa -b 2048 -m PEM -N '' -f $(KEY_DIR)/server_hostkey

$(KEY_DIR)/agent_hostkey:
	mkdir -p $(KEY_DIR)
	ssh-keygen -t rsa -b 2048 -m PEM -N '' -f $(KEY_DIR)/agent_hostkey

# Run SSH server (HTTP :8080, SSH :2222)
run-server: ssh ## Run ssh-server (requires port 8080/2222)
	@echo "[server] using in-memory generated key"
	@cd ssh && ./ssh-server

# Run Agent (connects to SERVER, uses DEVICE_ID)
run-agent: agent ## Run agent (SERVER, DEVICE_ID, [SINGLE_PASS])
	@echo "[agent] server=$(SERVER) id=$(DEVICE_ID) using in-memory generated key"
	cd agent && ./agent --server $(SERVER) --id $(DEVICE_ID) $(if $(SINGLE_PASS),--single-pass '$(SINGLE_PASS)',)

# Convenience: start server then agent (server in background)
up: build ## Start server (bg) then agent
	@echo "[up] starting server in background..."
	@cd ssh && nohup ./ssh-server >/dev/null 2>&1 & echo $$! > ../.server.pid
	@sleep 0.5
	@$(MAKE) --no-print-directory run-agent

# Stop background server
down: ## Stop server started by 'make up'
	@if [ -f .server.pid ]; then kill `cat .server.pid` || true; rm -f .server.pid; echo "[down] server stopped"; else echo "[down] no server pid"; fi

# Run go fmt on submodules
fmt: ## go fmt ./... (ssh, agent)
	cd ssh && go fmt ./...
	cd agent && go fmt ./...

# Run go mod tidy on submodules
tidy: ## go mod tidy (ssh, agent)
	cd ssh && go mod tidy
	cd agent && go mod tidy

# Clean build artifacts and keys
clean: ## Remove binaries and keys
	rm -f ssh/ssh-server agent/agent
	rm -rf $(KEY_DIR) .server.pid

# Show help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "%-15s %s\n", $$1, $$2}' | sort
