package transport

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/hashicorp/yamux"
)

func TestRelayYamuxHalfClose(t *testing.T) {
	wireA, wireB := net.Pipe()
	client, err := yamux.Client(wireA, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, err := yamux.Server(wireB, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	stream, err := client.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	incoming, err := server.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer incoming.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	app, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	target, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	deadline := time.Now().Add(3 * time.Second)
	stream.SetDeadline(deadline)
	incoming.SetDeadline(deadline)
	app.SetDeadline(deadline)
	target.SetDeadline(deadline)
	done := make(chan error, 1)
	go func() { done <- Relay(incoming, target) }()
	go func() {
		// Read until the request FIN, then reply in the reverse direction.
		data, readErr := io.ReadAll(app)
		if readErr == nil {
			app.Write(data)
			app.(*net.TCPConn).CloseWrite()
		}
	}()
	if _, err := stream.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	response, err := io.ReadAll(stream)
	if err != nil {
		t.Fatal(err)
	}
	if string(response) != "hello" {
		t.Fatalf("got %q", response)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("Relay stuck after EOF")
	}
}
