package index

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
)

// BookDocument represents the searchable fields of a book
type BookDocument struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	ISBN      string `json:"isbn"`
	Publisher string `json:"publisher"`
	Subject   string `json:"subject"`
	Notes     string `json:"notes"`
	Year      string `json:"year"`
}

type Manager struct {
	index bleve.Index
	path  string
}

// NewManager opens or creates a new Bleve index at the specified path
func NewManager(path string) (*Manager, error) {
	var index bleve.Index
	var err error

	// Check if index exists
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		// Create new mapping
		mapping := bleve.NewIndexMapping()
		
		// Configure default analyzer to "standard" (removes stopwords, lowercases)
		mapping.DefaultAnalyzer = standard.Name
		
		index, err = bleve.New(path, mapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
		slog.Info("created new bleve index", "path", path)
	} else {
		index, err = bleve.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open index: %w", err)
		}
		slog.Info("opened existing bleve index", "path", path)
	}

	return &Manager{
		index: index,
		path:  path,
	}, nil
}

func (m *Manager) Close() error {
	return m.index.Close()
}

// IndexBook adds or updates a book in the index
func (m *Manager) IndexBook(doc BookDocument) error {
	slog.Info("indexing book", "id", doc.ID, "title", doc.Title)
	return m.index.Index(doc.ID, doc)
}

// DeleteBook removes a book from the index
func (m *Manager) DeleteBook(id string) error {
	return m.index.Delete(id)
}

// Search performs a full-text search and returns matching IDs with scores
type SearchHit struct {
	ID    string
	Score float64
}

func (m *Manager) Search(queryStr string) ([]SearchHit, error) {
	// Create a MatchQuery (simple) or QueryStringQuery (advanced syntax like "title:go")
	query := bleve.NewQueryStringQuery(queryStr)
	
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = 50 // Default limit
	
	// Execute search
	searchResult, err := m.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	var hits []SearchHit
	for _, hit := range searchResult.Hits {
		hits = append(hits, SearchHit{
			ID:    hit.ID,
			Score: hit.Score,
		})
	}
	
	slog.Info("search executed", "query", queryStr, "hits", searchResult.Total, "took", searchResult.Took)
	return hits, nil
}

// Count returns the total number of indexed documents
func (m *Manager) Count() (uint64, error) {
	return m.index.DocCount()
}
