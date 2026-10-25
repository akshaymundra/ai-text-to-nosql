package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/config"
	"github.com/akshaymundra/ai-text-to-nosql/internal/routes"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
	"go.uber.org/zap"
)

type App struct {
	config *config.Config
	router http.Handler
	logger *zap.Logger
	db     *pgxpool.Pool
}

func NewApp(ctx context.Context, cfg *config.Config) (*App, error) {
	logger, _ := zap.NewProduction()

	db, err := newPostgresPool(ctx, cfg.DatabaseURL, cfg.DBPingTimeout)
	if err != nil {
		return nil, err
	}

	app := &App{
		config: cfg,
		logger: logger,
		db:     db,
	}

	app.router = routes.NewRoutes(logger, db, cfg)

	return app, nil
}

func (a *App) Close() {
	a.db.Close()
}

func newPostgresPool(ctx context.Context, url string, pingTimeout time.Duration) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parsing postgres url: %w", err)
	}
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("creating postgres pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}

	return pool, nil
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
