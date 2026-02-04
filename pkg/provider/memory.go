package provider

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

type MemoryProvider struct {
	mu          sync.RWMutex
	books       []SearchResult
	illRequests []ILLRequest
	users       []User
	targets     []Target
}

func NewMemoryProvider() *MemoryProvider {
	// Generate hash for "admin"
	adminHash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)

	return &MemoryProvider{
		books: []SearchResult{
			{ID: "1", Title: "Thinking in Go", Author: "Rob Pike", ISBN: "0201548550", Publisher: "Addison-Wesley", PubYear: "2012", Subject: "Programming"},
			{ID: "2", Title: "Z39.50 for Dummies", Author: "Index Data", ISBN: "1234567890", Publisher: "Dummy Press", PubYear: "1999", Subject: "Library Science", ISSN: "1234-5678"},
			{ID: "3", Title: "The Art of Protocol", Author: "Cerf & Kahn", ISBN: "0987654321", Publisher: "Network Books", PubYear: "1985", Subject: "Networking"},
			{ID: "4", Title: "SaaS Architecture", Author: "Gemini", ISBN: "9999999999", Publisher: "Cloud Pub", PubYear: "2025", Subject: "Cloud Computing"},
		},
		illRequests: []ILLRequest{},
		users: []User{
			{ID: 1, Username: "admin", PasswordHash: string(adminHash), Role: "admin"},
		},
		targets: []Target{
			{ID: 1, Name: "LCDB", Host: "lx2.loc.gov", Port: 210, DatabaseName: "LCDB", Encoding: "MARC21"},
			{ID: 2, Name: "Oxford", Host: "z3950.ox.ac.uk", Port: 210, DatabaseName: "OLIS", Encoding: "MARC21"},
			{ID: 3, Name: "Harvard", Host: "hollis.harvard.edu", Port: 210, DatabaseName: "Hollie", Encoding: "MARC21"},
			{ID: 4, Name: "Yale", Host: "orbis.library.yale.edu", Port: 210, DatabaseName: "Orbis", Encoding: "MARC21"},
		},
	}
}

func (m *MemoryProvider) AddBook(title, author, isbn, publisher, pubYear, issn, subject string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("%d", len(m.books)+1)
	m.books = append(m.books, SearchResult{
		ID: id, Title: title, Author: author, ISBN: isbn, 
		Publisher: publisher, PubYear: pubYear, ISSN: issn, Subject: subject,
	})
}

// evaluateQuery recursively checks if a book matches the query tree
func evaluateQuery(node z3950.QueryNode, book SearchResult) bool {
	if node == nil {
		return false
	}

	switch n := node.(type) {
	case z3950.QueryClause:
		term := strings.ToLower(n.Term)
		switch n.Attribute {
		case z3950.UseAttributeTitle:
			return strings.Contains(strings.ToLower(book.Title), term)
		case z3950.UseAttributeAuthor:
			return strings.Contains(strings.ToLower(book.Author), term)
		case z3950.UseAttributeISBN:
			return strings.Contains(book.ISBN, CleanISBN(n.Term))
		case z3950.UseAttributeISSN:
			return strings.Contains(book.ISSN, term)
		case z3950.UseAttributeSubject:
			return strings.Contains(strings.ToLower(book.Subject), term)
		case z3950.UseAttributeDatePub:
			return strings.Contains(book.PubYear, term)
		default:
			// Broad search
			return strings.Contains(strings.ToLower(book.Title), term) || strings.Contains(strings.ToLower(book.Author), term)
		}
	case z3950.QueryComplex:
		l := evaluateQuery(n.Left, book)
		r := evaluateQuery(n.Right, book)
		switch n.Operator {
		case "AND": return l && r
		case "OR": return l || r
		case "AND-NOT": return l && !r
		}
	}
	return false
}

