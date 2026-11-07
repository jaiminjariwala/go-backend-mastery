package main

import "testing"

// advance drives logical time without sleeping, so protocol tests are reproducible.
func advance(c *Cluster, n int) {
	for i := 0; i < n; i++ {
		c.Step()
	}
}
func leader(c *Cluster) *Node {
	for _, n := range c.Nodes {
		if n.Up && n.Role == "leader" {
			return n
		}
	}
	return nil
}
func TestElectionReplicationFailover(t *testing.T) {
	c := NewCluster()
	advance(c, 20)
	l := leader(c)
	if l == nil {
		t.Fatal("no leader")
	}
	if err := c.Write("x", "10"); err != nil {
		t.Fatal(err)
	}
	advance(c, 10)
	for _, n := range c.Nodes {
		if n.KV["x"] != "10" {
			t.Fatal("not replicated")
		}
	}
	c.Toggle(l.ID)
	advance(c, 35)
	next := leader(c)
	if next == nil || next.ID == l.ID || next.KV["x"] != "10" {
		t.Fatal("failed failover")
	}
	c.Toggle(l.ID)
	advance(c, 15)
	for _, n := range c.Nodes {
		if n.KV["x"] != "10" {
			t.Fatal("lost committed value")
		}
	}
}
func TestNoMajorityNoCommit(t *testing.T) {
	c := NewCluster()
	advance(c, 20)
	l := leader(c)
	for _, n := range c.Nodes {
		if n.ID != l.ID {
			c.Toggle(n.ID)
		}
	}
	before := l.Commit
	c.Write("x", "10")
	advance(c, 15)
	if l.Commit != before || l.KV["x"] != "" {
		t.Fatal("committed without majority")
	}
}
func TestDeterministic(t *testing.T) {
	a, b := NewCluster(), NewCluster()
	advance(a, 50)
	advance(b, 50)
	if a.Serial != b.Serial || len(a.Events) != len(b.Events) {
		t.Fatal("not deterministic")
	}
}

func TestProgressRequiresAcknowledgments(t *testing.T) {
	c := NewCluster()
	advance(c, 20)
	if c.Progress() != nil {
		t.Fatal("progress before a proposal")
	}
	if err := c.Write("x", "40"); err != nil {
		t.Fatal(err)
	}
	w := c.Progress()
	if w.Committed || len(w.Confirmed) != 1 || w.Phase != "proposed" {
		t.Fatalf("premature progress: %+v", w)
	}
	// Followers receive the write, but the leader has not received their replies.
	advance(c, 2)
	w = c.Progress()
	if w.Committed || len(w.Confirmed) != 1 {
		t.Fatalf("counted delivery as acknowledgment: %+v", w)
	}
	advance(c, 2)
	w = c.Progress()
	if !w.Committed || len(w.Confirmed) < 2 || len(w.Applied) < 1 {
		t.Fatalf("missing actual commit: %+v", w)
	}
	advance(c, 4)
	if len(c.Progress().Applied) != 3 {
		t.Fatal("followers did not apply")
	}
}

func TestProgressWithNoMajority(t *testing.T) {
	c := NewCluster()
	advance(c, 20)
	l := leader(c)
	for _, n := range c.Nodes {
		if n.ID != l.ID {
			c.Toggle(n.ID)
		}
	}
	if err := c.Write("x", "40"); err != nil {
		t.Fatal(err)
	}
	advance(c, 10)
	w := c.Progress()
	if w.Committed || len(w.Confirmed) != 1 || len(w.Applied) != 0 {
		t.Fatalf("invented write success: %+v", w)
	}
	dropped := false
	for _, d := range c.Deliveries {
		if d.Status == "dropped" && !c.Nodes[d.Message.To].Up {
			dropped = true
		}
	}
	if !dropped {
		t.Fatal("offline deliveries not recorded")
	}
}

func TestDeliveryToOfflineNodeDoesNotReply(t *testing.T) {
	c := NewCluster()
	c.Toggle(1)
	c.receive(Message{ID: 42, From: 0, To: 1, Term: 1, Kind: "append", Log: []Entry{{Term: 1, Key: "x", Value: "40"}}})
	if len(c.Messages) != 0 || len(c.Nodes[1].Log) != 0 {
		t.Fatal("offline node processed request")
	}
	if len(c.Deliveries) != 1 || c.Deliveries[0].Status != "dropped" || c.Deliveries[0].Message.ID != 42 {
		t.Fatal("missing drop observation")
	}
}

func TestDeliveryObservationsBounded(t *testing.T) {
	c := NewCluster()
	advance(c, 1000)
	if len(c.Deliveries) > 160 {
		t.Fatal("unbounded transport history")
	}
	for i := 1; i < len(c.Deliveries); i++ {
		if c.Deliveries[i].Sequence <= c.Deliveries[i-1].Sequence {
			t.Fatal("non-monotonic observation IDs")
		}
	}
}

func TestHeartbeatVisibilityMetadataDoesNotHideNewWrites(t *testing.T) {
	c := NewCluster()
	advance(c, 20)
	quiet := false
	for _, m := range c.Messages {
		if m.Kind == "append" && m.Quiet {
			quiet = true
		}
	}
	if !quiet {
		t.Fatal("expected quiet idle heartbeat after replicas acknowledged")
	}
	before := c.Serial
	if err := c.Write("x", "40"); err != nil {
		t.Fatal(err)
	}
	for _, m := range c.Messages {
		if m.ID > before && m.Kind == "append" && m.Quiet {
			t.Fatal("new write was marked as idle traffic")
		}
	}
	advance(c, 2)
	for _, m := range c.Messages {
		if m.Kind == "ack" && m.Index == 2 && m.Quiet {
			t.Fatal("write reply was hidden as idle traffic")
		}
	}
}

// A stale replica with the shortest timeout must not prevent fresher replicas
// from starting an election by repeatedly sending vote requests they reject.
func TestRejectedStaleVotesDoNotPostponeElection(t *testing.T) {
	c := NewCluster()
	for _, n := range c.Nodes {
		n.Term = 1
		if n.ID != 0 {
			n.Log = []Entry{{Term: 1, Key: "saved", Value: "yes"}}
			n.Commit = 1
			c.apply(n)
		}
	}
	advance(c, 100)
	elected := leader(c)
	if elected == nil || elected.KV["saved"] != "yes" {
		t.Fatal("stale candidate prevented an up-to-date leader election")
	}
}
