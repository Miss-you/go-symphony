package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Miss-you/go-symphony/internal/cli"
)

var run = cli.Main

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	code := run(ctx, os.Args[1:])
	slog.Info("symphony exiting", "code", code)
	os.Exit(code)
}
