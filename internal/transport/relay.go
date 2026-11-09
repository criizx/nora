package transport

import (
	"errors"
	"io"
	"net"
	"sync"
)

func copyOneWay(dst, src *net.TCPConn) error {
	_, err := io.CopyBuffer(dst, src, make([]byte, 1024))
	if err != nil {
		return err
	}

	err = dst.CloseWrite()
	if err != nil {
		return err
	}

	return nil
}

func Relay(sender, receiver net.Conn) error {
	wg := sync.WaitGroup{}

	senderTcpConn, ok := sender.(*net.TCPConn)
	if !ok {
		sender.Close()
		receiver.Close()
		return errors.New("sender is not a TCP connection")
	}
	receiverTcpConn, ok := receiver.(*net.TCPConn)

	if !ok {
		sender.Close()
		receiver.Close()
		return errors.New("receiver is not a TCP connection")
	}

	wg.Go(func() {
		err := copyOneWay(receiverTcpConn, senderTcpConn)
		if err != nil {
			receiver.Close()
			sender.Close()
			return
		}
	})
	wg.Go(func() {
		err := copyOneWay(senderTcpConn, receiverTcpConn)
		if err != nil {
			receiver.Close()
			sender.Close()
			return
		}
	})
	wg.Wait()
	return nil
}
