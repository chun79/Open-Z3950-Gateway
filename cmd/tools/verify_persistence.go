package main

import (
	"fmt"
	"os"

	"github.com/yourusername/open-z3950-gateway/pkg/provider"
)

func main() {
	dbPath := "persistence_test.db"
	// Cleanup previous run
	os.Remove(dbPath)
	defer os.Remove(dbPath)

	fmt.Println("1. Initializing SQLite Provider...")
	p, err := provider.NewSQLiteProvider(dbPath)
	if err != nil {
		panic(err)
	}

	fmt.Println("2. Creating Test Data...")
	target := &provider.Target{
		Name:         "PersistenceTestLib",
		Host:         "localhost",
		Port:         210,
		DatabaseName: "TEST",
		Encoding:     "MARC21",
	}
	if err := p.CreateTarget(target); err != nil {
		panic(err)
	}

	user := &provider.User{
		Username:     "testuser",
		PasswordHash: "hash123",
		Role:         "user",
	}
	if err := p.CreateUser(user); err != nil {
		panic(err)
	}

	// Verify Data exists in memory
	targets, _ := p.ListTargets()
	if len(targets) == 0 {
		panic("Failed to list targets immediately after creation")
	}
	fmt.Printf("   -> Created %d target(s)\n", len(targets))

	// Close (Simulate restart) - Implementation detail: SQL DB pool closes on GC or explicit close.
	// Since NewSQLiteProvider returns a struct with *sql.DB, we can't explicitly close it via the interface 
	// unless we add Close() to Provider interface. 
	// But for this test, we just open a NEW provider on the same file.
	p = nil 

	fmt.Println("3. Re-opening Database (Simulate Restart)...")
	p2, err := provider.NewSQLiteProvider(dbPath)
	if err != nil {
		panic(err)
	}

	fmt.Println("4. Verifying Persistence...")
	targets2, err := p2.ListTargets()
	if err != nil {
		panic(err)
	}

	found := false
	for _, t := range targets2 {
		if t.Name == "PersistenceTestLib" {
			found = true
			break
		}
	}

	if !found {
		panic("DATA LOSS DETECTED: Target 'PersistenceTestLib' not found after restart!")
	}

	u, err := p2.GetUserByUsername("testuser")
	if err != nil {
		panic("DATA LOSS DETECTED: User 'testuser' not found!")
	}
	if u.Role != "user" {
		panic("Data corruption: User role mismatch")
	}

	fmt.Println("✅ SUCCESS: Data persistence verified.")
}
