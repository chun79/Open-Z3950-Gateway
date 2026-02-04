package provider

import (
	"strings"

	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

type HybridProvider struct {
	local  Provider
	proxy  *ProxyProvider
}

func NewHybridProvider(local Provider) *HybridProvider {
	return &HybridProvider{
		local: local,
		proxy: NewProxyProvider(local), // Pass local as resolver
	}
}

func (h *HybridProvider) isLocalDB(db string) bool {
	return strings.EqualFold(db, "Default") || strings.EqualFold(db, "Local") || db == ""
}

func (h *HybridProvider) Search(db string, query z3950.StructuredQuery) ([]string, error) {
	if h.isLocalDB(db) {
		return h.local.Search(db, query)
	}
	// Check if target exists
	if _, err := h.local.GetTargetByName(db); err == nil {
		return h.proxy.Search(db, query)
	}
	return nil, nil // Or error?
}

func (h *HybridProvider) Fetch(db string, ids []string) ([]*z3950.MARCRecord, error) {
	if h.isLocalDB(db) {
		return h.local.Fetch(db, ids)
	}
	return h.proxy.Fetch(db, ids)
}

func (h *HybridProvider) CreateRecord(db string, record *z3950.MARCRecord) (string, error) {
	if h.isLocalDB(db) {
		return h.local.CreateRecord(db, record)
	}
	return "", fmt.Errorf("cannot create record in remote database")
}

func (h *HybridProvider) UpdateRecord(db string, id string, record *z3950.MARCRecord) error {
	if h.isLocalDB(db) {
		return h.local.UpdateRecord(db, id, record)
	}
	return fmt.Errorf("cannot update record in remote database")
}

func (h *HybridProvider) CreateItem(bibID string, item Item) error { return h.local.CreateItem(bibID, item) }
func (h *HybridProvider) GetItems(bibID string) ([]Item, error) { return h.local.GetItems(bibID) }
func (h *HybridProvider) GetItemByBarcode(barcode string) (*Item, error) { return h.local.GetItemByBarcode(barcode) }
func (h *HybridProvider) Checkout(itemBarcode, patronID string) (string, error) { return h.local.Checkout(itemBarcode, patronID) }
func (h *HybridProvider) Checkin(itemBarcode string) (float64, error) { return h.local.Checkin(itemBarcode) }

func (h *HybridProvider) GetDashboardStats() (map[string]interface{}, error) {
	return h.local.GetDashboardStats()
}

func (h *HybridProvider) GetNewArrivals(limit int) ([]SearchResult, error) { return h.local.GetNewArrivals(limit) }
func (h *HybridProvider) GetPopularBooks(limit int) ([]SearchResult, error) { return h.local.GetPopularBooks(limit) }
func (h *HybridProvider) RenewLoan(loanID int64) (string, error) { return h.local.RenewLoan(loanID) }
func (h *HybridProvider) PlaceHold(bibID string, patronID string) error { return h.local.PlaceHold(bibID, patronID) }
func (h *HybridProvider) GetPatronLoans(patronID string) ([]map[string]interface{}, error) { return h.local.GetPatronLoans(patronID) }

func (h *HybridProvider) Scan(db, field, startTerm string, opts z3950.ScanOptions) ([]ScanResult, error) {
	if h.isLocalDB(db) {
		return h.local.Scan(db, field, startTerm, opts)
	}
	return h.proxy.Scan(db, field, startTerm, opts)
}

// ILL operations ALWAYS go to local storage
func (h *HybridProvider) CreateILLRequest(req ILLRequest) error {
	return h.local.CreateILLRequest(req)
}

func (h *HybridProvider) GetILLRequest(id int64) (*ILLRequest, error) {
	return h.local.GetILLRequest(id)
}

func (h *HybridProvider) ListILLRequests() ([]ILLRequest, error) {
	return h.local.ListILLRequests()
}

func (h *HybridProvider) UpdateILLRequestStatus(id int64, status string) error {
	return h.local.UpdateILLRequestStatus(id, status)
}

// User operations ALWAYS go to local storage
func (h *HybridProvider) CreateUser(user *User) error {
	return h.local.CreateUser(user)
}

func (h *HybridProvider) GetUserByUsername(username string) (*User, error) {
	return h.local.GetUserByUsername(username)
}

// Target operations go to local storage
func (h *HybridProvider) CreateTarget(target *Target) error {
	return h.local.CreateTarget(target)
}

func (h *HybridProvider) ListTargets() ([]Target, error) {
	return h.local.ListTargets()
}

func (h *HybridProvider) DeleteTarget(id int64) error {
	return h.local.DeleteTarget(id)
}

func (h *HybridProvider) GetTargetByName(name string) (*Target, error) {
	return h.local.GetTargetByName(name)
}
