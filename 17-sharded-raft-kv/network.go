package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	pb "github.com/jaiminjariwala/go-backend-mastery/17-sharded-raft-kv/api/raftv1"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

// Runtime adapters: only the local node receives protocol events. The simulator
// and network runtime share election/receive/persistence transitions, not clocks
// or transport. Remote slots are routing identities, never replicas in memory.
type replicaRuntime struct {
	PodName, PodUID, VolumeClaim, Boot string
	PID                                int
}

type nodeSnapshot struct {
	Runtime              replicaRuntime
	PeerObservations     []PeerObservation
	Transport            string
	PeerRPCs, ClientRPCs int
	Node                 *Node
	PID                  int
	Boot                 string
	Events               []Event
	Progress             *WriteProgress
	Fault                bool
	Isolated             bool
	DroppedRPCs          int
	WALBytes             int64
	ShardID, ShardCount  int
}

// runNode owns one replica's memory, clock, WAL, and HTTP listener. The other
// Node slots are routing identities, never replicas managed by this process.
func runNode(id, base int, dir string, isolated bool) error {
	return runNodeWithShard(id, base, dir, isolated, 0, 1)
}

// runNodeWithShard binds one replica to a fixed shard. Peer envelopes and
// client keys are checked before they can mutate this replica’s Raft state.
func runNodeWithShard(id, base int, dir string, isolated bool, shardID, shardCount int) error {
	return runConfiguredNode(id, base, dir, isolated, shardID, shardCount, "", nil)
}

