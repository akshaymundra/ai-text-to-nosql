package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestEmbed(t *testing.T) {
	var gotRequest embedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/embed" {
			t.Errorf("request = %s %s, want POST /api/embed", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Errorf("decoding request: %v", err)
		}
		_, _ = w.Write([]byte(`{"model": "m", "embeddings": [[0.1, 0.2], [0.3, 0.4]]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	got, err := client.Embed(context.Background(), "qwen3-embedding:0.6b", []string{"first", "second"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}

	wantRequest := embedRequest{Model: "qwen3-embedding:0.6b", Input: []string{"first", "second"}}
	if diff := cmp.Diff(wantRequest, gotRequest); diff != "" {
		t.Errorf("request mismatch (-want +got):\n%s", diff)
	}
	want := [][]float32{{0.1, 0.2}, {0.3, 0.4}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("embeddings mismatch (-want +got):\n%s", diff)
	}
}

func TestEmbedErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{
			name:    "error status carries ollama's message",
			status:  http.StatusNotFound,
			body:    `{"error": "model \"qwen3-embedding:0.6b\" not found, try pulling it first"}`,
			wantErr: "not found, try pulling it first",
		},
		{
			name:    "fewer embeddings than inputs",
			status:  http.StatusOK,
			body:    `{"embeddings": [[0.1]]}`,
			wantErr: "got 1 embeddings for 2 inputs",
		},
		{
			name:    "invalid json",
			status:  http.StatusOK,
			body:    `not json`,
			wantErr: "decoding POST /api/embed response",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			_, err := NewClient(server.URL, time.Second).Embed(context.Background(), "m", []string{"a", "b"})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestEmbedTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	_, err := NewClient(server.URL, 50*time.Millisecond).Embed(context.Background(), "m", []string{"a"})
	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
}
