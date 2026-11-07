package main

import "testing"

func TestPeerObservationRequiresMatchingStoredMutation(t *testing.T) {
	entry := Entry{Term: 2, Key: "user:102", Delete: true, RequestID: "delete-102"}
	n := &Node{ID: 1, Term: 2}
	trace := peerObservations{boot: "replica-boot"}
	appendMessage := Message{From: 0, To: 1, Term: 2, Kind: "append", Log: []Entry{entry}}
	trace.record(n, appendMessage)
	if len(trace.events) != 0 {
		t.Fatal("unstored instruction appeared in the replay")
	}
	n.Log = []Entry{entry}
	trace.record(n, appendMessage)
	if len(trace.events) != 1 || !trace.events[0].Delete || trace.events[0].RequestID != entry.RequestID {
		t.Fatal("stored deletion lost its identity")
	}
	ack := Message{From: 1, To: 0, Term: 2, Kind: "ack", Index: 1}
	trace.record(n, ack)
	if len(trace.events) != 1 {
		t.Fatal("rejected reply was shown as an acknowledgement")
	}
	ack.OK = true
	trace.record(n, ack)
	if len(trace.events) != 2 || trace.events[1].Kind != "ack" {
		t.Fatal("successful reply missing")
	}
	ack.Quiet = true
	trace.record(n, ack)
	if len(trace.events) != 2 {
		t.Fatal("heartbeat entered the mutation replay")
	}
	ack.Quiet = false
	for i := 0; i < 100; i++ {
		trace.record(n, ack)
	}
	if len(trace.events) != 64 || trace.events[0].ID == trace.events[1].ID {
		t.Fatal("replay history is not bounded or uniquely identified")
	}
}