// Pod mode uses the same node engine with explicit stable peer addresses. It
// never starts other replicas, and a PVC stays bound to one fixed voter identity.
func runConfiguredNode(id, base int, dir string, isolated bool, shardID, shardCount int, listen string, addresses []string) error {
	if (shardCount != 1 && shardCount != fixedShardCount) || shardID < 0 || shardID >= shardCount {
		return fmt.Errorf("invalid shard configuration")
	}
	if id < 0 || id > 2 || base < 1024 || base > 65532 || dir == "" {
		return fmt.Errorf("invalid node configuration")
	}
	store, err := openStore(dir)
	if err != nil {
		return err
	}
	defer store.close()
	if len(addresses) != 0 {
		if err := validateGroup(addresses); err != nil {
			return err
		}
		if err := ensureReplicaIdentity(dir, id, shardID, shardCount, addresses); err != nil {
			return err
		}
	}
	if listen == "" {
		listen = fmt.Sprintf("127.0.0.1:%d", base+id)
	}
	c := NewCluster()
	c.store = store
	c.Durable = true
	for _, n := range c.Nodes {
		n.Up = n.ID == id
		if !n.Up {
			n.Role = "offline"
		}
	}
	n := c.Nodes[id]
	// First recover the persisted checkpoint and log. Only the committed prefix
	// is applied to the database; saved pending entries are not saved values.
	d, err := store.load(id)
	if err != nil {
		return err
	}
	n.Term = d.Term
	n.Vote = d.Vote
	n.Commit = d.Commit
	n.Log = d.Log
	n.Snapshot = d.Snapshot
	c.apply(n)
	if shardCount > 1 {
		// A copied/misconfigured data directory must not silently become a
		// different shard. Existing single-group data is kept separately.
		for key := range n.KV {
			if shardForKey(key, shardCount) != shardID {
				return fmt.Errorf("saved key belongs to another shard; refusing to start")
			}
		}
		for _, entry := range n.Log {
			if entry.Key != "" && shardForKey(entry.Key, shardCount) != shardID {
				return fmt.Errorf("saved log belongs to another shard")
			}
		}
	}
	c.Events = nil
	c.note(fmt.Sprintf("Server %d process started and recovered its WAL.", id+1))
	boot := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	observations := peerObservations{boot: boot}
	var mu sync.Mutex
	// The protocol timer, RPC handlers, and client waiters share this mutex.
	// Network I/O happens outside it so one slow peer cannot freeze the node.
	droppedRPCs := 0
	var peers rpcPool
	defer peers.close()
	peerRPCs, clientRPCs := 0, 0
	mux := http.NewServeMux()
	// Liveness checks only process responsiveness. Losing quorum must not cause
	// all replicas to restart. Readiness checks local recovery/storage health,
	// not leadership: followers must be reachable to elect and replicate.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if c.storageErr != nil {
			http.Error(w, "local storage fault", 503)
			return
		}
		w.WriteHeader(204)
	})
	mux.HandleFunc("GET /state", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		var walBytes int64
		if info, err := os.Stat(store.path(id)); err == nil {
			walBytes = info.Size()
		}
		json.NewEncoder(w).Encode(nodeSnapshot{
			Runtime: replicaRuntime{PodName: os.Getenv("POD_NAME"), PodUID: os.Getenv("POD_UID"), VolumeClaim: os.Getenv("POD_VOLUME_CLAIM"), Boot: boot, PID: os.Getpid()},
			Node:    n, PID: os.Getpid(), Boot: boot, PeerObservations: observations.events,
			Transport: "grpc", PeerRPCs: peerRPCs, ClientRPCs: clientRPCs,
			Events: c.Events, Progress: c.Progress(),
			Fault: c.storageErr != nil, Isolated: isolated,
			DroppedRPCs: droppedRPCs, WALBytes: walBytes,
			ShardID: shardID, ShardCount: shardCount,
		})
	})
	// A partition affects Raft traffic in both directions, not client/observer
	// access. The process keeps running: it may still believe it is the leader.
	mux.HandleFunc("POST /network", func(w http.ResponseWriter, r *http.Request) {
		var change struct{ Isolated bool }
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&change) != nil {
			http.Error(w, "invalid network configuration", 400)
			return
		}
		mu.Lock()
		isolated = change.Isolated
		c.note(fmt.Sprintf("Server %d peer isolation = %t.", id+1, isolated))
		mu.Unlock()
		w.WriteHeader(204)
	})
	mux.HandleFunc("POST /snapshot", func(w http.ResponseWriter, r *http.Request) {
		// This is local maintenance of already committed data. It does not
		// propose a new database write or require another round of voting.
		mu.Lock()
		defer mu.Unlock()
		if err := c.TakeSnapshot(n); err != nil {
			http.Error(w, err.Error(), 409)
			return
		}
		w.WriteHeader(204)
	})
	// A transport-independent ingress path preserves validation, isolation, and
	// persistence ordering for both protobuf RPCs and compatibility HTTP tests.
	deliver := func(m Message) (int, error) {
		if m.ShardID != shardID || m.To != id || m.From < 0 || m.From > 2 || m.From == id || m.Term < 0 || m.Commit < 0 || m.Index < 0 {
			return 400, fmt.Errorf("invalid RPC")
		}
		if _, ok := wireKinds[m.Kind]; !ok {
			return 400, fmt.Errorf("invalid RPC kind")
		}
		mu.Lock()
		defer mu.Unlock()
		if isolated {
			droppedRPCs++
			return 503, fmt.Errorf("peer link partitioned")
		}
		if (m.Kind == "append" || m.Kind == "snapshot") && validate(diskState{Version: 2, Term: m.Term, Vote: -1, Commit: m.Commit, Log: m.Log, Snapshot: m.Snapshot}) != nil {
			return 400, fmt.Errorf("invalid replicated state")
		}
		if c.storageErr != nil {
			return 503, fmt.Errorf("storage fault")
		}
		c.receive(m)
		if c.storageErr != nil {
			return 503, fmt.Errorf("storage fault")
		}
		observations.record(n, m)
		return 204, nil
	}
	mux.HandleFunc("POST /message", func(w http.ResponseWriter, r *http.Request) {
		var message Message
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRecord)).Decode(&message) != nil {
			http.Error(w, "invalid RPC", 400)
			return
		}
		code, err := deliver(message)
		if err != nil {
			http.Error(w, err.Error(), code)
			return
		}
		w.WriteHeader(code)
	})
	grpcServer := grpc.NewServer(grpc.MaxRecvMsgSize(maxRecord), grpc.MaxSendMsgSize(maxRecord), grpc.WaitForHandlers(true))
	defer grpcServer.Stop()
	pb.RegisterReplicaServer(grpcServer, &grpcReplica{
		shardID: shardID,
		deliver: func(message Message) (int, error) {
			code, err := deliver(message)
			if err == nil {
				mu.Lock()
				peerRPCs++
				mu.Unlock()
			}
			return code, err
		},
		execute: func(ctx context.Context, cmd command) (receipt, int, error) {
			return executeCommand(ctx, c, n, &mu, cmd, shardID, shardCount)
		},
		onClient: func() { mu.Lock(); clientRPCs++; mu.Unlock() },
	})
	// HTTP administration stays on the same loopback port. Only protobuf RPCs
	// take this HTTP/2 branch; there is no JSON codec hidden inside gRPC.
	handler := h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	}), &http2.Server{})
	mux.HandleFunc("POST /write", nodeClientHandlerForShard(c, n, &mu, "write", shardID, shardCount))
	mux.HandleFunc("POST /delete", nodeClientHandlerForShard(c, n, &mu, "delete", shardID, shardCount))
	mux.HandleFunc("POST /read", nodeClientHandlerForShard(c, n, &mu, "read", shardID, shardCount))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{
		Addr:              listen,
		Handler:           handler,
		ReadHeaderTimeout: time.Second,
		WriteTimeout:      4 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- server.ListenAndServe() }()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	// Delivery happens outside the state mutex. Bound concurrent sends so failed
	// peers cannot create unlimited goroutines. Heartbeats retry log replication.
	slots := make(chan struct{}, 16)
	for {
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return server.Shutdown(shutdown)
		case err := <-serveErrors:
			return err
		case <-ticker.C:
			// One local tick may start an election or send a heartbeat. The
			// observer/controller does not drive this clock in live mode.
			mu.Lock()
			if c.storageErr == nil {
				c.Tick++
				if n.Role == "leader" {
					if c.Tick%4 == 0 {
						c.broadcast(n)
					}
				} else if c.Tick >= n.Deadline {
					c.election(n)
				}
			}
			outgoing := c.Messages
			c.Messages = nil
			if isolated {
				droppedRPCs += len(outgoing)
				outgoing = nil
			}
			mu.Unlock()
			for _, m := range outgoing {
				select {
				case slots <- struct{}{}:
					go func(m Message) {
						defer func() { <-slots }()
						m.ShardID = shardID
						address := fmt.Sprintf("127.0.0.1:%d", base+m.To)
						if len(addresses) != 0 {
							address = addresses[m.To]
						}
						rpc, err := peers.clientAddress(address)
						if err != nil {
							return
						}
						callCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
						defer cancel()
						// Raft retries on subsequent heartbeats. This deadline
						// bounds transport work, not the protocol's logical clock.
						_, _ = rpc.Deliver(callCtx, messageToWire(m))
					}(m)
				default: /* next heartbeat/election retries */
				}
			}
		}
	}
}

