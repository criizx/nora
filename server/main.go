package main

import (
	"log"
)

func main() {
	firstTunnel, err := NewTunnel("9000", "8080")
	if err != nil {
		log.Fatal(err)
	}
	defer firstTunnel.Close()
	if err := firstTunnel.Serve(); err != nil {
		log.Println(err)
	}
}
