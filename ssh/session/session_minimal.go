package session

import (
    "errors"
    "fmt"
    "net"
    "net/http"
    "time"
    "strings"

    gliderssh "github.com/gliderlabs/ssh"
    "github.com/shellhub-io/shellhub/pkg/httptunnel"
    "github.com/shellhub-io/shellhub/pkg/models"
    "github.com/shellhub-io/mini-shellhub/ssh/pkg/host"
    "github.com/shellhub-io/mini-shellhub/ssh/pkg/target"
    gossh "golang.org/x/crypto/ssh"
)

// Data holds minimal metadata used by channel handlers and logging.
type Data struct {
    Target    *target.Target
    SSHID     string
    Device    *models.Device
    Namespace *models.Namespace
    IPAddress string
    Type      string
    Term      string
    Handled   bool
}

// AgentChannel represents a channel open between agent and server.
type AgentChannel struct {
    Channel  gossh.Channel
    Requests <-chan *gossh.Request
}

func (a *AgentChannel) Close() error { return a.Channel.Close() }

// Agent represents a connection to an agent.
type Agent struct {
    Conn     net.Conn
    Client   *gossh.Client
    Requests <-chan *gossh.Request
    Channels map[int]*AgentChannel
}

// ClientChannel represents a channel open between client and server.
type ClientChannel struct {
    Channel  gossh.Channel
    Requests <-chan *gossh.Request
}

func (c *ClientChannel) Close() error { return c.Channel.Close() }

// Client represents a connection to a client.
type Client struct {
    Channels map[int]*ClientChannel
}

// Seats control.
type Seat struct{ HasPty bool }
type Seats struct{ next int }

func NewSeats() Seats { return Seats{} }
func (s *Seats) NewSeat() (int, error) { id := s.next; s.next++; return id, nil }
func (s *Seats) SetPty(int, bool)      {}
func (s *Seats) Get(int) (*Seat, bool) { return &Seat{}, true }

// lightweight net.Conn wrapper interface to ease testing.
// helper to clear deadlines when needed
func clearReadDeadline(c net.Conn) error { return c.SetReadDeadline(time.Time{}) }

// Session is a minimal session used only to bridge SSH client <-> agent.
type Session struct {
    UID    string
    Agent  *Agent
    Client *Client

    tunnel *httptunnel.Tunnel

    Seats Seats
    Data  // embed to promote fields (SSHID, Device, Target, IPAddress, Type, ...)
}

// NewSession creates a new minimal session without API or cache.
func NewSession(ctx gliderssh.Context, tunnel *httptunnel.Tunnel) (*Session, error) {
    sshid := ctx.User()

    hos, err := host.NewHost(ctx.RemoteAddr().String())
    if err != nil {
        return nil, ErrHost
    }

    tgt, err := target.NewTarget(sshid)
    if err != nil {
        return nil, err
    }

    // In minimal mode, treat target.Data as device ID directly.
    deviceID := tgt.Data

    sess := &Session{
        UID:    ctx.SessionID(),
        tunnel: tunnel,
        Agent:  &Agent{Channels: make(map[int]*AgentChannel)},
        Client: &Client{Channels: make(map[int]*ClientChannel)},
        Seats:  NewSeats(),
        Data: Data{
            SSHID:    sshid,
            Target:   tgt,
            IPAddress: hos.Host,
            Device: &models.Device{
                UID:  deviceID,
                Name: deviceID,
                Info: &models.DeviceInfo{Version: "v0.9.3"},
            },
            Namespace: &models.Namespace{},
        },
    }

    snap := getSnapshot(ctx)
    snap.save(sess, StateCreated)

    return sess, nil
}

