# Nora

A self-hosted alternative to ngrok, written in Go. The goal is to expose local services to the internet through a public server and an agent running on your machine.

## Status

Early prototype: forwards TCP traffic through a single agent connection. Currently supports one visitor connection per run, with hardcoded ports and no authentication or encryption.

## Planned architecture

```mermaid
flowchart TB
    USERS["Visitors / TCP clients"]

    subgraph SERVER["Public server · Nora"]
        ENTRY["HTTP / HTTPS endpoints<br/>TCP ports"]
        ROUTER["Tunnel routing"]
        REGISTRY["Agent sessions<br/>Tunnel registry"]

        ENTRY --> ROUTER
        ROUTER --- REGISTRY
    end

    subgraph CONNECTION["TLS connection · Multiplexed"]
        CONTROL["Control stream<br/>Authentication · Tunnel management"]
        DATA["Data streams<br/>One per visitor connection"]
    end

    subgraph DEVICE["User device · Nora agent"]
        AGENT["Session management<br/>Configuration · Reconnection"]
        FORWARD["Local forwarding"]
        WEB["localhost:3000<br/>Web app"]
        SSH["localhost:22<br/>SSH"]

        AGENT --- FORWARD
        FORWARD <--> WEB
        FORWARD <--> SSH
    end

    USERS <--> ENTRY
    REGISTRY <--> CONTROL
    CONTROL <--> AGENT
    ROUTER <--> DATA
    DATA <--> FORWARD
```

One agent maintains one outbound TLS connection to the server. Multiple tunnels and visitor connections share it through separate logical streams.

## Planned

- Multiple tunnels and concurrent connections
- An API for creating and managing tunnels
- Custom subdomains for HTTP services
- Agent authentication and TLS
- Automatic reconnection
