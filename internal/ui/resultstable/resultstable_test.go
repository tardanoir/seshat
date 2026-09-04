package resultstable

import (
	"testing"

	"github.com/tardanoir/seshat/internal/db"
)

func TestColumnValues(t *testing.T) {
	m := New()
	m.SetResult(&db.QueryResult{
		Columns:  []string{"id", "name"},
		Rows:     [][]string{{"1", "ana"}, {"2", "bo"}, {"3", "cy"}},
		IsSelect: true,
	})

	if got, want := m.columnValues(0), "1\n2\n3"; got != want {
		t.Errorf("columnValues(0) = %q, want %q", got, want)
	}
	if got, want := m.columnValues(1), "ana\nbo\ncy"; got != want {
		t.Errorf("columnValues(1) = %q, want %q", got, want)
	}
}

// Copying must yield the raw stored values, not the display-formatted ones
// (NULL rendered as ∅, newlines flattened, long values truncated).
func TestColumnValuesUsesRawValues(t *testing.T) {
	long := ""
	for range 120 {
		long += "x"
	}
	m := New()
	m.SetResult(&db.QueryResult{
		Columns:  []string{"v"},
		Rows:     [][]string{{"NULL"}, {"a\tb"}, {long}},
		IsSelect: true,
	})

	want := "NULL\na\tb\n" + long
	if got := m.columnValues(0); got != want {
		t.Errorf("columnValues(0) = %q, want %q", got, want)
	}
}

// Ragged rows (a row shorter than the header) must be skipped rather than
// panicking on an out-of-range index.
func TestColumnValuesSkipsShortRows(t *testing.T) {
	m := New()
	m.SetResult(&db.QueryResult{
		Columns:  []string{"a", "b"},
		Rows:     [][]string{{"1", "x"}, {"2"}, {"3", "z"}},
		IsSelect: true,
	})

	if got, want := m.columnValues(1), "x\n\nz"; got != want {
		t.Errorf("columnValues(1) = %q, want %q", got, want)
	}
}
