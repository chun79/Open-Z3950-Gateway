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
