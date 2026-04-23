package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"raiseblinds/backend"
	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "file:raiseblinds.db?_pragma=foreign_keys(1)")
	if err != nil { log.Fatal(err) }
	s := backend.NewServer(db)
	if err := s.Migrate(); err != nil { log.Fatal(err) }
	addr := os.Getenv("ADDR")
	if addr == "" { addr = ":8080" }
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, s))
}
