package routes

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"

	appmiddleware "github.com/akshaymundra/ai-text-to-nosql/internal/middleware"
	"github.com/akshaymundra/ai-text-to-nosql/internal/utils"
)

func NewRoutes(logger *zap.Logger) http.Handler {
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

	r.Route("/api/v1", registerRoutes(logger))

	return r
}

func registerRoutes(logger *zap.Logger) func(router chi.Router) {
	return func(r chi.Router) {
		r.Route("/auth", AuthRoutes(logger))

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.Authenticate)
			r.Route("/workspace", WorkspaceRoutes(logger))
			r.Route("/schemas", TableSchemaRoutes(logger))
			r.Route("/queries", QueryRoutes(logger))
		})
	}
}
