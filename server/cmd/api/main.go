package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/akshaymundra/ai-text-to-nosql/internal/config"
)

func main() {
	config.Init()
	cfg := config.GetInstance()

	app := NewApp(cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := app.Start(ctx); err != nil {
		fmt.Println("failed to start app:", err)
		os.Exit(1)
	}
}
