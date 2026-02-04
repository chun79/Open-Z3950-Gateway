package provider

import (
	"net"
	"strings"
	"testing"

	"github.com/go-asn1-ber/asn1-ber"
	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

// Reusing MockServer concept locally for Provider test

type MockZServer struct {
	listener net.Listener
	Port     int
}

func StartMockZServer() (*MockZServer, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	addr := l.Addr().(*net.TCPAddr)
	s := &MockZServer{listener: l, Port: addr.Port}
	go s.serve()
	return s, nil
}

func (s *MockZServer) Close() {
	s.listener.Close()
}

func (s *MockZServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *MockZServer) handle(conn net.Conn) {
	defer conn.Close()
	for {
		pkt, err := ber.ReadPacket(conn)
		if err != nil {
			return
		}

		var resp *ber.Packet
		switch pkt.Tag {
		case 20: // Init
			resp = ber.Encode(ber.ClassContext, ber.TypeConstructed, 21, nil, "InitResp")
			// Result [12] IMPLICIT BOOLEAN
			resp.AppendChild(ber.NewBoolean(ber.ClassContext, ber.TypePrimitive, 12, true, "Result"))
		case 22: // Search
			resp = ber.Encode(ber.ClassContext, ber.TypeConstructed, 23, nil, "SearchResp")
			// ResultCount [23] IMPLICIT INTEGER
			resp.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 23, 1, "Count"))
			// SearchStatus [26] IMPLICIT BOOLEAN
			resp.AppendChild(ber.NewBoolean(ber.ClassContext, ber.TypePrimitive, 26, true, "Status"))
		case 24: // Present
			resp = ber.Encode(ber.ClassContext, ber.TypeConstructed, 25, nil, "PresentResp")
			recs := ber.Encode(ber.ClassContext, ber.TypeConstructed, 28, nil, "Records")
			rec := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "Record")
			dbrec := ber.Encode(ber.ClassContext, ber.TypeConstructed, 1, nil, "DBRecord")
			// "Remote Title"
			marc := z3950.BuildMARC(&z3950.ProfileMARC21, "999", "Remote Title", "Remote Author", "111", "RemotePub", "2024", "", "")
			dbrec.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, string(marc), "MARC"))
			rec.AppendChild(dbrec)
			recs.AppendChild(rec)
			resp.AppendChild(recs)
		default:
			return
		}
		conn.Write(resp.Bytes())
	}
}

func TestHybridProvider(t *testing.T) {
	// 1. Setup Local Provider (Memory)
	local := NewMemoryProvider()
	local.AddBook("Local Title", "Local Author", "000", "LocalPub", "2020", "", "LocalSub")

	// 2. Setup Hybrid
	hybrid := NewHybridProvider(local)

	// 3. Test Local Search
	query := z3950.StructuredQuery{Root: z3950.QueryClause{Attribute: z3950.UseAttributeTitle, Term: "Local"}}
	ids, err := hybrid.Search("Local", query)
	if err != nil {
		t.Fatalf("Local search failed: %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("Expected 1 local result, got %d", len(ids))
	}

	recs, err := hybrid.Fetch("Local", ids)
	if err != nil {
		t.Fatalf("Local fetch failed: %v", err)
	}
	if len(recs) > 0 && !strings.Contains(recs[0].GetTitle(nil), "Local Title") {
		t.Errorf("Expected 'Local Title', got '%s'", recs[0].GetTitle(nil))
	}

	// 4. Setup Remote Mock Server
	mockServer, err := StartMockZServer()
	if err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer mockServer.Close()

	// 5. Add Target pointing to Mock Server
	target := &Target{
		Name:         "MockRemote",
		Host:         "127.0.0.1",
		Port:         mockServer.Port,
		DatabaseName: "Default",
		Encoding:     "MARC21",
	}
	hybrid.CreateTarget(target)

	// 6. Test Remote Search via Hybrid
	rQuery := z3950.StructuredQuery{Root: z3950.QueryClause{Attribute: z3950.UseAttributeTitle, Term: "Remote"}}
	rIds, err := hybrid.Search("MockRemote", rQuery)
	if err != nil {
		t.Fatalf("Remote search failed: %v", err)
	}
	// Proxy provider mocks returning count based on server response (1 in mock)
	if len(rIds) != 1 {
		t.Errorf("Expected 1 remote result, got %d", len(rIds))
	}

	rRecs, err := hybrid.Fetch("MockRemote", rIds)
	if err != nil {
		t.Fatalf("Remote fetch failed: %v", err)
	}
	if len(rRecs) > 0 && !strings.Contains(rRecs[0].GetTitle(nil), "Remote Title") {
		t.Errorf("Expected 'Remote Title', got '%s'", rRecs[0].GetTitle(nil))
	}
}