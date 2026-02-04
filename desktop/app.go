package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

// App struct
type App struct {
	ctx    context.Context
	config *ConfigManager
	db     *DBManager
}

// NewApp creates a new App application struct
func NewApp() *App {
	cm, err := NewConfigManager()
	if err != nil {
		fmt.Printf("Error initializing config: %v\n", err)
	}
	
	db, err := NewDBManager()
	if err != nil {
		fmt.Printf("Error initializing db: %v\n", err)
	}

	return &App{
		config: cm,
		db:     db,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) SaveBook(book SearchResult) error {
	if a.db == nil { return fmt.Errorf("db not initialized") }
	return a.db.SaveBook(book, "")
}

func (a *App) ListSavedBooks() []SavedBook {
	if a.db == nil { return []SavedBook{} }
	list, _ := a.db.ListBooks()
	return list
}

func (a *App) DeleteSavedBook(id int64) error {
	if a.db == nil { return fmt.Errorf("db not initialized") }
	return a.db.DeleteBook(id)
}

func (a *App) ListSearchHistory() []string {
	if a.db == nil { return []string{} }
	list, _ := a.db.ListSearchHistory()
	return list
}

func (a *App) ClearSearchHistory() error {
	if a.db == nil { return fmt.Errorf("db not initialized") }
	return a.db.ClearSearchHistory()
}

// ExportBookshelf exports all saved books to a CSV file selected by the user
func (a *App) ExportBookshelf() error {
	if a.db == nil { return fmt.Errorf("db not initialized") }
	
	books, err := a.db.ListBooks()
	if err != nil { return err }
	if len(books) == 0 { return fmt.Errorf("no books to export") }

	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Export Bookshelf",
		DefaultFilename: "my_bookshelf.csv",
		Filters: []runtime.FileFilter{
			{ DisplayName: "CSV Files (*.csv)", Pattern: "*.csv" },
		},
	})
	if err != nil || filePath == "" { return err }

	file, err := os.Create(filePath)
	if err != nil { return err }
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Header
	writer.Write([]string{"ID", "Title", "Author", "ISBN", "Source DB", "Notes", "Saved At"})

	for _, b := range books {
		writer.Write([]string{
			fmt.Sprintf("%d", b.ID),
			b.Title,
			b.Author,
			b.ISBN,
			b.SourceDB,
			b.Notes,
			b.SavedAt.String(),
		})
	}

	return nil
}

// ListTargets returns the list of configured Z39.50 targets
func (a *App) ListTargets() []Target {
	if a.config == nil { return []Target{} }
	return a.config.ListTargets()
}

// AddTarget saves a new target
func (a *App) AddTarget(t Target) error {
	if a.config == nil { return fmt.Errorf("config not initialized") }
	return a.config.AddTarget(t)
}

// DeleteTarget removes a target by name
func (a *App) DeleteTarget(name string) error {
	if a.config == nil { return fmt.Errorf("config not initialized") }
	return a.config.DeleteTarget(name)
}

// TestResult mirrors the test connection response
type TestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// TestTarget tests connectivity to a Z39.50 server
func (a *App) TestTarget(host string, port int) TestResult {
	client := z3950.NewClient(host, port)
	if err := client.Connect(); err != nil {
		return TestResult{Success: false, Message: fmt.Sprintf("Connection failed: %v", err)}
	}
	defer client.Close()

	if err := client.Init(); err != nil {
		return TestResult{Success: false, Message: fmt.Sprintf("Init failed: %v", err)}
	}

	return TestResult{Success: true, Message: "Connection successful!"}
}

// SearchParams mirrors the frontend search request
type SearchParams struct {
	DBs  []string `json:"dbs"` // Changed from DB string to DBs array
	Term string   `json:"term"`
	Attr int      `json:"attr"`
}

// SearchResult mirrors the frontend book type
type SearchResult struct {
	SourceDB string `json:"source_db"` // Added source
	Title    string `json:"title"`
	Author   string `json:"author"`
	ISBN     string `json:"isbn"`
}

// BookDetail is a comprehensive struct for frontend
type BookDetail struct {
	Title     string              `json:"title"`
	Author    string              `json:"author"`
	ISBN      string              `json:"isbn"`
	Publisher string              `json:"publisher"`
	Edition   string              `json:"edition"`
	Summary   string              `json:"summary"`
	TOC       string              `json:"toc"`
	Physical  string              `json:"physical"`
	Series    string              `json:"series"`
	Notes     string              `json:"notes"`
	Fields    []z3950.MARCField   `json:"fields"`
	Holdings  []z3950.Holding     `json:"holdings"`
}

