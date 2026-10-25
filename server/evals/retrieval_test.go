package evals

import (
	"testing"

	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
)

func TestRetrievalRank(t *testing.T) {
	tables := []dynamoschema.Table{{Name: "invoices"}, {Name: "organizations"}, {Name: "users"}}

	tests := []struct {
		entity string
		want   int
	}{
		{entity: "invoices", want: 1},
		{entity: "users", want: 3},
		{entity: "tasks", want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.entity, func(t *testing.T) {
			if got := RetrievalRank(tc.entity, tables); got != tc.want {
				t.Errorf("RetrievalRank(%q) = %d, want %d", tc.entity, got, tc.want)
			}
		})
	}
}
