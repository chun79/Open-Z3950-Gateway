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

	// 模糊查 ISBN
	rows, err := db.Query("SELECT isbn FROM shared_bibliography WHERE isbn LIKE '%5325%'")
	if err != nil { log.Fatal(err) }
	defer rows.Close()

	fmt.Println("Matching ISBNs in shared_bibliography:")
		for rows.Next() {
			var isbn string
			rows.Scan(&isbn)
			fmt.Printf("'%s'\n", isbn)
		}}
