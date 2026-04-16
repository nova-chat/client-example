package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"novaclient/client"
)

func main() {
	cli, err := client.NewClient("localhost:7777")
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer cli.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cli.Run(ctx)

	if err := cli.DHHandshake(); err != nil {
		log.Fatalf("dh handshake: %v", err)
	}

	<-ctx.Done()
}