// Dial establishes a raw tunnel connection to the agent using the device ID.
func (s *Session) Dial(ctx gliderssh.Context) error {
    id := s.Data.Device.UID
    if !strings.Contains(id, ":") {
        id = "default:" + id
    }
    ctx.Lock()
    conn, err := s.tunnel.Dial(ctx, id)
    if err != nil {
        ctx.Unlock()
        return errors.Join(ErrDial, err)
    }
    req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/ssh/%s", s.UID), nil)
    if err := req.Write(conn); err != nil {
        ctx.Unlock()
        return err
    }
    s.Agent.Conn = conn
    ctx.Unlock()
    return nil
}

// Evaluate does nothing in minimal mode.
func (s *Session) Evaluate(ctx gliderssh.Context) error {
    snap := getSnapshot(ctx)
    snap.save(s, StateEvaluated)
    return nil
}

// Auth authenticates to the agent using the provided method and wires a client.
func (s *Session) Auth(ctx gliderssh.Context, auth Auth) error {
    snap := getSnapshot(ctx)
    sess, state := snap.retrieve()
    if state != StateEvaluated && state != StateRegistered {
        return errors.New("invalid session state")
    }
    cfg := &gossh.ClientConfig{
        User:            sess.Data.Target.Username,
        HostKeyCallback: gossh.InsecureIgnoreHostKey(), //nolint:gosec
    }
    if err := auth.Auth()(sess, cfg); err != nil {
        return err
    }
    if sess.Agent.Conn == nil {
        if err := sess.Dial(ctx); err != nil {
            return err
        }
    }
    conn, chans, reqs, err := gossh.NewClientConn(sess.Agent.Conn, "tcp", cfg)
    if err != nil {
        // reset so future attempts can redial
        sess.Agent.Conn = nil
        return err
    }
    ch := make(chan *gossh.Request)
    close(ch)
    sess.Agent.Client = gossh.NewClient(conn, chans, ch)
    sess.Agent.Requests = reqs

    snap.save(sess, StateFinished)
    return nil
}

// NewClientChannel accepts a new channel from a client and set a seat for it.
func (s *Session) NewClientChannel(newChannel gossh.NewChannel, seat int) (*ClientChannel, error) {
    if _, ok := s.Client.Channels[seat]; ok {
        return nil, ErrSeatAlreadySet
    }
    channel, requests, err := newChannel.Accept()
    if err != nil {
        return nil, err
    }
    c := &ClientChannel{Channel: channel, Requests: requests}
    s.Client.Channels[seat] = c
    return c, nil
}

// NewAgentChannel opens a new channel to agent and set a seat for it.
func (s *Session) NewAgentChannel(name string, seat int) (*AgentChannel, error) {
    if _, ok := s.Agent.Channels[seat]; ok {
        return nil, ErrSeatAlreadySet
    }
    if s.Agent == nil || s.Agent.Client == nil {
        return nil, errors.New("agent client not established")
    }
    channel, requests, err := s.Agent.Client.OpenChannel(name, nil)
    if err != nil {
        return nil, err
    }
    a := &AgentChannel{Channel: channel, Requests: requests}
    s.Agent.Channels[seat] = a
    return a, nil
}

// NewSeat delegates to Seats.NewSeat for channel handlers compatibility.
func (s *Session) NewSeat() (int, error) { return s.Seats.NewSeat() }

// KeepAlive is a no-op in minimal mode.
func (s *Session) KeepAlive() error { return nil }

// Event is a no-op in minimal mode.
func (s *Session) Event(string, any, int) {}

// Recorded is a no-op in minimal mode.
func (s *Session) Recorded(int) error { return nil }

// Event is a generic free function used by channel handlers; no-op here.
func Event[D any](_ *Session, _ string, _ []byte, _ int) {}

// Announce is a no-op in minimal mode.
func (s *Session) Announce(gossh.Channel) error { return nil }

// Finish closes server->agent side politely.
func (s *Session) Finish() error {
    if s.Agent != nil && s.Agent.Conn != nil {
        req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/ssh/close/%s", s.UID), nil)
        _ = req.Write(s.Agent.Conn)
    }
    return nil
}
