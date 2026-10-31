package main

import (
	"net"
	"nora/internal/transport"
	"os"

	"github.com/joho/godotenv"
)

type Agent struct {
	sender   net.Conn
	receiver net.Conn
}

func NewAgent(serverPort, receiverPort string) (*Agent, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}
	serverAddress := os.Getenv("SERVER_ADDRESS")
	sender, err := net.Dial("tcp", serverAddress+":"+serverPort)
	if err != nil {
		return nil, err
	}

	receiver, err := net.Dial("tcp", "127.0.0.1"+":"+receiverPort)
	if err != nil {
		sender.Close()
		return nil, err
	}

	return &Agent{
		sender:   sender,
		receiver: receiver,
	}, nil
}

func (a *Agent) Serve() error {
	return transport.Relay(a.sender, a.receiver)
}

func Close(agent *Agent) {
	agent.sender.Close()
	agent.receiver.Close()
}
