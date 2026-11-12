package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"nora/internal/protocol"

	"github.com/hashicorp/yamux"
)

type Client struct {
	account    Account
	session    *yamux.Session
	serverAddr string
	ctx        context.Context
	mu         sync.RWMutex

	agents map[string]*Agent
}

func (c *Client) handleIncomingStream(stream net.Conn) {
	defer stream.Close()

	var nameLen [1]byte
	if _, err := io.ReadFull(stream, nameLen[:]); err != nil {
		log.Printf("Failed to read agent name length: %v", err)
		return
	}

	size := int(nameLen[0])
	if size == 0 || size > 63 {
		log.Printf("Invalid agent name length: %d", size)
		return
	}

	nameBuf := make([]byte, size)
	if _, err := io.ReadFull(stream, nameBuf); err != nil {
		log.Printf("Failed to read agent name: %v", err)
		return
	}

	name := string(nameBuf)
	agent := c.GetAgent(name)
	if agent == nil {
		log.Printf("Unknown agent name: %s", name)
		return
	}

	if err := agent.Serve(stream); err != nil {
		log.Printf("Agent [%s] stream error: %v", name, err)
	}
}

func NewClient(ctx context.Context, acc Account, cfg *Config) (*Client, error) {
	c := &Client{
		serverAddr: cfg.ServerAddr,
		account:    acc,
		agents:     make(map[string]*Agent, len(cfg.Tunnels)),
		ctx:        ctx,
	}

	for name, tCfg := range cfg.Tunnels {
		if len(name) == 0 || len(name) > 63 {
			return nil, fmt.Errorf("Invalid tunnel name length: %s", name)
		}
		localAddr := fmt.Sprintf("127.0.0.1:%d", tCfg.Port)
		c.agents[name] = NewAgent(ctx, name, tCfg.Proto, localAddr)
	}
	return c, nil
}

func (c *Client) AddAgent(name, proto string, port int) {
	localAddr := fmt.Sprintf("127.0.0.1:%d", port)
	agent := NewAgent(c.ctx, name, proto, localAddr)

	c.mu.Lock()
	c.agents[name] = agent
	c.mu.Unlock()
}

func (c *Client) GetAgent(name string) *Agent {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.agents[name]
}

func (c *Client) RemoveAgent(name string) {
	c.mu.Lock()
	agent, ok := c.agents[name]
	if ok {
		agent.Close()
		delete(c.agents, name)
	}
	c.mu.Unlock()
}

func (c *Client) Close() error {
	if c.session != nil {
		return c.session.Close()
	}
	return nil
}

func (c *Client) collectTunnels() []protocol.TunnelInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tunnels := make([]protocol.TunnelInfo, 0, len(c.agents))
	for _, agent := range c.agents {
		tunnels = append(tunnels, protocol.TunnelInfo{
			ID:       agent.id,
			Protocol: agent.connType,
		})
	}
	return tunnels
}

func (c *Client) Run() error {
	var d net.Dialer
	conn, err := d.DialContext(c.ctx, "tcp", c.serverAddr)
	if err != nil {
		return err
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		conn.Close()
		return err
	}
	c.session = session

	controlStream, err := c.session.Open()
	if err != nil {
		return err
	}

	req := protocol.AuthRequest{
		Token:   c.account.Token,
		Tunnels: c.collectTunnels(),
	}
	if err := json.NewEncoder(controlStream).Encode(req); err != nil {
		return err
	}

	resp := protocol.AuthResponse{}
	if err := json.NewDecoder(controlStream).Decode(&resp); err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("Server error: %s", resp.Error)
	}
	go func() {
		<-c.ctx.Done()
		c.session.Close()
	}()
	for {
		stream, err := c.session.Accept()
		if err != nil {
			if c.ctx.Err() != nil {
				return nil
			}
			return err
		}

		go c.handleIncomingStream(stream)
	}
}
