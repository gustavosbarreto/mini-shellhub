package agent

import (
	"os/exec"
	"runtime"

	"github.com/shellhub-io/mini-shellhub/agent/pkg/agent/server"
	"github.com/shellhub-io/mini-shellhub/agent/pkg/agent/server/modes/host"
)

type Info struct {
	ID   string
	Name string
}

// Mode is the Agent execution mode.
//
// Check [HostMode] and [ConnectorMode] for more information.
type Mode interface {
	// Serve prepares the Agent for listening, setting up the SSH server, its modes and values on Agent's.
	Serve(agent *Agent)
	// GetInfo gets information about Agent according to Agent's mode.
	//
	// When Agent is running on [HostMode], the info got is from the system where the Agent is running, but when running
	// in [ConnectorMode], the data is retrieved from Docker Engine.
	GetInfo() (*Info, error)
}

// ModeHost is the Agent execution mode for `Host`.
//
// The host mode is the default mode one, and turns the host machine into a ShellHub's Agent. The host is
// responsible for the SSH server, authentication and authorization, `/etc/passwd`, `/etc/shadow`, and etc.
type HostMode struct{}

var _ Mode = new(HostMode)

func (m *HostMode) Serve(agent *Agent) {
	agent.server = server.NewServer(
		agent.cli,
		&host.Mode{
			Authenticator: *host.NewAuthenticator(agent.cli, agent.authData, agent.config.SingleUserPassword, &agent.authData.Name),
			Sessioner:     *host.NewSessioner(&agent.authData.Name, make(map[string]*exec.Cmd)),
		},
		&server.Config{
			PrivateKey:        agent.config.PrivateKey,
			KeepAliveInterval: agent.config.KeepAliveInterval,
			Features:          server.LocalPortForwardFeature,
		},
	)

	agent.server.SetDeviceName(agent.authData.Name)
}

func (m *HostMode) GetInfo() (*Info, error) {
	return &Info{
		ID:   runtime.GOOS,
		Name: runtime.GOOS,
	}, nil
}

