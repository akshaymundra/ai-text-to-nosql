package handler

import "github.com/akshaymundra/ai-text-to-nosql/internal/service"

type QueryHandler struct {
	QueryService *service.QueryService
}

func NewQueryHandler(svc *service.QueryService) *QueryHandler {
	return &QueryHandler{
		QueryService: svc,
	}
}
