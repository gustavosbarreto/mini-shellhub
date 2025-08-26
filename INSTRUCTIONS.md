Developer Instructions

Repo Layout (minimal)
- ssh/: SSH server entrypoint and runtime (HTTP + SSH) and session/channel handlers
  - main_minimal.go: main entry (now default) for the SSH+HTTP server
  - server/: GliderLabs SSH server setup and channel handlers
  - session/: Minimal session to bridge client <-> agent (no API/billing/firewall)
- agent/: Minimal agent main
  - main.go: agent entrypoint; sets up reverse tunnel and local SSH server in host mode
- pkg/: Shared libs used by both server and agent (httptunnel, revdial, wsconnadapter, connman, models, etc.)

Build
- make build (generates `bin/ssh-server` and `bin/agent`)
- Go versions: use the toolchain specified in the module files; `go mod tidy` on submodules via `make tidy`.

Run
1) Keys
   - make keys (creates `keys/server_hostkey` and `keys/agent_hostkey`)
2) Server
   - make run-server (binds :8080 HTTP for tunnel and :2222 for SSH)
3) Agent
   - make run-agent DEVICE_ID=DEVICE123 [SERVER=http://127.0.0.1:8080 SINGLE_PASS="$(openssl passwd -6)"]
4) Client SSH
   - ssh -p 2222 'root@DEVICE123'@127.0.0.1

Configuration
- Server
  - PRIVATE_KEY (env): path to SSH host private key (PEM). The Makefile sets this automatically when using `make run-server`.
- Agent CLI flags
  - --server: server base URL (http://host:8080)
  - --id: device id used to register the reverse tunnel
  - --key: path to agent’s SSH host private key (PEM)
  - --single-pass: (optional) hashed password for single-user mode (use `openssl passwd -6`)

Auth policy (test mode)
- Server side:
  - Password: forwarded to agent (agent accepts any).
  - Public key: accepted; server authenticates to agent using a dummy password to establish the bridge.
- Agent side:
  - Accepts any password and any public key (for local testing only).

Reverse Tunnel
- Endpoint: `GET /ssh/connection` (WebSocket)
- Header `X-Device-ID`:
  - Accepts `tenant:device` or `device` (single segment). The agent uses `device` by default.
- The server’s tunnel maps connections per device and lets the SSH server dial the agent over that mapping.

Common Issues
- Port 2222 busy:
  - ss -lntp | grep ':2222' to find listeners
  - `make down` if you started the server with `make up`.
- Agent panic (ticker interval):
  - KeepAlive interval now defaults to 30s when unset.
- Nil pointer on session pipe:
  - Minimal session sets a Device.Info (Version: v0.9.3) to avoid checks on nil.

Development Tips
- Use `make up` to start server in background and the agent in foreground for quick iteration.
- Use quotes around SSH users containing `@` in your shell (e.g., 'root@DEVICE123').
- Logs: both server and agent use logrus JSON logs; grep by `uid`, `sshid`, `device` for session tracking.

Production Hardening (optional, not implemented here)
- Replace “accept-any auth” with real PAM/public-key validation on agent.
- Gate test-mode behaviors with an env flag (e.g., TEST_MODE=1).
- Change tunnel key policy to require `tenant:device`.
- TLS termination for the tunnel and SSH server behind a proper proxy/gateway.

