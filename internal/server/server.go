package server

import "errors"

// ErrNotImplemented reports that the multiplexed server is still under development.
var ErrNotImplemented = errors.New("nora server is not implemented yet: authentication and yamux session handling are required")

// Run starts the server once its implementation is available.
func Run() error {
	return ErrNotImplemented
}
