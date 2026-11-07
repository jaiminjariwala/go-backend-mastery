package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDeploymentRejectsIdentityChanges(t *testing.T) {
	peers := []string{"shard-1-0.shard-1:8091", "shard-1-1.shard-1:8091", "shard-1-2.shard-1:8091"}
	dir := t.TempDir()
	if err := ensureReplicaIdentity(dir, 0, 0, 2, peers); err != nil {
		t.Fatal(err)
	}
	if err := ensureReplicaIdentity(dir, 0, 0, 2, peers); err != nil {
		t.Fatal("same PVC could not recover", err)
	}
	if err := ensureReplicaIdentity(dir, 1, 0, 2, peers); err == nil {
		t.Fatal("another voter accepted the PVC")
	}
	if err := ensureReplicaIdentity(dir, 0, 1, 2, peers); err == nil {
		t.Fatal("another shard accepted the PVC")
	}
	changed := append([]string(nil), peers...)
	changed[0], changed[1] = changed[1], changed[0]
	if err := ensureReplicaIdentity(dir, 0, 0, 2, changed); err == nil {
		t.Fatal("reordered membership accepted")
	}
	legacy := t.TempDir()
	if err := os.WriteFile(filepath.Join(legacy, "server-1.wal"), []byte("saved"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ensureReplicaIdentity(legacy, 0, 0, 2, peers); err == nil {
		t.Fatal("unidentified existing WAL silently adopted")
	}
}

func TestDeploymentConfigAndOrdinals(t *testing.T) {
	for _, address := range []string{"localhost:1", "http://host:8091", "host:99999", "host/path:8091", ":8091"} {
		if validatePeerAddress(address) == nil {
			t.Fatal("invalid address accepted", address)
		}
	}
	for _, name := range []string{"shard-1-3", "shard-1-x", "0", ""} {
		if _, err := podVoterID(name); err == nil {
			t.Fatal("invalid or additional voter accepted", name)
		}
	}
	if id, err := podVoterID("shard-2-1"); err != nil || id != 1 {
		t.Fatal(id, err)
	}
	configPath := filepath.Join(t.TempDir(), "peers.json")
	for _, input := range []string{
		`{"Groups":[["a:8091","b:8091","c:8091"]]}`,
		`{"Groups":[["a:8091","b:8091","c:8091"],["a:8091","e:8091","f:8091"]]}`,
		`{"Groups":[["a:8091","a:8091","c:8091"],["d:8091","e:8091","f:8091"]]}`,
		`{"Groups":[["a:8091","b:8091","c:8091"],["d:8091","e:8091","f:8091"]],"unexpected":true}`,
	} {
		os.WriteFile(configPath, []byte(input), 0600)
		if _, err := loadPeerConfig(configPath); err == nil {
			t.Fatal("unsafe configuration accepted", input)
		}
	}
}

func TestExternalControllerDoesNotManageProcesses(t *testing.T) {
	// Three containers may all report PID 1. Remote identity is the configured
	// endpoint plus shard/voter identity, not an unrelated local operating PID.
	var addresses []string
	for id := 0; id < 3; id++ {
		id := id
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(nodeSnapshot{PID: 1, ShardID: 0, ShardCount: 2, Node: &Node{ID: id, Up: true, Role: "follower"}})
		}))
		t.Cleanup(server.Close)
		addresses = append(addresses, server.Listener.Addr().String())
	}
	controller := &liveController{addresses: addresses, shardCount: 2, client: &http.Client{Timeout: time.Second}}
	for id, state := range controller.states() {
		if state == nil || state.Node.ID != id {
			t.Fatal("remote state missing", id)
		}
	}
	if controller.start(0) == nil || controller.kill(0) == nil {
		t.Fatal("router must not manage Pod processes")
	}
}