// GetBookDetails fetches a single record from a Z39.50 target
// We use ISBN or Title+Author search because standard Z39.50 doesn't always support recordID fetch directly across all servers easily
// But for "Fetch" we usually need a Result Set.
// Simpler approach for Desktop: Re-search by ISBN (precise) to get the full record.
// params: dbName, isbn (or title if isbn missing)
func (a *App) GetBookDetails(dbName string, query string, queryType string) (BookDetail, error) {
	target, found := a.config.GetTarget(dbName)
	if !found {
		// Try default fallback
		if dbName == "Library of Congress" {
			target = Target{Host: "lx2.loc.gov", Port: 210, DB: "LCDB", Encoding: "MARC21"}
		} else {
			return BookDetail{}, fmt.Errorf("target not found: %s", dbName)
		}
	}

	client := z3950.NewClient(target.Host, target.Port)
	if err := client.Connect(); err != nil {
		return BookDetail{}, err
	}
	defer client.Close()

	if err := client.Init(); err != nil {
		return BookDetail{}, err
	}

	// Determine search attribute
	attr := 4 // Title
	if queryType == "isbn" {
		attr = 7 // ISBN
	}

	zQuery := z3950.StructuredQuery{
		Root: z3950.QueryClause{Attribute: attr, Term: query},
	}

	count, err := client.StructuredSearch(target.DB, zQuery)
	if err != nil {
		return BookDetail{}, err
	}

	if count == 0 {
		return BookDetail{}, fmt.Errorf("book not found")
	}

	// Fetch the first record
	recs, err := client.Present(1, 1, z3950.OID_MARC21)
	if err != nil {
		return BookDetail{}, err
	}
	if len(recs) == 0 {
		return BookDetail{}, fmt.Errorf("failed to retrieve record")
	}

	r := recs[0]
	return BookDetail{
		Title:     r.GetTitle(nil),
		Author:    r.GetAuthor(nil),
		ISBN:      r.GetISBN(nil),
		Publisher: r.GetPublisher(nil),
		Edition:   r.Edition,
		Summary:   r.Summary,
		TOC:       r.TOC,
		Physical:  r.PhysicalDescription,
		Series:    r.Series,
		Notes:     r.Notes,
		Fields:    r.Fields,
		Holdings:  r.Holdings,
	}, nil
}

// RequestILL saves an ILL request to local DB
func (a *App) RequestILL(req ILLRequest) error {
	if a.db == nil { return fmt.Errorf("db not initialized") }
	return a.db.SaveILLRequest(req)
}

// ListILLRequests returns all local ILL requests
func (a *App) ListILLRequests() []ILLRequest {
	if a.db == nil { return []ILLRequest{} }
	list, _ := a.db.ListILLRequests()
	return list
}

func (a *App) DeleteILLRequest(id int64) error {
	if a.db == nil { return fmt.Errorf("db not initialized") }
	return a.db.DeleteILLRequest(id)
}

// Search performs a Z39.50 search concurrently
func (a *App) Search(params SearchParams) ([]SearchResult, error) {
	if a.db != nil && params.Term != "" {
		a.db.SaveSearchHistory(params.Term)
	}
	if len(params.DBs) == 0 {
		return nil, fmt.Errorf("no targets selected")
	}

	var wg sync.WaitGroup
	resultsChan := make(chan []SearchResult, len(params.DBs))
	errChan := make(chan error, len(params.DBs))

	attr := 4 // Title
	if params.Attr != 0 { attr = params.Attr }

	// Concurrently search each target
	for _, targetName := range params.DBs {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			
			// Resolve target
			target, found := a.config.GetTarget(name)
			if !found {
				// Fallback for default names if not in config
				if name == "Library of Congress" {
					target = Target{Host: "lx2.loc.gov", Port: 210, DB: "LCDB", Encoding: "MARC21"}
				} else if name == "Oxford University" {
					target = Target{Host: "library.ox.ac.uk", Port: 210, DB: "MAIN_BIB", Encoding: "MARC21"}
				} else {
					slog.Warn("target not found", "name", name)
					return
				}
			}

			client := z3950.NewClient(target.Host, target.Port)
			if err := client.Connect(); err != nil {
				slog.Error("connect failed", "target", name, "error", err)
				return
			}
			defer client.Close()

			if err := client.Init(); err != nil {
				slog.Error("init failed", "target", name, "error", err)
				return
			}

			query := z3950.StructuredQuery{
				Root: z3950.QueryClause{Attribute: attr, Term: params.Term},
			}

			count, err := client.StructuredSearch(target.DB, query)
			if err != nil {
				slog.Error("search failed", "target", name, "error", err)
				return
			}

			if count > 0 {
				limit := 10
				if count < limit { limit = count }
				recs, err := client.Present(1, limit, z3950.OID_MARC21)
				if err != nil {
					return
				}

				var localResults []SearchResult
				for _, r := range recs {
					localResults = append(localResults, SearchResult{
						SourceDB: name,
						Title:    r.GetTitle(nil),
						Author:   r.GetAuthor(nil),
						ISBN:     r.GetISBN(nil),
					})
				}
				resultsChan <- localResults
			}
		}(targetName)
	}

	wg.Wait()
	close(resultsChan)
	close(errChan)

	var allResults []SearchResult
	for res := range resultsChan {
		allResults = append(allResults, res...)
	}

	return allResults, nil
}
