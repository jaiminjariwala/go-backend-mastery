package main

import "fmt"

// PeerObservation is bounded display evidence, never input to Raft decisions.
// A page means a follower stored a matching entry. An ack means the leader
// received that follower's successful reply. Neither alone proves commitment.
type PeerObservation struct {
	ID, RequestID, Key    string
	From, To, Index, Term int
	Kind                  string
	Delete                bool
}

type peerObservations struct {
	boot   string
	serial int
	events []PeerObservation
}

// Called with the node lock held, after receive has persisted its transition.
// Ignore idle heartbeats, failed replies, reads, and unmatched proposals.
func (p *peerObservations) record(n *Node, m Message) {
	if m.Quiet || (m.Kind != "append" && m.Kind != "ack") || (m.Kind == "ack" && !m.OK) {
		return
	}
	index := m.Index
	if m.Kind == "append" {
		index = m.Snapshot.Index + len(m.Log)
		if len(m.Log) == 0 || !n.matches(index, m.Log[len(m.Log)-1]) {
			return
		}
	}
	entry, ok := n.entryAt(index)
	if !ok || entry.Key == "" || entry.ReadKey != "" || m.Term != n.Term {
		return
	}
	p.serial++
	p.events = append(p.events, PeerObservation{
		ID: fmt.Sprintf("%s:%d", p.boot, p.serial), RequestID: entry.RequestID,
		Key: entry.Key, From: m.From, To: m.To, Index: index, Term: entry.Term,
		Kind: m.Kind, Delete: entry.Delete,
	})
	if len(p.events) > 64 {
		p.events = p.events[len(p.events)-64:]
	}
}
