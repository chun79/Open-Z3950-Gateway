package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		fmt.Println("Error: DB_DSN is not set")
		return
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil { log.Fatal(err) }
	defer db.Close()

	isbn := "7-5325-1989-9"
	fmt.Printf("--- Inspecting Record for ISBN: %s ---\n", isbn)

	var title string
	var rawRecord, rawFormat sql.NullString
	
	err = db.QueryRow("SELECT title, raw_record, raw_record_format FROM shared_bibliography WHERE TRIM(isbn) = $1", isbn).Scan(&title, &rawRecord, &rawFormat)

	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Printf("Title: %s\n", title)
	fmt.Printf("Format Column: %s\n", rawFormat.String)
	
	if !rawRecord.Valid || rawRecord.String == "" {
		fmt.Println("❌ raw_record IS EMPTY (NULL or '')")
		fmt.Println("   -> System generated a BASIC record dynamically.")
	} else {
		fmt.Println("✅ raw_record HAS DATA:")
		fmt.Printf("   Length: %d bytes\n", len(rawRecord.String))
		if len(rawRecord.String) > 100 {
			fmt.Printf("   Content (First 100): %s...\n", rawRecord.String[:100])
		} else {
			fmt.Printf("   Content: %s\n", rawRecord.String)
		}
		
		if len(rawRecord.String) > 0 && rawRecord.String[0] == '{' {
			fmt.Println("   Format: JSON MARC")
		} else {
			fmt.Println("   Format: ISO 2709 / Other")
		}
	}
}