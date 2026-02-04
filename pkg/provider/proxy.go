package provider

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

// friendlyError maps technical errors to user-friendly messages
func friendlyError(target string, action string, err error) error {
	msg := err.Error()
	friendly := msg

	if strings.Contains(msg, "i/o timeout") {
		friendly = fmt.Sprintf("Connection to %s timed out.", target)
	} else if strings.Contains(msg, "connection refused") {
		friendly = fmt.Sprintf("%s server refused the connection.", target)
	} else if strings.Contains(msg, "no such host") {
		friendly = fmt.Sprintf("Could not resolve hostname for %s.", target)
	} else if strings.Contains(msg, "server rejected connection") {
		friendly = fmt.Sprintf("%s rejected the connection (Invalid credentials/options).", target)
	} else if strings.Contains(msg, "reset by peer") {
		friendly = fmt.Sprintf("%s closed the connection unexpectedly.", target)
	}

	// Log original error for debugging but return friendly one
	slog.Error("Z39.50 Error", "target", target, "action", action, "original_error", err)
	return errors.New(friendly)
}

// TargetConfig holds connection details for a remote Z39.50 server
type TargetConfig struct {
	Host         string
	Port         int
	DatabaseName string
	Encoding     string // "MARC21", "UNIMARC", "SUTRS"
}

type TargetResolver interface {
	GetTargetByName(name string) (*Target, error)
}

type ProxyProvider struct {
	resolver   TargetResolver
	queryCache sync.Map
}

func NewProxyProvider(resolver TargetResolver) *ProxyProvider {
	return &ProxyProvider{
		resolver: resolver,
	}
}

// connectToTarget connects and initializes a session
func (p *ProxyProvider) connectToTarget(targetName string) (*z3950.Client, TargetConfig, error) {
	// Resolve target from DB
	t, err := p.resolver.GetTargetByName(targetName)
	if err != nil {
		return nil, TargetConfig{}, fmt.Errorf("unknown target: %s", targetName)
	}

	config := TargetConfig{
		Host:         t.Host,
		Port:         t.Port,
		DatabaseName: t.DatabaseName,
		Encoding:     t.Encoding,
	}

	client := z3950.NewClient(config.Host, config.Port)
	if err := client.Connect(); err != nil {
		return nil, config, friendlyError(targetName, "connect", err)
	}

	if err := client.Init(); err != nil {
		client.Close()
		return nil, config, friendlyError(targetName, "init", err)
	}

	return client, config, nil
}

// executeRemoteSearch connects, initializes, searches, and returns the client, count AND config.
func (p *ProxyProvider) executeRemoteSearch(targetName string, query z3950.StructuredQuery) (*z3950.Client, int, TargetConfig, error) {
	client, config, err := p.connectToTarget(targetName)
	if err != nil {
		return nil, 0, config, err
	}

	count, err := client.StructuredSearch(config.DatabaseName, query)
	if err != nil {
		client.Close()
		return nil, 0, config, friendlyError(targetName, "search", err)
	}

	// Perform Sort if requested
	if len(query.SortKeys) > 0 && count > 0 {
		if err := client.Sort("default", query.SortKeys); err != nil {
			slog.Warn("sort failed", "target", targetName, "error", err)
			// Don't fail the search, just log warning
		}
	}

	return client, count, config, nil
}

func (p *ProxyProvider) Search(db string, query z3950.StructuredQuery) ([]string, error) {
	client, count, _, err := p.executeRemoteSearch(db, query)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if count > 20 {
		count = 20
	}

	// Generate a unique session ID for this search result set
	sessionID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(100000))
	p.queryCache.Store(sessionID, query)

	ids := make([]string, count)
	for i := 0; i < count; i++ {
		// Return IDs in format "sessionID:index"
		ids[i] = fmt.Sprintf("%s:%d", sessionID, i+1)
	}

	return ids, nil
}

func (p *ProxyProvider) Fetch(db string, ids []string) ([]*z3950.MARCRecord, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Parse the first ID to get the sessionID
	parts := strings.Split(ids[0], ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid id format")
	}
	sessionID := parts[0]

	val, ok := p.queryCache.Load(sessionID)
	if !ok {
		return nil, fmt.Errorf("session expired or unknown query for db: %s", db)
	}
	query := val.(z3950.StructuredQuery)

	client, _, config, err := p.executeRemoteSearch(db, query)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// Determine Syntax OID
	syntaxOID := z3950.OID_MARC21
	if config.Encoding == "UNIMARC" {
		syntaxOID = z3950.OID_UNIMARC
	} else if config.Encoding == "SUTRS" {
		syntaxOID = z3950.OID_SUTRS
	}

	var records []*z3950.MARCRecord
	for _, id := range ids {
		parts := strings.Split(id, ":")
		if len(parts) != 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}

		recs, err := client.Present(idx, 1, syntaxOID)
		if err != nil {
			slog.Warn("failed to fetch record", "db", db, "index", idx, "error", err)
			continue
		}
		if len(recs) > 0 {
			records = append(records, recs[0])
		}
	}

	return records, nil
}

