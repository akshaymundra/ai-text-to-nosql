package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/retrieval"
)

type textEmbedder struct {
	vectors map[string][]float32
	called  bool
	err     error
}

func (f *textEmbedder) Embed(ctx context.Context, model string, inputs []string) ([][]float32, error) {
	f.called = true
	if f.err != nil {
		return nil, f.err
	}
	out := make([][]float32, len(inputs))
	for i, in := range inputs {
		out[i] = f.vectors[in]
	}
	return out, nil
}

var suggestTable = dynamoschema.Table{
	Name: "users",
	Attributes: []dynamoschema.Attribute{
		{Name: "name", Description: "Full name."},
		{Name: "status", Description: "Account status."},
	},
}

func suggestEmbedder() *textEmbedder {
	return &textEmbedder{vectors: map[string][]float32{
		retrieval.FieldText(suggestTable.Attributes[0]): {1, 0},
		retrieval.FieldText(suggestTable.Attributes[1]): {0, 1},
		retrieval.TermText("first name"):                {0.8, 0.6},
		retrieval.TermText("phone number"):              {0.5, 0.5},
	}}
}

func TestSuggest(t *testing.T) {
	rejections := []ir.Rejection{
		{Code: constants.RejectCodeUnknownField, Field: "first name"},
		{Code: constants.RejectCodeUnknownField, Field: "phone number"},
		{Code: constants.RejectCodeValueTypeMismatch, Field: "status"},
	}

	got, err := NewFieldSuggester(suggestEmbedder(), testEmbeddingModel).
		Suggest(context.Background(), rejections, suggestTable)
	if err != nil {
		t.Fatalf("suggest: %v", err)
	}

	want := []ir.Rejection{
		{Code: constants.RejectCodeUnknownField, Field: "first name", Suggestion: "name"},
		{Code: constants.RejectCodeUnknownField, Field: "phone number"},
		{Code: constants.RejectCodeValueTypeMismatch, Field: "status"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("rejections mismatch (-want +got):\n%s", diff)
	}
	if rejections[0].Suggestion != "" {
		t.Error("input rejections were modified")
	}
}

func TestSuggestSkipsEmbeddingWithoutUnknownFields(t *testing.T) {
	embedder := suggestEmbedder()
	rejections := []ir.Rejection{{Code: constants.RejectCodeScanRequired}}

	got, err := NewFieldSuggester(embedder, testEmbeddingModel).Suggest(context.Background(), rejections, suggestTable)
	if err != nil {
		t.Fatalf("suggest: %v", err)
	}
	if embedder.called {
		t.Error("embedder called without any unknown field")
	}
	if diff := cmp.Diff(rejections, got); diff != "" {
		t.Errorf("rejections mismatch (-want +got):\n%s", diff)
	}
}

func TestSuggestEmbedFailure(t *testing.T) {
	failure := errors.New("connection refused")
	rejections := []ir.Rejection{{Code: constants.RejectCodeUnknownField, Field: "first name"}}

	_, err := NewFieldSuggester(&textEmbedder{err: failure}, testEmbeddingModel).
		Suggest(context.Background(), rejections, suggestTable)
	if !errors.Is(err, failure) {
		t.Fatalf("err = %v, want it to wrap %v", err, failure)
	}
}
