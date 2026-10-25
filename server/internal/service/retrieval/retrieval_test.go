package retrieval

import (
	"testing"

	"github.com/google/go-cmp/cmp"

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

func TestSelectTables(t *testing.T) {
	users := dynamoschema.Table{Name: "users"}
	invoices := dynamoschema.Table{Name: "invoices"}
	tasks := dynamoschema.Table{Name: "tasks"}
	projects := dynamoschema.Table{Name: "projects"}

	tests := []struct {
		name   string
		ranked []dynamoschema.Table
		max    int
		want   []string
	}{
		{name: "nothing retrieved", ranked: nil, max: 3, want: []string{}},
		{name: "fewer tables than the limit", ranked: []dynamoschema.Table{users, invoices}, max: 3, want: []string{"users", "invoices"}},
		{
			name:   "duplicates keep the best rank",
			ranked: []dynamoschema.Table{users, invoices, users, tasks},
			max:    3,
			want:   []string{"users", "invoices", "tasks"},
		},
		{
			name:   "capped at the limit",
			ranked: []dynamoschema.Table{users, invoices, tasks, projects},
			max:    3,
			want:   []string{"users", "invoices", "tasks"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := []string{}
			for _, table := range SelectTables(tc.ranked, tc.max) {
				got = append(got, table.Name)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("tables mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
