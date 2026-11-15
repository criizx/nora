package transport

import (
	"io"
	"net"
	"sync"
)

func copyOneWay(dst, src net.Conn) error {
	if _, err := io.CopyBuffer(dst, src, make([]byte, 1024)); err != nil {
		return err
	}

	if cw, ok := dst.(interface{ CloseWrite() error }); ok {
		return cw.CloseWrite()
	}

	return nil
}

func Relay(sender, receiver net.Conn) error {
	wg := sync.WaitGroup{}

	wg.Go(func() {
		if err := copyOneWay(receiver, sender); err != nil {
			receiver.Close()
			sender.Close()
			return
		}
	})
	wg.Go(func() {
		if err := copyOneWay(sender, receiver); err != nil {
			receiver.Close()
			sender.Close()
			return
		}
	})
	wg.Wait()
	return nil
}
