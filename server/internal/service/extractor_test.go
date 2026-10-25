package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/integrations/ollama"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/extraction"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
)

type fakeChatter struct {
	gotRequest ollama.ChatRequest
	called     bool
	answer     string
	err        error
}

func (f *fakeChatter) Chat(ctx context.Context, req ollama.ChatRequest) (string, error) {
	f.gotRequest = req
	f.called = true
	return f.answer, f.err
}

var extractNow = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

func TestExtract(t *testing.T) {
	chat := &fakeChatter{answer: `{"entity": "users", "conditions": [` +
		`{"field": "status", "operator": "eq", "value": "active"}], "unmapped_terms": []}`}
	extractor := NewExtractor(chat, "qwen3.5:9b", 42)

	got, rejection, err := extractor.Extract(context.Background(), usersTables, "Active users", extractNow)
	if err != nil || rejection != nil {
		t.Fatalf("extract: err %v, rejection %v", err, rejection)
	}

	wantSchema, err := extraction.Schema(usersTables)
	if err != nil {
		t.Fatalf("schema: %v", err)
	}
	wantRequest := ollama.ChatRequest{
		Model: "qwen3.5:9b",
		Messages: []ollama.Message{
			{Role: ollama.RoleSystem, Content: extraction.SystemPrompt(usersTables, extractNow)},
			{Role: ollama.RoleUser, Content: "Active users"},
		},
		Format:  wantSchema,
		Options: ollama.Options{Temperature: 0, Seed: 42, NumPredict: maxAnswerTokens},
	}
	if diff := cmp.Diff(wantRequest, chat.gotRequest); diff != "" {
		t.Errorf("chat request mismatch (-want +got):\n%s", diff)
	}

	want := extraction.Extraction{
		Query: ir.Query{Entity: "users", Conditions: []ir.Condition{
			{Field: "status", Operator: ir.OpEq, Value: "active"},
		}},
		UnmappedTerms: []string{},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("extraction mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractRejections(t *testing.T) {
	tests := []struct {
		name       string
		tables     []dynamoschema.Table
		chat       *fakeChatter
		wantCode   constants.RejectionCode
		wantCalled bool
	}{
		{
			name:     "workspace without schemas",
			tables:   nil,
			chat:     &fakeChatter{},
			wantCode: constants.RejectCodeNoSchemas,
		},
		{
			name:       "answer is not the expected json",
			tables:     usersTables,
			chat:       &fakeChatter{answer: `{"entity": "users", "index": "x"}`},
			wantCode:   constants.RejectCodeExtractionFailed,
			wantCalled: true,
		},
		{
			name:       "answer cut off",
			tables:     usersTables,
			chat:       &fakeChatter{err: fmt.Errorf("ollama chat: %w: stopped early", ollama.ErrIncompleteAnswer)},
			wantCode:   constants.RejectCodeExtractionFailed,
			wantCalled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, rejection, err := NewExtractor(tc.chat, "m", 42).
				Extract(context.Background(), tc.tables, "Active users", extractNow)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(&ir.Rejection{Code: tc.wantCode}, rejection); diff != "" {
				t.Errorf("rejection mismatch (-want +got):\n%s", diff)
			}
			if tc.chat.called != tc.wantCalled {
				t.Errorf("model called = %v, want %v", tc.chat.called, tc.wantCalled)
			}
		})
	}
}

func TestExtractOllamaFailureIsAnError(t *testing.T) {
	failure := errors.New("connection refused")
	_, rejection, err := NewExtractor(&fakeChatter{err: failure}, "m", 42).
		Extract(context.Background(), usersTables, "Active users", extractNow)

	if !errors.Is(err, failure) || rejection != nil {
		t.Fatalf("err = %v, rejection = %v; want the Ollama error and no rejection", err, rejection)
	}
}
