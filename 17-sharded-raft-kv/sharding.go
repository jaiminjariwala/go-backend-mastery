package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const fixedShardCount = 2
const routingAlgorithm = "fnv1a32-mod-v1"

// Hashing the same UTF-8 key must always pick the same group, including after
// restart. Do not use Go's randomized map hashing or silently change the count.
func shardForKey(key string, count int) int {
	hash := fnv.New32a()
	hash.Write([]byte(key))
	return int(hash.Sum32() % uint32(count))
}

// routingInfo explains the stable ownership calculation shown beside the form.
type routingInfo struct {
	Key        string
	ShardID    int
	ShardCount int
	Algorithm  string
}

// shardSummary is a diagnostic sample, not a quorum certificate. Observed keys
// may lag; verified reads still go through the owning group’s read barrier.
type shardSummary struct {
	ID, Online, Connected, BasePort int
	Leader                          *int
	Term                            int
	Keys                            []string
}

// shardedController routes requests and supervises groups; it never votes.
type shardedController struct{ groups []*liveController }

// The router assigns a key to a group; only that group's Raft protocol decides
// commitment. An unavailable shard must never fall back to a different shard.
func (s *shardedController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/kv" {
		if r.Method != "POST" {
			http.Error(w, "POST required", 405)
			return
		}
		var input struct{ Key string }
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
		r.Body.Close()
		if err != nil || json.Unmarshal(body, &input) != nil || len(input.Key) == 0 || len(input.Key) > 64 {
			http.Error(w, "Use a key of 1-64 bytes.", 400)
			return
		}
		owner := shardForKey(input.Key, len(s.groups))
		r.Body = http.NoBody
		// Re-create the consumed request body before handing it to the group.
		r.Body = io.NopCloser(bytes.NewReader(body))
		s.groups[owner].serveKV(w, r)
		return
	}
	if r.URL.Path == "/api/shards" {
		if r.Method != "GET" {
			http.Error(w, "GET required", 405)
			return
		}
		summaries := make([]shardSummary, len(s.groups))
		var wait sync.WaitGroup
		for id, group := range s.groups {
			wait.Add(1)
			go func(id int, group *liveController) {
				defer wait.Done()
				group.mu.Lock()
				states := group.states()
				group.mu.Unlock()
				summary := shardSummary{ID: id, BasePort: group.base, Keys: []string{}}
				keys := map[string]bool{}
				for nodeID, state := range states {
					if state == nil {
						continue
					}
					summary.Online++
					if !state.Isolated {
						summary.Connected++
					}
					if state.Node.Role == "leader" && (summary.Leader == nil || state.Node.Term > summary.Term) {
						copyID := nodeID
						summary.Leader = &copyID
						summary.Term = state.Node.Term
					}
					for key := range state.Node.KV {
						keys[key] = true
					}
				}
				for key := range keys {
					summary.Keys = append(summary.Keys, key)
				}
				sort.Strings(summary.Keys)
				summaries[id] = summary
			}(id, group)
		}
		wait.Wait()
		key := r.URL.Query().Get("key")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(struct {
			Shards []shardSummary
			Route  routingInfo
		}{summaries, routingInfo{key, shardForKey(key, len(s.groups)), len(s.groups), routingAlgorithm}})
		return
	}
	if r.URL.Path != "/api/cluster" {
		http.NotFound(w, r)
		return
	}
	id := 0
	if value := r.URL.Query().Get("shard"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 || parsed >= len(s.groups) {
			http.Error(w, "Unknown shard.", 400)
			return
		}
		id = parsed
	}
	s.groups[id].ServeHTTP(w, r)
}

// A manifest makes the routing contract part of the saved database. Changing
// the hash or shard count would otherwise make existing keys appear missing.
func ensureShardLayout(dir string) error {
	layout := struct {
		Version, ShardCount int
		Algorithm           string
	}{1, fixedShardCount, routingAlgorithm}
	path := filepath.Join(dir, "layout.json")
	data, err := os.ReadFile(path)
	if err == nil {
		var existing struct {
			Version, ShardCount int
			Algorithm           string
		}
		if json.Unmarshal(data, &existing) != nil || existing != layout {
			return fmt.Errorf("saved shard layout differs; refusing to remap existing data")
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "shard-") {
			return fmt.Errorf("shard directories exist without layout.json; refusing to guess their routing contract")
		}
	}
	data, _ = json.MarshalIndent(layout, "", "  ")
	file, err := os.CreateTemp(dir, ".layout-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, path); err != nil {
		return err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

// runSharded reuses the existing three-replica runtime twice. Each group owns
// separate ports, WALs, elections, snapshots, partitions, and retry receipts.
func runSharded(listen string, base int, dir string) error {
	host, port, err := net.SplitHostPort(listen)
	parsedPort, parseErr := strconv.Atoi(port)
	if err != nil || host != "127.0.0.1" || parseErr != nil || parsedPort < 1024 || parsedPort > 65535 {
		return fmt.Errorf("sharded controller must bind a valid loopback port")
	}
	if base < 1024 || base > 65535-fixedShardCount*replicaCount+1 {
		return fmt.Errorf("invalid shard peer ports")
	}
	if dir == "" {
		dir = filepath.Join("17-sharded-raft-kv", "data-sharded")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	lock, err := openStore(dir)
	if err != nil {
		return err
	}
	defer lock.close()
	if err = ensureShardLayout(dir); err != nil {
		return err
	}
	// Reserve all six ports before spawning any children. Never share a peer
	// endpoint between groups, even though local node IDs are 0-2 in both.
	var reservations []net.Listener
	for offset := 0; offset < fixedShardCount*replicaCount; offset++ {
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", base+offset))
		if err != nil {
			for _, open := range reservations {
				open.Close()
			}
			return err
		}
		reservations = append(reservations, listener)
	}
	for _, listener := range reservations {
		listener.Close()
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	router := &shardedController{}
	defer func() {
		for _, group := range router.groups {
			group.rpc.close()
			for id := 0; id < replicaCount; id++ {
				group.kill(id)
			}
		}
	}()
	for id := 0; id < fixedShardCount; id++ {
		group := &liveController{executable: executable, dir: filepath.Join(dir, fmt.Sprintf("shard-%d", id+1)), base: base + id*replicaCount, client: &http.Client{Timeout: 400 * time.Millisecond}, shardID: id, shardCount: fixedShardCount}
		router.groups = append(router.groups, group)
		for nodeID := 0; nodeID < replicaCount; nodeID++ {
			if err = group.start(nodeID); err != nil {
				return err
			}
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Addr: listen, Handler: router, ReadHeaderTimeout: time.Second, WriteTimeout: 5 * time.Second, BaseContext: func(net.Listener) context.Context { return ctx }}
	result := make(chan error, 1)
	go func() { result <- server.ListenAndServe() }()
	log.Printf("Sharded Raft: http://%s · 2 groups, 6 replicas · data: %s", listen, dir)
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
