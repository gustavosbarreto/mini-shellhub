package main

import (
    "context"
    "flag"
    "io"
    "net"
    "net/http"
    "os"
    "os/exec"
    "time"

    "github.com/gorilla/websocket"
    "github.com/labstack/echo/v4"
    agentsrv "github.com/shellhub-io/shellhub/pkg/agent/server"
    hostmode "github.com/shellhub-io/shellhub/pkg/agent/server/modes/host"
    apiclient "github.com/shellhub-io/shellhub/pkg/api/client"
    "github.com/shellhub-io/shellhub/pkg/revdial"
    "github.com/shellhub-io/shellhub/pkg/wsconnadapter"
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

    if serverURL == "" || deviceID == "" || privKey == "" {
        log.Fatal("missing required params: --server, --id, --key")
    }

    deviceName := deviceID

    // Build host mode server with password auth (public key auth disabled without API).
    mode := &hostmode.Mode{
        Authenticator: *hostmode.NewAuthenticator(nil, nil, singleUserPass, &deviceName),
        Sessioner:     *hostmode.NewSessioner(&deviceName, make(map[string]*exec.Cmd)),
    }

    srv := agentsrv.NewServer(nil, mode, &agentsrv.Config{PrivateKey: privKey})

    // Bridge handlers for SSH over reverse tunnel.
    tunnel := hostTunnel(srv)

    // Connect to server via websocket and create reverse listener.
    ctx := context.Background()
    // Send only device ID; server tunnel accepts one-part keys (device-only)
    conn, _, err := apiclient.DialContext(ctx, serverURL+"/ssh/connection", http.Header{"X-Device-ID": []string{deviceID}})
    if err != nil {
        log.WithError(err).Fatal("failed to connect to server")
    }
    listener := revdial.NewListener(wsconnadapter.New(conn), func(ctx context.Context, path string) (*websocket.Conn, *http.Response, error) {
        return apiclient.DialContext(ctx, serverURL+path, nil)
    })

    log.WithFields(log.Fields{"server": serverURL, "id": deviceID}).Info("connected; listening for SSH")

    if err := tunnel.Listen(listener); err != nil {
        log.WithError(err).Error("tunnel listener closed")
    }

    time.Sleep(time.Second)
}

// hostTunnel builds a minimal echo server binding SSH handlers for reverse listener.
func hostTunnel(serv *agentsrv.Server) *hostTunnelWrap {
    e := echo.New()
    e.GET("/ssh/:id", func(c echo.Context) error { return sshHandler(serv)(c) })
    e.GET("/ssh/close/:id", func(c echo.Context) error { return sshCloseHandler(serv)(c) })
    e.CONNECT("/http/proxy/:addr", func(c echo.Context) error { return httpProxyHandler()(c) })
    return &hostTunnelWrap{router: e}
}

type hostTunnelWrap struct{ router *echo.Echo }
type hostServer struct{ *http.Server }

func (t *hostTunnelWrap) Listen(l *revdial.Listener) error {
    srv := &http.Server{ //nolint:gosec
        Handler: t.router,
        ConnContext: func(ctx context.Context, c net.Conn) context.Context {
            return context.WithValue(ctx, "http-conn", c) //nolint:revive
        },
    }
    return srv.Serve(l)
}
func (t *hostTunnelWrap) Close() error { return t.router.Close() }

// sshHandler proxies TCP stream from reverse tunnel into the agent SSH server.
func sshHandler(serv *agentsrv.Server) func(c echo.Context) error {
    return func(c echo.Context) error {
        hj, ok := c.Response().Writer.(http.Hijacker)
        if !ok {
            return c.String(http.StatusInternalServerError, "webserver doesn't support hijacking")
        }
        conn := c.Request().Context().Value("http-conn").(net.Conn)
        _, buf, err := hj.Hijack()
        if err != nil {
            return c.String(http.StatusInternalServerError, err.Error())
        }
        // Write a HTTP 200 OK to finish the handshake and start raw TCP bridge.
        if _, err := buf.WriteString("HTTP/1.1 200 OK\r\n\r\n"); err != nil { return err }
        if err := buf.Flush(); err != nil { return err }
        serv.HandleConn(conn)
        return nil
    }
}

func sshCloseHandler(serv *agentsrv.Server) func(c echo.Context) error {
    return func(c echo.Context) error {
        id := c.Param("id")
        serv.CloseSession(id)
        return c.NoContent(http.StatusOK)
    }
}

func httpProxyHandler() func(c echo.Context) error {
    return func(c echo.Context) error {
        hj, ok := c.Response().Writer.(http.Hijacker)
        if !ok { return c.String(http.StatusInternalServerError, "webserver doesn't support hijacking") }
        out := c.Request().Context().Value("http-conn").(net.Conn)
        in, buf, err := hj.Hijack()
        if err != nil { return c.String(http.StatusInternalServerError, err.Error()) }
        if _, err := buf.WriteString("HTTP/1.1 200 OK\r\n\r\n"); err != nil { return err }
        if err := buf.Flush(); err != nil { return err }
        go func() { defer out.Close(); defer in.Close(); io.Copy(out, in) }()
        go func() { defer out.Close(); defer in.Close(); io.Copy(in, out) }()
        return nil
    }
}
