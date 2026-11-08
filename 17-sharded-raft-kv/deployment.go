package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// peerConfig is a fixed routing contract, not a membership-change API. Each
// array index is a voter ID. Reordering addresses would assign the wrong voter.
type peerConfig struct{ Groups [][]string }

func validatePeerAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	number, numberErr := strconv.Atoi(port)
	if err != nil || host == "" || strings.ContainsAny(host, "/?#@ \\ \t\r\n") || numberErr != nil || number < 1024 || number > 65535 {
		return fmt.Errorf("invalid configured peer address %q", address)
	}
	return nil
}

func validateGroup(addresses []string) error {
	if len(addresses) != 3 {
		return fmt.Errorf("a fixed group requires exactly three peer addresses")
	}
	seen := map[string]bool{}
	for _, address := range addresses {
		if err := validatePeerAddress(address); err != nil {
			return err
		}
		if seen[address] {
			return fmt.Errorf("duplicate peer address")
		}
		seen[address] = true
	}
	return nil
}

func loadPeerConfig(path string) (peerConfig, error) {
	var config peerConfig
	file, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 8192))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&config); err != nil {
		return config, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return config, fmt.Errorf("unexpected trailing configuration")
	}
	if len(config.Groups) != fixedShardCount {
		return config, fmt.Errorf("exactly two fixed shard groups required")
	}
	seen := map[string]bool{}
	for _, addresses := range config.Groups {
		if err = validateGroup(addresses); err != nil {
			return config, err
		}
		for _, address := range addresses {
			if seen[address] {
				return config, fmt.Errorf("peer reused across shard groups")
			}
			seen[address] = true
		}
	}
	return config, nil
}

// The StatefulSet ordinal is the voter identity, never a random Pod UID or IP.
// Ordinals beyond two are rejected: kubectl scale cannot add Raft voters.
func podVoterID(name string) (int, error) {
	_, suffix, ok := strings.Cut(name, "-")
	if last := strings.LastIndex(name, "-"); last >= 0 {
		suffix = name[last+1:]
	}
	id, err := strconv.Atoi(suffix)
	if !ok || err != nil || id < 0 || id > 2 {
		return 0, fmt.Errorf("Pod name must end in voter ordinal 0, 1, or 2")
	}
	return id, nil
}

// A PVC cannot silently become another shard or voter. Save this small manifest
// atomically before creating the first WAL, and retain it across Pod replacement.
func ensureReplicaIdentity(dir string, id, shardID, count int, peers []string) error {
	want, _ := json.Marshal(struct {
		Version, Voter, Shard, Count int
		Algorithm                    string
		Peers                        []string
	}{1, id, shardID, count, routingAlgorithm, peers})
	path := filepath.Join(dir, "identity.json")
	if data, err := os.ReadFile(path); err == nil {
		if string(data) != string(want) {
			return fmt.Errorf("PVC identity differs from configured voter/shard; refusing to remap data")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() != "LOCK" {
			return fmt.Errorf("existing data has no replica identity; use a new deployment volume")
		}
	}
	file, err := os.CreateTemp(dir, ".identity-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(want); err != nil {
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
	if err = os.Rename(file.Name(), path); err != nil {
		return err
	}
	folder, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer folder.Close()
	return folder.Sync()
}

func (l *liveController) external() bool { return len(l.addresses) != 0 }
func (l *liveController) address(id int) string {
	if l.external() {
		return l.addresses[id]
	}
	return fmt.Sprintf("127.0.0.1:%d", l.base+id)
}
func (l *liveController) deployment() string {
	if l.external() {
		return "kubernetes"
	}
	return "local"
}

// runRouter observes independently managed Pods. It owns no children and no WAL.
// Losing/restarting this router never changes the fixed voting configuration.
func runRouter(listen string, config peerConfig) error {
	s := &shardedController{}
	for shardID, addresses := range config.Groups {
		s.groups = append(s.groups, &liveController{addresses: addresses, shardID: shardID, shardCount: fixedShardCount, client: &http.Client{Timeout: 800 * time.Millisecond}})
	}
	defer func() {
		for _, group := range s.groups {
			group.rpc.close()
		}
	}()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	// Router readiness means the routing API is running. A failed shard must not
	// remove access to the other shard or hide diagnostics for the failed group.
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	mux.Handle("/", s)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Addr: listen, Handler: mux, ReadHeaderTimeout: 3 * time.Second, WriteTimeout: 10 * time.Second}
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
