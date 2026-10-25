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

// ErrIncompleteAnswer means the model replied but the answer is unusable (cut off or empty), as opposed to
// Ollama being unreachable or failing.
var ErrIncompleteAnswer = errors.New("incomplete answer")

const (
	RoleSystem = "system"
	RoleUser   = "user"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Options struct {
	Temperature float64 `json:"temperature"`
	Seed        int     `json:"seed"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// ChatRequest asks for one non-streamed answer. Format is a JSON schema the answer must follow.
type ChatRequest struct {
	Model    string
	Messages []Message
	Format   json.RawMessage
	Options  Options
}

type chatRequest struct {
	Model    string          `json:"model"`
	Messages []Message       `json:"messages"`
	Format   json.RawMessage `json:"format,omitempty"`
	Stream   bool            `json:"stream"`
	Think    bool            `json:"think"`
	Options  Options         `json:"options"`
}

type chatResponse struct {
	Message    Message `json:"message"`
	DoneReason string  `json:"done_reason"`
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

// Chat never enables thinking: extraction wants the structured answer only, and thinking tokens cost seconds.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (string, error) {
	var resp chatResponse
	err := c.http.Post(ctx, "/api/chat", chatRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Format:   req.Format,
		Options:  req.Options,
	}, &resp)
	if err != nil {
		return "", fmt.Errorf("ollama chat: %w", withOllamaMessage(err))
	}
	if resp.DoneReason != "stop" {
		return "", fmt.Errorf("ollama chat: %w: stopped early: %q", ErrIncompleteAnswer, resp.DoneReason)
	}
	if resp.Message.Content == "" {
		return "", fmt.Errorf("ollama chat: %w: empty response", ErrIncompleteAnswer)
	}
	return resp.Message.Content, nil
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
