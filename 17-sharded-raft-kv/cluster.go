package main

import "fmt"

// Each transition engine belongs to one fixed group. A quorum means more than
// half the group: two replicas out of three, not merely two reachable processes.
const (
	replicaCount = 3
	quorumSize   = replicaCount/2 + 1
)

// Entry is one ordered instruction. A normal write changes a key. A read barrier
// or leader no-op occupies a log position without changing the saved values.
type Entry struct {
	Delete     bool `json:",omitempty"` // A replicated row removal; absence of this flag means a normal write.
	Term       int
	Key, Value string
	RequestID  string `json:",omitempty"`
	ReadKey    string `json:",omitempty"`
}

// Node holds one replica’s protocol state. KV contains only applied, committed
// commands; Log may also contain proposals that can still be replaced.
type Node struct {
	ID       int // Internal IDs are 0-2; the interface displays Server 1-3.
	Role     string
	Term     int // Election round, not a timestamp or time in milliseconds.
	Vote     int // Candidate chosen in this term; -1 means no vote yet.
	Deadline int // Local logical tick when another election may begin.
	Commit   int // Highest absolute log index known to be committed.
	Up       bool
	Log      []Entry           // Only entries AFTER Snapshot.Index remain here.
	KV       map[string]string // Applied database values, never pending proposals.
	Votes    map[int]bool      // A candidate's received votes for the current term.
	Match    map[int]int       // Leader's acknowledged replication index per peer.
	Snapshot Snapshot
	Requests map[string]RequestResult // Applied write IDs used to deduplicate retries.
}

// Message is the small learning protocol's shared RPC envelope. For a vote
// request, Index/Commit carry the candidate's last log index/term. For append
// RPCs, Commit is the commit index; an acknowledgment uses Index as its receipt.
type Message struct {
	ShardID                       int // Transport boundary: a peer must belong to this same Raft group.
	ID, From, To, Term, Sent, Due int
	Kind                          string
	Log                           []Entry
	Commit                        int
	OK                            bool
	Quiet                         bool // idle heartbeat and its reply; presentation metadata only
	Index                         int
	Snapshot                      Snapshot
}

// Event is a bounded teaching annotation, separate from durable protocol state.
type Event struct {
	At   int
	Text string
}

// Transport observations describe delivery, not protocol acceptance or commitment.
type Delivery struct {
	Sequence int
	At       int
	Status   string
	Message  Message
}

// Proposal identifies the latest simulated operation so the UI can follow its
// replication without treating a matching value as proof of commitment.
type Proposal struct {
	Committed         bool
	Leader, Index, At int
	Entry             Entry
}

// WriteProgress separates recorded copies from applied copies of one log entry.
type WriteProgress struct {
	Delete             bool `json:",omitempty"`
	Key, Value, Phase  string
	RequestID          string
	Index, Term        int
	Confirmed, Applied []int
	Committed          bool
}

// Cluster contains the shared transition engine. The simulator owns all nodes;
// a live child drives only its local node and sends peer messages over HTTP.
type Cluster struct {
	Durable        bool
	store          *walStore
	storageErr     error
	Tick, Serial   int
	Nodes          []*Node
	Messages       []Message
	Events         []Event
	DeliverySerial int
	Deliveries     []Delivery
	Proposal       *Proposal
}

// NewCluster creates the deterministic test/simulation fixture. Live processes
// reuse these transitions for ONE local node; remote slots are peer identities.
func NewCluster() *Cluster {
	c := &Cluster{}
	for i := 0; i < replicaCount; i++ {
		c.Nodes = append(c.Nodes, &Node{
			ID:       i,
			Role:     "follower",
			Vote:     -1,
			Deadline: 8 + i*3,
			Up:       true,
			KV:       map[string]string{},
			Votes:    map[int]bool{},
			Match:    map[int]int{},
		})
	}
	c.note("Three nodes started. Waiting for an election timeout.")
	return c
}

// note keeps a bounded explanation history for the console; it is not the WAL.
func (c *Cluster) note(s string) {
	c.Events = append(c.Events, Event{c.Tick, s})
	if len(c.Events) > 80 {
		c.Events = c.Events[1:]
	}
}

