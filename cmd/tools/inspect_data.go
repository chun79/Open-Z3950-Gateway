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
	db, err := sql.Open("postgres", dsn)
	if err != nil { log.Fatal(err) }
	defer db.Close()

	tables := []string{"bibliography", "shared_bibliography"}
	for _, table := range tables {
		fmt.Printf("\n=== Table: %s ===\n", table)
		
		// 1. 检查字段是否存在
	
rows, _ := db.Query(fmt.Sprintf("SELECT column_name FROM information_schema.columns WHERE table_name = '%s'", table))
		hasRaw := false
		hasJSON := false
		for rows.Next() {
			var name string
			rows.Scan(&name)
			if name == "raw_record" { hasRaw = true }
			if name == "marcjson" { hasJSON = true }
		}
		fmt.Printf("Fields: raw_record=%v, marcjson=%v\n", hasRaw, hasJSON)

		// 2. 抽样内容
		var query string
		if hasRaw && hasJSON {
			query = fmt.Sprintf("SELECT raw_record, marcjson FROM %s WHERE raw_record IS NOT NULL OR marcjson IS NOT NULL LIMIT 1", table)
		} else if hasRaw {
			query = fmt.Sprintf("SELECT raw_record, 'N/A' FROM %s WHERE raw_record IS NOT NULL LIMIT 1", table)
		} else {
			continue
		}

		var raw, mjson sql.NullString
		err := db.QueryRow(query).Scan(&raw, &mjson)
		if err != nil {
			fmt.Printf("No data samples found.\n")
			continue
		}

		fmt.Println("--- Data Sample ---")
		if raw.Valid && len(raw.String) > 0 {
			fmt.Printf("[raw_record] (First 100 chars): %s\n", truncate(raw.String, 100))
			fmt.Printf("             Total Length: %d\n", len(raw.String))
		} else {
			fmt.Println("[raw_record] IS EMPTY")
		}

		if mjson.Valid && mjson.String != "N/A" && len(mjson.String) > 0 {
			fmt.Printf("[marcjson]   (First 100 chars): %s\n", truncate(mjson.String, 100))
			fmt.Printf("             Total Length: %d\n", len(mjson.String))
		} else {
			fmt.Println("[marcjson]   NOT PRESENT OR EMPTY")
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n { return s }
	return s[:n] + "..."
}
