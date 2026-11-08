package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	pb "github.com/jaiminjariwala/go-backend-mastery/17-sharded-raft-kv/api/raftv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// rpcPool reuses HTTP/2 connections. gRPC reconnects the same channel after a
// replica restarts; a caller's deadline bounds each individual attempt.
// Destinations are validated startup addresses, never browser-supplied hosts.
type rpcPool struct {
	mu          sync.Mutex
	connections map[string]*grpc.ClientConn
	closed      bool
}

func (p *rpcPool) client(port int) (pb.ReplicaClient, error) {
	return p.clientAddress(fmt.Sprintf("127.0.0.1:%d", port))
}

// Addresses come from startup configuration, never from browser request data.
// Local mode supplies loopback ports; Kubernetes supplies stable Pod DNS names.
func (p *rpcPool) clientAddress(address string) (pb.ReplicaClient, error) {
	if err := validatePeerAddress(address); err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, fmt.Errorf("RPC pool closed")
	}
	if p.connections == nil {
		p.connections = map[string]*grpc.ClientConn{}
	}
	conn := p.connections[address]
	if conn == nil {
		var err error
		// This learning deployment uses cleartext within its private network.
		// Remote/public deployments require authenticated TLS, not just DNS.
		conn, err = grpc.NewClient("dns:///"+address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxRecord), grpc.MaxCallSendMsgSize(maxRecord)))
		if err != nil {
			return nil, err
		}
		p.connections[address] = conn
	}
	return pb.NewReplicaClient(conn), nil
}
func (p *rpcPool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	for _, conn := range p.connections {
		conn.Close()
	}
}

var wireKinds = map[string]pb.MessageKind{"vote": pb.MessageKind_VOTE, "ballot": pb.MessageKind_BALLOT, "append": pb.MessageKind_APPEND, "snapshot": pb.MessageKind_SNAPSHOT, "ack": pb.MessageKind_ACK}
var wireOperations = map[string]pb.Operation{"write": pb.Operation_WRITE, "read": pb.Operation_READ, "delete": pb.Operation_DELETE}

func operationName(operation pb.Operation) string {
	for name, value := range wireOperations {
		if value == operation {
			return name
		}
	}
	return ""
}

