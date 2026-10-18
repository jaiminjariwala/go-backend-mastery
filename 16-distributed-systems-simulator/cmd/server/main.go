package main

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"time"

	lb "github.com/jaiminjariwala/go-backend-mastery/16-distributed-systems-simulator/concepts/loadbalancer"
	rl "github.com/jaiminjariwala/go-backend-mastery/16-distributed-systems-simulator/concepts/ratelimit"
	"github.com/jaiminjariwala/go-backend-mastery/16-distributed-systems-simulator/web"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/load-balancer", simulate)
	mux.HandleFunc("POST /api/rate-limit", simulateRateLimit)
	files, err := fs.Sub(web.Files, ".")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(files)))
	server := &http.Server{Addr: "127.0.0.1:8082", Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	log.Println("Systems lab: http://127.0.0.1:8082")
	log.Fatal(server.ListenAndServe())
}

func simulateRateLimit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var config rl.Config
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&config) != nil {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid simulation input"})
		return
	}
	result, err := rl.Run(config)
	if err != nil {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(result)
}

func simulate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var config lb.Config
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	if decoder.Decode(&config) != nil {
		http.Error(w, `{"error":"Invalid simulation input"}`, 400)
		return
	}
	result, err := lb.Run(config)
	if err != nil {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(result)
}
