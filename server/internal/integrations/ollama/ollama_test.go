package ollama

import (
	"context"
	"encoding/json"
	"errors"
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

func TestChat(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/chat" {
			t.Errorf("request = %s %s, want POST /api/chat", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decoding request: %v", err)
		}
		_, _ = w.Write([]byte(`{"message": {"role": "assistant", "content": "{\"entity\": \"users\"}"}, "done": true, "done_reason": "stop"}`))
	}))
	defer server.Close()

	got, err := NewClient(server.URL, time.Second).Chat(context.Background(), ChatRequest{
		Model: "qwen3.5:9b",
		Messages: []Message{
			{Role: RoleSystem, Content: "rules"},
			{Role: RoleUser, Content: "Get active users"},
		},
		Format:  json.RawMessage(`{"type": "object"}`),
		Options: Options{Temperature: 0, Seed: 42},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}

	wantBody := map[string]any{
		"model": "qwen3.5:9b",
		"messages": []any{
			map[string]any{"role": "system", "content": "rules"},
			map[string]any{"role": "user", "content": "Get active users"},
		},
		"format":  map[string]any{"type": "object"},
		"stream":  false,
		"think":   false,
		"options": map[string]any{"temperature": float64(0), "seed": float64(42)},
	}
	if diff := cmp.Diff(wantBody, gotBody); diff != "" {
		t.Errorf("request body mismatch (-want +got):\n%s", diff)
	}
	if got != `{"entity": "users"}` {
		t.Errorf("content = %q, want the assistant message content", got)
	}
}

func TestChatErrors(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		body           string
		wantErr        string
		wantIncomplete bool
	}{
		{
			name:    "error status carries ollama's message",
			status:  http.StatusNotFound,
			body:    `{"error": "model \"qwen3.5:9b\" not found, try pulling it first"}`,
			wantErr: "not found, try pulling it first",
		},
		{
			name:           "output cut off by the token limit",
			status:         http.StatusOK,
			body:           `{"message": {"role": "assistant", "content": "{\"entity\":"}, "done": true, "done_reason": "length"}`,
			wantErr:        `stopped early: "length"`,
			wantIncomplete: true,
		},
		{
			name:           "empty content",
			status:         http.StatusOK,
			body:           `{"message": {"role": "assistant", "content": ""}, "done": true, "done_reason": "stop"}`,
			wantErr:        "empty response",
			wantIncomplete: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			_, err := NewClient(server.URL, time.Second).Chat(context.Background(), ChatRequest{Model: "m"})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.wantErr)
			}
			if errors.Is(err, ErrIncompleteAnswer) != tc.wantIncomplete {
				t.Errorf("errors.Is(err, ErrIncompleteAnswer) = %v, want %v", !tc.wantIncomplete, tc.wantIncomplete)
			}
		})
	}
}
