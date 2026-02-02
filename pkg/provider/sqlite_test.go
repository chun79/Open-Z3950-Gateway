package provider

import (
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

// setupTestDB is a helper function to create a temporary database for testing.
// It returns a new SQLiteProvider instance and a cleanup function to be called with defer.
func setupTestDB(t *testing.T) (*SQLiteProvider, func()) {
	t.Helper()

	tmpfile, err := os.CreateTemp("", "testdb_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file for test database: %v", err)
	}
	path := tmpfile.Name()
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	provider, err := NewSQLiteProvider(path)
	if err != nil {
		os.Remove(path)
		t.Fatalf("Failed to create SQLiteProvider for test: %v", err)
	}

	cleanup := func() {
		provider.db.Close()
		os.Remove(path)
	}

	return provider, cleanup
}

func TestSearch(t *testing.T) {
	provider, cleanup := setupTestDB(t)
	defer cleanup()

	testCases := []struct {
		name        string
		query       z3950.StructuredQuery
		expectedIDs []string
	}{
		{
			name:        "Search by Title",
			query:       z3950.StructuredQuery{Root: z3950.QueryClause{Attribute: z3950.UseAttributeTitle, Term: "Go Programming Language"}},
			expectedIDs: []string{"1"},
		},
		{
			name:        "Search by Clean ISBN",
			query:       z3950.StructuredQuery{Root: z3950.QueryClause{Attribute: z3950.UseAttributeISBN, Term: "9780134190440"}},
			expectedIDs: []string{"1"},
		},
		{
			name:        "Search by Dirty ISBN",
			query:       z3950.StructuredQuery{Root: z3950.QueryClause{Attribute: z3950.UseAttributeISBN, Term: "ISBN-13: 978-159327865-1"}},
			expectedIDs: []string{"4"},
		},
		{
			name:        "Search by Author",
			query:       z3950.StructuredQuery{Root: z3950.QueryClause{Attribute: z3950.UseAttributeAuthor, Term: "Rob Pike"}},
			expectedIDs: []string{"2"},
		},
		{
			name:        "Search with no results",
			query:       z3950.StructuredQuery{Root: z3950.QueryClause{Attribute: z3950.UseAttributeTitle, Term: "NonExistentBook"}},
			expectedIDs: nil,
		},
		{
			name: "Search with partial term (default attribute)",
			query: z3950.StructuredQuery{Root: z3950.QueryClause{Attribute: 0, Term: "go"}},
			expectedIDs: []string{"1", "2", "3", "4"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ids, err := provider.Search("bibliography", tc.query)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			sort.Strings(ids)
			sort.Strings(tc.expectedIDs)

			if !reflect.DeepEqual(ids, tc.expectedIDs) {
				t.Errorf("Expected IDs %v, but got %v", tc.expectedIDs, ids)
			}
		})
	}
}

func TestFetch(t *testing.T) {
	provider, cleanup := setupTestDB(t)
	defer cleanup()

	idsToFetch := []string{"1", "3"}
	records, err := provider.Fetch("bibliography", idsToFetch)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if len(records) != len(idsToFetch) {
		t.Errorf("Expected to fetch %d records, but got %d", len(idsToFetch), len(records))
	}

	var gotTitles []string
	for _, rec := range records {
		gotTitles = append(gotTitles, strings.TrimSpace(rec.GetTitle(nil)))
	}
	sort.Strings(gotTitles)

	expectedTitles := []string{
		"Go in Practice",
		"The Go Programming Language",
	}
	sort.Strings(expectedTitles)

	if !reflect.DeepEqual(gotTitles, expectedTitles) {
		t.Errorf("Fetched titles do not match expected. Got %v, want %v", gotTitles, expectedTitles)
	}
}

func TestScan(t *testing.T) {
	provider, cleanup := setupTestDB(t)
	defer cleanup()

	startTerm := "Go"
	results, err := provider.Scan("bibliography", "title", startTerm)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	expectedTerms := []string{
		"Go in Practice",
		"The Go Programming Language",
		"Thinking in Go",
	}
	
	if len(results) != len(expectedTerms) {
		var gotTerms []string
		for _, r := range results {
			gotTerms = append(gotTerms, r.Term)
		}
		t.Fatalf("Expected %d scan results, but got %d. Results: %v", len(expectedTerms), len(results), gotTerms)
	}

	for i, res := range results {
		if res.Term != expectedTerms[i] {
			t.Errorf("Scan result at index %d: expected term '%s', got '%s'", i, expectedTerms[i], res.Term)
		}
		if res.Count != 1 {
			t.Errorf("Scan result for '%s': expected count 1, got %d", res.Term, res.Count)
		}
	}
}
