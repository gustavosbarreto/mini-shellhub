ShellHub Minimal (SSH + Agent)

Overview
- Minimal ShellHub core focused on SSH access via a reverse HTTP tunnel.
- Components:
  - ssh-server: Listens on TCP 2222 for SSH clients and on HTTP 8080 for the agent’s reverse tunnel.
  - agent: Runs on the target host and keeps a reverse tunnel to the server; executes SSH sessions locally.
- Test-friendly: both server and agent accept any password and any public key for authentication (intended for local testing only).

Quick Start
1) Requirements
   - Go installed
   - Ports: 8080 (HTTP, reverse tunnel) and 2222 (SSH server)

2) Build binaries
   - make build
   - Binaries are placed in `bin/`: `bin/ssh-server` and `bin/agent`

3) Generate host keys
   - make keys
   - Keys are placed in `keys/`: `keys/server_hostkey` and `keys/agent_hostkey`

4) Run the server
   - make run-server
   - Uses env `PRIVATE_KEY` pointing to `keys/server_hostkey`

5) Run the agent (same host or another machine)
   - Default server URL is `http://127.0.0.1:8080`
   - Example:
     - make run-agent DEVICE_ID=DEVICE123
   - Single-user mode (no root):
     - make run-agent DEVICE_ID=DEVICE123 SINGLE_PASS="$(openssl passwd -6)"

6) Connect via SSH
   - User format: `user@device-id`
   - Example:
     - ssh -p 2222 'root@DEVICE123'@127.0.0.1
   - Notes:
     - Quote the remote user (`'root@DEVICE123'`) to avoid shell parsing issues with multiple '@'.
     - Any password or public key is accepted in this minimal build for testing.

Makefile Targets
- make build: Build both server and agent.
- make keys: Generate RSA host keys for server and agent.
- make run-server: Start the server (HTTP 8080, SSH 2222).
- make run-agent: Start the agent; variables:
  - SERVER (default http://127.0.0.1:8080)
  - DEVICE_ID (default DEVICE123)
  - SINGLE_PASS (optional; hashed password for single-user mode)
- make up: Launch server in background, then run agent in foreground.
- make down: Stop background server started by `make up`.
- make tidy / make fmt: Go module tidy / formatting.
- make clean: Remove `bin/`, `keys/`, `.server.pid`.

How It Works
- Agent connects to the server’s reverse tunnel endpoint (`/ssh/connection`) via WebSocket.
- Server maps each connection to the provided `X-Device-ID` header (accepts `device` or `tenant:device`).
- When an SSH client connects to the server, it resolves the target device ID and dials the agent through the tunnel.
- The SSH channel is bridged to the agent’s local SSH server (host-mode), executing commands on the target host.

Troubleshooting
- Port in use (2222):
  - ss -lntp | grep ':2222'
  - Kill conflicting process or run `make down` if you used `make up`.
- Reverse tunnel errors:
  - Ensure the agent can reach `SERVER:8080` and that `X-Device-ID` matches the device ID used in your SSH target (`user@DEVICE123`).
- Authentication:
  - Minimal build accepts any password and public key to simplify testing.

Security Notes
- This build is intended for local development/testing. Do not expose it to untrusted networks.
- Consider adding gates (env flags) to turn off “accept-any” auth in production usage.

