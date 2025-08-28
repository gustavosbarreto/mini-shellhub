package server

import (
    gliderssh "github.com/gliderlabs/ssh"
)

func (s *Server) passwordHandler(ctx gliderssh.Context, pass string) bool {
    // Accept any password for testing
    _ = pass
    return true
}

func (s *Server) publicKeyHandler(ctx gliderssh.Context, key gliderssh.PublicKey) bool {
    // Accept any public key for testing
    _ = key
    return true
}
