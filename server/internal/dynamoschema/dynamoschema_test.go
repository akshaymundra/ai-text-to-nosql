package dynamoschema

import (
	"strings"
	"testing"
)

func TestIsDate(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "2026-09-01", want: true},
		{value: "2026-9-1", want: false},
		{value: "2026-02-30", want: false},
		{value: "2026-09-01T00:00:00Z", want: false},
		{value: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.value, func(t *testing.T) {
			if got := IsDate(tc.value); got != tc.want {
				t.Errorf("IsDate(%q) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func TestIsTimestamp(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "2026-09-01T10:00:00Z", want: true},
		{value: "2026-09-01T10:00:00.5Z", want: false},
		{value: "2026-09-01T10:00:00+05:30", want: false},
		{value: "2026-09-01T25:00:00Z", want: false},
		{value: "2026-09-01", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.value, func(t *testing.T) {
			if got := IsTimestamp(tc.value); got != tc.want {
				t.Errorf("IsTimestamp(%q) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func validTable() Table {
	return Table{
		Name:         "tasks",
		Description:  "Work items inside a project.",
		PartitionKey: "project_id",
		SortKey:      "task_id",
		Attributes: []Attribute{
			{Name: "project_id", Type: TypeString},
			{Name: "task_id", Type: TypeString},
			{Name: "assignee_id", Type: TypeString},
			{Name: "due_date", Type: TypeString, Format: FormatDate},
			{Name: "estimate", Type: TypeNumber},
		},
		Indexes: []Index{{Name: "assignee-due-date-index", PartitionKey: "assignee_id", SortKey: "due_date"}},
		AccessPatterns: []AccessPattern{
			{Description: "List all tasks of a project."},
			{Description: "List tasks assigned to a user.", Index: "assignee-due-date-index"},
		},
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(t *Table)
		wantErr string
	}{
		{name: "valid table", mutate: func(t *Table) {}},
		{name: "table name too short", mutate: func(t *Table) { t.Name = "ab" }, wantErr: `table name "ab"`},
		{name: "table name with space", mutate: func(t *Table) { t.Name = "my tasks" }, wantErr: `table name "my tasks"`},
		{name: "missing description", mutate: func(t *Table) { t.Description = "" }, wantErr: "description is required"},
		{name: "missing partition key", mutate: func(t *Table) { t.PartitionKey = "" }, wantErr: "partition key is required"},
		{
			name:    "partition key not in attributes",
			mutate:  func(t *Table) { t.PartitionKey = "org_id" },
			wantErr: `key "org_id" of table is not in attributes`,
		},
		{
			name:    "sort key not in attributes",
			mutate:  func(t *Table) { t.SortKey = "created_at" },
			wantErr: `key "created_at" of table is not in attributes`,
		},
		{
			name: "unsupported attribute type",
			mutate: func(t *Table) {
				t.Attributes = append(t.Attributes, Attribute{Name: "done", Type: "boolean"})
			},
			wantErr: `attribute "done" has unsupported type "boolean"`,
		},
		{
			name:    "duplicate attribute",
			mutate:  func(t *Table) { t.Attributes = append(t.Attributes, Attribute{Name: "task_id", Type: TypeString}) },
			wantErr: `attribute "task_id" is listed twice`,
		},
		{
			name:    "attribute without name",
			mutate:  func(t *Table) { t.Attributes = append(t.Attributes, Attribute{Type: TypeString}) },
			wantErr: "attribute name is required",
		},
		{
			name:    "unknown format",
			mutate:  func(t *Table) { t.Attributes[3].Format = "datetime" },
			wantErr: `attribute "due_date" has unsupported format "datetime"`,
		},
		{
			name:    "format on a number attribute",
			mutate:  func(t *Table) { t.Attributes[4].Format = FormatDate },
			wantErr: `attribute "estimate" has a format but is not a string`,
		},
		{
			name:    "index key not in attributes",
			mutate:  func(t *Table) { t.Indexes[0].PartitionKey = "owner_id" },
			wantErr: `key "owner_id" of index "assignee-due-date-index" is not in attributes`,
		},
		{
			name:    "index name invalid",
			mutate:  func(t *Table) { t.Indexes[0].Name = "by assignee" },
			wantErr: `index name "by assignee"`,
		},
		{
			name: "duplicate index name",
			mutate: func(t *Table) {
				t.Indexes = append(t.Indexes, Index{Name: "assignee-due-date-index", PartitionKey: "assignee_id"})
			},
			wantErr: `index "assignee-due-date-index" is listed twice`,
		},
		{
			name:    "no access patterns",
			mutate:  func(t *Table) { t.AccessPatterns = nil },
			wantErr: "at least one access pattern is required",
		},
		{
			name:    "access pattern without description",
			mutate:  func(t *Table) { t.AccessPatterns[0].Description = "" },
			wantErr: "access pattern 1 has no description",
		},
		{
			name:    "access pattern on unknown index",
			mutate:  func(t *Table) { t.AccessPatterns[1].Index = "status-index" },
			wantErr: `access pattern 2 uses unknown index "status-index"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			table := validTable()
			tc.mutate(&table)

			err := table.Check()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestCheckReportsEveryProblem(t *testing.T) {
	table := validTable()
	table.Description = ""
	table.AccessPatterns = nil

	err := table.Check()
	for _, want := range []string{"description is required", "at least one access pattern is required"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("err = %v, want it to contain %q", err, want)
		}
	}
}

func TestDemoSchemasPassCheck(t *testing.T) {
	tables, err := LoadDir("../../schemas-local")
	if err != nil {
		t.Fatalf("loading demo schemas: %v", err)
	}
	if len(tables) == 0 {
		t.Fatal("no demo schemas found")
	}
	for _, table := range tables {
		if err := table.Check(); err != nil {
			t.Errorf("%s: %v", table.Name, err)
		}
	}
}

func TestContentHash(t *testing.T) {
	first, err := validTable().ContentHash()
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	again, err := validTable().ContentHash()
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	changed := validTable()
	changed.AccessPatterns[0].Description = "List every task of a project."
	different, err := changed.ContentHash()
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	if len(first) != 64 {
		t.Errorf("hash length = %d, want 64 hex characters", len(first))
	}
	if first != again {
		t.Error("same table hashed differently")
	}
	if first == different {
		t.Error("changed table kept the same hash")
	}
}
