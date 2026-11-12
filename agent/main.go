package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"
)

func main() {
	configPath := flag.String("config", "", "Path to YAML configuration (default: user configuration directory/nora/nora.yml)")
	flag.Parse()

	path := *configPath
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			log.Fatal(err)
		}
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		log.Fatal(err)
	}

	acc, err := Authorize(cfg)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client, err := NewClient(ctx, *acc, cfg)
	if err != nil {
		log.Fatal(err)
	}

	defer client.Close()

	if err := client.Run(); err != nil {
		log.Println("Client stopped with error: " + err.Error())
	}
}
