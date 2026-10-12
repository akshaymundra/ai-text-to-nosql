package handler

import (
	"net/http"

	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
	"github.com/akshaymundra/ai-text-to-nosql/internal/utils"
)

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req types.RegisterRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}
	if err := req.Validate(); err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	token, err := h.AuthService.Register(r.Context(), req)
	if err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	utils.JSON(w, http.StatusCreated, token)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req types.LoginRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}
	if err := req.Validate(); err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	token, err := h.AuthService.Login(r.Context(), req)
	if err != nil {
		utils.WriteError(w, r, h.Logger, err)
		return
	}

	utils.JSON(w, http.StatusOK, token)
}
