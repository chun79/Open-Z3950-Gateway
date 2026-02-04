package sip2

import (
	"testing"
	"time"
)

func TestIntegration(t *testing.T) {
	// 1. Start Server
	port := 6001
	server := NewMockServer(port)
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer server.Close()
	
	// Give server time to bind
	time.Sleep(100 * time.Millisecond)
	
	// 2. Client Connect
	client := NewClient("localhost", port)
	client.Location = "MainBranch"
	
	if err := client.Connect(); err != nil {
		t.Fatalf("Client connect failed: %v", err)
	}
	defer client.Close()
	
	// 3. Login
	ok, err := client.Login("sipuser", "sippass", "MainBranch")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if !ok {
		t.Errorf("Login rejected")
	}
	
	// 4. Patron Info
	info, err := client.GetPatronInfo("12345678", "")
	if err != nil {
		t.Fatalf("GetPatronInfo failed: %v", err)
	}
	
	if info["AE"] != "John Doe" {
		t.Errorf("Expected Patron Name 'John Doe', got '%s'", info["AE"])
	}
}
