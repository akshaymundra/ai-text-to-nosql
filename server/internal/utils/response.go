package utils

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
)

type Response struct {
	Data  any `json:"data,omitempty"`
	Error any `json:"error,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Data: data})
}

func JSONError(w http.ResponseWriter, status int, errPayload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Error: errPayload})
}

func WriteError(w http.ResponseWriter, r *http.Request, logger *zap.Logger, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		JSONError(w, appErr.HTTPStatus, appErr)
		return
	}

	logger.Error("request failed",
		zap.String("request_id", middleware.GetReqID(r.Context())),
		zap.Error(err),
	)
	internal := apperror.Internal("internal server error")
	JSONError(w, internal.HTTPStatus, internal)
}
