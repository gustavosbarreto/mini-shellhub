package session

import (
    gossh "golang.org/x/crypto/ssh"
)

type authFunc func(*Session, *gossh.ClientConfig) error

type authMethod int8

const (
    AuthMethodPassword authMethod = iota
)

type Auth interface {
    Method() authMethod
    Auth() authFunc
    Evaluate(*Session) error
}

type passwordAuth struct{ pwd string }

func AuthPassword(pwd string) Auth { return &passwordAuth{pwd: pwd} }
func (*passwordAuth) Method() authMethod { return AuthMethodPassword }
func (p *passwordAuth) Auth() authFunc {
    return func(_ *Session, cfg *gossh.ClientConfig) error {
        cfg.Auth = []gossh.AuthMethod{gossh.Password(p.pwd)}
        return nil
    }
}
func (*passwordAuth) Evaluate(*Session) error { return nil }
