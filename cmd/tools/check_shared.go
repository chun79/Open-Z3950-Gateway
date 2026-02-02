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
	if dsn == "" { fmt.Println("Error: DB_DSN is not set"); return }
	db, err := sql.Open("postgres", dsn)
	if err != nil { log.Fatalf("Connect failed: %v", err) }
	defer db.Close()

	// 查 shared_bibliography 的字段
	rows, err := db.Query(`
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = 'shared_bibliography'
	`)
	if err != nil { log.Fatalf("Query failed: %v", err) }
	defer rows.Close()

	fmt.Println("--- shared_bibliography Columns ---")
		for rows.Next() {
			var name, dtype string
			rows.Scan(&name, &dtype)
			fmt.Printf("%s (%s)\n", name, dtype)
		}	
	// 顺便查查有没有数据
	var count int
	db.QueryRow("SELECT COUNT(*) FROM shared_bibliography").Scan(&count)
	fmt.Printf("\nRow Count: %d\n", count)
}
