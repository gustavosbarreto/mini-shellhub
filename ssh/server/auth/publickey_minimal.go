package auth

import (
    gliderssh "github.com/gliderlabs/ssh"
    "github.com/shellhub-io/mini-shellhub/ssh/session"
    log "github.com/sirupsen/logrus"
)

// PublicKeyHandler: accept any public key and attempt to auth to agent with dummy password.
// This mirrors the lenient behavior used previously in minimal mode.
func PublicKeyHandler(ctx gliderssh.Context, _ gliderssh.PublicKey) bool {
    logger := log.WithFields(log.Fields{"uid": ctx.SessionID(), "user": ctx.User()})
    sess, state := session.ObtainSession(ctx)
    if state < session.StateEvaluated {
        logger.Trace("session not evaluated yet on public key handler; accepting")
        return true
    }
    if err := sess.Auth(ctx, session.AuthPassword("any")); err != nil {
        logger.WithError(err).Warn("failed to authenticate on agent after pubkey; accepting anyway")
        return true
    }
    logger.Info("accepted public key")
    return true
}
