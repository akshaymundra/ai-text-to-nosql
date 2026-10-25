package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/akshaymundra/ai-text-to-nosql/internal/integrations/httpclient"
)

type Client struct {
	http *httpclient.Client
}

type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{http: httpclient.New(baseURL, timeout)}
}

func (c *Client) Embed(ctx context.Context, model string, inputs []string) ([][]float32, error) {
	var resp embedResponse
	if err := c.http.Post(ctx, "/api/embed", embedRequest{Model: model, Input: inputs}, &resp); err != nil {
		return nil, fmt.Errorf("ollama embed: %w", withOllamaMessage(err))
	}
	if len(resp.Embeddings) != len(inputs) {
		return nil, fmt.Errorf("ollama embed: got %d embeddings for %d inputs", len(resp.Embeddings), len(inputs))
	}
	return resp.Embeddings, nil
}

func withOllamaMessage(err error) error {
	var statusErr *httpclient.StatusError
	if !errors.As(err, &statusErr) {
		return err
	}
	var e errorResponse
	if json.Unmarshal(statusErr.Body, &e) != nil || e.Error == "" {
		return err
	}
	return fmt.Errorf("status %d: %s", statusErr.StatusCode, e.Error)
}
