package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/config"
	"github.com/akshaymundra/ai-text-to-nosql/internal/routes"
	"go.uber.org/zap"
)

type App struct {
	config *config.Config
	router http.Handler
	logger *zap.Logger
}

func NewApp(cfg *config.Config) *App {
	logger, _ := zap.NewProduction()

	app := &App{
		config: cfg,
		logger: logger,
	}

	app.router = routes.NewRoutes(logger)

	return app
}

func (a *App) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", a.config.ServerPort),
		Handler:      a.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	a.logger.Info("starting server", zap.String("port", a.config.ServerPort))

	ch := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ch <- fmt.Errorf("server error: %w", err)
		}
		close(ch)
	}()

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	}
}
