package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Miss-you/go-symphony/internal/cli"
)

var run = cli.Main

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:]))
}
