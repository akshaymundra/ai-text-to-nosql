package routes

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/akshaymundra/ai-text-to-nosql/internal/handler"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service"
)

func AuthRoutes(logger *zap.Logger) func(router chi.Router) {
	svc := service.NewAuthService()
	h := handler.NewAuthHandler(svc, logger)
	return func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
	}
}
