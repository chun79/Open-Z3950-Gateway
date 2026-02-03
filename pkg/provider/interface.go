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

// ScanResult 代表浏览结果
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

	Status    string `json:"status"` // e.g., "pending", "approved", "rejected"

	Requestor string `json:"requestor"` // User ID or Name

	Comments  string `json:"comments"` // User comments/notes
}



type User struct {

	ID           int64  `json:"id"`

		Username     string `json:"username"`

		PasswordHash string `json:"-"` // Never export password hash

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

	

	type Provider interface {

		// Search now accepts a StructuredQuery from the z3950 package.

		Search(db string, query z3950.StructuredQuery) ([]string, error)

	

			Fetch(db string, ids []string) ([]*z3950.MARCRecord, error)

	

		

	

			// Scan 浏览索引

	

			Scan(db, field, startTerm string) ([]ScanResult, error)

	

		

	

				// CreateILLRequest creates a new Inter-Library Loan request.
			
				CreateILLRequest(req ILLRequest) error
			
				// GetILLRequest retrieves a single ILL request by ID.
				GetILLRequest(id int64) (*ILLRequest, error)
			
				// ListILLRequests retrieves all Inter-Library Loan requests.
			
				ListILLRequests() ([]ILLRequest, error)
	

		// UpdateILLRequestStatus updates the status of an ILL request.

		UpdateILLRequestStatus(id int64, status string) error

	

		// User Management

		CreateUser(user *User) error

		GetUserByUsername(username string) (*User, error)

	

		// Target Management (Dynamic Z39.50 Targets)

		CreateTarget(target *Target) error

		ListTargets() ([]Target, error)

		DeleteTarget(id int64) error

		GetTargetByName(name string) (*Target, error)

	}

	
