package host

import (
	gliderssh "github.com/gliderlabs/ssh"
	"github.com/shellhub-io/mini-shellhub/agent/pkg/agent/server/modes"
	log "github.com/sirupsen/logrus"
)

// NOTICE: Ensures the Authenticator interface is implemented.
var _ modes.Authenticator = (*Authenticator)(nil)

// Authenticator implements the Authenticator interface when the server is running in host mode.
type Authenticator struct {
	// singleUserPassword is the password of the single user.
	// When it is empty, it means that the single user is disabled.
	singleUserPassword string
	// deviceName is the device name.
	//
	// NOTICE: Uses a pointer for later assignment.
	deviceName *string
}

// NewAuthenticator creates a new instance of Authenticator for the host mode.
func NewAuthenticator(api interface{}, authData interface{}, singleUserPassword string, deviceName *string) *Authenticator {
	return &Authenticator{
		singleUserPassword: singleUserPassword,
		deviceName:         deviceName,
	}
}

// Password handles the server's SSH password authentication when server is running in host mode.
func (a *Authenticator) Password(ctx gliderssh.Context, _ string, pass string) bool {
	log := log.WithFields(log.Fields{
		"user": ctx.User(),
	})
	
	// For mini-shellhub, accept any password for simplicity
	ok := true
	if a.singleUserPassword != "" {
		ok = (pass == a.singleUserPassword)
	}

	if ok {
		log.Info("Using password authentication")
	} else {
		log.Info("Failed to authenticate using password")
	}

	return ok
}

// PublicKey handles the server's SSH public key authentication when server is running in host mode.
func (a *Authenticator) PublicKey(ctx gliderssh.Context, _ string, key gliderssh.PublicKey) bool {
	// For mini-shellhub, accept any public key for simplicity
	if key == nil {
		return false
	}

	log.WithFields(
		log.Fields{
			"username": ctx.User(),
		},
	).Info("using public key authentication")

	return true
}
