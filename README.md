# tailscale-client-go

[![Go Reference](https://pkg.go.dev/badge/github.com/tailscale/tailscale-client-go.svg)](https://pkg.go.dev/github.com/tailscale/tailscale-client-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/tailscale/tailscale-client-go)](https://goreportcard.com/report/github.com/tailscale/tailscale-client-go)
![Github Actions](https://github.com/tailscale/tailscale-client-go/actions/workflows/ci.yml/badge.svg?branch=master)


A client implementation for the [Tailscale](https://tailscale.com) HTTP API.
For more details, please see [API documentation](https://github.com/tailscale/tailscale/blob/main/api.md).

A [V2](v2) is currently under active development, but has not yet been released. Use at your own risk.

# Example

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/tailscale/tailscale-client-go/tailscale"
)

func main() {
	apiKey := os.Getenv("TAILSCALE_API_KEY")
	tailnet := os.Getenv("TAILSCALE_TAILNET")

	client, err := tailscale.NewClient(apiKey, tailnet)
	if err != nil {
		log.Fatalln(err)
	}

	// List all your devices
	devices, err := client.Devices(context.Background())
}
```
