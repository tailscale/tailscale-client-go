# tailscale-client-go

[![Go Reference](https://pkg.go.dev/badge/github.com/davidsbond/tailscale-client-go.svg)](https://pkg.go.dev/github.com/davidsbond/tailscale-client-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/davidsbond/tailscale-client-go)](https://goreportcard.com/report/github.com/davidsbond/tailscale-client-go)
![Github Actions](https://github.com/davidsbond/tailscale-client-go/actions/workflows/ci.yml/badge.svg?branch=master)


A client implementation for the [Tailscale](https://tailscale.com) HTTP API

# Example

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/davidsbond/tailscale-client-go/tailscale"
)

func main() {
	apiKey := os.GetEnv("TAILSCALE_API_KEY")
	tailnet := os.GetEnv("TAILSCALE_TAILNET")

	client, err := tailscale.NewClient(apiKey, tailnet)
	if err != nil {
		log.Fatalln(err)
	}

	// List all your devices
	devices, err := client.Devices(context.Background())
}
```
