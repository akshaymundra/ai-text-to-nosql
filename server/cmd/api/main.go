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
	if err := config.Validate(); err != nil {
		fmt.Println("invalid config:", err)
		os.Exit(1)
	}
	cfg := config.GetInstance()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	app, err := NewApp(ctx, cfg)
	if err != nil {
		fmt.Println("failed to init app:", err)
		os.Exit(1)
	}
	defer app.Close()

	if err := app.Start(ctx); err != nil {
		fmt.Println("failed to start app:", err)
		os.Exit(1)
	}
}
