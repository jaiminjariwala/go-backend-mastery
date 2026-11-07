package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	pb "github.com/jaiminjariwala/go-backend-mastery/17-sharded-raft-kv/api/raftv1"
)

// command describes the client's intent. A write's RequestID identifies the
// operation across retries; choosing another server does not create a new ID.
type command struct {
	Operation, RequestID, Key, Value string
}

// receipt is returned only after the exact operation is committed. A write
// receipt describes that original write, even if a later write changed the key.
type receipt struct {
	ShardID, ServerID                int
	Operation, RequestID, Key, Value string
	Index, Term                      int
	Committed, Replayed, Found       bool
}

// propose runs with the node mutex held. It finds an existing request or appends
// one entry, persists before sending, and returns its identity to the waiter.
// Returning from this function alone does NOT mean the write is committed.
func (c *Cluster) propose(n *Node, cmd command) (int, Entry, bool, error) {
	return c.proposeWithReadID(n, cmd, "")
}

// Tests supply deterministic read identities. Live requests still use fresh
// cryptographic nonces, through the ordinary propose method above.
func (c *Cluster) proposeWithReadID(n *Node, cmd command, readID string) (int, Entry, bool, error) {
	if c.storageErr != nil {
		return 0, Entry{}, false, c.storageErr
	}
	if n.Role != "leader" {
		return 0, Entry{}, false, fmt.Errorf("not leader")
	}
	if cmd.Operation != "read" {
		// Applied receipts survive snapshotting. Check them before scanning the
		// retained log, because the original entry may already be compacted.
		if result, ok := n.Requests[cmd.RequestID]; ok {
			if result.Key != cmd.Key || result.Value != cmd.Value || result.Delete != (cmd.Operation == "delete") {
				return 0, Entry{}, false, fmt.Errorf("request ID already belongs to a different write")
			}
			e := Entry{Term: result.Term, Key: result.Key, Value: result.Value, Delete: result.Delete, RequestID: cmd.RequestID}
			c.Proposal = &Proposal{Leader: n.ID, Index: result.Index, At: c.Tick, Entry: e, Committed: true}
			return result.Index, e, true, nil
		}
		for i, e := range n.Log {
			// An earlier attempt can still be pending. Reuse it instead of
			// appending a second operation with the same client request ID.
			if e.RequestID == cmd.RequestID && e.ReadKey == "" && e.Key != "" {
				if e.Key != cmd.Key || e.Value != cmd.Value || e.Delete != (cmd.Operation == "delete") {
					return 0, Entry{}, false, fmt.Errorf("request ID already belongs to a different write")
				}
				index := n.Snapshot.Index + i + 1
				c.Proposal = &Proposal{Leader: n.ID, Index: index, At: c.Tick, Entry: e}
				return index, e, true, nil
			}
		}
	}
	e := Entry{Term: n.Term, Key: cmd.Key, Value: cmd.Value, Delete: cmd.Operation == "delete", RequestID: cmd.RequestID}
	if cmd.Operation == "read" {
		// A fresh barrier must be replicated in THIS request's lifetime. Merely
		// checking Role == leader would let an isolated old leader serve stale data.
		if readID == "" {
			var nonce [16]byte
			if _, err := rand.Read(nonce[:]); err != nil {
				return 0, Entry{}, false, err
			}
			readID = hex.EncodeToString(nonce[:])
		}
		e = Entry{Term: n.Term, ReadKey: cmd.Key, RequestID: readID}
	}
	n.Log = append(n.Log, e)
	n.Match[n.ID] = n.lastIndex()
	if cmd.Operation != "read" {
		c.Proposal = &Proposal{Leader: n.ID, Index: n.lastIndex(), At: c.Tick, Entry: e}
	}
	c.note(fmt.Sprintf("%s for %s proposed at entry %d; waiting for commitment.", cmd.Operation, cmd.Key, n.lastIndex()))
	c.broadcast(n)
	return n.lastIndex(), e, false, c.storageErr
}

// nodeClientHandler validates an operation, proposes it, then waits for its
// exact log identity to become committed. It releases the mutex between polls
// so votes, replication replies, and maintenance can continue while it waits.
func nodeClientHandler(c *Cluster, n *Node, mu *sync.Mutex, operation string) http.HandlerFunc {
	return nodeClientHandlerForShard(c, n, mu, operation, 0, 1)
}

// nodeClientHandlerForShard adds the ownership boundary to the commit-aware
// handler: a misplaced key is rejected rather than written into another shard.
func nodeClientHandlerForShard(c *Cluster, n *Node, mu *sync.Mutex, operation string, shardID, shardCount int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var cmd command
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&cmd) != nil {
			http.Error(w, "JSON command required", 400)
			return
		}
		cmd.Operation = operation
		result, code, err := executeCommand(r.Context(), c, n, mu, cmd, shardID, shardCount)
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(result)
	}
}

