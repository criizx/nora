package transport

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

const testTimeout = 3 * time.Second

// tcpPair returns connected endpoints backed by real TCP sockets so FIN and
// CloseWrite behave as they do in production (net.Pipe has no half-close).
func tcpPair(t *testing.T) (*net.TCPConn, *net.TCPConn) {
	t.Helper()
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := listener.SetDeadline(time.Now().Add(testTimeout)); err != nil {
		t.Fatal(err)
	}
	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	server, err := listener.AcceptTCP()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Close() })
	for _, conn := range []*net.TCPConn{client, server} {
		if err := conn.SetDeadline(time.Now().Add(testTimeout)); err != nil {
			t.Fatal(err)
		}
	}
	return client, server
}

func awaitRelay(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(testTimeout):
		t.Fatal("Relay did not finish")
		return nil
	}
}

func TestRelayHalfClosePreservesReverseTraffic(t *testing.T) {
	// Exercise both directions: the responder waits for EOF before replying.
	for _, reverse := range []bool{false, true} {
		name := "forward"
		if reverse {
			name = "reverse"
		}
		t.Run(name, func(t *testing.T) {
			client, left := tcpPair(t)
			service, right := tcpPair(t)
			done := make(chan error, 1)
			go func() { done <- Relay(left, right) }()
			if reverse {
				client, service = service, client
			}
			request := bytes.Repeat([]byte("request\x00"), 16384)
			response := bytes.Repeat([]byte("response\xff"), 16384)
			serviceDone := make(chan error, 1)
			go func() {
				got, err := io.ReadAll(service)
				if err == nil && !bytes.Equal(got, request) {
					err = io.ErrUnexpectedEOF
				}
				if err == nil {
					_, err = io.Copy(service, bytes.NewReader(response))
				}
				if err == nil {
					err = service.CloseWrite()
				}
				serviceDone <- err
			}()
			if _, err := io.Copy(client, bytes.NewReader(request)); err != nil {
				t.Fatal(err)
			}
			if err := client.CloseWrite(); err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(client)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, response) {
				t.Fatalf("response mismatch: got %d bytes, want %d", len(got), len(response))
			}
			if err := awaitRelay(t, serviceDone); err != nil {
				t.Fatalf("service: %v", err)
			}
			if err := awaitRelay(t, done); err != nil {
				t.Fatalf("Relay: %v", err)
			}
		})
	}
}

func TestRelayCopyErrorIsReturnedAndStopsOtherDirection(t *testing.T) {
	_, left := tcpPair(t)
	_, right := tcpPair(t)
	// A closed TCP socket guarantees a copy error without relying on OS reset timing.
	left.Close()
	done := make(chan error, 1)
	go func() { done <- Relay(left, right) }()
	if err := awaitRelay(t, done); err == nil {
		t.Fatal("Relay returned nil after copying through a closed socket")
	}
	// The other direction must be interrupted, not left waiting for its peer.
	if _, err := right.Write([]byte("x")); err == nil {
		t.Fatal("other relay connection is still writable after failure")
	}
}

func TestRelayRejectsNonTCPConnections(t *testing.T) {
	for _, invalidSender := range []bool{true, false} {
		name := "receiver"
		if invalidSender {
			name = "sender"
		}
		t.Run(name, func(t *testing.T) {
			pipe, peer := net.Pipe()
			defer pipe.Close()
			defer peer.Close()
			_, tcp := tcpPair(t)
			var sender, receiver net.Conn = tcp, pipe
			if invalidSender {
				sender, receiver = pipe, tcp
			}
			if err := Relay(sender, receiver); err == nil {
				t.Fatal("expected non-TCP error")
			}
			if _, err := tcp.Write([]byte("x")); err == nil {
				t.Fatal("TCP connection was not closed")
			}
			if _, err := peer.Read(make([]byte, 1)); err != io.EOF {
				t.Fatalf("pipe peer: got %v, want EOF", err)
			}
		})
	}
}
