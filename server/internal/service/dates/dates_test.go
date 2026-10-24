package dates

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
)

var tasksTable = dynamoschema.Table{
	Name: "tasks",
	Attributes: []dynamoschema.Attribute{
		{Name: "created_at", Type: dynamoschema.TypeString, Format: dynamoschema.FormatTimestamp},
		{Name: "due_date", Type: dynamoschema.TypeString, Format: dynamoschema.FormatDate},
		{Name: "status", Type: dynamoschema.TypeString},
	},
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name           string
		condition      ir.Condition
		want           ir.Condition
		wantRejections []ir.Rejection
	}{
		{
			name:      "attribute without format is untouched",
			condition: ir.Condition{Field: "status", Operator: ir.OpEq, Value: "2026-09-01"},
			want:      ir.Condition{Field: "status", Operator: ir.OpEq, Value: "2026-09-01"},
		},
		{
			name:      "between dates on timestamp covers both whole days",
			condition: ir.Condition{Field: "created_at", Operator: ir.OpBetween, Value: []any{"2026-09-01", "2026-09-30"}},
			want: ir.Condition{
				Field:    "created_at",
				Operator: ir.OpBetween,
				Value:    []any{"2026-09-01T00:00:00Z", "2026-09-30T23:59:59Z"},
			},
		},
		{
			name: "between date and timestamp widens only the date",
			condition: ir.Condition{
				Field:    "created_at",
				Operator: ir.OpBetween,
				Value:    []any{"2026-09-01", "2026-09-15T12:00:00Z"},
			},
			want: ir.Condition{
				Field:    "created_at",
				Operator: ir.OpBetween,
				Value:    []any{"2026-09-01T00:00:00Z", "2026-09-15T12:00:00Z"},
			},
		},
		{
			name:      "gte date starts at midnight",
			condition: ir.Condition{Field: "created_at", Operator: ir.OpGte, Value: "2026-09-01"},
			want:      ir.Condition{Field: "created_at", Operator: ir.OpGte, Value: "2026-09-01T00:00:00Z"},
		},
		{
			name:      "lt date stops before midnight",
			condition: ir.Condition{Field: "created_at", Operator: ir.OpLt, Value: "2026-09-01"},
			want:      ir.Condition{Field: "created_at", Operator: ir.OpLt, Value: "2026-09-01T00:00:00Z"},
		},
		{
			name:      "gt date skips the whole day",
			condition: ir.Condition{Field: "created_at", Operator: ir.OpGt, Value: "2026-09-01"},
			want:      ir.Condition{Field: "created_at", Operator: ir.OpGt, Value: "2026-09-01T23:59:59Z"},
		},
		{
			name:      "lte date includes the whole day",
			condition: ir.Condition{Field: "created_at", Operator: ir.OpLte, Value: "2026-09-01"},
			want:      ir.Condition{Field: "created_at", Operator: ir.OpLte, Value: "2026-09-01T23:59:59Z"},
		},
		{
			name:      "eq date on timestamp becomes a day prefix",
			condition: ir.Condition{Field: "created_at", Operator: ir.OpEq, Value: "2026-09-01"},
			want:      ir.Condition{Field: "created_at", Operator: ir.OpBeginsWith, Value: "2026-09-01"},
		},
		{
			name:      "eq timestamp is untouched",
			condition: ir.Condition{Field: "created_at", Operator: ir.OpEq, Value: "2026-09-01T10:00:00Z"},
			want:      ir.Condition{Field: "created_at", Operator: ir.OpEq, Value: "2026-09-01T10:00:00Z"},
		},
		{
			name:      "begins_with month prefix is untouched",
			condition: ir.Condition{Field: "created_at", Operator: ir.OpBeginsWith, Value: "2026-09"},
			want:      ir.Condition{Field: "created_at", Operator: ir.OpBeginsWith, Value: "2026-09"},
		},
		{
			name:      "exists has no value to check",
			condition: ir.Condition{Field: "created_at", Operator: ir.OpExists},
			want:      ir.Condition{Field: "created_at", Operator: ir.OpExists},
		},
		{
			name: "in with timestamps is untouched",
			condition: ir.Condition{
				Field:    "created_at",
				Operator: ir.OpIn,
				Value:    []any{"2026-09-01T10:00:00Z", "2026-09-02T10:00:00Z"},
			},
			want: ir.Condition{
				Field:    "created_at",
				Operator: ir.OpIn,
				Value:    []any{"2026-09-01T10:00:00Z", "2026-09-02T10:00:00Z"},
			},
		},
		{
			name:           "ne date on timestamp cannot be widened",
			condition:      ir.Condition{Field: "created_at", Operator: ir.OpNe, Value: "2026-09-01"},
			want:           ir.Condition{Field: "created_at", Operator: ir.OpNe, Value: "2026-09-01"},
			wantRejections: []ir.Rejection{{Code: constants.RejectCodeInvalidDateFormat, Field: "created_at"}},
		},
		{
			name:           "in dates on timestamp cannot be widened",
			condition:      ir.Condition{Field: "created_at", Operator: ir.OpIn, Value: []any{"2026-09-01"}},
			want:           ir.Condition{Field: "created_at", Operator: ir.OpIn, Value: []any{"2026-09-01"}},
			wantRejections: []ir.Rejection{{Code: constants.RejectCodeInvalidDateFormat, Field: "created_at"}},
		},
		{
			name:           "timestamp with millis is rejected",
			condition:      ir.Condition{Field: "created_at", Operator: ir.OpEq, Value: "2026-09-01T10:00:00.5Z"},
			want:           ir.Condition{Field: "created_at", Operator: ir.OpEq, Value: "2026-09-01T10:00:00.5Z"},
			wantRejections: []ir.Rejection{{Code: constants.RejectCodeInvalidDateFormat, Field: "created_at"}},
		},
		{
			name:           "not a date at all on timestamp",
			condition:      ir.Condition{Field: "created_at", Operator: ir.OpGte, Value: "last month"},
			want:           ir.Condition{Field: "created_at", Operator: ir.OpGte, Value: "last month"},
			wantRejections: []ir.Rejection{{Code: constants.RejectCodeInvalidDateFormat, Field: "created_at"}},
		},
		{
			name:      "between dates on date attribute is untouched",
			condition: ir.Condition{Field: "due_date", Operator: ir.OpBetween, Value: []any{"2026-09-01", "2026-09-30"}},
			want:      ir.Condition{Field: "due_date", Operator: ir.OpBetween, Value: []any{"2026-09-01", "2026-09-30"}},
		},
		{
			name:           "timestamp on date attribute is rejected",
			condition:      ir.Condition{Field: "due_date", Operator: ir.OpEq, Value: "2026-09-01T10:00:00Z"},
			want:           ir.Condition{Field: "due_date", Operator: ir.OpEq, Value: "2026-09-01T10:00:00Z"},
			wantRejections: []ir.Rejection{{Code: constants.RejectCodeInvalidDateFormat, Field: "due_date"}},
		},
		{
			name:           "impossible date on date attribute is rejected",
			condition:      ir.Condition{Field: "due_date", Operator: ir.OpLt, Value: "2026-13-01"},
			want:           ir.Condition{Field: "due_date", Operator: ir.OpLt, Value: "2026-13-01"},
			wantRejections: []ir.Rejection{{Code: constants.RejectCodeInvalidDateFormat, Field: "due_date"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := ir.Query{Entity: "tasks", Conditions: []ir.Condition{tc.condition}}
			got, rejections := Normalize(q, tasksTable)

			want := ir.Query{Entity: "tasks", Conditions: []ir.Condition{tc.want}}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("query mismatch (-want +got):\n%s", diff)
			}
			wantRejections := tc.wantRejections
			if wantRejections == nil {
				wantRejections = []ir.Rejection{}
			}
			if diff := cmp.Diff(wantRejections, rejections); diff != "" {
				t.Errorf("rejections mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNormalizeDoesNotModifyInput(t *testing.T) {
	bounds := []any{"2026-09-01", "2026-09-30"}
	q := ir.Query{
		Entity:     "tasks",
		Conditions: []ir.Condition{{Field: "created_at", Operator: ir.OpBetween, Value: bounds}},
	}

	Normalize(q, tasksTable)

	if diff := cmp.Diff([]any{"2026-09-01", "2026-09-30"}, bounds); diff != "" {
		t.Errorf("input bounds changed (-want +got):\n%s", diff)
	}
}
