package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"context"
	pb "github.com/jaiminjariwala/go-backend-mastery/17-sharded-raft-kv/api/raftv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Pin the persisted routing contract. An accidental hash/count change would
// make existing keys unreachable even though all replica files still exist.
func TestFixedShardLayoutAndHash(t *testing.T) {
	// Known FNV-1a values pin the exact hash variant, not just a distribution.
	for key, want := range map[string]int{"": 1, "a": 0, "b": 1, "hello": 1} {
		if got := shardForKey(key, 2); got != want {
			t.Fatalf("hash(%q): got %d, want %d", key, got, want)
		}
	}
	dir := t.TempDir()
	if err := ensureShardLayout(dir); err != nil {
		t.Fatal(err)
	}
	if err := ensureShardLayout(dir); err != nil {
		t.Fatal("layout changed across startup", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "layout.json"), []byte(`{"Version":1,"ShardCount":3,"Algorithm":"fnv1a32-mod-v1"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if ensureShardLayout(dir) == nil {
		t.Fatal("silently remapped a saved database")
	}
	missing := t.TempDir()
	if err := os.Mkdir(filepath.Join(missing, "shard-1"), 0700); err != nil {
		t.Fatal(err)
	}
	if ensureShardLayout(missing) == nil {
		t.Fatal("guessed routing for shard directories without a manifest")
	}
}

// Build the actual executable so this exercises the production supervisor's
// flags, storage layout, and six real child processes rather than mock groups.
func TestFixedShardsProcesses(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "raft")
	build := exec.Command("go", "build", "-race", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	// Hold seven adjacent ephemeral ports until a complete range is available.
	base := 0
	for attempt := 0; attempt < 100; attempt++ {
		first, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		candidate := first.Addr().(*net.TCPAddr).Port
		open := []net.Listener{first}
		ok := candidate <= 65529
		if ok {
			for offset := 1; offset < 7; offset++ {
				listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", candidate+offset))
				if err != nil {
					ok = false
					break
				}
				open = append(open, listener)
			}
		}
		for _, listener := range open {
			listener.Close()
		}
		if ok {
			base = candidate
			break
		}
	}
	if base == 0 {
		t.Fatal("could not reserve ports")
	}
	address := fmt.Sprintf("http://127.0.0.1:%d", base)
	data := filepath.Join(dir, "data")
	var process *exec.Cmd
	start := func() {
		process = exec.Command(binary, "-mode=sharded", "-listen", fmt.Sprintf("127.0.0.1:%d", base), fmt.Sprintf("-base-port=%d", base+1), "-data-dir", data)
		process.Stderr = os.Stderr
		if err := process.Start(); err != nil {
			t.Fatal(err)
		}
	}
	stop := func() {
		if process != nil {
			process.Process.Signal(os.Interrupt)
			process.Wait()
			process = nil
		}
	}
	t.Cleanup(stop)
	client := &http.Client{Timeout: 5 * time.Second}
	observe := func(shard int) *struct {
		Nodes   []*Node
		PIDs    []int
		ShardID int
	} {
		response, err := client.Get(fmt.Sprintf("%s/api/cluster?shard=%d", address, shard))
		if err != nil {
			return nil
		}
		defer response.Body.Close()
		var view struct {
			Nodes   []*Node
			PIDs    []int
			ShardID int
		}
		if response.StatusCode != 200 || json.NewDecoder(response.Body).Decode(&view) != nil {
			return nil
		}
		return &view
	}
	await := func(label string, condition func() bool) {
		t.Helper()
		until := time.Now().Add(12 * time.Second)
		for time.Now().Before(until) {
			if condition() {
				return
			}
			time.Sleep(25 * time.Millisecond)
		}
		t.Fatal("timed out:", label)
	}
	ready := func() bool {
		pids := map[int]bool{}
		for shard := 0; shard < 2; shard++ {
			view := observe(shard)
			if view == nil || view.ShardID != shard || len(view.Nodes) != 3 {
				return false
			}
			leader := false
			for id, n := range view.Nodes {
				if !n.Up || view.PIDs[id] == 0 {
					return false
				}
				pids[view.PIDs[id]] = true
				leader = leader || n.Role == "leader"
			}
			if !leader {
				return false
			}
		}
		return len(pids) == 6
	}
	invoke := func(path string, body any) (receipt, int) {
		t.Helper()
		data, _ := json.Marshal(body)
		response, err := client.Post(address+path, "application/json", bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var result receipt
		if response.StatusCode == 200 && path == "/api/kv" {
			if json.NewDecoder(response.Body).Decode(&result) != nil {
				t.Fatal("bad receipt")
			}
		}
		return result, response.StatusCode
	}
	write := func(key, value, id string) receipt {
		t.Helper()
		var out receipt
		// Availability samples are not a leadership lease. A recovered peer may
		// start an election between observation and the request. Retry only the
		// same operation identity when that transition leaves its outcome unknown.
		await("committed response for "+id, func() bool {
			result, status := invoke("/api/kv", command{Operation: "write", Key: key, Value: value, RequestID: id})
			if status == 409 || status == 503 || status == 504 {
				return false
			}
			if status != 200 || !result.Committed || result.ShardID != shardForKey(key, 2) || result.RequestID != id {
				t.Fatal("wrong shard/write response", result, status)
			}
			out = result
			return true
		})
		return out
	}
	start()
	await("six independent replicas with leaders", ready)
	await("protobuf peer traffic on all six replicas", func() bool {
		for offset := 0; offset < 6; offset++ {
			response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/state", base+1+offset))
			if err != nil {
				return false
			}
			var state nodeSnapshot
			err = json.NewDecoder(response.Body).Decode(&state)
			response.Body.Close()
			if err != nil || state.Transport != "grpc" || state.PeerRPCs == 0 {
				return false
			}
		}
		return true
	})

	keys := [2]string{"a", "b"}
	first := [2]receipt{}
	for shard, key := range keys {
		first[shard] = write(key, "initial", "request-"+key)
	}
	for shard := 0; shard < 2; shard++ {
		seenClient := false
		for id := 0; id < 3; id++ {
			response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/state", base+1+shard*3+id))
			if err != nil {
				t.Fatal(err)
			}
			var state nodeSnapshot
			err = json.NewDecoder(response.Body).Decode(&state)
			response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			seenClient = seenClient || state.ClientRPCs > 0
		}
		if !seenClient {
			t.Fatal("controller did not use the protobuf Execute RPC", shard)
		}
	}
	await("keys only on their owning groups", func() bool {
		for shard, key := range keys {
			view := observe(shard)
			if view == nil {
				return false
			}
			for _, n := range view.Nodes {
				if n.KV[key] != "initial" || n.KV[keys[1-shard]] != "" {
					return false
				}
			}
		}
		return true
	})
	// A direct request to the wrong group's leader must fail too: guarding only
	// the router would let a client create duplicate ownership by bypassing it.
	wrong := observe(1)
	wrongLeader := 0
	for _, n := range wrong.Nodes {
		if n.Role == "leader" {
			wrongLeader = n.ID
		}
	}
	bad, _ := json.Marshal(command{Operation: "write", Key: keys[0], Value: "wrong", RequestID: "bad"})
	response, err := client.Post(fmt.Sprintf("http://127.0.0.1:%d/write", base+1+3+wrongLeader), "application/json", bytes.NewReader(bad))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 421 {
		t.Fatal("wrong-shard write accepted")
	}
	// Peer identities are group-local. A valid local node ID from another
	// group is still an invalid sender for this replica's consensus traffic.
	peer := 0
	if wrongLeader == 0 {
		peer = 1
	}
	// Exercise the protobuf ingress itself, not only its HTTP compatibility path.
	var rpcClients rpcPool
	defer rpcClients.close()
	wrongRPC, err := rpcClients.client(base + 1 + 3 + wrongLeader)
	if err != nil {
		t.Fatal(err)
	}
	rpcCtx, cancelRPC := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelRPC()
	_, err = wrongRPC.Deliver(rpcCtx, &pb.PeerMessage{ShardId: 0, From: int32((wrongLeader + 1) % 3), To: int32(wrongLeader), Kind: pb.MessageKind_VOTE, Term: 100})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatal("protobuf accepted a foreign shard message", err)
	}
	_, err = wrongRPC.Execute(rpcCtx, &pb.ClientCommand{ShardId: 1, Operation: pb.Operation_WRITE, Key: keys[0], Value: "wrong", RequestId: "wrong-grpc"})
	if status.Code(err) != codes.OutOfRange {
		t.Fatal("protobuf accepted a key owned by another shard", err)
	}
	foreign, _ := json.Marshal(Message{ShardID: 0, From: peer, To: wrongLeader, Kind: "vote", Term: 100})
	response, err = client.Post(fmt.Sprintf("http://127.0.0.1:%d/message", base+1+3+wrongLeader), "application/json", bytes.NewReader(foreign))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 400 {
		t.Fatal("foreign-shard RPC accepted")
	}
	for shard, key := range keys {
		write(key, "latest", "updated-"+key)
		view := observe(shard)
		leader := 0
		for _, n := range view.Nodes {
			if n.Role == "leader" {
				leader = n.ID
			}
		}
		_, status := invoke(fmt.Sprintf("/api/cluster?shard=%d", shard), map[string]any{"Action": "snapshot", "ID": leader})
		if status != 200 {
			t.Fatal("snapshot failed", status)
		}
		retry := write(key, "initial", "request-"+key)
		if !retry.Replayed || retry.Index != first[shard].Index {
			t.Fatal("deduplication lost across snapshot", retry)
		}
		out, status := invoke("/api/kv", command{Operation: "read", Key: key})
		if status != 200 || out.Value != "latest" || out.ShardID != shard {
			t.Fatal("read failed after snapshot", out, status)
		}
	}
	// Remove shard 1's quorum. Shard 2 must continue; its term, data, and votes
	// are independent. The router must not put shard 1's failed write in shard 2.
	view := observe(0)
	leader := 0
	for _, n := range view.Nodes {
		if n.Role == "leader" {
			leader = n.ID
		}
	}
	for _, n := range view.Nodes {
		if n.ID != leader {
			_, status := invoke("/api/cluster?shard=0", map[string]any{"Action": "toggle", "ID": n.ID})
			if status != 200 {
				t.Fatal("stop failed")
			}
		}
	}
	_, status := invoke("/api/kv", command{Operation: "write", Key: keys[0], Value: "pending", RequestID: "uncertain-a"})
	if status != 504 {
		t.Fatal("shard without quorum returned success", status)
	}
	write(keys[1], "still-available", "independent-b")
	for _, n := range observe(1).Nodes {
		if _, exists := n.KV[keys[0]]; exists {
			t.Fatal("router failed over to the wrong shard")
		}
	}
	_, status = invoke("/api/cluster?shard=0", map[string]any{"Action": "heal"})
	if status != 200 {
		t.Fatal("heal failed")
	}
	running := 0
	for _, node := range observe(0).Nodes {
		if node.Up {
			running++
		}
	}
	if running != 1 {
		t.Fatal("healing unexpectedly restarted stopped servers")
	}
	_, status = invoke("/api/cluster?shard=0", map[string]any{"Action": "recover"})
	if status != 200 {
		t.Fatal("group recovery failed", status)
	}

	await("first shard recovered", ready)
	out := write(keys[0], "pending", "uncertain-a")
	// An uncommitted proposal may have been replaced during the election.
	// Its first eventual success need not be a replay. Once acknowledged, a
	// second retry must return the same receipt without executing again.
	again := write(keys[0], "pending", "uncertain-a")
	if !again.Replayed || again.Index != out.Index {
		t.Fatal("acknowledged retry was duplicated")
	}
	// Route deletion through the real HTTP adapter, then verify that its absence
	// survives snapshotting and a restart of all six replicas.
	removedKey := "removed:user"
	write(removedKey, "temporary", "create-removal")
	deleted, status := invoke("/api/kv", command{Operation: "delete", Key: removedKey, RequestID: "delete-row"})
	if status != 200 || deleted.Operation != "delete" || !deleted.Committed || deleted.Found {
		t.Fatal("bad deletion response", deleted, status)
	}
	removedShard := shardForKey(removedKey, 2)
	await("deletion applied to all owning replicas", func() bool {
		for _, replica := range observe(removedShard).Nodes {
			if _, exists := replica.KV[removedKey]; exists {
				return false
			}
		}
		return true
	})
	for _, replica := range observe(removedShard).Nodes {
		if replica.Role == "leader" {
			_, status = invoke(fmt.Sprintf("/api/cluster?shard=%d", removedShard), map[string]any{"Action": "snapshot", "ID": replica.ID})
			if status != 200 {
				t.Fatal("snapshot deletion", status)
			}
		}
	}
	stop()
	start()
	await("all groups recover after supervisor restart", ready)
	missing, status := invoke("/api/kv", command{Operation: "read", Key: removedKey})
	if status != 200 || missing.Found {
		t.Fatal("deleted row returned after restart", missing, status)
	}
	retried, status := invoke("/api/kv", command{Operation: "delete", Key: removedKey, RequestID: "delete-row"})
	if status != 200 || !retried.Replayed || retried.Index != deleted.Index {
		t.Fatal("deletion identity lost", retried, status)
	}
	for shard, key := range keys {
		out, status := invoke("/api/kv", command{Operation: "read", Key: key})
		want := "pending"
		if shard == 1 {
			want = "still-available"
		}
		if status != 200 || out.Value != want || out.ShardID != shard {
			t.Fatal("lost shard state after restart", out, status)
		}
	}
}
