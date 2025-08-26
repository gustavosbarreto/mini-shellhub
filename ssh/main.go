package main

import (
    "fmt"
    "net/http"

    "github.com/shellhub-io/shellhub/pkg/httptunnel"
    "github.com/shellhub-io/shellhub/ssh/server"
    log "github.com/sirupsen/logrus"
)

const ListenAddress = ":8080"

func init() {
    log.SetFormatter(&log.JSONFormatter{})
}

// minimalMain starts only the SSH server and a basic reverse tunnel endpoint.
func main() {
    // Setup a bare tunnel with simple ID routing: key == device ID.
    tun := httptunnel.NewTunnel("/ssh/connection", "/ssh/revdial")

    tun.ConnectionHandler = func(r *http.Request) (string, error) {
        // Accept an optional X-Device-ID header for simple mapping.
        id := r.Header.Get("X-Device-ID")
        if id == "" {
            // Fallback to legacy headers if present.
            uid := r.Header.Get("X-Device-UID")
            if uid != "" {
                id = uid
            }
        }
        if id == "" {
            return "", fmt.Errorf("missing X-Device-ID header")
        }
        return id, nil
    }

    router := tun.Router()

    // Profiling not enabled in minimal build.

    errs := make(chan error)

    go func() {
        errs <- http.ListenAndServe(ListenAddress, router) //nolint:gosec
    }()

    go func() {
    errs <- server.NewServer(&server.Options{ // nolint:exhaustruct
            ConnectTimeout:               0,
            AllowPublickeyAccessBelow060: false,
        }, tun).ListenAndServe()
    }()

    if err := <-errs; err != nil {
        log.WithError(err).Fatal("fatal error from HTTP or SSH server")
    }

    log.Warn("ssh service is closed")
}
