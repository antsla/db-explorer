package main

import (
	"database/sql"
	"fmt"
	"net/http"
	_ "github.com/go-sql-driver/mysql"
)

var (
	// DSN = "golang:12345@tcp(localhost)/observer?charset=utf8"
	DSN = "root:12345@tcp(db:3306)/observer?charset=utf8"
)

func main() {
	db, err := sql.Open("mysql", DSN)
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	handler, err := NewDbExplorer(db)
	if err != nil {
		panic(err)
	}

	fmt.Println("starting server at :8084")
	http.ListenAndServe(":8084", handler)
}