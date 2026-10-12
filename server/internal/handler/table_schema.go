package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/akshaymundra/ai-text-to-nosql/internal/middleware"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
	"github.com/akshaymundra/ai-text-to-nosql/internal/utils"
)

func (h *TableSchemaHandler) List(w http.ResponseWriter, r *http.Request) {
	schemas, err := h.TableSchemaService.List(r.Context(), middleware.WorkspaceID(r.Context()))
	if err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	utils.JSON(w, http.StatusOK, schemas)
}

func (h *TableSchemaHandler) Get(w http.ResponseWriter, r *http.Request) {
	schema, err := h.TableSchemaService.Get(
		r.Context(),
		middleware.WorkspaceID(r.Context()),
		chi.URLParam(r, "tableName"),
	)
	if err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	utils.JSON(w, http.StatusOK, schema)
}

func (h *TableSchemaHandler) Put(w http.ResponseWriter, r *http.Request) {
	var req types.PutTableSchemaRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	schema, err := h.TableSchemaService.Put(
		r.Context(),
		middleware.WorkspaceID(r.Context()),
		chi.URLParam(r, "tableName"),
		req,
	)
	if err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	utils.JSON(w, http.StatusOK, schema)
}

func (h *TableSchemaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	err := h.TableSchemaService.Delete(
		r.Context(),
		middleware.WorkspaceID(r.Context()),
		chi.URLParam(r, "tableName"),
	)
	if err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
