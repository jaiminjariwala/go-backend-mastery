package main

import "fmt"

// RequestResult is part of the database state, not a disposable cache. Keeping
// it in snapshots prevents an old retry from executing again after compaction.
type RequestResult struct {
	Delete      bool `json:",omitempty"`
	Key, Value  string
	Index, Term int
}

// Snapshot replaces the committed prefix. Log[0] is now entry Index+1; Raft
// indexes never restart at zero when we discard old entries.
type Snapshot struct {
	Index, Term int
	KV          map[string]string
	Requests    map[string]RequestResult
}

// lastIndex stays absolute. After compacting through entry 12, a two-entry
// retained log ends at entry 14, not entry 2.
func (n *Node) lastIndex() int { return n.Snapshot.Index + len(n.Log) }

// termAt also recognizes the compacted boundary, whose term is needed to
// compare log freshness even though its original entry has been discarded.
func (n *Node) termAt(index int) (int, bool) {
	if index == n.Snapshot.Index {
		return n.Snapshot.Term, true
	}
	e, ok := n.entryAt(index)
	return e.Term, ok
}

// entryAt translates an absolute Raft index into the retained slice offset.
// Snapshot-covered entries cannot be retrieved from this slice.
func (n *Node) entryAt(index int) (Entry, bool) {
	// Example: with snapshot index 12, absolute entry 13 is array offset 0.
	offset := index - n.Snapshot.Index - 1
	if offset < 0 || offset >= len(n.Log) {
		return Entry{}, false
	}
	return n.Log[offset], true
}

// lastTerm works for both a retained suffix and a fully compacted log.
func (n *Node) lastTerm() int { term, _ := n.termAt(n.lastIndex()); return term }

// matches checks operation identity, including durable receipts when a write
// has already moved out of the retained log into the snapshot.
func (n *Node) matches(index int, e Entry) bool {
	if stored, ok := n.entryAt(index); ok {
		return stored == e
	}
	// Compacted writes can still be identified by their durable retry receipt.
	result, ok := n.Requests[e.RequestID]
	return e.RequestID != "" && e.ReadKey == "" && ok && result.Index == index && result.Term == e.Term && result.Key == e.Key && result.Value == e.Value && result.Delete == e.Delete
}

// cloneSnapshot prevents later map mutations from changing a checkpoint.
func cloneSnapshot(s Snapshot) Snapshot {
	// A checkpoint must not change later when the live database map changes.
	cloned := Snapshot{Index: s.Index, Term: s.Term, KV: map[string]string{}, Requests: map[string]RequestResult{}}
	for k, v := range s.KV {
		cloned.KV[k] = v
	}
	for k, v := range s.Requests {
		cloned.Requests[k] = v
	}
	return cloned
}

// Snapshot is a local maintenance operation: only an already committed prefix
// can be replaced. The uncommitted suffix remains available for Raft to resolve.
func (c *Cluster) TakeSnapshot(n *Node) error {
	if c.storageErr != nil {
		return c.storageErr
	}
	if !n.Up {
		return fmt.Errorf("server is stopped")
	}
	if n.Commit <= n.Snapshot.Index {
		return fmt.Errorf("no newly committed entries to snapshot")
	}
	term, ok := n.termAt(n.Commit)
	if !ok {
		return fmt.Errorf("commit index is unavailable")
	}
	updated := *n
	updated.Snapshot = cloneSnapshot(Snapshot{Index: n.Commit, Term: term, KV: n.KV, Requests: n.Requests})
	updated.Log = append([]Entry{}, n.Log[n.Commit-n.Snapshot.Index:]...)
	// Do not discard memory until the replacement WAL is atomically installed.
	if c.store != nil {
		if err := c.store.compact(&updated); err != nil {
			c.storageErr = err
			c.Messages = nil
			return err
		}
	}
	n.Snapshot = updated.Snapshot
	n.Log = updated.Log
	if c.Proposal != nil && c.Proposal.Leader == n.ID && c.Proposal.Index <= n.Commit {
		c.Proposal.Committed = true
	}
	c.note(fmt.Sprintf("Server %d snapshotted through entry %d; retained %d newer log entries.", n.ID+1, n.Snapshot.Index, len(n.Log)))
	return nil
}

// A sender includes its snapshot boundary and retained suffix. This learning
// transport sends the full suffix, rather than optimized AppendEntries batches.
// A lagging follower can therefore install a snapshot and catch up in one RPC.
func (c *Cluster) acceptAppend(n *Node, m Message) bool {
	base := m.Snapshot.Index
	end := base + len(m.Log)
	if base < 0 || m.Commit < base || m.Commit > end || end < n.Commit {
		return false
	}
	sender := &Node{Snapshot: m.Snapshot, Log: m.Log}
	if base < n.Snapshot.Index {
		// This follower compacted further than the sender. The sender still
		// has to agree with the follower's already committed boundary.
		term, ok := sender.termAt(n.Snapshot.Index)
		if !ok || term != n.Snapshot.Term {
			return false
		}
	}
	if base > n.Snapshot.Index && base <= n.Commit {
		term, ok := n.termAt(base)
		if !ok || term != m.Snapshot.Term {
			return false
		}
	}
	// A committed entry can never be replaced, even by a higher-term sender.
	for index := max(base, n.Snapshot.Index) + 1; index <= n.Commit; index++ {
		old, oldOK := n.entryAt(index)
		next, nextOK := sender.entryAt(index)
		if !oldOK || !nextOK || old != next {
			return false
		}
	}
	if base > n.Snapshot.Index {
		boundaryTerm, known := n.termAt(base)
		var suffix []Entry
		if known && boundaryTerm == m.Snapshot.Term {
			// A matching boundary lets us retain newer local entries. A
			// conflicting uncommitted tail has to be replaced by the leader.
			suffix = append([]Entry{}, n.Log[base-n.Snapshot.Index:]...)
		}
		n.Snapshot = cloneSnapshot(m.Snapshot)
		n.Log = suffix
		n.Commit = max(n.Commit, base)
	}
	// A delayed, shorter RPC must not remove a matching suffix that this server
	// has already acknowledged. Replace only at the first actual conflict.
	incoming := m.Log[n.Snapshot.Index-base:]
	i := 0
	for i < len(n.Log) && i < len(incoming) && n.Log[i] == incoming[i] {
		i++
	}
	if i < len(incoming) {
		n.Log = append(append([]Entry{}, n.Log[:i]...), incoming[i:]...)
	}
	n.Commit = max(n.Commit, m.Commit)
	c.apply(n)
	return true
}
