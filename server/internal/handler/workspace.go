package handler

import (
	"net/http"

	"github.com/akshaymundra/ai-text-to-nosql/internal/middleware"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
	"github.com/akshaymundra/ai-text-to-nosql/internal/utils"
)

func (h *WorkspaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	workspace, err := h.WorkspaceService.Get(r.Context(), middleware.WorkspaceID(r.Context()))
	if err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	utils.JSON(w, http.StatusOK, workspace)
}

func (h *WorkspaceHandler) UpdateConnection(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateConnectionRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	workspace, err := h.WorkspaceService.UpdateConnection(r.Context(), middleware.WorkspaceID(r.Context()), req)
	if err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	utils.JSON(w, http.StatusOK, workspace)
}
