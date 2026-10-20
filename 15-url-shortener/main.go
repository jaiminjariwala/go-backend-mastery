package main

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
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
	instance := os.Getenv("INSTANCE_ID")
	if instance == "" {
		instance = "go-1"
	}
	app := &App{db: db, instance: instance}
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		options, parseErr := redis.ParseURL(redisURL)
		if parseErr != nil {
			log.Fatal("Invalid REDIS_URL (credentials not logged)")
		}
		options.DialTimeout = 250 * time.Millisecond
		options.ReadTimeout = 250 * time.Millisecond
		options.WriteTimeout = 250 * time.Millisecond
		options.MaxRetries = -1
		options.ContextTimeoutEnabled = true
		client := redis.NewClient(options)
		defer client.Close()
		app.infra = &redisInfra{client: client}
		app.cacheEnabled = os.Getenv("CACHE_ENABLED") != "false"
		app.limitEnabled = os.Getenv("RATE_LIMIT_ENABLED") != "false"
	} else {
		log.Println("Redis not configured: cache and shared rate limiting are disabled")
	}
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8081"
	}
	host, _, addrErr := net.SplitHostPort(addr)
	if addrErr != nil || host != "127.0.0.1" {
		log.Fatal("HTTP_ADDR must bind to 127.0.0.1 for this local learning app")
	}
	server := &http.Server{Addr: addr, Handler: app.Routes(), ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	log.Println("URL shortener:", "http://"+addr, "instance:", instance)
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