// child tracks only processes this controller created. Closing done makes it
// safe to reuse that slot for a restarted process with the same storage path.
type child struct {
	cmd  *exec.Cmd
	done chan struct{}
}

// liveController manages process lifecycle and samples observations. It is not
// a Raft voter, database replica, or source of truth for commitment.
type liveController struct {
	rpc                 rpcPool
	addresses           []string // Nonempty only when Kubernetes owns process lifecycle.
	mu                  sync.Mutex
	children            [3]*child
	executable, dir     string
	base                int
	client              *http.Client
	poll                int
	isolated            *int // Desired policy; reapplied if this child is restarted.
	shardID, shardCount int
}

// start launches a fresh address space and reuses this node's WAL directory.
func (l *liveController) start(id int) error {
	if l.external() {
		return fmt.Errorf("Kubernetes owns replica lifecycle")
	}
	if old := l.children[id]; old != nil {
		select {
		case <-old.done:
		default:
			return fmt.Errorf("server is already running")
		}
	}
	dir := filepath.Join(l.dir, fmt.Sprintf("server-%d", id+1))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	cmd := exec.Command(l.executable, "-mode=node", fmt.Sprintf("-id=%d", id), fmt.Sprintf("-base-port=%d", l.base), "-data-dir", dir)
	cmd.Args = append(cmd.Args, fmt.Sprintf("-shard-id=%d", l.shardID), fmt.Sprintf("-shard-count=%d", max(1, l.shardCount)))
	if l.isolated != nil && *l.isolated == id {
		cmd.Args = append(cmd.Args, "-isolated")
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	ch := &child{cmd: cmd, done: make(chan struct{})}
	l.children[id] = ch
	go func() { cmd.Wait(); close(ch.done) }()
	return nil
}

// kill deliberately uses SIGKILL for the Stop experiment. No graceful shutdown
// can rescue volatile state; restarting must recover from already synced data.
func (l *liveController) kill(id int) error {
	if l.external() {
		return fmt.Errorf("Kubernetes owns replica lifecycle")
	}
	ch := l.children[id]
	if ch == nil {
		return nil
	}
	select {
	case <-ch.done:
		return nil
	default:
	}
	if err := ch.cmd.Process.Kill(); err != nil {
		return err
	}
	<-ch.done
	return nil
}

// states samples children concurrently. These observations are not simultaneous
// and do not constitute a linearizable database read. PIDs verify ownership.
func (l *liveController) states() []*nodeSnapshot {
	result := make([]*nodeSnapshot, 3)
	var wg sync.WaitGroup
	for id := 0; id < 3; id++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ch := l.children[id]
			if !l.external() {
				if ch == nil {
					return
				}
				select {
				case <-ch.done:
					return
				default:
				}
			}
			resp, err := l.client.Get("http://" + l.address(id) + "/state")
			if err != nil {
				return
			}
			defer resp.Body.Close()
			var s nodeSnapshot
			if resp.StatusCode == 200 && json.NewDecoder(resp.Body).Decode(&s) == nil && s.ShardID == l.shardID && (l.external() || s.PID == ch.cmd.Process.Pid) && s.ShardCount == max(1, l.shardCount) && !s.Fault && s.Node != nil && s.Node.ID == id {
				result[id] = &s
			}
		}(id)
	}
	wg.Wait()
	return result
}

