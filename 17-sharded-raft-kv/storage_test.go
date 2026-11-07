package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// durable opens a disk-backed fixture and registers cleanup even when a test fails.
func durable(t *testing.T, dir string) *Cluster {
	t.Helper()
	c, err := NewDurableCluster(dir)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestDiskRestart(t *testing.T) {
	dir := t.TempDir()
	c := durable(t, dir)
	advance(c, 20)
	if err := c.Write("x", "40"); err != nil {
		t.Fatal(err)
	}
	advance(c, 10)
	term := leader(c).Term
	c.store.close()
	c = durable(t, dir)
	defer c.store.close()
	for _, n := range c.Nodes {
		if n.KV["x"] != "40" || n.Role != "follower" || n.Term < term {
			t.Fatal("bad recovery", n)
		}
	}
	advance(c, 30)
	if leader(c) == nil {
		t.Fatal("no election after recovery")
	}
}
func TestUncommittedNotAppliedAndNodeReload(t *testing.T) {
	dir := t.TempDir()
	c := durable(t, dir)
	defer func() { c.store.close() }()
	advance(c, 20)
	l := leader(c)
	for _, n := range c.Nodes {
		if n.ID != l.ID {
			c.Toggle(n.ID)
		}
	}
	if err := c.Write("pending", "no"); err != nil {
		t.Fatal(err)
	}
	c.Toggle(l.ID)
	if len(c.Nodes[l.ID].Log) != 0 {
		t.Fatal("memory not discarded")
	}
	c.Toggle(l.ID)
	n := c.Nodes[l.ID]
	if len(n.Log) != 2 || n.KV["pending"] != "" || n.Commit != 1 {
		t.Fatal("treated durable as committed", n)
	}
}
func TestVotePersistsBeforeReply(t *testing.T) {
	dir := t.TempDir()
	c := durable(t, dir)
	c.receive(Message{From: 0, To: 1, Term: 2, Kind: "vote"})
	if len(c.Messages) != 1 || !c.Messages[0].OK {
		t.Fatal("no vote")
	}
	c.store.close()
	c = durable(t, dir)
	defer c.store.close()
	if c.Nodes[1].Vote != 0 || c.Nodes[1].Term != 2 {
		t.Fatal("lost vote")
	}
	c.receive(Message{From: 2, To: 1, Term: 2, Kind: "vote"})
	if c.Messages[len(c.Messages)-1].OK {
		t.Fatal("voted twice")
	}
}
func TestTornTailAndCorruption(t *testing.T) {
	for _, tail := range [][]byte{{0, 0}, {0, 0, 0, 10, 0, 0, 0, 0, 1, 2}} {
		dir := t.TempDir()
		c := durable(t, dir)
		advance(c, 20)
		term := c.Nodes[0].Term
		path := c.store.path(0)
		c.store.close()
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		f.Write(tail)
		f.Close()
		c = durable(t, dir)
		if c.Nodes[0].Term != term {
			t.Fatal("lost prior record")
		}
		advance(c, 10)
		c.store.close()
		c = durable(t, dir)
		c.store.close()
	}
	dir := t.TempDir()
	c := durable(t, dir)
	advance(c, 20)
	path := c.store.path(0)
	c.store.close()
	b, _ := os.ReadFile(path)
	b[len(b)-1] ^= 1
	os.WriteFile(path, b, 0600)
	if c, err := NewDurableCluster(dir); err == nil {
		c.store.close()
		t.Fatal("accepted checksum corruption")
	}
}
func TestStorageFailureSuppressesAck(t *testing.T) {
	c := durable(t, t.TempDir())
	defer c.store.close()
	// Removing this test-only WAL makes the next append fail before a reply.
	if err := os.Remove(c.store.path(1)); err != nil {
		t.Fatal(err)
	}
	c.receive(Message{From: 0, To: 1, Term: 1, Kind: "append", Log: []Entry{{Term: 1, Key: "x", Value: "40"}}})
	if c.storageErr == nil || len(c.Messages) != 0 {
		t.Fatal("acknowledged failed persistence")
	}
}
func TestExclusiveStore(t *testing.T) {
	dir := t.TempDir()
	c := durable(t, dir)
	defer c.store.close()
	if other, err := NewDurableCluster(dir); err == nil {
		other.store.close()
		t.Fatal("multiple writers allowed")
	}
}

// The helper is terminated without Close or any graceful-shutdown handler.
func TestCrashRecoveryProcess(t *testing.T) {
	if os.Getenv("RAFT_CRASH_HELPER") == "1" {
		c, err := NewDurableCluster(os.Getenv("RAFT_CRASH_DIR"))
		if err != nil {
			os.Exit(3)
		}
		advance(c, 20)
		if c.Write("x", "40") != nil {
			os.Exit(4)
		}
		advance(c, 10)
		if os.WriteFile(filepath.Join(os.Getenv("RAFT_CRASH_DIR"), "ready"), []byte("ready"), 0600) != nil {
			os.Exit(5)
		}
		// SIGKILL cannot run deferred closes.
		p, _ := os.FindProcess(os.Getpid())
		p.Kill()
		os.Exit(6)
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestCrashRecoveryProcess$")
	cmd.Env = append(os.Environ(), "RAFT_CRASH_HELPER=1", "RAFT_CRASH_DIR="+dir)
	if cmd.Run() == nil {
		t.Fatal("helper did not crash")
	}
	if _, err := os.Stat(filepath.Join(dir, "ready")); err != nil {
		t.Fatal("helper failed before committed write", err)
	}
	c := durable(t, dir)
	defer c.store.close()
	for _, n := range c.Nodes {
		if n.KV["x"] != "40" {
			t.Fatal("lost acknowledged history after process death")
		}
	}
}
