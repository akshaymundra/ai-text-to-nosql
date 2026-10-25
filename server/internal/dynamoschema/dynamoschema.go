package dynamoschema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	TypeString = "string"
	TypeNumber = "number"

	FormatDate      = "date"
	FormatTimestamp = "timestamp"

	dateLayout      = "2006-01-02"
	timestampLayout = "2006-01-02T15:04:05Z"
)

type Attribute struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Format      string `json:"format,omitempty"`
	Description string `json:"description"`
}

type Index struct {
	Name         string `json:"name"`
	PartitionKey string `json:"partitionKey"`
	SortKey      string `json:"sortKey,omitempty"`
}

type AccessPattern struct {
	Description string `json:"description"`
	Index       string `json:"index,omitempty"`
}

type Table struct {
	Name           string           `json:"table"`
	Description    string           `json:"description"`
	PartitionKey   string           `json:"partitionKey"`
	SortKey        string           `json:"sortKey,omitempty"`
	Attributes     []Attribute      `json:"attributes"`
	Indexes        []Index          `json:"indexes"`
	AccessPatterns []AccessPattern  `json:"accessPatterns"`
	ExampleRecords []map[string]any `json:"exampleRecords"`
}

func (t Table) Attribute(name string) (Attribute, bool) {
	for _, a := range t.Attributes {
		if a.Name == name {
			return a, true
		}
	}
	return Attribute{}, false
}

func FindTable(tables []Table, name string) (Table, bool) {
	for _, t := range tables {
		if t.Name == name {
			return t, true
		}
	}
	return Table{}, false
}

func IsDate(v string) bool {
	return matchesLayout(v, dateLayout)
}

func IsTimestamp(v string) bool {
	return matchesLayout(v, timestampLayout)
}

// time.Parse accepts fractional seconds the layout does not mention, so the length check keeps values fixed-width.
func matchesLayout(v, layout string) bool {
	if len(v) != len(layout) {
		return false
	}
	_, err := time.Parse(layout, v)
	return err == nil
}

func LoadDir(dir string) ([]Table, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	tables := make([]Table, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var t Table
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		tables = append(tables, t)
	}
	return tables, nil
}