// send persists the sender first. In particular, a vote or replication reply
// must not promise state that would disappear if the process crashed now.
func (c *Cluster) send(m Message) {
	if !c.persist(c.Nodes[m.From]) {
		return
	}
	c.Serial++
	m.ID = c.Serial
	m.Sent = c.Tick
	m.Due = c.Tick + 2
	c.Messages = append(c.Messages, m)
}

// recordDelivery retains bounded transport evidence for animation. Delivery
// alone does not imply that the receiver accepted or committed a command.
func (c *Cluster) recordDelivery(m Message, status string) {
	c.DeliverySerial++
	// Payloads already live in the node logs. Retain transport metadata only.
	m.Log = nil
	m.Snapshot = Snapshot{} // Telemetry must not retain another copy of database state.
	c.Deliveries = append(c.Deliveries, Delivery{c.DeliverySerial, c.Tick, status, m})
	if len(c.Deliveries) > 160 {
		c.Deliveries = c.Deliveries[len(c.Deliveries)-160:]
	}
}

// Progress reports evidence for the latest proposal. Node availability alone
// never counts as a replication acknowledgment. Applied means the node's state
// machine has processed this log index, not necessarily that its current KV value
// still equals an older write after later updates.
func (c *Cluster) Progress() *WriteProgress {
	p := c.Proposal
	if p == nil {
		return nil
	}
	w := &WriteProgress{Delete: p.Entry.Delete, Key: p.Entry.Key, Value: p.Entry.Value, RequestID: p.Entry.RequestID, Index: p.Index, Term: p.Entry.Term, Phase: "proposed", Confirmed: []int{}, Applied: []int{}}
	confirmed := map[int]bool{}
	for _, n := range c.Nodes {
		matches := n.matches(p.Index, p.Entry)
		if !matches {
			continue
		}
		if n.Commit >= p.Index {
			w.Committed = true
			w.Applied = append(w.Applied, n.ID)
			confirmed[n.ID] = true
		}
		if n.Role == "leader" || n.ID == p.Leader {
			// Match retains actual acknowledgments; count the origin's local append too.
			confirmed[n.ID] = true
			for id, index := range n.Match {
				if index >= p.Index {
					confirmed[id] = true
				}
			}
		}
	}
	for id := 0; id < 3; id++ {
		if confirmed[id] {
			w.Confirmed = append(w.Confirmed, id)
		}
	}
	if c.Tick > p.At {
		w.Phase = "replicating"
	}
	if w.Committed {
		p.Committed = true
	}
	w.Committed = p.Committed
	if w.Committed {
		w.Phase = "applied"
	}
	// If every log has lost the entry, it is no longer a pending proposal.
	found := false
	for _, n := range c.Nodes {
		if !n.Up {
			found = true
		} // its durable log is not loaded into memory
		if n.matches(p.Index, p.Entry) {
			found = true
		}
	}
	if !found {
		w.Phase = "superseded"
	}
	return w
}

// A newer term invalidates a node's old vote and leadership claim. Hearing from
// a valid leader postpones the next election only in the append handler.
func (c *Cluster) follower(n *Node, t int) {
	if t > n.Term {
		n.Term = t
		n.Vote = -1
	}
	n.Role = "follower"
	// Merely learning a newer term must not postpone an election. Otherwise a
	// stale candidate with the shortest timer can repeatedly reset fresher peers
	// through rejected vote requests, preventing anyone from becoming leader.
}

// broadcast retries replication even for unreachable peers. A leader cannot
// know with certainty whether a peer is stopped, isolated, or simply slow.
func (c *Cluster) broadcast(n *Node) {
	for _, p := range c.Nodes {
		if p.ID != n.ID {
			kind := "append"
			if n.Match[p.ID] < n.Snapshot.Index {
				kind = "snapshot"
			}
			c.send(Message{
				From: n.ID, To: p.ID, Term: n.Term,
				Kind: kind, Quiet: n.Match[p.ID] >= n.lastIndex(),
				Log:      append([]Entry{}, n.Log...),
				Snapshot: n.Snapshot, Commit: n.Commit,
			})
		}
	}
}

