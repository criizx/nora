package main

import (
	"io"
	"log"
	"net"
)

func main() {
	sender, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		log.Fatal(err)
	}
	defer sender.Close()

	log.Println("Connected to server on port 9000")
	receiver, err := net.Dial("tcp", "127.0.0.1:3000")
	if err != nil {
		log.Fatal(err)
	}
	defer receiver.Close()

	go func() {
		log.Println("Forwarding data from receiver to sender")
		_, err := io.Copy(sender, receiver)
		if err != nil {
			log.Println("Error copying from receiver to sender:", err)
		}
		sender.Close()
		receiver.Close()
	}()

	_, err = io.Copy(receiver, sender)
	if err != nil {
		log.Println("Error copying from sender to receiver:", err)
	}
}
