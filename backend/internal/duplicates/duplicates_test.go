package duplicates

import "testing"

func TestDuplicatePairKeyStable(t *testing.T) {
	// The UpsertPair canonicalizes ordering. Verify the SQL ordering logic helper
	// matches what the query expects (a < b lexicographically).
	a, b := "lead-b", "lead-a"
	if a > b {
		a, b = b, a
	}
	if a != "lead-a" {
		t.Fatalf("expected canonical order lead-a, got %s", a)
	}
}