// ServeHTTP keeps control/observation separate from client requests. A write
// waiting for a quorum must not prevent the user from healing a partition.
func (l *liveController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/kv" {
		l.serveKV(w, r)
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Path != "/api/cluster" {
		http.NotFound(w, r)
		return
	}
	states := l.states()
	if r.Method == "POST" {
		var a struct {
			Action, Key, Value string
			ID                 int
		}
		if r.Header.Get("Content-Type") != "application/json" || json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&a) != nil {
			http.Error(w, "JSON required", 400)
			return
		}
		if l.external() && (a.Action == "toggle" || a.Action == "recover") {
			http.Error(w, "Kubernetes owns Pod restarts. Use kubectl delete pod; never scale the fixed voting group.", 409)
			return
		}
		switch a.Action {
		case "partition":
			if a.ID < 0 || a.ID > 2 || states[a.ID] == nil || l.isolated != nil {
				http.Error(w, "Choose a running server and heal any existing partition first.", 409)
				return
			}
			id := a.ID
			l.isolated = &id
			if err := l.control(id, "network", map[string]bool{"Isolated": true}); err != nil {
				http.Error(w, "Could not confirm isolation; refresh state or heal the network.", 503)
				return
			}
		case "heal":
			l.isolated = nil
			for id, s := range states {
				if s != nil {
					if err := l.control(id, "network", map[string]bool{"Isolated": false}); err != nil {
						http.Error(w, "Could not confirm all links healed; retry Heal network.", 503)
						return
					}
				}
			}
		case "recover":
			// Recovery restores availability without clearing logs, rows, terms,
			// snapshots, or client retry identities. Unlike heal, it starts stopped peers.
			l.isolated = nil
			for id, state := range states {
				if state == nil {
					if err := l.kill(id); err != nil {
						http.Error(w, err.Error(), 503)
						return
					}
					if err := l.start(id); err != nil {
						http.Error(w, err.Error(), 503)
						return
					}
				} else if err := l.control(id, "network", map[string]bool{"Isolated": false}); err != nil {
					http.Error(w, err.Error(), 503)
					return
				}
			}
		case "snapshot":
			if a.ID < 0 || a.ID > 2 || states[a.ID] == nil {
				http.Error(w, "Choose a running server.", 409)
				return
			}
			if err := l.control(a.ID, "snapshot", nil); err != nil {
				http.Error(w, err.Error(), 409)
				return
			}
		case "toggle":
			if a.ID < 0 || a.ID > 2 {
				http.Error(w, "invalid server", 400)
				return
			}
			var err error
			if states[a.ID] != nil {
				err = l.kill(a.ID)
			} else {
				// A child with a latched storage fault or an unresponsive HTTP
				// endpoint may still exist. Restart replaces that owned process.
				err = l.kill(a.ID)
				if err == nil {
					err = l.start(a.ID)
				}
			}
			if err != nil {
				http.Error(w, err.Error(), 409)
				return
			}
		case "write":
			http.Error(w, "Use POST /api/kv with Operation=write and a RequestID.", 400)
			return
		default:
			http.Error(w, "Step/reset are unavailable in live process mode.", 409)
			return
		}
		states = l.states()
	} else if r.Method != "GET" {
		http.Error(w, "method not allowed", 405)
		return
	}
	l.poll++
	c := NewCluster()
	c.Durable = true
	c.Tick = l.poll
	c.Events = nil
	c.Messages = nil
	c.Deliveries = nil
	pids := make([]int, 3)
	runtimes := make([]replicaRuntime, 3)
	isolatedNodes := make([]bool, 3)
	drops := make([]int, 3)
	walBytes := make([]int64, 3)
	peerCounts, clientCounts := make([]int, 3), make([]int, 3)
	var progress *WriteProgress
	var peerEvents []PeerObservation
	var leaderEvent *Event
	leaderTerm := -1
	for id, s := range states {
		if s == nil {
			c.Nodes[id] = &Node{ID: id, Role: "offline", KV: map[string]string{}}
			continue
		}
		c.Nodes[id] = s.Node
		peerEvents = append(peerEvents, s.PeerObservations...)
		pids[id] = s.PID
		runtimes[id] = s.Runtime
		isolatedNodes[id] = s.Isolated
		drops[id] = s.DroppedRPCs
		walBytes[id] = s.WALBytes
		peerCounts[id], clientCounts[id] = s.PeerRPCs, s.ClientRPCs
		if s.Node.Role == "leader" && s.Node.Term > leaderTerm {
			leaderTerm = s.Node.Term
			progress = s.Progress
			if len(s.Events) > 0 {
				e := s.Events[len(s.Events)-1]
				leaderEvent = &e
			}
		}
		if len(s.Events) > 0 && s.Node.Role != "leader" {
			e := s.Events[len(s.Events)-1]
			e.Text = fmt.Sprintf("Server %d: %s", id+1, e.Text)
			c.Events = append(c.Events, e)
		}
	}
	if leaderEvent != nil {
		c.Events = append(c.Events, *leaderEvent)
	}
	if progress != nil {
		progress.Applied = []int{}
		for id, s := range states {
			if s != nil && s.Node.Commit >= progress.Index {
				if s.Node.matches(progress.Index, Entry{Delete: progress.Delete, Term: progress.Term, Key: progress.Key, Value: progress.Value, RequestID: progress.RequestID}) {
					progress.Applied = append(progress.Applied, id)
				}
			}
		}
	}
	json.NewEncoder(w).Encode(struct {
		*Cluster
		PeerObservations     []PeerObservation
		Transport            string
		Deployment           string
		ReplicaRuntime       []replicaRuntime
		PeerRPCs, ClientRPCs []int
		Live                 bool
		PIDs                 []int
		Progress             *WriteProgress
		Isolated             []bool
		DroppedRPCs          []int
		PartitionTarget      *int
		WALBytes             []int64
		ShardID, ShardCount  int
	}{
		ReplicaRuntime: runtimes, Cluster: c, PeerObservations: peerEvents, Transport: "grpc", Deployment: l.deployment(), PeerRPCs: peerCounts, ClientRPCs: clientCounts, Live: true, PIDs: pids, Progress: progress,
		Isolated: isolatedNodes, DroppedRPCs: drops,
		PartitionTarget: l.isolated, WALBytes: walBytes,
		ShardID: l.shardID, ShardCount: max(1, l.shardCount),
	})
}

