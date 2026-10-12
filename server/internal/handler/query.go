package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/akshaymundra/ai-text-to-nosql/internal/middleware"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
	"github.com/akshaymundra/ai-text-to-nosql/internal/utils"
)

func (h *QueryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req types.CreateQueryRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	query, err := h.QueryService.Create(r.Context(), middleware.WorkspaceID(r.Context()), req)
	if err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	utils.JSON(w, http.StatusCreated, query)
}

func (h *QueryHandler) List(w http.ResponseWriter, r *http.Request) {
	queries, err := h.QueryService.List(r.Context(), middleware.WorkspaceID(r.Context()))
	if err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	utils.JSON(w, http.StatusOK, queries)
}

func (h *QueryHandler) Get(w http.ResponseWriter, r *http.Request) {
	query, err := h.QueryService.Get(r.Context(), middleware.WorkspaceID(r.Context()), chi.URLParam(r, "queryID"))
	if err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	utils.JSON(w, http.StatusOK, query)
}

func (h *QueryHandler) Execute(w http.ResponseWriter, r *http.Request) {
	var req types.ExecuteQueryRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	result, err := h.QueryService.Execute(
		r.Context(),
		middleware.WorkspaceID(r.Context()),
		chi.URLParam(r, "queryID"),
		req,
	)
	if err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	utils.JSON(w, http.StatusOK, result)
}

func (h *QueryHandler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	executions, err := h.QueryService.ListExecutions(
		r.Context(),
		middleware.WorkspaceID(r.Context()),
		chi.URLParam(r, "queryID"),
	)
	if err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	utils.JSON(w, http.StatusOK, executions)
}
