package main

import (
	"log"
)

func main() {
	agent, err := NewAgent("9000", "3000")
	if err != nil {
		log.Fatal(err)
	}
	defer Close(agent)
	if err := agent.Serve(); err != nil {
		log.Println(err)
	}
}
