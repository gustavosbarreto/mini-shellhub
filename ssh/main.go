package main

import (
    "fmt"
    "io"
    "net/http"
    "sync"

    "github.com/gorilla/websocket"
    "github.com/hashicorp/yamux"
    "github.com/labstack/echo/v4"
    "github.com/shellhub-io/mini-shellhub/pkg/yamuxws"
    "github.com/shellhub-io/mini-shellhub/ssh/server"
    log "github.com/sirupsen/logrus"
)

const ListenAddress = ":8080"

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(_ *http.Request) bool {
        return true
    },
}

// DeviceManager manages yamux sessions per device
type DeviceManager struct {
    sessions map[string]*yamux.Session
    mutex    sync.RWMutex
}

func NewDeviceManager() *DeviceManager {
    return &DeviceManager{
        sessions: make(map[string]*yamux.Session),
    }
}

func (dm *DeviceManager) AddDevice(deviceID string, session *yamux.Session) {
    dm.mutex.Lock()
    defer dm.mutex.Unlock()
    
    // Close existing session if any
    if oldSession, exists := dm.sessions[deviceID]; exists {
        oldSession.Close()
    }
    
    dm.sessions[deviceID] = session
    log.WithFields(log.Fields{"device": deviceID}).Info("device connected via yamux")
}

func (dm *DeviceManager) RemoveDevice(deviceID string) {
    dm.mutex.Lock()
    defer dm.mutex.Unlock()
    
    if session, exists := dm.sessions[deviceID]; exists {
        session.Close()
        delete(dm.sessions, deviceID)
        log.WithFields(log.Fields{"device": deviceID}).Info("device disconnected")
    }
}

func (dm *DeviceManager) OpenStream(deviceID string) (io.ReadWriteCloser, error) {
    dm.mutex.RLock()
    defer dm.mutex.RUnlock()
    
    session, exists := dm.sessions[deviceID]
    if !exists {
        return nil, fmt.Errorf("device %s not connected", deviceID)
    }
    
    return session.Open()
}

func init() {
    log.SetFormatter(&log.JSONFormatter{})
}

// main starts the SSH server with yamux-based device connections
func main() {
    deviceManager := NewDeviceManager()
    
    // Setup Echo router
    e := echo.New()
    e.HideBanner = true
    
    // WebSocket endpoint for device connections
    e.GET("/ssh/connection", func(c echo.Context) error {
        return handleDeviceConnection(c, deviceManager)
    })
    
    errs := make(chan error)
    
    // Start HTTP server
    go func() {
        errs <- e.Start(ListenAddress)
    }()
    
    // Create tunnel wrapper for device manager
    tunnel := server.NewDeviceManagerTunnel(deviceManager)
    
    // Start SSH server with yamux support
    go func() {
        errs <- server.NewServer(&server.Options{
            ConnectTimeout:               0,
            AllowPublickeyAccessBelow060: false,
        }, tunnel).ListenAndServe()
    }()
    
    if err := <-errs; err != nil {
        log.WithError(err).Fatal("fatal error from HTTP or SSH server")
    }
    
    log.Warn("ssh service is closed")
}

// handleDeviceConnection handles WebSocket upgrade and yamux session creation
func handleDeviceConnection(c echo.Context, dm *DeviceManager) error {
    // Get device ID from header
    deviceID := c.Request().Header.Get("X-Device-ID")
    if deviceID == "" {
        deviceID = c.Request().Header.Get("X-Device-UID") // Fallback
    }
    if deviceID == "" {
        return c.String(http.StatusBadRequest, "missing X-Device-ID header")
    }
    
    // Upgrade to WebSocket
    conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
    if err != nil {
        log.WithError(err).Error("failed to upgrade websocket")
        return err
    }
    defer conn.Close()
    
    // Create yamux session
    wsConn := yamuxws.NewWSConn(conn)
    session, err := yamux.Server(wsConn, yamux.DefaultConfig())
    if err != nil {
        log.WithError(err).Error("failed to create yamux session")
        return err
    }
    defer session.Close()
    
    // Register device
    dm.AddDevice(deviceID, session)
    defer dm.RemoveDevice(deviceID)
    
    // Keep session alive until it closes
    <-session.CloseChan()
    
    return nil
}
