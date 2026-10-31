package main

import (
	"net"
	"nora/internal/transport"
)

type Tunnel struct {
	agentListener  net.Listener
	publicListener net.Listener
}

func NewTunnel(agentPort, publicPort string) (*Tunnel, error) {
	agentListener, err := net.Listen("tcp", ":"+agentPort)
	if err != nil {
		return nil, err
	}

	publicListener, err := net.Listen("tcp", ":"+publicPort)
	if err != nil {
		agentListener.Close()
		return nil, err
	}

	return &Tunnel{
		agentListener:  agentListener,
		publicListener: publicListener,
	}, nil
}

func (t *Tunnel) Serve() error {
	agent, err := t.agentListener.Accept()
	if err != nil {
		return err
	}
	defer agent.Close()

	public, err := t.publicListener.Accept()
	if err != nil {
		return err
	}
	defer public.Close()

	return transport.Relay(agent, public)
}

func (t *Tunnel) Close() {
	t.agentListener.Close()
	t.publicListener.Close()
}
