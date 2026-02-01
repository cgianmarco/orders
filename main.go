package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

type Config struct {
	DB_HOST     string
	DB_PORT     string
	DB_USER     string
	DB_PASSWORD string
	DB_NAME     string
}

func getHandler(db *sql.DB) http.Handler {
	var mux = http.NewServeMux()
	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) { GetProductsHandler(db, w, r) })
	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) { PlaceOrderHandler(db, w, r) })
	return mux
}

func main() {

	config := Config{
		DB_HOST:     os.Getenv("DB_HOST"),
		DB_PORT:     os.Getenv("DB_PORT"),
		DB_USER:     os.Getenv("DB_USER"),
		DB_PASSWORD: os.Getenv("DB_PASSWORD"),
		DB_NAME:     os.Getenv("DB_NAME"),
	}

	var db, err = sql.Open(
		"postgres",
		fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			config.DB_HOST,
			config.DB_PORT,
			config.DB_USER,
			config.DB_PASSWORD,
			config.DB_NAME,
		),
	)
	if err != nil {
		log.Println("Error connecting to database:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Println("Error connecting to database:", err)
	}

	handler := getHandler(db)

	if err = http.ListenAndServe(":8080", handler); err != nil {
		log.Println("Error starting server:", err)
	}
}
