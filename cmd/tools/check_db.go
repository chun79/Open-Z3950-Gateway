package main

import (
	"fmt"
	"log"
	"os"

	"github.com/yourusername/open-z3950-gateway/pkg/provider"
)

func main() {
	if os.Getenv("DB_DSN") == "" {
		fmt.Println("Error: DB_DSN is not set")
		return
	}

	fmt.Println("--- Connecting to Database ---")
	prov, err := provider.NewPostgresProvider()
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	ids, err := prov.Search("Default", "% ")
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	fmt.Printf("Total records found: %d\n", len(ids))

	records, err := prov.Fetch("Default", ids[:3]) // Check top 3
	if err != nil {
		log.Fatalf("Fetch failed: %v", err)
	}

	fmt.Println("\n--- Rich Data Sample ---")
	for i, rec := range records {
		fmt.Printf("[%d] Title: %s\n", i+1, rec.GetTitle(nil))
		fmt.Printf("    Publisher/Year (Tag 260): %s\n", rec.GetPublisher(nil))
		fmt.Println("    ---------------------------")
	}
}
