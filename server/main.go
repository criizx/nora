package main

import (
	"log"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server, err := NewServer(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	
	defer server.Close()
	
	if err := server.Run(); err != nil {
		log.Println("Server stopped" + err)
	}
}
