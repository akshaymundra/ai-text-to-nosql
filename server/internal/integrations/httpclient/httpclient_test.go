package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type payload struct {
	Name string `json:"name"`
}

type recorded struct {
	method      string
	path        string
	contentType string
	body        string
}

func echoServer(t *testing.T, got *recorded) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*got = recorded{
			method:      r.Method,
			path:        r.URL.Path,
			contentType: r.Header.Get("Content-Type"),
			body:        string(body),
		}
		_, _ = w.Write([]byte(`{"name": "from server"}`))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestMethods(t *testing.T) {
	in := payload{Name: "acme"}

	tests := []struct {
		name string
		call func(c *Client, out *payload) error
		want recorded
	}{
		{
			name: "get",
			call: func(c *Client, out *payload) error { return c.Get(context.Background(), "/items", out) },
			want: recorded{method: http.MethodGet, path: "/items"},
		},
		{
			name: "post",
			call: func(c *Client, out *payload) error { return c.Post(context.Background(), "/items", in, out) },
			want: recorded{method: http.MethodPost, path: "/items", contentType: "application/json", body: `{"name":"acme"}`},
		},
		{
			name: "put",
			call: func(c *Client, out *payload) error { return c.Put(context.Background(), "/items/1", in, out) },
			want: recorded{method: http.MethodPut, path: "/items/1", contentType: "application/json", body: `{"name":"acme"}`},
		},
		{
			name: "patch",
			call: func(c *Client, out *payload) error { return c.Patch(context.Background(), "/items/1", in, out) },
			want: recorded{method: http.MethodPatch, path: "/items/1", contentType: "application/json", body: `{"name":"acme"}`},
		},
		{
			name: "delete",
			call: func(c *Client, out *payload) error { return c.Delete(context.Background(), "/items/1", out) },
			want: recorded{method: http.MethodDelete, path: "/items/1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got recorded
			server := echoServer(t, &got)

			var out payload
			if err := tc.call(New(server.URL, time.Second), &out); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(recorded{})); diff != "" {
				t.Errorf("request mismatch (-want +got):\n%s", diff)
			}
			if out.Name != "from server" {
				t.Errorf("decoded name = %q, want %q", out.Name, "from server")
			}
		})
	}
}

func TestNilOutSkipsDecoding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := New(server.URL, time.Second).Delete(context.Background(), "/items/1", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatusError(t *testing.T) {
	longBody := strings.Repeat("x", maxErrorBodyBytes+100)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(longBody))
	}))
	defer server.Close()

	err := New(server.URL, time.Second).Get(context.Background(), "/items", nil)

	var statusErr *StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("err = %v, want *StatusError", err)
	}
	if statusErr.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusBadGateway)
	}
	if len(statusErr.Body) != maxErrorBodyBytes {
		t.Errorf("body length = %d, want it capped at %d", len(statusErr.Body), maxErrorBodyBytes)
	}
}

func TestDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	var out payload
	err := New(server.URL, time.Second).Get(context.Background(), "/items", &out)
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("err = %v, want a JSON syntax error", err)
	}
}

func TestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	if err := New(server.URL, 50*time.Millisecond).Get(context.Background(), "/slow", nil); err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
}

func TestWithHeader(t *testing.T) {
	var gotAuth, gotTrace []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))
		gotTrace = append(gotTrace, r.Header.Get("X-Trace"))
	}))
	defer server.Close()

	base := New(server.URL, time.Second)
	authed := base.WithHeader("Authorization", "Bearer token").WithHeader("X-Trace", "abc")

	for _, c := range []*Client{authed, base} {
		if err := c.Get(context.Background(), "/", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if diff := cmp.Diff([]string{"Bearer token", ""}, gotAuth); diff != "" {
		t.Errorf("authorization headers mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"abc", ""}, gotTrace); diff != "" {
		t.Errorf("trace headers mismatch (-want +got):\n%s", diff)
	}
}