// Expiring a timeout starts candidacy, not leadership. The candidate increments
// its term, votes for itself, and still needs votes from a majority to win.
func (c *Cluster) election(n *Node) {
	n.Term++
	n.Role = "candidate"
	n.Vote = n.ID
	n.Votes = map[int]bool{n.ID: true}
	n.Deadline = c.Tick + 9 + n.ID*3
	c.note(fmt.Sprintf("Node %d starts election for term %d.", n.ID+1, n.Term))
	for _, p := range c.Nodes {
		if p.ID != n.ID {
			c.send(Message{From: n.ID, To: p.ID, Term: n.Term, Kind: "vote", Index: n.lastIndex(), Commit: n.lastTerm()})
		}
	}
}

// apply rebuilds the database from its checkpoint plus committed suffix. This
// simple replay is easier to inspect than an incremental state machine, but is
// less efficient. Entries after Commit are deliberately excluded.
func (c *Cluster) apply(n *Node) {
	n.KV = map[string]string{}
	n.Requests = map[string]RequestResult{}
	for k, v := range n.Snapshot.KV {
		n.KV[k] = v
	}
	for k, v := range n.Snapshot.Requests {
		n.Requests[k] = v
	}
	for i, e := range n.Log[:n.Commit-n.Snapshot.Index] {
		if e.Key != "" && e.ReadKey == "" {
			if _, seen := n.Requests[e.RequestID]; e.RequestID != "" && seen {
				continue
			}
			// Deletion is an ordered command, so offline replicas learn it on catch-up.
			if e.Delete {
				delete(n.KV, e.Key)
			} else {
				n.KV[e.Key] = e.Value
			}
			if e.RequestID != "" {
				n.Requests[e.RequestID] = RequestResult{
					Key: e.Key, Value: e.Value, Delete: e.Delete,
					Index: n.Snapshot.Index + i + 1, Term: e.Term,
				}
			}
		}
	}
}

// receive handles one protocol event while the runtime owns the node mutex.
// Every return path preserves term/vote/log changes before later events run.
func (c *Cluster) receive(m Message) {
	if c.storageErr != nil {
		return
	}
	n := c.Nodes[m.To]
	if !n.Up {
		c.recordDelivery(m, "dropped")
		return
	}
	c.recordDelivery(m, "delivered")
	defer c.persist(n)
	if m.Term > n.Term {
		c.follower(n, m.Term)
	}
	switch m.Kind {
	case "vote":
		// Log freshness compares last term first, then index. Wall-clock time
		// does not make a candidate's log more up to date.
		fresh := m.Commit > n.lastTerm() || (m.Commit == n.lastTerm() && m.Index >= n.lastIndex())
		ok := m.Term == n.Term && (n.Vote < 0 || n.Vote == m.From) && fresh
		if ok {
			n.Vote = m.From
			n.Deadline = c.Tick + 9 + n.ID*3
		}
		c.send(Message{From: n.ID, To: m.From, Term: n.Term, Kind: "ballot", OK: ok})
	case "ballot":
		if n.Role != "candidate" || m.Term != n.Term || !m.OK {
			return
		}
		n.Votes[m.From] = true
		if len(n.Votes) >= quorumSize {
			n.Role = "leader"
			n.Match = map[int]int{n.ID: n.lastIndex()}
			c.note(fmt.Sprintf("Node %d wins a majority in term %d.", n.ID+1, n.Term))
			n.Log = append(n.Log, Entry{Term: n.Term})
			// Committing this current-term no-op also establishes which older
			// entries precede it in the newly elected leader's committed history.
			n.Match[n.ID] = n.lastIndex()
			c.broadcast(n)
		}
	case "append", "snapshot":
		if m.Term != n.Term {
			return
		}
		c.follower(n, m.Term)
		n.Deadline = c.Tick + 9 + n.ID*3
		previousSnapshot := n.Snapshot.Index
		if !c.acceptAppend(n, m) {
			return
		}
		c.send(Message{From: n.ID, To: m.From, Term: n.Term, Kind: "ack", Quiet: m.Quiet, OK: true, Index: m.Snapshot.Index + len(m.Log)})
		if c.storageErr == nil && n.Snapshot.Index > previousSnapshot {
			c.note(fmt.Sprintf("Server %d installed and synced snapshot through entry %d; values and retry receipts restored.", n.ID+1, n.Snapshot.Index))
		}
	case "ack":
		if n.Role != "leader" || m.Term != n.Term {
			return
		}
		n.Match[m.From] = max(n.Match[m.From], m.Index)
		c.advanceCommit(n)
	}
}

