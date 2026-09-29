package main

import (
	"context"
	"fmt"
	"github.com/shoshindev/shoshinvim/internal/cli"
	"os"
	"os/signal"
)

var version = "0.1.0-dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := cli.New(version).ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "shoshinvim:", err)
		os.Exit(1)
	}
}
