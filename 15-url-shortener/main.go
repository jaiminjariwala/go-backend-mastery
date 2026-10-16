package main

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed index.html
var page string

func main() {
	uri := os.Getenv("DATABASE_URL")
	if uri == "" {
		log.Fatal("Set DATABASE_URL privately before starting.")
	}
	db, err := sql.Open("pgx", uri)
	if err != nil {
		log.Fatal("Invalid database configuration (credentials not logged).")
	}
	defer db.Close()
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = db.PingContext(ctx)
	cancel()
	if err != nil {
		log.Fatal("Database connection failed; check credentials, TLS and network.")
	}
	app := &App{db: db}
	server := &http.Server{Addr: "127.0.0.1:8081", Handler: app.Routes(), ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	log.Println("URL shortener: http://127.0.0.1:8081")
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Println("Server stopped:", err)
		}
	case <-ctx.Done():
		stop()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if server.Shutdown(shutdown) != nil {
			server.Close()
		}
		<-done
	}
}
