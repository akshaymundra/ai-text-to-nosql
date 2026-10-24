package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/akshaymundra/ai-text-to-nosql/internal/model"
	"github.com/akshaymundra/ai-text-to-nosql/internal/repo"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service"
	"github.com/akshaymundra/ai-text-to-nosql/internal/types"
)

type singleWorkspaceStore struct {
	ws model.Workspace
}

func (s *singleWorkspaceStore) Create(ctx context.Context, ws model.Workspace) (model.Workspace, error) {
	ws.ID = "ws-1"
	s.ws = ws
	return ws, nil
}

func (s *singleWorkspaceStore) GetByName(ctx context.Context, name string) (model.Workspace, error) {
	if name != s.ws.Name {
		return model.Workspace{}, repo.ErrNotFound
	}
	return s.ws, nil
}

func TestAuthenticate(t *testing.T) {
	authSvc := service.NewAuthService(&singleWorkspaceStore{}, "test-secret-at-least-32-bytes-long!!", time.Hour, false)
	token, err := authSvc.Register(context.Background(), types.RegisterRequest{
		Name:           "acme",
		Password:       "correct-horse",
		ConnectionType: model.ConnectionTypeTemporaryCredentials,
		AWSRegion:      "ap-south-1",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	tests := []struct {
		name          string
		header        string
		wantStatus    int
		wantWorkspace string
	}{
		{name: "valid bearer token", header: "Bearer " + token.Token, wantStatus: http.StatusOK, wantWorkspace: "ws-1"},
		{name: "missing header", header: "", wantStatus: http.StatusUnauthorized},
		{name: "wrong scheme", header: "Basic " + token.Token, wantStatus: http.StatusUnauthorized},
		{name: "bearer without token", header: "Bearer ", wantStatus: http.StatusUnauthorized},
		{name: "invalid token", header: "Bearer not-a-jwt", wantStatus: http.StatusUnauthorized},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotWorkspace string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotWorkspace = WorkspaceID(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/queries", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			Authenticate(authSvc, zap.NewNop())(next).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if gotWorkspace != tc.wantWorkspace {
				t.Fatalf("workspace in context = %q, want %q", gotWorkspace, tc.wantWorkspace)
			}
		})
	}
}
