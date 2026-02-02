package main

import (
	"fmt"
	"log"
	
	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

func main() {
	// Test against Index Data
	host := "z3950.indexdata.com"
	port := 210
	dbName := "gils"
	term := "computer"

	fmt.Printf("Connecting to %s:%d...\n", host, port)
	client := z3950.NewClient(host, port)
	if err := client.Connect(); err != nil {
		log.Fatalf("Connect failed: %v", err)
	}
	defer client.Close()
	fmt.Println("Connected.")

	if err := client.Init(); err != nil {
		log.Fatalf("Init failed: %v", err)
	}
	// Init doesn't return the result boolean in the current client.go implementation (it just checks Tag 21).
	// We need to trust the log or modify Client.Init. 
	// But let's assume it's True for now. 
	// Wait, if it was False, the server would likely close or ignore subsequent search.
	fmt.Println("Init PDU received (Tag 21). Assuming success.")

	fmt.Printf("Searching for '%s' in %s...\n", term, dbName)
	query := z3950.StructuredQuery{
		Root: z3950.QueryClause{
			Attribute: z3950.UseAttributeAny, // 1016
			Term:      term,
		},
	}

	count, err := client.StructuredSearch(dbName, query)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}
	fmt.Printf("Search success! Found %d records.\n", count)

	if count > 0 {
		fmt.Println("Fetching first record...")
		recs, err := client.Present(1, 1, z3950.OID_MARC21)
		if err != nil {
			log.Fatalf("Present failed: %v", err)
		}
		if len(recs) > 0 {
			fmt.Printf("Title: %s\n", recs[0].GetTitle(nil))
		}
	}
}
