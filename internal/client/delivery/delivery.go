package delivery

import "fmt"

// Delivery binds user-facing handlers (CLI, future TUI, etc.) to auth, vault, and sync use cases.
type Delivery struct {
	auth   AuthCommands
	secret SecretCommands
	sync   SyncCommands
}

// New requires all three ports; wiring lives outside this package (e.g. cmd/client).
func New(auth AuthCommands, secret SecretCommands, sync SyncCommands) (*Delivery, error) {
	if auth == nil || secret == nil || sync == nil {
		return nil, fmt.Errorf("delivery.New: auth, secret, and sync are required")
	}
	return &Delivery{auth: auth, secret: secret, sync: sync}, nil
}
