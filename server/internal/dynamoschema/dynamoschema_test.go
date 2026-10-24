package dynamoschema

import "testing"

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