func (m *MemoryProvider) Search(db string, query z3950.StructuredQuery) ([]string, error) {
	if query.Root == nil {
		return nil, nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var matchingIds []string
	for _, book := range m.books {
		if evaluateQuery(query.Root, book) {
			matchingIds = append(matchingIds, book.ID)
		}
	}

	// Apply pagination
	offset := 0
	if query.Offset > 0 {
		offset = query.Offset
	}

	limit := len(matchingIds)
	if query.Limit > 0 {
		limit = query.Limit
	}

	start := offset
	if start > len(matchingIds) {
		start = len(matchingIds)
	}

	end := start + limit
	if end > len(matchingIds) {
		end = len(matchingIds)
	}

	return matchingIds[start:end], nil
}


func (m *MemoryProvider) Fetch(db string, ids []string) ([]*z3950.MARCRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var records []*z3950.MARCRecord
	for _, id := range ids {
		for _, book := range m.books {
			if book.ID == id {
				rawBytes := z3950.BuildMARC(nil, book.ID, book.Title, book.Author, book.ISBN, book.Publisher, book.PubYear, book.ISSN, book.Subject)
				rec, err := z3950.ParseMARC(rawBytes)
				if err == nil {
					records = append(records, rec)
				}
				break
			}
		}
	}
	return records, nil
}

func (m *MemoryProvider) Scan(db, field, startTerm string, opts z3950.ScanOptions) ([]ScanResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var terms []string
	for _, b := range m.books {
		terms = append(terms, b.Title)
	}
	sort.Strings(terms)
	
	var results []ScanResult
	start := strings.ToLower(startTerm)
	count := 0
	
	for _, t := range terms {
		if count >= 10 { break }
		if strings.ToLower(t) >= start {
			results = append(results, ScanResult{Term: t, Count: 1})
			count++
		}
	}
	return results, nil
}

func (m *MemoryProvider) CreateILLRequest(req ILLRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	req.ID = int64(len(m.illRequests) + 1)
	m.illRequests = append(m.illRequests, req)
	return nil
}

func (m *MemoryProvider) GetILLRequest(id int64) (*ILLRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, req := range m.illRequests {
		if req.ID == id {
			// Return a copy
			r := req
			return &r, nil
		}
	}
	return nil, fmt.Errorf("request not found")
}

func (m *MemoryProvider) ListILLRequests() ([]ILLRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	requests := make([]ILLRequest, len(m.illRequests))
	copy(requests, m.illRequests)
	return requests, nil
}

func (m *MemoryProvider) UpdateILLRequestStatus(id int64, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, req := range m.illRequests {
		if req.ID == id {
			m.illRequests[i].Status = status
			return nil
		}
	}
	return fmt.Errorf("request with id %d not found", id)
}

func (m *MemoryProvider) CreateRecord(db string, record *z3950.MARCRecord) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("%d", len(m.books)+1)
	m.books = append(m.books, SearchResult{
		ID: id, Title: record.Title, Author: record.Author, ISBN: record.ISBN,
		Publisher: record.Publisher, PubYear: record.GetPubYear(nil), Subject: record.Subject,
	})
	return id, nil
}

func (m *MemoryProvider) UpdateRecord(db string, id string, record *z3950.MARCRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, b := range m.books {
		if b.ID == id {
			m.books[i] = SearchResult{
				ID: id, Title: record.Title, Author: record.Author, ISBN: record.ISBN,
				Publisher: record.Publisher, PubYear: record.GetPubYear(nil), Subject: record.Subject,
			}
			return nil
		}
	}
	return fmt.Errorf("record not found")
}

func (m *MemoryProvider) CreateItem(bibID string, item Item) error { return nil }
func (m *MemoryProvider) GetItems(bibID string) ([]Item, error) { return []Item{}, nil }
func (m *MemoryProvider) GetItemByBarcode(barcode string) (*Item, error) { return nil, fmt.Errorf("not found") }
func (m *MemoryProvider) Checkout(itemBarcode, patronID string) (string, error) { return "", fmt.Errorf("not implemented") }
func (m *MemoryProvider) Checkin(itemBarcode string) (float64, error) { return 0, fmt.Errorf("not implemented") }

func (m *MemoryProvider) CreateUser(user *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	user.ID = int64(len(m.users) + 1)
	m.users = append(m.users, *user)
	return nil
}

func (m *MemoryProvider) GetUserByUsername(username string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range m.users {
		if u.Username == username {
			// Return a copy
			return &u, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (m *MemoryProvider) CreateTarget(target *Target) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target.ID = int64(len(m.targets) + 1)
	m.targets = append(m.targets, *target)
	return nil
}

func (m *MemoryProvider) ListTargets() ([]Target, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]Target, len(m.targets))
	copy(list, m.targets)
	return list, nil
}

func (m *MemoryProvider) DeleteTarget(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.targets {
		if t.ID == id {
			m.targets = append(m.targets[:i], m.targets[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("target not found")
}

func (m *MemoryProvider) GetTargetByName(name string) (*Target, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.targets {
		if strings.EqualFold(t.Name, name) {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("target not found")
}