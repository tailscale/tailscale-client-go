package main

import (
	"context"
	"log"
	"os"

	"github.com/tailscale/tailscale-client-go/tailscale"
)

func main() {
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	ctx := context.Background()

	client, err := tailscale.NewClient("", "-", tailscale.WithClientCredentials(ctx, clientID, clientSecret, nil))
	if err != nil {
		log.Fatal(err)
	}

	devices, err := client.Devices(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for _, device := range devices {
		log.Println(device.Hostname)
	}
}
