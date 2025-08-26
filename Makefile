BIN_DIR := bin
KEY_DIR := keys
SERVER  ?= http://127.0.0.1:8080
DEVICE_ID ?= DEVICE123
SINGLE_PASS ?=

.PHONY: all build ssh agent keys run-server run-agent up down clean tidy fmt help

all: build ## Build ssh-server and agent

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

# Build SSH server binary
ssh: $(BIN_DIR) ## Build only ssh-server
	cd ssh && go build -o ../$(BIN_DIR)/ssh-server

# Build Agent binary
agent: $(BIN_DIR) ## Build only agent
	cd agent && go build -o ../$(BIN_DIR)/agent

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
run-server: build keys ## Run ssh-server (requires port 8080/2222)
	@echo "[server] PRIVATE_KEY=$(PWD)/$(KEY_DIR)/server_hostkey"
	@PRIVATE_KEY=$(PWD)/$(KEY_DIR)/server_hostkey $(BIN_DIR)/ssh-server

# Run Agent (connects to SERVER, uses DEVICE_ID)
run-agent: build keys ## Run agent (SERVER, DEVICE_ID, [SINGLE_PASS])
	@echo "[agent] server=$(SERVER) id=$(DEVICE_ID) key=$(PWD)/$(KEY_DIR)/agent_hostkey"
	$(BIN_DIR)/agent --server $(SERVER) --id $(DEVICE_ID) --key $(PWD)/$(KEY_DIR)/agent_hostkey $(if $(SINGLE_PASS),--single-pass '$(SINGLE_PASS)',)

# Convenience: start server then agent (server in background)
up: build keys ## Start server (bg) then agent
	@echo "[up] starting server in background..."
	@nohup env PRIVATE_KEY=$(PWD)/$(KEY_DIR)/server_hostkey $(BIN_DIR)/ssh-server >/dev/null 2>&1 & echo $$! > .server.pid
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
clean: ## Remove bin and keys
	rm -rf $(BIN_DIR) $(KEY_DIR) .server.pid

# Show help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "%-15s %s\n", $$1, $$2}' | sort