// The controller changes transport policy or requests local maintenance; it
// never grants votes or decides whether database writes are committed.
func (l *liveController) control(id int, path string, body any) error {
	b, _ := json.Marshal(body)
	response, err := (&http.Client{Timeout: 4 * time.Second}).Post("http://"+l.address(id)+"/"+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 204 {
		return fmt.Errorf("server rejected %s (HTTP %d)", path, response.StatusCode)
	}
	return nil
}

// runLive starts the supervisor and its three child processes. Shutting down
// this parent normally stops its children; SIGKILL cannot execute that cleanup.
func runLive(listen string, base int, dir string) error {
	if listen != "127.0.0.1:8084" || base < 1024 || base > 65532 {
		return fmt.Errorf("live controller must use loopback :8084 and valid peer ports")
	}
	if dir == "" {
		dir = filepath.Join("17-sharded-raft-kv", "data-live")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	// A separate controller lock prevents two supervisors from managing one set
	// of WALs. Each child also has its own exclusive storage lock.
	lock, err := openStore(dir)
	if err != nil {
		return err
	}
	defer lock.close()
	// Refuse occupied peer ports before starting any children. Snapshots also
	// verify the owned PID, so an unrelated listener cannot impersonate a child.
	var listeners []net.Listener
	for id := 0; id < 3; id++ {
		listener, e := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", base+id))
		if e != nil {
			for _, previous := range listeners {
				previous.Close()
			}
			return fmt.Errorf("peer port unavailable: %w", e)
		}
		listeners = append(listeners, listener)
	}
	for _, listener := range listeners {
		listener.Close()
	}
	l := &liveController{executable: executable, dir: dir, base: base, client: &http.Client{Timeout: 400 * time.Millisecond}}
	defer l.rpc.close()
	defer func() {
		for id := 0; id < 3; id++ {
			l.kill(id)
		}
	}()
	for id := 0; id < 3; id++ {
		if err = l.start(id); err != nil {
			return err
		}
	}
	log.Printf("Live Raft controller: http://%s · peers: %d-%d · data: %s", listen, base, base+2, dir)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Addr: listen, Handler: l, ReadHeaderTimeout: time.Second, WriteTimeout: 5 * time.Second, BaseContext: func(net.Listener) context.Context { return ctx }}
	result := make(chan error, 1)
	go func() { result <- server.ListenAndServe() }()
	select {
	case err = <-result:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
