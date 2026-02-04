package main

import (
	"net"
	"testing"
	
	"github.com/yourusername/open-z3950-gateway/pkg/provider"
	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

func TestServer_HandleSearch(t *testing.T) {
	// Start Server
	mem := provider.NewMemoryProvider()
	srv := NewServer(mem)
	
	// Create a Listener
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer l.Close()

	go func() {
		conn, err := l.Accept()
		if err == nil {
			srv.handleConnection(conn)
		}
	}()

	// Connect Client
	client := z3950.NewClient("127.0.0.1", l.Addr().(*net.TCPAddr).Port)
	if err := client.Connect(); err != nil {
		t.Fatalf("Client connect failed: %v", err)
	}
	defer client.Close()

	if err := client.Init(); err != nil {
		t.Fatalf("Client init failed: %v", err)
	}

	// Add data to memory provider
	mem.AddBook("Server Test Title", "Author", "ISBN", "Pub", "2024", "", "")

	// Test Search
	count, err := client.Search("Default", "Server")
	if err != nil {
		t.Fatalf("Client search failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 result, got %d", count)
	}

	recs, err := client.Present(1, 1, z3950.OID_MARC21)
	if err != nil {
		t.Fatalf("Client present failed: %v", err)
	}
	if len(recs) != 1 {
		t.Errorf("Expected 1 record, got %d", len(recs))
	}
}
