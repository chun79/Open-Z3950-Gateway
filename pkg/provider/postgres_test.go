package provider

import (
	"database/sql"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

// setupPostgresTestDB connects to the test postgres DB, truncates it, and seeds it.
func setupPostgresTestDB(t *testing.T) (*PostgresProvider, func()) {
	t.Helper()

	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set, skipping postgres tests")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("Failed to open test postgres db: %v", err)
	}

	// Clean and setup table for a fresh test run
	_, err = db.Exec(`DROP TABLE IF EXISTS bibliography;`)
	if err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}

	createTableSQL := `
	CREATE TABLE bibliography (
		id INTEGER PRIMARY KEY,
		title TEXT,
		author TEXT,
		isbn TEXT,
		publisher TEXT,
		pub_year TEXT,
		raw_record TEXT,
		raw_record_format TEXT
	);`
	if _, err := db.Exec(createTableSQL); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	
	insertSQL := `
	INSERT INTO bibliography (id, title, author, isbn, publisher, pub_year) VALUES
	(1, 'The Go Programming Language', 'Alan A. A. Donovan, Brian W. Kernighan', '978-0134190440', 'Addison-Wesley', '2015'),
	(2, 'Thinking in Go', 'Rob Pike', '0201548550', 'PublisherX', '2018'),
	(3, 'Go in Practice', 'Matt Butcher, Matt Farina', '978-1617291784', 'Manning', '2016'),
	(4, 'Black Hat Go', 'Tom Steele, Chris Patten, Dan Kottmann', '978-1593278651', 'No Starch Press', '2020');
	`
	if _, err := db.Exec(insertSQL); err != nil {
		t.Fatalf("Failed to seed data: %v", err)
	}
	
	// The NewPostgresProvider doesn't need the db object, just the dsn.
	provider, err := NewPostgresProvider(dsn)
	if err != nil {
		t.Fatalf("Failed to create PostgresProvider for test: %v", err)
	}

	cleanup := func() {
		provider.db.Close()
	}

	return provider, cleanup
}

func TestPostgresSearch(t *testing.T) {
	provider, cleanup := setupPostgresTestDB(t)
	defer cleanup()

	testCases := []struct {
		name        string
		query       z3950.StructuredQuery
		expectedIDs []string
	}{
		{
			name: "Search by Title",
			query: z3950.StructuredQuery{Root: z3950.QueryClause{Attribute: z3950.UseAttributeTitle, Term: "Go Programming"}},
			expectedIDs: []string{"1"},
		},
		{
			name: "Search by ISBN",
			query: z3950.StructuredQuery{Root: z3950.QueryClause{Attribute: z3950.UseAttributeISBN, Term: "978-161729178-4"}},
			expectedIDs: []string{"3"},
		},
		{
			name: "Search by Author",
			query: z3950.StructuredQuery{Root: z3950.QueryClause{Attribute: z3950.UseAttributeAuthor, Term: "Rob Pike"}},
			expectedIDs: []string{"2"},
		},
		{
			name:        "Search with no results",
			query:       z3950.StructuredQuery{Root: z3950.QueryClause{Attribute: z3950.UseAttributeTitle, Term: "NonExistentBook"}},
			expectedIDs: nil,
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

func TestPostgresFetch(t *testing.T) {
	provider, cleanup := setupPostgresTestDB(t)
	defer cleanup()

	idsToFetch := []string{"2", "4"}
	records, err := provider.Fetch("bibliography", idsToFetch)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if len(records) != len(idsToFetch) {
		t.Errorf("Expected to fetch %d records, but got %d", len(idsToFetch), len(records))
	}

	var gotTitles []string
	for _, rec := range records {
		// Unlike SQLite, Postgres doesn't seem to add padding, so TrimSpace may not be needed,
		// but it's good practice to keep it for robustness.
		gotTitles = append(gotTitles, strings.TrimSpace(rec.GetTitle(nil)))
	}
	sort.Strings(gotTitles)

	expectedTitles := []string{
		"Black Hat Go",
		"Thinking in Go",
	}
	sort.Strings(expectedTitles)

	if !reflect.DeepEqual(gotTitles, expectedTitles) {
		t.Errorf("Fetched titles do not match expected. Got %v, want %v", gotTitles, expectedTitles)
	}
}

func TestPostgresScan(t *testing.T) {
	provider, cleanup := setupPostgresTestDB(t)
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
		for _, r := range results { gotTerms = append(gotTerms, r.Term) }
		t.Fatalf("Expected %d scan results, but got %d. Results: %v", len(expectedTerms), len(results), gotTerms)
	}

	for i, res := range results {
		if res.Term != expectedTerms[i] {
			t.Errorf("Scan result at index %d: expected term '%s', got '%s'", i, expectedTerms[i], res.Term)
		}
	}
}
