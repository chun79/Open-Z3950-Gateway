package provider

import "github.com/yourusername/open-z3950-gateway/pkg/z3950"

type SearchResult struct {
	ID        string
	Title     string
	Author    string
	ISBN      string
	ISSN      string
	Subject   string
	Publisher string
	PubYear   string
}

// ScanResult represents an index browsing result
type ScanResult struct {
	Term  string `json:"term"`
	Count int    `json:"count"`
}

// ILLRequest represents an Inter-Library Loan request.
type ILLRequest struct {
	ID        int64  `json:"id"`
	TargetDB  string `json:"target_db"`
	RecordID  string `json:"record_id"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	ISBN      string `json:"isbn"`
	Status    string `json:"status"`    // e.g., "pending", "approved", "rejected"
	Requestor string `json:"requestor"` // User ID or Name
	Comments  string `json:"comments"`  // User comments/notes
}

type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`    // Never export password hash
	Role         string `json:"role"` // "admin", "user"
}

type Target struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	DatabaseName string `json:"database_name"`
	Encoding     string `json:"encoding"`      // "MARC21", "UNIMARC"
	AuthUser     string `json:"auth_user"`     // Optional
	AuthPass     string `json:"auth_password"` // Optional
}

type Item struct {
	ID         int64  `json:"id"`
	BibID      int64  `json:"bib_id"`
	Barcode    string `json:"barcode"`
	CallNumber string `json:"call_number"`
	Status     string `json:"status"` // "Available", "Checked Out"
	Location   string `json:"location"`
}

type Provider interface {
	// Search performs a Z39.50 search.
	Search(db string, query z3950.StructuredQuery) ([]string, error)

	// Fetch retrieves full records by their local or session IDs.
	Fetch(db string, ids []string) ([]*z3950.MARCRecord, error)

	// --- Cataloging Operations (LSP Core) ---

	// CreateRecord saves a new MARC record to the local database.
	CreateRecord(db string, record *z3950.MARCRecord) (string, error)

	// UpdateRecord modifies an existing local MARC record.
	UpdateRecord(db string, id string, record *z3950.MARCRecord) error

	// --- Item Management (Circulation Base) ---

	CreateItem(bibID string, item Item) error
	GetItems(bibID string) ([]Item, error)
	GetItemByBarcode(barcode string) (*Item, error)

	// --- Circulation Operations ---
	
	// Checkout performs a loan transaction. Returns due date.
	Checkout(itemBarcode, patronID string) (string, error)
	
	// Checkin returns an item. Returns fine amount if overdue.
	Checkin(itemBarcode string) (float64, error)

	// --- Index Browsing ---

	// Scan browses the index.
	// opts allows setting count, step size, and position.
	Scan(db, field, startTerm string, opts z3950.ScanOptions) ([]ScanResult, error)

	// --- ILL Management ---

	// CreateILLRequest creates a new Inter-Library Loan request.
	CreateILLRequest(req ILLRequest) error
	// GetILLRequest retrieves a single ILL request by ID.
	GetILLRequest(id int64) (*ILLRequest, error)
	// ListILLRequests retrieves all Inter-Library Loan requests.
	ListILLRequests() ([]ILLRequest, error)
	// UpdateILLRequestStatus updates the status of an ILL request.
	UpdateILLRequestStatus(id int64, status string) error

	// --- User Management ---

	CreateUser(user *User) error
	GetUserByUsername(username string) (*User, error)

	// --- Target Management ---

	CreateTarget(target *Target) error
	ListTargets() ([]Target, error)
	DeleteTarget(id int64) error
	GetTargetByName(name string) (*Target, error)
}