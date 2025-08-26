# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a minimal implementation of ShellHub focused on SSH access via reverse HTTP tunnel. It consists of two main components:
- **ssh-server**: Listens on HTTP :8080 (reverse tunnel) and SSH :2222 (SSH clients)  
- **agent**: Runs on target hosts, maintains reverse tunnel to server, executes SSH sessions locally

The codebase is written in Go and uses a modular architecture with shared packages in `pkg/`.

## Build Commands

- `make build` - Build both ssh-server and agent binaries (outputs to `bin/`)
- `make ssh` - Build only ssh-server  
- `make agent` - Build only agent
- `make keys` - Generate RSA host keys for server and agent (outputs to `keys/`)
- `make clean` - Remove build artifacts (`bin/`, `keys/`, `.server.pid`)

## Development Commands

- `make fmt` - Format Go code in ssh and agent modules
- `make tidy` - Run `go mod tidy` on ssh and agent modules
- `make run-server` - Start ssh-server (requires keys)
- `make run-agent` - Start agent with default settings
- `make up` - Start server in background, then run agent in foreground
- `make down` - Stop background server started by `make up`

### Agent Configuration

The agent accepts these parameters:
- `SERVER` - Server URL (default: http://127.0.0.1:8080)
- `DEVICE_ID` - Device identifier (default: DEVICE123)  
- `SINGLE_PASS` - Optional hashed password for single-user mode

Example: `make run-agent DEVICE_ID=mydevice SINGLE_PASS="$(openssl passwd -6)"`

## Architecture

### Core Components

- **ssh/**: SSH server implementation using gliderlabs/ssh
  - `main.go` - Entry point, sets up HTTP tunnel and SSH server
  - `server/` - SSH server logic with authentication and channel handling
  - `session/` - Session management
- **agent/**: Agent implementation  
  - `main.go` - Entry point, connects to server via WebSocket, handles SSH bridging
- **pkg/**: Shared packages
  - `httptunnel/` - HTTP tunnel implementation for reverse connections
  - `agent/` - Agent core functionality and server modes
  - `revdial/` - Reverse dial implementation for tunneling
  - `api/client/` - API client utilities

### Communication Flow

1. Agent connects to server's `/ssh/connection` endpoint via WebSocket with `X-Device-ID` header
2. Server maps connections by device ID
3. SSH clients connect to server on port 2222 using format `user@device-id`  
4. Server routes SSH traffic through established tunnel to appropriate agent
5. Agent bridges SSH traffic to local SSH server running in host mode

### Key Dependencies

- `github.com/gorilla/websocket` - WebSocket connections
- `github.com/labstack/echo/v4` - HTTP router
- `github.com/sirupsen/logrus` - Logging
- `github.com/pires/go-proxyproto` - Proxy protocol support

## Testing Notes

- This is a minimal build that accepts any password/public key for authentication
- Intended for local development/testing only
- SSH connection format: `ssh -p 2222 'root@DEVICE123'@127.0.0.1`
- Quote the remote user to avoid shell parsing issues with multiple '@' symbols

## Multi-module Structure

The project uses Go modules in subdirectories:
- Root `go.mod` - Main module
- `ssh/go.mod` - SSH server module  
- `agent/go.mod` - Agent module

When making changes, run `make tidy` to update all module dependencies.