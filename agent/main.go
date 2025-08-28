package main

import (
    "context"
    "flag"
    "net"
    "net/http"
    "os"
    "os/exec"
    "time"

    "github.com/hashicorp/yamux"
    agentsrv "github.com/shellhub-io/mini-shellhub/agent/pkg/agent/server"
    hostmode "github.com/shellhub-io/mini-shellhub/agent/pkg/agent/server/modes/host"
    apiclient "github.com/shellhub-io/shellhub/pkg/api/client"
    "github.com/shellhub-io/mini-shellhub/pkg/yamuxws"
    log "github.com/sirupsen/logrus"
)

func main() {
    var serverURL string
    var deviceID string
    var privKey string
    var singleUserPass string

    flag.StringVar(&serverURL, "server", os.Getenv("MINIMAL_SERVER"), "Server base URL, e.g. http://127.0.0.1:8080")
    flag.StringVar(&deviceID, "id", os.Getenv("MINIMAL_DEVICE_ID"), "Device ID for registration")
    flag.StringVar(&privKey, "key", os.Getenv("MINIMAL_PRIVATE_KEY"), "Path to SSH host private key (PEM)")
    flag.StringVar(&singleUserPass, "single-pass", os.Getenv("MINIMAL_SINGLE_USER_PASSWORD"), "Enable single-user mode with this password hash")
    flag.Parse()

    if serverURL == "" || deviceID == "" {
        log.Fatal("missing required params: --server, --id")
    }

    deviceName := deviceID

    // Build host mode server with password auth (public key auth disabled without API).
    mode := &hostmode.Mode{
        Authenticator: *hostmode.NewAuthenticator(nil, nil, singleUserPass, &deviceName),
        Sessioner:     *hostmode.NewSessioner(&deviceName, make(map[string]*exec.Cmd)),
    }

    srv := agentsrv.NewServer(nil, mode, &agentsrv.Config{PrivateKey: privKey})

    // Connect to server via websocket
    ctx := context.Background()
    conn, _, err := apiclient.DialContext(ctx, serverURL+"/ssh/connection", http.Header{"X-Device-ID": []string{deviceID}})
    if err != nil {
        log.WithError(err).Fatal("failed to connect to server")
    }

    // Create yamux session over websocket
    wsConn := yamuxws.NewWSConn(conn)
    session, err := yamux.Client(wsConn, yamux.DefaultConfig())
    if err != nil {
        log.WithError(err).Fatal("failed to create yamux session")
    }
    defer session.Close()

    log.WithFields(log.Fields{"server": serverURL, "id": deviceID}).Info("connected; listening for SSH via yamux")

    // Accept incoming streams (SSH connections)
    for {
        stream, err := session.Accept()
        if err != nil {
            log.WithError(err).Error("failed to accept yamux stream")
            break
        }
        
        go handleSSHStream(srv, stream)
    }

    time.Sleep(time.Second)
}

// handleSSHStream handles a yamux stream as an SSH connection
func handleSSHStream(serv *agentsrv.Server, stream net.Conn) {
    defer stream.Close()
    
    log.WithFields(log.Fields{
        "remote": stream.RemoteAddr(),
    }).Info("handling SSH stream")
    
    // Handle the connection directly with the SSH server
    serv.HandleConn(stream)
}
