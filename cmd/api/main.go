package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	fmt.Println("Starting connection with Postgres Db")
	connStr := "postgres://postgres:postgres@localhost:5432/postgres"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		log.Println("DB Ping Failed")
		log.Fatal(err)
	}
	log.Println("DB Connection started successfully")
}
