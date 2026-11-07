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
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/anishathalye/porcupine"
)

func TestNetworkProcess(t *testing.T) {
	if os.Getenv("RAFT_NODE_HELPER") != "1" {
		return
	}
	id, _ := strconv.Atoi(os.Getenv("RAFT_NODE_ID"))
	base, _ := strconv.Atoi(os.Getenv("RAFT_NODE_BASE"))
	if err := runNode(id, base, os.Getenv("RAFT_NODE_DIR"), false); err != nil {
		t.Fatal(err)
	}
}

// Real sockets, independent address spaces, SIGKILL, and recovery from each
// node's own directory. This checks one failure history, not full correctness.
// Exercise actual child processes: kill leaders, preserve retry IDs, and check
// completed client histories. In-memory fixtures cannot prove crash recovery.
func TestIndependentProcessesFailover(t *testing.T) {
	var base int
	for attempt := 0; attempt < 100; attempt++ {
		first, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		candidate := first.Addr().(*net.TCPAddr).Port
		second, e2 := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", candidate+1))
		third, e3 := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", candidate+2))
		first.Close()
		if second != nil {
			second.Close()
		}
		if third != nil {
			third.Close()
		}
		if e2 == nil && e3 == nil && candidate <= 65532 {
			base = candidate
			break
		}
	}
	if base == 0 {
		t.Fatal("no adjacent ports")
	}
	root := t.TempDir()
	var processes [3]*exec.Cmd
	stop := func(id int) {
		if processes[id] != nil {
			processes[id].Process.Kill()
			processes[id].Wait()
			processes[id] = nil
		}
	}
	t.Cleanup(func() {
		for id := range processes {
			stop(id)
		}
	})
	start := func(id int) {
		dir := filepath.Join(root, fmt.Sprint(id))
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(os.Args[0], "-test.run=^TestNetworkProcess$")
		cmd.Env = append(os.Environ(), "RAFT_NODE_HELPER=1", fmt.Sprintf("RAFT_NODE_ID=%d", id), fmt.Sprintf("RAFT_NODE_BASE=%d", base), "RAFT_NODE_DIR="+dir)
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		processes[id] = cmd
	}
	client := &http.Client{Timeout: 4 * time.Second}
	snapshot := func(id int) *nodeSnapshot {
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/state", base+id))
		if err != nil {
			return nil
		}
		defer resp.Body.Close()
		var s nodeSnapshot
		if resp.StatusCode != 200 || json.NewDecoder(resp.Body).Decode(&s) != nil || s.Fault {
			return nil
		}
		return &s
	}
	await := func(label string, condition func() bool) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if condition() {
				return
			}
			time.Sleep(25 * time.Millisecond)
		}
		t.Fatal("timed out:", label)
	}
	findLeader := func(exclude int) int {
		for id := 0; id < 3; id++ {
			if id == exclude {
				continue
			}
			s := snapshot(id)
			if s != nil && s.Node.Role == "leader" {
				return id
			}
		}
		return -1
	}
	var historyMu sync.Mutex
	var history []porcupine.Operation
	invoke := func(id int, cmd command, clientID int) (receipt, int, error) {
		b, _ := json.Marshal(cmd)
		call := time.Now().UnixNano()
		resp, err := client.Post(fmt.Sprintf("http://127.0.0.1:%d/%s", base+id, cmd.Operation), "application/json", bytes.NewReader(b))
		if err != nil {
			return receipt{}, 0, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return receipt{}, resp.StatusCode, nil
		}
		var out receipt
		if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return out, resp.StatusCode, err
		}
		historyMu.Lock()
		history = append(history, porcupine.Operation{ClientId: clientID, Input: cmd, Output: out, Call: call, Return: time.Now().UnixNano()})
		historyMu.Unlock()
		return out, resp.StatusCode, nil
	}
	write := func(id int, key, value string) receipt {
		t.Helper()
		out, status, err := invoke(id, command{Operation: "write", RequestID: "request-" + key, Key: key, Value: value}, 0)
		if err != nil || status != 200 || !out.Committed {
			t.Fatalf("write not committed: %d %v", status, err)
		}
		return out
	}
	for id := 0; id < 3; id++ {
		start(id)
	}
	await("three separate PIDs and leader", func() bool {
		seen := map[int]bool{}
		for id := 0; id < 3; id++ {
			s := snapshot(id)
			if s == nil || s.PID != processes[id].Process.Pid {
				return false
			}
			seen[s.PID] = true
		}
		return len(seen) == 3 && findLeader(-1) >= 0
	})
	old := findLeader(-1)
	firstReceipt := write(old, "x", "40")
	await("x applied everywhere", func() bool {
		for id := 0; id < 3; id++ {
			s := snapshot(id)
			if s == nil || s.Node.KV["x"] != "40" {
				return false
			}
		}
		return true
	})
	stop(old)
	await("replacement leader", func() bool { return findLeader(old) >= 0 })
	replacement := findLeader(old)
	write(replacement, "y", "20")
	// The same ID after failover returns the original entry; conflicting reuse fails.
	retry := write(replacement, "x", "40")
	if !retry.Replayed || retry.Index != firstReceipt.Index || retry.Term != firstReceipt.Term {
		t.Fatal("retry was not deduplicated after failover")
	}
	_, status, err := invoke(replacement, command{Operation: "write", RequestID: "request-x", Key: "x", Value: "wrong"}, 0)
	if err != nil || status != 409 {
		t.Fatal("conflicting ID reuse accepted", status, err)
	}
	await("survivors apply y", func() bool {
		for id := 0; id < 3; id++ {
			if id == old {
				continue
			}
			s := snapshot(id)
			if s == nil || s.Node.KV["y"] != "20" {
				return false
			}
		}
		return true
	})
	start(old)
	await("restarted server recovers and catches up", func() bool {
		s := snapshot(old)
		return s != nil && s.PID == processes[old].Process.Pid && s.Node.KV["x"] == "40" && s.Node.KV["y"] == "20"
	})
	// Concurrent overlapping writes/reads through independent HTTP handlers.
	var writers sync.WaitGroup
	for worker := 1; worker <= 3; worker++ {
		writers.Add(1)
		go func(worker int) {
			defer writers.Done()
			for i := 0; i < 4; i++ {
				cmd := command{Operation: "write", RequestID: fmt.Sprintf("w%d-%d", worker, i), Key: "shared", Value: fmt.Sprintf("%d-%d", worker, i)}
				if i%2 == 1 {
					cmd = command{Operation: "read", Key: "shared"}
				}
				_, status, err := invoke(replacement, cmd, worker)
				if err != nil || status != 200 {
					t.Errorf("concurrent operation failed: %d %v", status, err)
					return
				}
			}
		}(worker)
	}
	writers.Wait()
	if t.Failed() {
		return
	}
	// Retry an older write after a newer one; it must not restore the old value.
	write(replacement, "x", "40")
	_, status, err = invoke(replacement, command{Operation: "write", RequestID: "x-new", Key: "x", Value: "99"}, 0)
	if err != nil || status != 200 {
		t.Fatal("new x failed", status, err)
	}
	write(replacement, "x", "40")
	read, status, err := invoke(replacement, command{Operation: "read", Key: "x"}, 0)
	if err != nil || status != 200 || read.Value != "99" {
		t.Fatal("retry restored stale value", read, status, err)
	}
	control := func(id int, path string, body any) {
		t.Helper()
		b, _ := json.Marshal(body)
		response, err := client.Post(fmt.Sprintf("http://127.0.0.1:%d/%s", base+id, path), "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != 204 {
			t.Fatalf("%s failed: %d", path, response.StatusCode)
		}
	}
	await("all replicas applied before compaction", func() bool {
		for id := 0; id < 3; id++ {
			s := snapshot(id)
			if s == nil || s.Node.Commit < read.Index {
				return false
			}
		}
		return true
	})
	for id := 0; id < 3; id++ {
		control(id, "snapshot", nil)
		s := snapshot(id)
		if s == nil || s.Node.Snapshot.Index < read.Index || len(s.Node.Log) != 0 || s.Node.KV["x"] != "99" {
			t.Fatal("snapshot did not compact committed prefix", id)
		}
	}
	// Restart the leader itself: deduplication must survive its own WAL recovery.
	stop(replacement)
	start(replacement)
	await("leader after restart", func() bool { return findLeader(-1) >= 0 })
	currentLeader := findLeader(-1)
	retry = write(currentLeader, "x", "40")
	if !retry.Replayed || retry.Index != firstReceipt.Index {
		t.Fatal("WAL recovery lost deduplication")
	}
	read, status, err = invoke(currentLeader, command{Operation: "read", Key: "x"}, 0)
	if err != nil || status != 200 || read.Value != "99" {
		t.Fatal("read after recovery failed", read, status, err)
	}
	result, info := porcupine.CheckOperationsVerbose(kvModel, history, 5*time.Second)
	if artifactDir := os.Getenv("RAFT_TEST_ARTIFACT_DIR"); artifactDir != "" {
		if err := os.MkdirAll(artifactDir, 0700); err != nil {
			t.Fatal(err)
		}
		payload, _ := json.MarshalIndent(history, "", "  ")
		if err := os.WriteFile(filepath.Join(artifactDir, "history.json"), payload, 0600); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(filepath.Join(artifactDir, "history.html"))
		if err != nil {
			t.Fatal(err)
		}
		if err = porcupine.Visualize(kvModel, info, file); err != nil {
			file.Close()
			t.Fatal(err)
		}
		file.Close()
	}
	if result != porcupine.Ok {
		t.Fatalf("Porcupine did not validate history: %v", result)
	}
	t.Logf("Porcupine: %d completed operations linearizable against KV + request-ID model", len(history))
	// Both sides remain alive. The former leader cannot commit, while the
	// connected pair elects a higher-term leader and continues accepting work.
	partitioned := findLeader(-1)
	before := snapshot(partitioned)
	control(partitioned, "network", map[string]bool{"Isolated": true})
	await("leader on majority side", func() bool { return findLeader(partitioned) >= 0 })
	majority := findLeader(partitioned)
	_, status, err = invoke(partitioned, command{Operation: "write", RequestID: "partition-uncommitted", Key: "discarded", Value: "no"}, 0)
	if err != nil || status != 504 {
		t.Fatal("partitioned leader acknowledged a write", status, err)
	}
	_, status, err = invoke(partitioned, command{Operation: "read", Key: "x"}, 0)
	if err != nil || status != 504 {
		t.Fatal("partitioned leader served a read", status, err)
	}
	write(majority, "partition-good", "yes")
	control(majority, "snapshot", nil)
	compacted := snapshot(majority)
	cutOff := snapshot(partitioned)
	if cutOff == nil || cutOff.PID != before.PID || !cutOff.Isolated || cutOff.DroppedRPCs == 0 || cutOff.Node.KV["partition-good"] != "" || cutOff.Node.KV["discarded"] != "" {
		t.Fatal("partition did not isolate a running process")
	}
	control(partitioned, "network", map[string]bool{"Isolated": false})
	await("snapshot installation after healing", func() bool {
		s := snapshot(partitioned)
		return s != nil && s.PID == before.PID && s.Node.Role == "follower" && s.Node.Snapshot.Index >= compacted.Node.Snapshot.Index && s.Node.KV["partition-good"] == "yes" && s.Node.KV["discarded"] == ""
	})
	// A request compacted out of the log still returns its original receipt.
	retry = write(majority, "x", "40")
	if !retry.Replayed || retry.Index != firstReceipt.Index {
		t.Fatal("snapshot installation lost retry history")
	}
	// Keep only the current leader alive. A locally accepted proposal must not
	// become applied without replication acknowledgments from a majority.
	current := findLeader(-1)
	if current < 0 {
		t.Fatal("lost leader")
	}
	for id := 0; id < 3; id++ {
		if id != current {
			stop(id)
		}
	}
	_, status, err = invoke(current, command{Operation: "write", RequestID: "isolated", Key: "no-quorum", Value: "pending"}, 0)
	if err != nil || status != 504 {
		t.Fatal("isolated write returned success or wrong status", status, err)
	}
	_, status, err = invoke(current, command{Operation: "read", Key: "x"}, 0)
	if err != nil || status != 504 {
		t.Fatal("isolated leader returned a read", status, err)
	}
	s := snapshot(current)
	if s == nil || s.Node.KV["no-quorum"] != "" || s.Progress == nil || s.Progress.Committed {
		t.Fatal("write committed without majority")
	}
	// A timeout is not an abort. Restore a majority and retry exactly that ID.
	start((current + 1) % 3)
	await("majority restored", func() bool { peer := snapshot((current + 1) % 3); return peer != nil })
	cmd := command{Operation: "write", RequestID: "isolated", Key: "no-quorum", Value: "pending"}
	var acknowledged receipt
	await("uncertain write retried through leader changes", func() bool {
		leader := findLeader(-1)
		if leader < 0 {
			return false
		}
		out, status, err := invoke(leader, cmd, 0)
		if err != nil || status == 409 || status == 503 || status == 504 {
			return false
		}
		if status != 200 || !out.Committed {
			t.Fatal("uncertain write could not be safely retried", out, status, err)
		}
		acknowledged = out
		return true
	})
	// An uncommitted proposal can be replaced during an election. Once a retry
	// succeeds, every later retry must return that exact committed entry.
	await("acknowledged retry retains its receipt", func() bool {
		leader := findLeader(-1)
		if leader < 0 {
			return false
		}
		out, status, err := invoke(leader, cmd, 0)
		if err != nil || status == 409 || status == 503 || status == 504 {
			return false
		}
		if status != 200 || !out.Replayed || !out.Committed || out.Index != acknowledged.Index || out.Term != acknowledged.Term {
			t.Fatal("acknowledged retry lost its original receipt", out, status, err)
		}
		return true
	})
}