// executeCommand is shared by the protobuf service and the compatibility HTTP
// adapter. A deadline cancels waiting, never rolls back an already proposed entry.
func executeCommand(ctx context.Context, c *Cluster, n *Node, mu *sync.Mutex, cmd command, shardID, shardCount int) (receipt, int, error) {
	fail := func(code int, message string) (receipt, int, error) {
		return receipt{}, code, fmt.Errorf("%s", message)
	}
	if cmd.Operation != "write" && cmd.Operation != "read" && cmd.Operation != "delete" {
		return fail(400, "Operation write/read/delete required.")
	}
	if len(cmd.Key) == 0 || len(cmd.Key) > 64 || len(cmd.Value) > 256 || (cmd.Operation != "read" && (len(cmd.RequestID) == 0 || len(cmd.RequestID) > 128)) {
		return fail(400, "Use a key, a write request ID (1-128 bytes), and a value up to 256 bytes.")
	}
	if shardForKey(cmd.Key, shardCount) != shardID {
		return fail(421, "Key belongs to another shard; send it through the router.")
	}
	if cmd.Operation == "delete" {
		cmd.Value = ""
	}
	mu.Lock()
	if err := ctx.Err(); err != nil {
		mu.Unlock()
		return fail(499, "Request canceled before proposal.")
	}
	index, entry, replayed, err := c.propose(n, cmd)
	fault := c.storageErr != nil
	mu.Unlock()
	if err != nil {
		code := 409
		if fault {
			code = 503
		}
		return fail(code, err.Error())
	}
	deadline := time.NewTimer(2500 * time.Millisecond)
	defer deadline.Stop()
	poll := time.NewTicker(10 * time.Millisecond)
	defer poll.Stop()
	for {
		mu.Lock()
		valid := n.matches(index, entry)
		if c.storageErr != nil {
			mu.Unlock()
			return fail(503, "Storage fault; no committed response confirmed. Retry the same mutation ID after recovery.")
		}
		if valid && n.Commit >= index {
			value, found := n.KV[cmd.Key]
			if cmd.Operation == "write" {
				value = cmd.Value
				found = true
			}
			if cmd.Operation == "delete" {
				value = ""
				found = false
			}
			result := receipt{ServerID: n.ID, ShardID: shardID, Operation: cmd.Operation, RequestID: cmd.RequestID, Key: cmd.Key, Value: value, Index: index, Term: entry.Term, Committed: true, Replayed: replayed, Found: found}
			c.note(fmt.Sprintf("%s for %s committed at entry %d; returning success to client.", cmd.Operation, cmd.Key, index))
			mu.Unlock()
			return result, 200, nil
		}
		lost := n.Role != "leader" || !valid
		mu.Unlock()
		if lost {
			return fail(409, "Leadership changed or the barrier was replaced; no committed response confirmed. Retry the same mutation ID or repeat the read.")
		}
		select {
		case <-ctx.Done():
			return fail(499, "Request canceled while awaiting commitment; outcome unknown. Retry the same mutation ID.")
		case <-deadline.C:
			if cmd.Operation == "read" {
				return fail(504, "Read could not confirm a majority before timeout. No value returned. Heal the network or route to the current leader, then retry.")
			}
			return fail(504, "Commitment timed out; outcome unknown. The operation may commit later. Retry the same request ID.")
		case <-poll.C:
		}
	}
}

// This path deliberately releases the supervisor lock while awaiting consensus:
// observation and process-stop requests must remain available during a write.
func (l *liveController) serveKV(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", 405)
		return
	}
	var cmd struct {
		Operation, RequestID, Key, Value string
		Target                           *int
	}
	if r.Header.Get("Content-Type") != "application/json" || json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&cmd) != nil || (cmd.Operation != "write" && cmd.Operation != "read" && cmd.Operation != "delete") {
		http.Error(w, "JSON operation write/read/delete required", 400)
		return
	}
	if cmd.Target != nil && (*cmd.Target < 0 || *cmd.Target > 2) {
		http.Error(w, "Target must be server ID 0, 1, or 2.", 400)
		return
	}
	l.mu.Lock()
	states := l.states()
	leader := -1
	term := -1
	// This is a routing hint from observations, not a quorum decision. Even a
	// selected leader must still pass its own commit/barrier checks.
	for id, s := range states {
		if s != nil && s.Node.Role == "leader" && s.Node.Term > term {
			leader = id
			term = s.Node.Term
		}
	}
	// Explicit targeting lets students probe an isolated former leader. It does
	// not bypass that node's consensus checks or turn a follower into a leader.
	if cmd.Target != nil {
		leader = *cmd.Target
		if states[leader] == nil {
			leader = -1
		}
	}
	l.mu.Unlock()
	if leader < 0 {
		message := "No leader available. Retry after election using the same write ID."
		if cmd.Operation == "read" {
			message = "No read target available. Wait for an election or choose another server, then retry."
		}
		http.Error(w, message, 409)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3500*time.Millisecond)
	defer cancel()
	rpc, err := l.rpc.clientAddress(l.address(leader))
	if err != nil {
		http.Error(w, err.Error(), 503)
		return
	}
	result, err := rpc.Execute(ctx, &pb.ClientCommand{ShardId: int32(l.shardID), Operation: wireOperations[cmd.Operation], RequestId: cmd.RequestID, Key: cmd.Key, Value: cmd.Value})
	if err != nil {
		http.Error(w, err.Error(), rpcHTTPCode(err))
		return
	}
	decoded := receiptFromWire(result)
	if decoded.ShardID != l.shardID || decoded.ServerID != leader || !decoded.Committed {
		http.Error(w, "Invalid replica receipt; outcome unknown. Retry the same mutation ID.", 503)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(decoded)
}
