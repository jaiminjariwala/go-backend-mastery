package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Register the pgx PostgreSQL driver.

	httpbasics "github.com/jaiminjariwala/go-backend-mastery/14-backend-labs/concepts/01-http-request-response"
	jsonbasics "github.com/jaiminjariwala/go-backend-mastery/14-backend-labs/concepts/02-json-validation"
	notes "github.com/jaiminjariwala/go-backend-mastery/14-backend-labs/concepts/03-rest-notes"
)

// main takes no parameters and returns nothing. It starts the local lab server.
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/hello", httpbasics.Hello)
	mux.HandleFunc("/api/greet", jsonbasics.Greet)
	mux.HandleFunc("GET /concepts/notes", notes.Page)
	// Earlier lessons remain usable without database configuration.
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL != "" {
		db, err := sql.Open("pgx", databaseURL)
		if err != nil {
			log.Fatal("Invalid database configuration; check DATABASE_URL (not printed for privacy).")
		}
		defer db.Close()
		db.SetMaxOpenConns(5)
		db.SetMaxIdleConns(2)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err != nil {
			log.Fatal("Could not connect to PostgreSQL; check credentials, TLS, and network. Credentials are not logged.")
		}
		noteStore := notes.NewStore(db)
		noteStore.Register(mux)
		log.Println("Database connected. Notes API: http://127.0.0.1:8080/api/notes")
	} else {
		log.Println("Lesson 3 disabled: set DATABASE_URL to enable PostgreSQL notes.")
	}

	server := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Println("HTTP lesson: http://127.0.0.1:8080/api/hello?name=Jaimin")
	log.Println("JSON lesson: POST http://127.0.0.1:8080/api/greet")
	log.Fatal(server.ListenAndServe())
}
