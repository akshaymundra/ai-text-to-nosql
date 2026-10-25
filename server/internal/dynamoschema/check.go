package dynamoschema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

var namePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{3,255}$`)

// Check reports every problem in a table definition at once, so a client can fix a schema in one round trip.
func (t Table) Check() error {
	var problems []error
	if !namePattern.MatchString(t.Name) {
		problems = append(problems, fmt.Errorf("table name %q must be 3 to 255 letters, digits, '_', '-' or '.'", t.Name))
	}
	if t.Description == "" {
		problems = append(problems, errors.New("description is required"))
	}
	if t.PartitionKey == "" {
		problems = append(problems, errors.New("partition key is required"))
	}
	problems = append(problems, t.checkAttributes()...)
	problems = append(problems, t.checkKeys("table", t.PartitionKey, t.SortKey)...)
	problems = append(problems, t.checkIndexes()...)
	problems = append(problems, t.checkAccessPatterns()...)
	return errors.Join(problems...)
}

// ContentHash versions a table definition. json.Marshal is deterministic here: struct fields keep their order
// and map keys are sorted.
func (t Table) ContentHash() (string, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("hashing table %q: %w", t.Name, err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (t Table) checkAttributes() []error {
	var problems []error
	seen := map[string]bool{}
	for _, a := range t.Attributes {
		switch {
		case a.Name == "":
			problems = append(problems, errors.New("attribute name is required"))
			continue
		case seen[a.Name]:
			problems = append(problems, fmt.Errorf("attribute %q is listed twice", a.Name))
		}
		seen[a.Name] = true

		if a.Type != TypeString && a.Type != TypeNumber {
			problems = append(problems, fmt.Errorf("attribute %q has unsupported type %q", a.Name, a.Type))
		}
		if a.Format != "" && a.Format != FormatDate && a.Format != FormatTimestamp {
			problems = append(problems, fmt.Errorf("attribute %q has unsupported format %q", a.Name, a.Format))
		}
		if a.Format != "" && a.Type != TypeString {
			problems = append(problems, fmt.Errorf("attribute %q has a format but is not a string", a.Name))
		}
	}
	return problems
}

func (t Table) checkKeys(owner, partitionKey, sortKey string) []error {
	var problems []error
	for _, key := range []string{partitionKey, sortKey} {
		if key == "" {
			continue
		}
		if _, ok := t.Attribute(key); !ok {
			problems = append(problems, fmt.Errorf("key %q of %s is not in attributes", key, owner))
		}
	}
	return problems
}

func (t Table) checkIndexes() []error {
	var problems []error
	seen := map[string]bool{}
	for _, idx := range t.Indexes {
		if !namePattern.MatchString(idx.Name) {
			problems = append(problems, fmt.Errorf("index name %q must be 3 to 255 letters, digits, '_', '-' or '.'", idx.Name))
		}
		if seen[idx.Name] {
			problems = append(problems, fmt.Errorf("index %q is listed twice", idx.Name))
		}
		seen[idx.Name] = true

		if idx.PartitionKey == "" {
			problems = append(problems, fmt.Errorf("index %q has no partition key", idx.Name))
		}
		problems = append(problems, t.checkKeys(fmt.Sprintf("index %q", idx.Name), idx.PartitionKey, idx.SortKey)...)
	}
	return problems
}

func (t Table) checkAccessPatterns() []error {
	if len(t.AccessPatterns) == 0 {
		return []error{errors.New("at least one access pattern is required")}
	}

	var problems []error
	for i, p := range t.AccessPatterns {
		number := i + 1
		if p.Description == "" {
			problems = append(problems, fmt.Errorf("access pattern %d has no description", number))
		}
		if p.Index != "" && !t.hasIndex(p.Index) {
			problems = append(problems, fmt.Errorf("access pattern %d uses unknown index %q", number, p.Index))
		}
	}
	return problems
}

func (t Table) hasIndex(name string) bool {
	for _, idx := range t.Indexes {
		if idx.Name == name {
			return true
		}
	}
	return false
}
