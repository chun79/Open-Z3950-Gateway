package z3950

import (
	"fmt"
	"net"
	"strings"
	"testing"

	ber "github.com/go-asn1-ber/asn1-ber"
)

// MockServer represents a mock Z39.50 server for testing
type MockServer struct {
	listener net.Listener
	Addr     string
	stop     chan struct{}
}

func NewMockServer() (*MockServer, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0") // Random port
	if err != nil {
		return nil, err
	}
	s := &MockServer{
		listener: l,
		Addr:     l.Addr().String(),
		stop:     make(chan struct{}),
	}
	go s.serve()
	return s, nil
}

func (s *MockServer) Close() {
	close(s.stop)
	s.listener.Close()
}

func (s *MockServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stop:
				return
			default:
				// continue
			}
			continue
		}
		go s.handle(conn)
	}
}

func (s *MockServer) handle(conn net.Conn) {
	defer conn.Close()
	for {
		pkt, err := ber.ReadPacket(conn)
		if err != nil {
			return
		}
		
		var resp *ber.Packet
		
		switch pkt.Tag {
		case 20: // InitializeRequest
			resp = ber.Encode(ber.ClassContext, ber.TypeConstructed, 21, nil, "InitializeResponse")
			resp.AppendChild(ber.NewBoolean(ber.ClassUniversal, ber.TypePrimitive, ber.TagBoolean, true, "Result"))
		
		case 22: // SearchRequest
			resp = ber.Encode(ber.ClassContext, ber.TypeConstructed, 23, nil, "SearchResponse")
			resp.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 23, 5, "Count"))
			resp.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 24, 0, "Returned"))
			resp.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 25, 0, "NextPos"))
			resp.AppendChild(ber.NewBoolean(ber.ClassContext, ber.TypePrimitive, 26, true, "Status"))

		case 24: // PresentRequest
			resp = ber.Encode(ber.ClassContext, ber.TypeConstructed, 25, nil, "PresentResponse")
			// Create a dummy MARC record
			recordsWrapper := ber.Encode(ber.ClassContext, ber.TypeConstructed, 28, nil, "Records")
			
			// Mock one record
			namePlusRecord := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "Record")
			dbRecord := ber.Encode(ber.ClassContext, ber.TypeConstructed, 1, nil, "DBRecord")
			
			// Minimal MARC
			marcData := BuildMARC(&ProfileMARC21, "001", "Mock Title", "Mock Author", "1234567890", "Mock Pub", "2024", "", "Mock Subj")
			octet := ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, string(marcData), "MARC")
			
			dbRecord.AppendChild(octet)
			namePlusRecord.AppendChild(dbRecord)
			recordsWrapper.AppendChild(namePlusRecord)
			
			resp.AppendChild(recordsWrapper)

		case 35: // ScanRequest
			resp = ber.Encode(ber.ClassContext, ber.TypeConstructed, 36, nil, "ScanResponse")
			entriesWrapper := ber.Encode(ber.ClassContext, ber.TypeConstructed, 7, nil, "Entries")
			listEntries := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "List")
			
			// Entry 1
			entry := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "Entry")
			termInfo := ber.Encode(ber.ClassContext, ber.TypeConstructed, 1, nil, "TermInfo")
			termInfo.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 45, "MockTerm1", "Term"))
			termInfo.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 2, 10, "Count"))
			entry.AppendChild(termInfo)
			listEntries.AppendChild(entry)

			entriesWrapper.AppendChild(listEntries)
			resp.AppendChild(entriesWrapper)
		
		case 30: // DeleteResultSet
			resp = ber.Encode(ber.ClassContext, ber.TypeConstructed, 31, nil, "DeleteResponse")
			resp.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 0, 0, "Status"))

		default:
			// Unknown, close
			return
		}

		conn.Write(resp.Bytes())
	}
}

func TestClient_ConnectAndInit(t *testing.T) {
	server, err := NewMockServer()
	if err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer server.Close()

	host, portStr, _ := net.SplitHostPort(server.Addr)
	port := 0
	fmt.Sscanf(portStr, "%d", &port)

	client := NewClient(host, port)
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Close()

	if err := client.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
}

func TestClient_SearchAndPresent(t *testing.T) {
	server, err := NewMockServer()
	if err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer server.Close()

	host, portStr, _ := net.SplitHostPort(server.Addr)
	port := 0
	fmt.Sscanf(portStr, "%d", &port)

	client := NewClient(host, port)
	client.Connect()
	defer client.Close()
	client.Init()

	// Test Structured Search
	query := StructuredQuery{
		Root: QueryClause{Attribute: UseAttributeTitle, Term: "Mock"},
	}
	count, err := client.StructuredSearch("Default", query)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected 5 results, got %d", count)
	}

	// Test Present
	recs, err := client.Present(1, 1, OID_MARC21)
	if err != nil {
		t.Fatalf("Present failed: %v", err)
	}
	if len(recs) == 0 {
		t.Fatal("Expected at least one record")
	}
	
	title := recs[0].GetTitle(nil)
	// BuildMARC implementation adds subfield markers.
	if !strings.Contains(title, "Mock Title") {
		t.Errorf("Expected title 'Mock Title', got '%s'", title)
	}
}

func TestClient_Scan(t *testing.T) {
	server, err := NewMockServer()
	if err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer server.Close()

	host, portStr, _ := net.SplitHostPort(server.Addr)
	port := 0
	fmt.Sscanf(portStr, "%d", &port)

	client := NewClient(host, port)
	client.Connect()
	defer client.Close()
	client.Init()

	results, err := client.Scan("Default", "Mock", nil)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected scan results")
	}
	if results[0].Term != "MockTerm1" {
		t.Errorf("Expected term 'MockTerm1', got '%s'", results[0].Term)
	}
}