// Conversion lives at the transport boundary. The Raft transition engine and
// persisted WAL keep their existing types and absolute indexes.
func messageToWire(m Message) *pb.PeerMessage {
	out := &pb.PeerMessage{ShardId: int32(m.ShardID), Id: int64(m.ID), From: int32(m.From), To: int32(m.To), Term: int64(m.Term), Sent: int64(m.Sent), Due: int64(m.Due), Kind: wireKinds[m.Kind], Commit: int64(m.Commit), Ok: m.OK, Quiet: m.Quiet, Index: int64(m.Index), Snapshot: &pb.Checkpoint{Index: int64(m.Snapshot.Index), Term: int64(m.Snapshot.Term), Values: m.Snapshot.KV, Requests: map[string]*pb.AppliedRequest{}}}
	for _, entry := range m.Log {
		out.Log = append(out.Log, &pb.LogEntry{Term: int64(entry.Term), Key: entry.Key, Value: entry.Value, RequestId: entry.RequestID, ReadKey: entry.ReadKey, Delete: entry.Delete})
	}
	for id, result := range m.Snapshot.Requests {
		out.Snapshot.Requests[id] = &pb.AppliedRequest{Key: result.Key, Value: result.Value, Index: int64(result.Index), Term: int64(result.Term), Delete: result.Delete}
	}
	return out
}
func messageFromWire(in *pb.PeerMessage) (Message, error) {
	if in == nil {
		return Message{}, fmt.Errorf("missing peer message")
	}
	kind := ""
	for name, value := range wireKinds {
		if value == in.Kind {
			kind = name
		}
	}
	if kind == "" {
		return Message{}, fmt.Errorf("invalid RPC kind")
	}
	m := Message{ShardID: int(in.ShardId), ID: int(in.Id), From: int(in.From), To: int(in.To), Term: int(in.Term), Sent: int(in.Sent), Due: int(in.Due), Kind: kind, Commit: int(in.Commit), OK: in.Ok, Quiet: in.Quiet, Index: int(in.Index)}
	for _, entry := range in.Log {
		if entry == nil {
			return Message{}, fmt.Errorf("missing log entry")
		}
		m.Log = append(m.Log, Entry{Term: int(entry.Term), Key: entry.Key, Value: entry.Value, RequestID: entry.RequestId, ReadKey: entry.ReadKey, Delete: entry.Delete})
	}
	if checkpoint := in.Snapshot; checkpoint != nil {
		m.Snapshot = Snapshot{Index: int(checkpoint.Index), Term: int(checkpoint.Term), KV: checkpoint.Values, Requests: map[string]RequestResult{}}
		for id, result := range checkpoint.Requests {
			if result == nil {
				return Message{}, fmt.Errorf("missing retry receipt")
			}
			m.Snapshot.Requests[id] = RequestResult{Key: result.Key, Value: result.Value, Index: int(result.Index), Term: int(result.Term), Delete: result.Delete}
		}
	}
	return m, nil
}
func receiptToWire(result receipt) *pb.ClientReceipt {
	return &pb.ClientReceipt{ShardId: int32(result.ShardID), ServerId: int32(result.ServerID), Operation: wireOperations[result.Operation], RequestId: result.RequestID, Key: result.Key, Value: result.Value, Index: int64(result.Index), Term: int64(result.Term), Committed: result.Committed, Replayed: result.Replayed, Found: result.Found}
}
func receiptFromWire(result *pb.ClientReceipt) receipt {
	return receipt{ShardID: int(result.ShardId), ServerID: int(result.ServerId), Operation: operationName(result.Operation), RequestID: result.RequestId, Key: result.Key, Value: result.Value, Index: int(result.Index), Term: int(result.Term), Committed: result.Committed, Replayed: result.Replayed, Found: result.Found}
}

// grpcReplica is deliberately thin: both transports invoke the same validation,
// persistence, partition policy, and commit-aware client execution.
type grpcReplica struct {
	pb.UnimplementedReplicaServer
	shardID  int
	deliver  func(Message) (int, error)
	execute  func(context.Context, command) (receipt, int, error)
	onClient func()
}

func (s *grpcReplica) Deliver(ctx context.Context, in *pb.PeerMessage) (*pb.DeliveryReceipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}
	message, err := messageFromWire(in)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	code, err := s.deliver(message)
	if err != nil {
		return nil, rpcError(code, err)
	}
	return &pb.DeliveryReceipt{}, nil
}
func (s *grpcReplica) Execute(ctx context.Context, in *pb.ClientCommand) (*pb.ClientReceipt, error) {
	if in == nil || int(in.ShardId) != s.shardID {
		return nil, status.Error(codes.OutOfRange, "Request belongs to another shard.")
	}
	if s.onClient != nil {
		s.onClient()
	}
	result, code, err := s.execute(ctx, command{Operation: operationName(in.Operation), RequestID: in.RequestId, Key: in.Key, Value: in.Value})
	if err != nil {
		return nil, rpcError(code, err)
	}
	return receiptToWire(result), nil
}
func rpcError(httpCode int, err error) error {
	code := codes.Unavailable
	switch httpCode {
	case http.StatusBadRequest:
		code = codes.InvalidArgument
	case http.StatusConflict:
		code = codes.FailedPrecondition
	case http.StatusMisdirectedRequest:
		code = codes.OutOfRange
	case http.StatusGatewayTimeout:
		code = codes.DeadlineExceeded
	case 499:
		code = codes.Canceled
	}
	return status.Error(code, err.Error())
}
func rpcHTTPCode(err error) int {
	switch status.Code(err) {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.FailedPrecondition:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusMisdirectedRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	default:
		return http.StatusServiceUnavailable
	}
}
