package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// newHandler gives HTTP tests a fresh in-memory cluster. Normal startup uses
// live mode; the deterministic API below remains useful for protocol exercises.
func newHandler() http.Handler {
	return clusterHandler(NewCluster())
}

// clusterHandler owns the simulator's shared state. Its mutex makes each API
// action and the following JSON observation one consistent simulator step.
func clusterHandler(c *Cluster) http.Handler {
	var mu sync.Mutex
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "http://127.0.0.1:8083/", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/api/cluster", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if c.storageErr != nil {
			http.Error(w, "Storage unavailable; restart after repairing storage", 503)
			return
		}
		if r.Method == "POST" {
			if r.Header.Get("Content-Type") != "application/json" {
				http.Error(w, "JSON required", 415)
				return
			}
			var a struct {
				Action, Key, Value string
				ID                 int
			}
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&a) != nil {
				http.Error(w, "Invalid request", 400)
				return
			}
			switch a.Action {
			case "step":
				c.Step()
			case "reset":
				if c.Durable {
					http.Error(w, "Reset is disabled in durable mode to protect saved data. Use a new RAFT_DATA_DIR for a fresh experiment.", 409)
					return
				}
				c = NewCluster()
			case "toggle":
				if a.ID < 0 || a.ID > 2 {
					http.Error(w, "Invalid node", 400)
					return
				}
				c.Toggle(a.ID)
			case "write":
				if len(a.Key) == 0 || len(a.Key) > 64 || len(a.Value) > 256 {
					http.Error(w, "Use a key of 1-64 characters and value up to 256 characters", 400)
					return
				}
				if err := c.Write(a.Key, a.Value); err != nil {
					if c.storageErr != nil {
						http.Error(w, "Storage failed; write success cannot be confirmed", 503)
						return
					}
					http.Error(w, err.Error(), 409)
					return
				}
			default:
				http.Error(w, "Unknown action", 400)
				return
			}
		} else if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		if c.storageErr != nil {
			http.Error(w, "Storage failed; no operation success can be confirmed", 503)
			return
		}
		json.NewEncoder(w).Encode(struct {
			*Cluster
			Progress *WriteProgress
		}{c, c.Progress()})
	})
	return mux
}

// main selects the supervisor, individual replica, or deterministic teaching
// runtime. All modes share the same consensus and persistence implementation.
func main() {
	// The executable can supervise two fixed shards, run the earlier single
	// group, run one child replica, or expose the deterministic simulator.
	mode := flag.String("mode", "sharded", "sharded, live, node, router, or simulation")
	id := flag.Int("id", 0, "node ID, 0..2")
	base := flag.Int("base-port", 8091, "first server port")
	listen := flag.String("listen", "127.0.0.1:8084", "controller address")
	data := flag.String("data-dir", "", "persistent data directory")
	isolated := flag.Bool("isolated", false, "start node with peer communication blocked")
	shardID := flag.Int("shard-id", 0, "zero-based shard identity for a child")
	shardCount := flag.Int("shard-count", 1, "fixed shard count for a child")
	peerFile := flag.String("peer-config", "", "fixed two-group peer configuration for Kubernetes")

	nodeListen := flag.String("node-listen", "", "explicit Pod listener; local children default to loopback")
	flag.Parse()
	var peers peerConfig
	if *peerFile != "" {
		var err error
		peers, err = loadPeerConfig(*peerFile)
		if err != nil {
			log.Fatal(err)
		}
	}
	if *mode == "router" {
		if *peerFile == "" {
			log.Fatal("router requires -peer-config")
		}
		if err := runRouter(*listen, peers); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *mode == "node" {
		// The supervisor launches this mode once per replica, with a separate
		// port and storage directory. These children run the actual protocol.
		var addresses []string
		if *peerFile != "" {
			if *shardCount != fixedShardCount || *shardID < 0 || *shardID >= len(peers.Groups) {
				log.Fatal("invalid fixed shard")
			}
			if *id == -1 {
				var err error
				*id, err = podVoterID(os.Getenv("POD_NAME"))
				if err != nil {
					log.Fatal(err)
				}
			}
			addresses = peers.Groups[*shardID]
		} else if *nodeListen != "" {
			log.Fatal("explicit node listener requires -peer-config")
		}
		if err := runConfiguredNode(*id, *base, *data, *isolated, *shardID, *shardCount, *nodeListen, addresses); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *mode == "sharded" {
		if err := runSharded(*listen, *base, *data); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *mode == "live" {
		if err := runLive(*listen, *base, *data); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *mode != "simulation" {
		log.Fatal("unknown mode")
	}
	// Simulation keeps its own data directory so experiments cannot silently
	// replace the live servers' saved databases.
	dir := os.Getenv("RAFT_DATA_DIR")
	if dir == "" {
		dir = filepath.Join("17-sharded-raft-kv", "data")
	}
	c, err := NewDurableCluster(dir)
	if err != nil {
		log.Fatal(err)
	}
	defer c.store.close()
	log.Println("Raft API: http://127.0.0.1:8084 · Next.js UI: http://127.0.0.1:8083")
	server := &http.Server{
		Addr:              "127.0.0.1:8084",
		Handler:           clusterHandler(c),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
