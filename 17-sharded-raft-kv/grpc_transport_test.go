package main

import (
	"context"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	pb "github.com/jaiminjariwala/go-backend-mastery/17-sharded-raft-kv/api/raftv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

// Keep every field, including compacted retry identities and delete markers,
// across the typed protobuf boundary. JSON remains only an external adapter.
func TestProtobufPeerRoundTrip(t *testing.T) {
	original := Message{ShardID: 1, ID: 12, From: 1, To: 2, Term: 7, Sent: 3, Due: 4, Kind: "snapshot", Commit: 11, OK: true, Quiet: true, Index: 12, Log: []Entry{{Term: 7, Key: "user:1", RequestID: "remove", Delete: true}}, Snapshot: Snapshot{Index: 10, Term: 6, KV: map[string]string{"user:1": "Maya"}, Requests: map[string]RequestResult{"old": {Key: "user:2", Value: "", Index: 8, Term: 5, Delete: true}}}}
	wireBytes, err := proto.Marshal(messageToWire(original))
	if err != nil {
		t.Fatal(err)
	}
	var decoded pb.PeerMessage
	if err := proto.Unmarshal(wireBytes, &decoded); err != nil {
		t.Fatal(err)
	}
	restored, err := messageFromWire(&decoded)
	if err != nil || !reflect.DeepEqual(restored, original) {
		t.Fatal("protobuf changed protocol state", restored, err)
	}
	if _, err := messageFromWire(&pb.PeerMessage{}); err == nil {
		t.Fatal("accepted unspecified message kind")
	}
}

// A real generated client exercises service status mapping and cancellation.
// Cancellation must stop waiting without claiming that a proposal was rolled back.
func TestGRPCClientValidationAndDeadline(t *testing.T) {
	cluster := NewCluster()
	advance(cluster, 24)
	node := leader(cluster)
	if node == nil {
		t.Fatal("no leader")
	}
	for _, peer := range cluster.Nodes {
		if peer.ID != node.ID {
			cluster.Toggle(peer.ID)
		}
	}
	var mu sync.Mutex
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	pb.RegisterReplicaServer(server, &grpcReplica{shardID: 0, deliver: func(Message) (int, error) { return 204, nil }, execute: func(ctx context.Context, cmd command) (receipt, int, error) {
		return executeCommand(ctx, cluster, node, &mu, cmd, 0, 1)
	}})
	go server.Serve(listener)
	t.Cleanup(func() { server.Stop(); listener.Close() })
	conn, err := grpc.NewClient("passthrough:///test", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	rpc := pb.NewReplicaClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err = rpc.Execute(ctx, &pb.ClientCommand{ShardId: 1, Operation: pb.Operation_WRITE, Key: "x", Value: "40", RequestId: "wrong"}); status.Code(err) != codes.OutOfRange {
		t.Fatal("wrong shard accepted", err)
	}
	if _, err = rpc.Execute(ctx, &pb.ClientCommand{Operation: pb.Operation_OPERATION_UNSPECIFIED, Key: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Fatal("invalid operation accepted", err)
	}
	// Warm the connection first so this deadline covers command execution.
	short, stop := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer stop()
	_, err = rpc.Execute(short, &pb.ClientCommand{Operation: pb.Operation_WRITE, Key: "x", Value: "40", RequestId: "uncertain"})
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatal("deadline was not propagated", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if node.KV["x"] != "" || node.lastIndex() <= node.Commit {
		t.Fatal("uncommitted RPC changed rows or lost its proposal")
	}
}
