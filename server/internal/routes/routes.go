package routes

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/akshaymundra/ai-text-to-nosql/internal/config"
	appmiddleware "github.com/akshaymundra/ai-text-to-nosql/internal/middleware"
	"github.com/akshaymundra/ai-text-to-nosql/internal/repo"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service"
	"github.com/akshaymundra/ai-text-to-nosql/internal/utils"
)

func NewRoutes(logger *zap.Logger, db *pgxpool.Pool, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(utils.RequestLogger(logger))

	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}).Handler)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	r.Route("/api/v1", registerRoutes(logger, db, cfg))

	return r
}

func registerRoutes(logger *zap.Logger, db *pgxpool.Pool, cfg *config.Config) func(router chi.Router) {
	allowLocalConnection := cfg.AppEnv == config.EnvDevelopment
	authService := service.NewAuthService(
		repo.NewWorkspaceRepo(db),
		cfg.JWTSecret,
		cfg.JWTTTL,
		allowLocalConnection,
	)

	return func(r chi.Router) {
		r.Route("/auth", AuthRoutes(authService, logger))

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.Authenticate(authService, logger))
			r.Route("/workspace", WorkspaceRoutes(logger, allowLocalConnection))
			r.Route("/schemas", TableSchemaRoutes(logger))
			r.Route("/queries", QueryRoutes(logger))
		})
	}
}
