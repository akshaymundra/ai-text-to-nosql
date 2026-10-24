package planner

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/akshaymundra/ai-text-to-nosql/internal/constants"
	"github.com/akshaymundra/ai-text-to-nosql/internal/dynamoschema"
	"github.com/akshaymundra/ai-text-to-nosql/internal/service/ir"
)

var tasksTable = dynamoschema.Table{
	Name:         "tasks",
	PartitionKey: "project_id",
	SortKey:      "task_id",
	Indexes: []dynamoschema.Index{
		{Name: "assignee-due-date-index", PartitionKey: "assignee_id", SortKey: "due_date"},
		{Name: "status-index", PartitionKey: "status"},
	},
}

var (
	projectEq   = ir.Condition{Field: "project_id", Operator: ir.OpEq, Value: "p1"}
	assigneeEq  = ir.Condition{Field: "assignee_id", Operator: ir.OpEq, Value: "u1"}
	statusEq    = ir.Condition{Field: "status", Operator: ir.OpEq, Value: "open"}
	taskPrefix  = ir.Condition{Field: "task_id", Operator: ir.OpBeginsWith, Value: "t_"}
	taskNe      = ir.Condition{Field: "task_id", Operator: ir.OpNe, Value: "t_1"}
	dueBetween  = ir.Condition{Field: "due_date", Operator: ir.OpBetween, Value: []any{"2026-09-01", "2026-09-30"}}
	priorityGte = ir.Condition{Field: "priority", Operator: ir.OpGte, Value: float64(3)}
)

func tasksQuery(conditions ...ir.Condition) ir.Query {
	return ir.Query{Entity: "tasks", Conditions: conditions}
}

func TestPlan(t *testing.T) {
	tests := []struct {
		name          string
		query         ir.Query
		want          AccessPath
		wantRejection *ir.Rejection
	}{
		{
			name:  "base table partition key only",
			query: tasksQuery(projectEq),
			want:  AccessPath{Table: "tasks", PartitionKey: projectEq, Filters: []ir.Condition{}},
		},
		{
			name:  "base table with sort key condition",
			query: tasksQuery(projectEq, taskPrefix),
			want:  AccessPath{Table: "tasks", PartitionKey: projectEq, SortKey: &taskPrefix, Filters: []ir.Condition{}},
		},
		{
			name:  "non-key condition becomes a filter",
			query: tasksQuery(projectEq, priorityGte),
			want:  AccessPath{Table: "tasks", PartitionKey: projectEq, Filters: []ir.Condition{priorityGte}},
		},
		{
			name:  "GSI with sort key condition",
			query: tasksQuery(assigneeEq, dueBetween),
			want: AccessPath{
				Table:        "tasks",
				Index:        "assignee-due-date-index",
				PartitionKey: assigneeEq,
				SortKey:      &dueBetween,
				Filters:      []ir.Condition{},
			},
		},
		{
			name:  "two usable paths without sort key: base table first",
			query: tasksQuery(assigneeEq, projectEq),
			want:  AccessPath{Table: "tasks", PartitionKey: projectEq, Filters: []ir.Condition{assigneeEq}},
		},
		{
			name:  "path that also uses its sort key wins",
			query: tasksQuery(projectEq, assigneeEq, dueBetween),
			want: AccessPath{
				Table:        "tasks",
				Index:        "assignee-due-date-index",
				PartitionKey: assigneeEq,
				SortKey:      &dueBetween,
				Filters:      []ir.Condition{projectEq},
			},
		},
		{
			name:  "rejected base table falls back to usable GSI",
			query: tasksQuery(statusEq, projectEq, taskNe),
			want: AccessPath{
				Table:        "tasks",
				Index:        "status-index",
				PartitionKey: statusEq,
				Filters:      []ir.Condition{projectEq, taskNe},
			},
		},
		{
			name:          "no partition key equality requires a scan",
			query:         tasksQuery(priorityGte),
			wantRejection: &ir.Rejection{Code: constants.RejectCodeScanRequired},
		},
		{
			name:          "no conditions requires a scan",
			query:         tasksQuery(),
			wantRejection: &ir.Rejection{Code: constants.RejectCodeScanRequired},
		},
		{
			name:          "partition key without eq requires a scan",
			query:         tasksQuery(ir.Condition{Field: "project_id", Operator: ir.OpBeginsWith, Value: "p"}),
			wantRejection: &ir.Rejection{Code: constants.RejectCodeScanRequired},
		},
		{
			name:          "filter-only operator on sort key",
			query:         tasksQuery(projectEq, taskNe),
			wantRejection: &ir.Rejection{Code: constants.RejectCodeInvalidKeyCondition, Field: "task_id"},
		},
		{
			name: "filter-only operator on partition key",
			query: tasksQuery(
				ir.Condition{Field: "project_id", Operator: ir.OpIn, Value: []any{"p1", "p2"}},
				projectEq,
			),
			wantRejection: &ir.Rejection{Code: constants.RejectCodeInvalidKeyCondition, Field: "project_id"},
		},
		{
			name: "two conditions on partition key",
			query: tasksQuery(
				projectEq,
				ir.Condition{Field: "project_id", Operator: ir.OpEq, Value: "p2"},
			),
			wantRejection: &ir.Rejection{Code: constants.RejectCodeMultipleKeyConditions, Field: "project_id"},
		},
		{
			name: "two conditions on sort key",
			query: tasksQuery(
				projectEq,
				ir.Condition{Field: "task_id", Operator: ir.OpGte, Value: "t_1"},
				ir.Condition{Field: "task_id", Operator: ir.OpLte, Value: "t_9"},
			),
			wantRejection: &ir.Rejection{Code: constants.RejectCodeMultipleKeyConditions, Field: "task_id"},
		},
		{
			name:          "empty partition key value",
			query:         tasksQuery(ir.Condition{Field: "project_id", Operator: ir.OpEq, Value: ""}),
			wantRejection: &ir.Rejection{Code: constants.RejectCodeEmptyKeyValue, Field: "project_id"},
		},
		{
			name: "empty sort key value inside between",
			query: tasksQuery(
				projectEq,
				ir.Condition{Field: "task_id", Operator: ir.OpBetween, Value: []any{"", "t_9"}},
			),
			wantRejection: &ir.Rejection{Code: constants.RejectCodeEmptyKeyValue, Field: "task_id"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, rejection := Plan(tc.query, tasksTable)
			if diff := cmp.Diff(tc.wantRejection, rejection); diff != "" {
				t.Fatalf("rejection mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("access path mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWarnings(t *testing.T) {
	tests := []struct {
		name string
		path AccessPath
		want []Warning
	}{
		{
			name: "base table without filters",
			path: AccessPath{Table: "tasks", PartitionKey: projectEq, Filters: []ir.Condition{}},
			want: []Warning{},
		},
		{
			name: "GSI with filters",
			path: AccessPath{
				Table:        "tasks",
				Index:        "status-index",
				PartitionKey: statusEq,
				Filters:      []ir.Condition{projectEq, priorityGte},
			},
			want: []Warning{
				{Code: constants.WarnCodeFilterExpression, Field: "project_id"},
				{Code: constants.WarnCodeFilterExpression, Field: "priority"},
				{Code: constants.WarnCodeEventuallyConsistent},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, tc.path.Warnings()); diff != "" {
				t.Errorf("warnings mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
