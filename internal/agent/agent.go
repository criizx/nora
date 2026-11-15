package agent

import (
	"context"
	"net"
	"sync"
	"time"

	"nora/internal/transport"
)

type Agent struct {
	id        string
	connType  string
	localAddr string
	mu        sync.Mutex
	closed    bool
	nextID    uint64
	active    map[uint64]context.CancelFunc
}

func NewAgent(id, connType, localAddr string) *Agent {
	return &Agent{
		id:        id,
		connType:  connType,
		localAddr: localAddr,
		active:    make(map[uint64]context.CancelFunc),
	}
}

func (a *Agent) Serve(ctx context.Context, stream net.Conn) error {
	defer stream.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return net.ErrClosed
	}
	id := a.nextID
	a.nextID++
	a.active[id] = cancel
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		delete(a.active, id)
		a.mu.Unlock()
	}()

	var d net.Dialer
	receiver, err := d.DialContext(ctx, "tcp", a.localAddr)
	if err != nil {
		return err
	}
	defer receiver.Close()
	stop := context.AfterFunc(ctx, func() {
		// A yamux stream Close sends FIN but may leave reads blocked.
		stream.SetDeadline(time.Now())
		receiver.SetDeadline(time.Now())
		stream.Close()
		receiver.Close()
	})
	defer stop()

	return transport.Relay(stream, receiver)
}

// Close prevents new operations and cancels all active operations.
func (a *Agent) Close() {
	a.mu.Lock()
	a.closed = true
	cancels := make([]context.CancelFunc, 0, len(a.active))
	for _, cancel := range a.active {
		cancels = append(cancels, cancel)
	}
	a.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}
