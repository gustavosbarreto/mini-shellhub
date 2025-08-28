package host

import (
	"fmt"
	"os"
	"os/exec"

	gliderssh "github.com/gliderlabs/ssh"
)

func generateShellCmd(deviceName string, session gliderssh.Session, term string) *exec.Cmd {
	envs := session.Environ()

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	if term == "" {
		term = "xterm"
	}

	authSock := session.Context().Value("SSH_AUTH_SOCK")
	if authSock != nil {
		envs = append(envs, fmt.Sprintf("%s=%s", "SSH_AUTH_SOCK", authSock.(string)))
	}

	cmd := exec.Command(shell, "--login")
	cmd.Env = envs
	
	return cmd
}
