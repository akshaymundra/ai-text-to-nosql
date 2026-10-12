package middleware

import (
	"context"
	"net/http"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
	"github.com/akshaymundra/ai-text-to-nosql/internal/utils"
)

type contextKey string

const workspaceIDKey contextKey = "workspace_id"

// Authenticate rejects every request until JWT verification exists, so protected routes are never open by accident.
func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := apperror.NotImplemented("authentication is not implemented")
		utils.JSONError(w, err.HTTPStatus, err)
	})
}

func WorkspaceID(ctx context.Context) string {
	id, _ := ctx.Value(workspaceIDKey).(string)
	return id
}