// advanceCommit considers real replication acknowledgments. Being online is
// not a vote for a write. Raft directly commits only current-term entries;
// committing one also commits the older prefix before it.
func (c *Cluster) advanceCommit(n *Node) {
	for i := n.lastIndex(); i > n.Commit; i-- {
		if term, _ := n.termAt(i); term != n.Term {
			continue
		}
		votes := 1
		for id, v := range n.Match {
			if id != n.ID && v >= i {
				votes++
			}
		}
		if votes >= quorumSize {
			n.Commit = i
			c.apply(n)
			c.note(fmt.Sprintf("Node %d commits through entry %d with a majority.", n.ID+1, i))
			c.broadcast(n)
			break
		}
	}
}

// Step advances simulated time by one tick. Live nodes use independent timers
// in network.go instead; pausing the interface cannot pause those processes.
func (c *Cluster) Step() {
	if c.storageErr != nil {
		return
	}
	c.Tick++
	pending := c.Messages
	c.Messages = nil
	for _, m := range pending {
		if m.Due <= c.Tick {
			c.receive(m)
		} else {
			c.Messages = append(c.Messages, m)
		}
	}
	for _, n := range c.Nodes {
		if !n.Up {
			continue
		}
		if n.Role == "leader" {
			if c.Tick%4 == 0 {
				c.broadcast(n)
			}
		} else if c.Tick >= n.Deadline {
			c.election(n)
		}
	}
}

// Write proposes a simulation/test write. It does not return a client success
// receipt. The live client handler in client.go waits for commitment instead.
func (c *Cluster) Write(key, value string) error {
	if c.storageErr != nil {
		return c.storageErr
	}
	for _, n := range c.Nodes {
		if n.Up && n.Role == "leader" {
			n.Log = append(n.Log, Entry{Term: n.Term, Key: key, Value: value})
			c.Proposal = &Proposal{Leader: n.ID, Index: n.lastIndex(), At: c.Tick, Entry: n.Log[len(n.Log)-1]}
			n.Match[n.ID] = n.lastIndex()
			c.note(fmt.Sprintf("Write %s = %s proposed at entry %d; not committed yet.", key, value, n.lastIndex()))
			c.broadcast(n)
			return c.storageErr
		}
	}
	return fmt.Errorf("no leader yet: advance the clock to elect one")
}

// Toggle simulates stop/restart. Durable fixtures discard volatile memory and
// reload the WAL; the live controller actually kills/restarts a child process.
func (c *Cluster) Toggle(id int) {
	if c.storageErr != nil {
		return
	}
	n := c.Nodes[id]
	if c.store != nil {
		if n.Up {
			if !c.persist(n) {
				return
			}
			c.Nodes[id] = &Node{ID: id, Role: "offline", Up: false, Vote: -1, KV: map[string]string{}, Match: map[int]int{}, Votes: map[int]bool{}}
			c.note(fmt.Sprintf("Server %d stopped; volatile memory discarded.", id+1))
			return
		}
		d, err := c.store.load(id)
		if err != nil {
			c.storageErr = err
			return
		}
		n.Term = d.Term
		n.Vote = d.Vote
		n.Log = d.Log
		n.Commit = d.Commit
		n.Snapshot = d.Snapshot
		n.Up = true
		n.Match = map[int]int{}
		n.Votes = map[int]bool{}
		c.follower(n, n.Term)
		c.apply(n)
		c.note(fmt.Sprintf("Server %d recovered from disk as a follower.", id+1))
		return
	}
	n.Up = !n.Up
	if !n.Up {
		n.Role = "offline"
		c.note(fmt.Sprintf("Node %d stopped. In-memory log retained for this simulated restart.", id+1))
	} else {
		c.follower(n, n.Term)
		c.note(fmt.Sprintf("Node %d restarted as a follower.", id+1))
	}
}
