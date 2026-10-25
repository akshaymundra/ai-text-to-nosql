package utils

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/akshaymundra/ai-text-to-nosql/internal/apperror"
	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
)

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode constants.QueryErrCode
	}{
		{name: "valid", body: `{"name": "acme"}`},
		{name: "invalid json", body: `{"name":`, wantCode: constants.ErrCodeBadRequest},
		{
			name:     "body over the limit",
			body:     `{"name": "` + strings.Repeat("a", MaxBodyBytes) + `"}`,
			wantCode: constants.ErrCodePayloadTooLarge,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotErr error
			handler := middleware.RequestSize(MaxBodyBytes)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var dst struct {
					Name string `json:"name"`
				}
				gotErr = DecodeJSON(r, &dst)
			}))
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body)))

			if tc.wantCode == "" {
				if gotErr != nil {
					t.Fatalf("unexpected error: %v", gotErr)
				}
				return
			}
			var appErr *apperror.AppError
			if !errors.As(gotErr, &appErr) || appErr.Code != tc.wantCode {
				t.Fatalf("err = %v, want AppError %s", gotErr, tc.wantCode)
			}
		})
	}
}
