package main

import (
	"database/sql"
	"log"
	"net/http"
	"ta17eee-serial-auth/handler"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./serials.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/api/verify", handler.VerifyHandler(db))

	log.Println("Server listening on :8080")
	http.ListenAndServe(":8080", nil)
}
