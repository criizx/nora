package transport

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/hashicorp/yamux"
)

func copyOneWay(dst, src net.Conn) error {
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	switch conn := dst.(type) {
	case interface{ CloseWrite() error }:
		return conn.CloseWrite()
	case *yamux.Stream:
		// yamux Close sends FIN but still permits reads from the stream.
		return conn.Close()
	default:
		return fmt.Errorf("connection %T does not support half-close", dst)
	}
}

// Relay waits for both directions. The caller owns final connection cleanup.
// On failure it interrupts both directions, including reads on yamux streams.
func Relay(sender, receiver net.Conn) error {
	results := make(chan error, 2)
	go func() { results <- copyOneWay(receiver, sender) }()
	go func() { results <- copyOneWay(sender, receiver) }()
	first := <-results
	if first != nil {
		for _, conn := range []net.Conn{sender, receiver} {
			// yamux Close alone allows reads to continue, so expire pending I/O too.
			conn.SetDeadline(time.Now())
			conn.Close()
		}
	}
	second := <-results
	if first != nil {
		return first
	}
	return second
}
