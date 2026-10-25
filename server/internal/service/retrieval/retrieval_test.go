package retrieval

import (
	"testing"

	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
)

func TestDocumentText(t *testing.T) {
	table := dynamoschema.Table{Name: "invoices", Description: "Invoices billed to organizations."}
	pattern := dynamoschema.AccessPattern{
		Description: "List invoices with a given payment status.",
		Index:       "status-issued-at-index",
	}

	got := DocumentText(table, pattern)

	want := "Table invoices: Invoices billed to organizations.\nAccess pattern: List invoices with a given payment status."
	if got != want {
		t.Errorf("DocumentText() = %q, want %q", got, want)
	}
}

func TestQueryText(t *testing.T) {
	got := QueryText("Show all overdue invoices")

	want := "Instruct: Given a request for data, retrieve the database table access patterns that can answer it\n" +
		"Query:Show all overdue invoices"
	if got != want {
		t.Errorf("QueryText() = %q, want %q", got, want)
	}
}
