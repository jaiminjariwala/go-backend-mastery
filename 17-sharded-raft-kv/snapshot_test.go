package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Compaction must preserve pending work and retry identity while replacing
// only the committed prefix; discarding either can change client-visible results.
func TestSnapshotRetainsUncommittedSuffixAndRetryIDs(t *testing.T) {
	dir := t.TempDir()
	c := durable(t, dir)
	advance(c, 20)
	n := leader(c)
	index, _, _, err := c.propose(n, command{Operation: "write", RequestID: "original", Key: "x", Value: "40"})
	if err != nil {
		t.Fatal(err)
	}
	advance(c, 10)
	if err = c.Write("x", "99"); err != nil {
		t.Fatal(err)
	}
	advance(c, 10)
	for id := 0; id < 3; id++ {
		if id != n.ID {
			c.Toggle(id)
		}
	}
	if err = c.Write("pending", "uncommitted"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(c.store.path(n.ID))
	boundary := n.Commit
	if err = c.TakeSnapshot(n); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(c.store.path(n.ID))
	if n.Snapshot.Index != boundary || len(n.Log) != 1 || n.KV["pending"] != "" || n.Requests["original"].Index != index || after.Size() >= before.Size() {
		t.Fatal("bad compaction or premature application")
	}
	// A process dying before rename may leave an incomplete temp file. Recovery
	// reads only the named WAL and must ignore this abandoned candidate.
	if err = os.WriteFile(filepath.Join(dir, ".snapshot-abandoned"), []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	id := n.ID
	c.store.close()
	c = durable(t, dir)
	defer c.store.close()
	n = c.Nodes[id]
	if n.Snapshot.Index != boundary || n.lastIndex() != boundary+1 || n.KV["x"] != "99" || n.KV["pending"] != "" {
		t.Fatal("incorrect snapshot recovery")
	}
	n.Role = "leader"
	retry, _, replayed, err := c.propose(n, command{Operation: "write", RequestID: "original", Key: "x", Value: "40"})
	if err != nil || !replayed || retry != index || n.KV["x"] != "99" {
		t.Fatal("retry information lost during compaction")
	}
}

func TestSnapshotSuffixSurvivesDelayedShorterAppend(t *testing.T) {
	c := NewCluster()
	advance(c, 20)
	n := leader(c)
	if err := c.Write("x", "40"); err != nil {
		t.Fatal(err)
	}
	advance(c, 10)
	if err := c.TakeSnapshot(n); err != nil {
		t.Fatal(err)
	}
	old := Message{From: n.ID, To: (n.ID + 1) % 3, Term: n.Term, Kind: "snapshot", Snapshot: n.Snapshot, Commit: n.Commit}
	if err := c.Write("y", "20"); err != nil {
		t.Fatal(err)
	}
	advance(c, 10)
	follower := c.Nodes[old.To]
	last := follower.lastIndex()
	c.receive(old)
	if follower.lastIndex() != last || follower.KV["y"] != "20" {
		t.Fatal("delayed snapshot removed a newer committed suffix")
	}
}

func TestFailedSnapshotInstallDoesNotAcknowledge(t *testing.T) {
	dir := t.TempDir()
	c := durable(t, dir)
	defer c.store.close()
	advance(c, 20)
	n := leader(c)
	if c.Write("x", "40") != nil {
		t.Fatal("write failed")
	}
	advance(c, 10)
	if err := c.TakeSnapshot(n); err != nil {
		t.Fatal(err)
	}
	c.Messages = nil
	// Force CreateTemp to fail without damaging the previously synced WAL.
	// The receiver must latch a fault and suppress its replication reply.
	c.store.dir = filepath.Join(dir, "missing-directory")
	c.receive(Message{From: n.ID, To: (n.ID + 1) % 3, Term: n.Term, Kind: "snapshot", Snapshot: n.Snapshot, Commit: n.Commit})
	if c.storageErr == nil || len(c.Messages) != 0 {
		t.Fatal("acknowledged an unpersisted snapshot")
	}
	c.store.dir = dir
}

func TestSnapshotRecoveryAfterSIGKILL(t *testing.T) {
	if os.Getenv("RAFT_SNAPSHOT_CRASH") == "1" {
		c, err := NewDurableCluster(os.Getenv("RAFT_SNAPSHOT_DIR"))
		if err != nil {
			os.Exit(2)
		}
		advance(c, 20)
		n := leader(c)
		if _, _, _, err = c.propose(n, command{Operation: "write", RequestID: "survives", Key: "x", Value: "40"}); err != nil {
			os.Exit(3)
		}
		advance(c, 10)
		for _, node := range c.Nodes {
			if c.TakeSnapshot(node) != nil {
				os.Exit(4)
			}
		}
		if os.WriteFile(filepath.Join(os.Getenv("RAFT_SNAPSHOT_DIR"), "ready"), nil, 0600) != nil {
			os.Exit(5)
		}
		process, _ := os.FindProcess(os.Getpid())
		process.Kill()
		os.Exit(6)
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestSnapshotRecoveryAfterSIGKILL$")
	cmd.Env = append(os.Environ(), "RAFT_SNAPSHOT_CRASH=1", "RAFT_SNAPSHOT_DIR="+dir)
	if cmd.Run() == nil {
		t.Fatal("helper did not crash")
	}
	if _, err := os.Stat(filepath.Join(dir, "ready")); err != nil {
		t.Fatal("snapshot was not installed", err)
	}
	c := durable(t, dir)
	defer c.store.close()
	for _, n := range c.Nodes {
		if n.Snapshot.Index == 0 || len(n.Log) != 0 || n.KV["x"] != "40" || n.Requests["survives"].Index == 0 {
			t.Fatal("lost snapshotted state after crash")
		}
	}
	advance(c, 30)
	if leader(c) == nil {
		t.Fatal("could not elect after compacted-log recovery")
	}
}
