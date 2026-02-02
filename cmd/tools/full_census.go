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

	// 拉取前 50 条数据
	rows, err := db.Query("SELECT isbn, title FROM shared_bibliography LIMIT 50")
	if err != nil { log.Fatal(err) }
	defer rows.Close()

	fmt.Println("=== Data Census (Top 50 books in shared_bibliography) ===")
	count := 0
		for rows.Next() {
			var isbn, title string
			rows.Scan(&isbn, &title)
			fmt.Printf("[%d] ISBN: '%s' | Title: '%s'\n", count+1, isbn, title)
			count++
		}	
	if count == 0 {
		fmt.Println("Warning: Table shared_bibliography is EMPTY.")
	} else {
		fmt.Printf("\nTotal displayed: %d\n", count)
	}
}

