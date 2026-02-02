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
	if err != nil {
		log.Fatalf("Connect failed: %v", err)
	}
	defer db.Close()

	// 查询所有表
	rows, err := db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		ORDER BY table_name
	`)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	fmt.Println("--- Tables in library_db ---")
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tables = append(tables, name)
			fmt.Printf("- %s\n", name)
		}
	}
	
	fmt.Printf("\nTotal: %d tables\n", len(tables))
}
