# tailscale-client-go/v2

[![Go Reference](https://pkg.go.dev/badge/github.com/tailscale/tailscale-client-go/v2.svg)](https://pkg.go.dev/github.com/tailscale/tailscale-client-go/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/tailscale/tailscale-client-go/v2)](https://goreportcard.com/report/github.com/tailscale/tailscale-client-go/v2)
![Github Actions](https://github.com/tailscale/tailscale-client-go/actions/workflows/ci.yml/badge.svg?branch=main)

A client implementation for the [Tailscale](https://tailscale.com) HTTP API.
For more details, please see [API documentation](https://tailscale.com/api).

## Example (Using API Key)

```go
package main

import (
	"context"
	"log"
	"os"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func main() {
	apiKey := os.Getenv("TAILSCALE_API_KEY")
	tailnet := os.Getenv("TAILSCALE_TAILNET")

	&tsclient.Client{
		APIKey:    apiKey,
		Tailnet:   tailnet,
	}

	devices, err := client.Devices().List(context.Background())
}
```

## Example (Using OAuth)

```go
package main

import (
	"context"
	"log"
	"os"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
)

func main() {
	oauthClientID := os.Getenv("TAILSCALE_OAUTH_CLIENT_ID")
	tailnet := os.Getenv("TAILSCALE_OAUTH_CLIENT_SECRET")
	oauthScopes := []string{"all:write"}

	&tsclient.Client{
		APIKey:    apiKey,
		Tailnet:   tailnet,
	}
	clientV2.UseOAuth(oauthClientID, oauthClientSecret, oauthScopes)

	devices, err := client.Devices().List(context.Background())
}
```
