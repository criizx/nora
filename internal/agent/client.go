package agent

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
	mu         sync.RWMutex

	agents map[string]*Agent
}

func (c *Client) handleIncomingStream(ctx context.Context, stream net.Conn) {
	defer stream.Close()

	var nameLen [1]byte
	if _, err := io.ReadFull(stream, nameLen[:]); err != nil {
		log.Printf("reading agent name length: %v", err)
		return
	}

	size := int(nameLen[0])
	if size == 0 || size > 63 {
		log.Printf("invalid agent name length: %d", size)
		return
	}

	nameBuf := make([]byte, size)
	if _, err := io.ReadFull(stream, nameBuf); err != nil {
		log.Printf("reading agent name: %v", err)
		return
	}

	name := string(nameBuf)
	agent := c.GetAgent(name)
	if agent == nil {
		log.Printf("unknown agent name: %s", name)
		return
	}

	if err := agent.Serve(ctx, stream); err != nil {
		log.Printf("agent [%s] stream error: %v", name, err)
	}
}

func NewClient(acc Account, cfg *Config) (*Client, error) {
	c := &Client{
		serverAddr: cfg.ServerAddr,
		account:    acc,
		agents:     make(map[string]*Agent, len(cfg.Tunnels)),
	}

	for name, tCfg := range cfg.Tunnels {
		if len(name) == 0 || len(name) > 63 {
			return nil, fmt.Errorf("invalid tunnel name length: %s", name)
		}
		localAddr := fmt.Sprintf("127.0.0.1:%d", tCfg.Port)
		c.agents[name] = NewAgent(name, tCfg.Proto, localAddr)
	}
	return c, nil
}

func (c *Client) AddAgent(name, proto string, port int) {
	localAddr := fmt.Sprintf("127.0.0.1:%d", port)
	agent := NewAgent(name, proto, localAddr)

	c.mu.Lock()
	previous := c.agents[name]
	c.agents[name] = agent
	c.mu.Unlock()
	if previous != nil {
		previous.Close()
	}
}

func (c *Client) GetAgent(name string) *Agent {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.agents[name]
}

func (c *Client) RemoveAgent(name string) {
	c.mu.Lock()
	agent := c.agents[name]
	delete(c.agents, name)
	c.mu.Unlock()
	if agent != nil {
		agent.Close()
	}
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

func (c *Client) Run(ctx context.Context) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", c.serverAddr)
	if err != nil {
		return err
	}

	session, err := yamux.Client(conn, nil)
	if err != nil {
		conn.Close()
		return err
	}
	c.session = session
	defer session.Close()

	stop := context.AfterFunc(ctx, func() {
		session.Close()
	})
	defer stop()
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
		return fmt.Errorf("server error: %s", resp.Error)
	}

	for {
		stream, err := c.session.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}

		go c.handleIncomingStream(ctx, stream)
	}
}
