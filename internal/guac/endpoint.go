package guac

import (
	"context"
	"net"
	"path/filepath"
)

// Packaged engines use a filesystem socket. TCP is used by the local development backend.
func DialContext(ctx context.Context, address string) (net.Conn, error) {
	network := "tcp"
	if filepath.IsAbs(address) {
		network = "unix"
	}
	return (&net.Dialer{}).DialContext(ctx, network, address)
}
