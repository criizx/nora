package main

import (
	"net"
	"nora/internal/transport"
	"context"
)

type Agent struct {
	id        string
	connType  string
	localAddr string
	status    string
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewAgent(parentCtx context.Context, id, connType, localAddr string) *Agent {
	ctx, cancel := context.WithCancel(parentCtx)
	return &Agent{
		id:        id,
		connType:  connType,
		localAddr: localAddr,
		status:    "active",
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (a *Agent) Serve(stream net.Conn) error {
	defer stream.Close()

	var d net.Dialer
	receiver, err := d.DialContext(a.ctx, "tcp", a.localAddr)
	if err != nil {
		return err
	}
	defer receiver.Close()

	return transport.Relay(stream, receiver)
}

func (a *Agent) Close() {
	a.status = "closed"
	a.cancel()
}