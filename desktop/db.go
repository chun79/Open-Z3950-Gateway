package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SavedBook struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	ISBN      string    `json:"isbn"`
	SourceDB  string    `json:"source_db"`
	Notes     string    `json:"notes"`
	SavedAt   time.Time `json:"saved_at"`
}

type ILLRequest struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	ISBN      string    `json:"isbn"`
	TargetDB  string    `json:"target_db"`
	Status    string    `json:"status"` // pending, approved, rejected
	Comments  string    `json:"comments"`
	Requestor string    `json:"requestor"`
	CreatedAt time.Time `json:"created_at"`
}

type DBManager struct {
	db *sql.DB
}

func NewDBManager() (*DBManager, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	appDir := filepath.Join(configDir, "OpenZ3950Desktop")
	os.MkdirAll(appDir, 0755)
	
	dbPath := filepath.Join(appDir, "books.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Init Schema
	queries := []string{
		`CREATE TABLE IF NOT EXISTS saved_books (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			author TEXT,
			isbn TEXT,
			source_db TEXT,
			notes TEXT,
			saved_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS ill_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			author TEXT,
			isbn TEXT,
			target_db TEXT,
			status TEXT DEFAULT 'pending',
			comments TEXT,
			requestor TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS search_history (
			term TEXT PRIMARY KEY,
			saved_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}
	
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return nil, err
		}
	}

	return &DBManager{db: db}, nil
}

func (m *DBManager) SaveBook(book SearchResult, notes string) (int64, error) {
	res, err := m.db.Exec("INSERT INTO saved_books (title, author, isbn, source_db, notes) VALUES (?, ?, ?, ?, ?)",
		book.Title, book.Author, book.ISBN, book.SourceDB, notes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (m *DBManager) ListBooks() ([]SavedBook, error) {
	rows, err := m.db.Query("SELECT id, title, author, isbn, source_db, notes, saved_at FROM saved_books ORDER BY saved_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []SavedBook
	for rows.Next() {
		var b SavedBook
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.ISBN, &b.SourceDB, &b.Notes, &b.SavedAt); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, nil
}

func (m *DBManager) DeleteBook(id int64) error {
	_, err := m.db.Exec("DELETE FROM saved_books WHERE id = ?", id)
	return err
}

func (m *DBManager) SaveILLRequest(req ILLRequest) error {
	_, err := m.db.Exec(`INSERT INTO ill_requests 
		(title, author, isbn, target_db, status, comments, requestor) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		req.Title, req.Author, req.ISBN, req.TargetDB, "pending", req.Comments, "me")
	return err
}

func (m *DBManager) ListILLRequests() ([]ILLRequest, error) {
	rows, err := m.db.Query("SELECT id, title, author, isbn, target_db, status, comments, requestor, created_at FROM ill_requests ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reqs []ILLRequest
	for rows.Next() {
		var r ILLRequest
		if err := rows.Scan(&r.ID, &r.Title, &r.Author, &r.ISBN, &r.TargetDB, &r.Status, &r.Comments, &r.Requestor, &r.CreatedAt); err != nil {
			return nil, err
		}
		reqs = append(reqs, r)
	}
	return reqs, nil
}

func (m *DBManager) SaveSearchHistory(term string) error {
	if term == "" { return nil }
	_, err := m.db.Exec("INSERT OR REPLACE INTO search_history (term, saved_at) VALUES (?, CURRENT_TIMESTAMP)", term)
	return err
}

func (m *DBManager) ListSearchHistory() ([]string, error) {
	rows, err := m.db.Query("SELECT term FROM search_history ORDER BY saved_at DESC LIMIT 10")
	if err != nil { return nil, err }
	defer rows.Close()
	var terms []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err == nil {
			terms = append(terms, t)
		}
	}
	return terms, nil
}

func (m *DBManager) DeleteILLRequest(id int64) error {
	_, err := m.db.Exec("DELETE FROM ill_requests WHERE id = ?", id)
	return err
}

func (m *DBManager) ClearSearchHistory() error {
	_, err := m.db.Exec("DELETE FROM search_history")
	return err
}