func (p *ProxyProvider) Scan(db, field, startTerm string, opts z3950.ScanOptions) ([]ScanResult, error) {
	client, config, err := p.connectToTarget(db)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// Map field string to Bib-1 Use Attribute
	if opts.Attributes == nil {
		opts.Attributes = make(map[int]int)
	}
	// Only set attribute if not already provided in opts
	if _, ok := opts.Attributes[1]; !ok {
		switch field {
		case "author":
			opts.Attributes[1] = 1003
		case "subject":
			opts.Attributes[1] = 21
		case "isbn":
			opts.Attributes[1] = 7
		case "issn":
			opts.Attributes[1] = 8
		case "title":
			opts.Attributes[1] = 4
		default:
			opts.Attributes[1] = 4 // Default to Title
		}
	}

	entries, err := client.ScanWithOpts(config.DatabaseName, startTerm, opts)
	if err != nil {
		return nil, fmt.Errorf("remote scan failed: %w", err)
	}

	results := make([]ScanResult, len(entries))
	for i, entry := range entries {
		results[i] = ScanResult{
			Term:  entry.Term,
			Count: entry.Count,
		}
	}

	return results, nil
}

func (p *ProxyProvider) CreateRecord(db string, record *z3950.MARCRecord) (string, error) {
	return "", fmt.Errorf("proxy provider is read-only")
}

func (p *ProxyProvider) UpdateRecord(db string, id string, record *z3950.MARCRecord) error {
	return fmt.Errorf("proxy provider is read-only")
}

func (p *ProxyProvider) CreateItem(bibID string, item Item) error { return fmt.Errorf("proxy is read-only") }
func (p *ProxyProvider) GetItems(bibID string) ([]Item, error) { return []Item{}, nil }
func (p *ProxyProvider) GetItemByBarcode(barcode string) (*Item, error) { return nil, fmt.Errorf("not found") }
func (p *ProxyProvider) Checkout(itemBarcode, patronID string) (string, error) { return "", fmt.Errorf("proxy is read-only") }
func (p *ProxyProvider) Checkin(itemBarcode string) (float64, error) { return 0, fmt.Errorf("proxy is read-only") }

// --- Discovery & Patron Stubs ---
func (p *ProxyProvider) GetDashboardStats() (map[string]interface{}, error) { return nil, nil }
func (p *ProxyProvider) GetNewArrivals(limit int) ([]SearchResult, error) { return nil, nil }
func (p *ProxyProvider) GetPopularBooks(limit int) ([]SearchResult, error) { return nil, nil }
func (p *ProxyProvider) RenewLoan(loanID int64) (string, error) { return "", fmt.Errorf("not implemented") }
func (p *ProxyProvider) PlaceHold(bibID string, patronID string) error { return fmt.Errorf("not implemented") }
func (p *ProxyProvider) GetPatronLoans(patronID string) ([]map[string]interface{}, error) { return nil, nil }

// Stub implementations for unsupported methods
func (p *ProxyProvider) CreateILLRequest(req ILLRequest) error {
	return fmt.Errorf("proxy provider does not support creating ILL requests locally")
}

func (p *ProxyProvider) GetILLRequest(id int64) (*ILLRequest, error) {
	return nil, fmt.Errorf("proxy provider does not support ILL")
}

func (p *ProxyProvider) ListILLRequests() ([]ILLRequest, error) {
	return []ILLRequest{}, nil
}

func (p *ProxyProvider) UpdateILLRequestStatus(id int64, status string) error {
	return fmt.Errorf("proxy provider does not support updating ILL requests")
}

func (p *ProxyProvider) CreateUser(user *User) error {
	return fmt.Errorf("proxy provider does not support user management")
}

func (p *ProxyProvider) GetUserByUsername(username string) (*User, error) {
	return nil, fmt.Errorf("user not found in proxy provider")
}

func (p *ProxyProvider) CreateTarget(target *Target) error {
	return fmt.Errorf("proxy provider does not support managing targets")
}

func (p *ProxyProvider) ListTargets() ([]Target, error) {
	return []Target{}, nil
}

func (p *ProxyProvider) DeleteTarget(id int64) error {
	return fmt.Errorf("proxy provider does not support managing targets")
}

func (p *ProxyProvider) GetTargetByName(name string) (*Target, error) {
	return nil, fmt.Errorf("proxy provider does not store targets")
}