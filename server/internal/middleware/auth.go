package middleware

import (
	"context"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service"
	"github.com/akshaymundra/ai-text-to-nosql/internal/utils"
)

type contextKey string

const (
	workspaceIDKey contextKey = "workspace_id"
	bearerPrefix              = "Bearer "
)

func Authenticate(authService *service.AuthService, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), bearerPrefix)
			if !ok || token == "" {
				err := apperror.Unauthorized("missing bearer token")
				utils.JSONError(w, err.HTTPStatus, err)
				return
			}

			workspaceID, err := authService.VerifyToken(token)
			if err != nil {
				utils.WriteError(w, r, logger, err)
				return
			}

			ctx := context.WithValue(r.Context(), workspaceIDKey, workspaceID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func WorkspaceID(ctx context.Context) string {
	id, _ := ctx.Value(workspaceIDKey).(string)
	return id
}
