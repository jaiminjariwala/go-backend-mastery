package main

import "testing"

// A deletion is a durable, retryable instruction. Replaying an old deletion
// after a later recreation must return its receipt without deleting the new row.
func TestDeleteCommitSnapshotRecoveryAndRetry(t *testing.T) {
	dir := t.TempDir()
	c := durable(t, dir)
	advance(c, 24)
	n := leader(c)
	if n == nil {
		t.Fatal("no leader")
	}
	propose := func(operation, value, requestID string) (int, Entry) {
		t.Helper()
		index, entry, _, err := c.propose(n, command{Operation: operation, Key: "person:1", Value: value, RequestID: requestID})
		if err != nil {
			t.Fatal(err)
		}
		advance(c, 12)
		if n.Commit < index {
			t.Fatal("operation did not commit")
		}
		return index, entry
	}
	propose("write", "Maya", "create")
	index, deletion := propose("delete", "", "remove")
	for _, replica := range c.Nodes {
		if _, exists := replica.KV["person:1"]; exists {
			t.Fatal("committed deletion not applied", replica.ID)
		}
	}
	if err := c.TakeSnapshot(n); err != nil {
		t.Fatal(err)
	}
	if !n.matches(index, deletion) {
		t.Fatal("snapshot lost deletion receipt")
	}
	propose("write", "Maya Chen", "recreate")
	oldIndex, _, replayed, err := c.propose(n, command{Operation: "delete", Key: "person:1", RequestID: "remove"})
	if err != nil || !replayed || oldIndex != index || n.KV["person:1"] != "Maya Chen" {
		t.Fatal("retry executed an old deletion again", err)
	}
	if _, _, _, err := c.propose(n, command{Operation: "write", Key: "person:1", RequestID: "remove"}); err == nil {
		t.Fatal("write reused a deletion ID")
	}
	if err := c.TakeSnapshot(n); err != nil {
		t.Fatal(err)
	}
	if err := c.store.close(); err != nil {
		t.Fatal(err)
	}
	recovered, err := NewDurableCluster(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.store.close()
	if recovered.Nodes[n.ID].KV["person:1"] != "Maya Chen" || !recovered.Nodes[n.ID].Requests["remove"].Delete {
		t.Fatal("recovery lost rows or deletion identity")
	}
}

// A single replica must not apply a proposed deletion before majority agreement.
func TestDeleteWithoutQuorumRemainsPending(t *testing.T) {
	c := NewCluster()
	advance(c, 24)
	n := leader(c)
	if n == nil {
		t.Fatal("no leader")
	}
	_, _, _, err := c.propose(n, command{Operation: "write", Key: "person:1", Value: "Maya", RequestID: "create"})
	if err != nil {
		t.Fatal(err)
	}
	advance(c, 12)
	for _, peer := range c.Nodes {
		if peer.ID != n.ID {
			c.Toggle(peer.ID)
		}
	}
	index, _, _, err := c.propose(n, command{Operation: "delete", Key: "person:1", RequestID: "remove"})
	if err != nil {
		t.Fatal(err)
	}
	advance(c, 20)
	if n.Commit >= index || n.KV["person:1"] != "Maya" {
		t.Fatal("uncommitted deletion changed visible rows")
	}
}
