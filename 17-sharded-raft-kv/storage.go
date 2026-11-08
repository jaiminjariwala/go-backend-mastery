package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// Each checksummed record is a complete persistent-state revision. Revisions can
// replace an uncommitted suffix; this is a deliberately unoptimized WAL, not an
// append of KV commands alone. Commit is persisted explicitly as recovery metadata.
type diskState struct {
	Version, Term, Vote, Commit int
	Log                         []Entry
	Snapshot                    Snapshot
}

// walStore owns the directory lock and the last synced revision per replica.
// Its bookkeeping avoids duplicate writes without weakening fsync guarantees.
type walStore struct {
	dir   string
	lock  *os.File
	last  map[int][]byte
	bases map[int]int
}

const maxRecord = 16 << 20

// openStore gives one process exclusive ownership of a storage directory. The
// directory lock protects WAL replacement as well as ordinary appends.
func openStore(dir string) (*walStore, error) {
	if err := os.Mkdir(dir, 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	parent, err := os.Open(filepath.Dir(dir))
	if err != nil {
		return nil, err
	}
	err = parent.Sync()
	parent.Close()
	if err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filepath.Join(dir, "LOCK"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return nil, fmt.Errorf("data directory is already in use: %w", err)
	}
	return &walStore{dir: dir, lock: lock, last: map[int][]byte{}, bases: map[int]int{}}, nil
}

// close releases directory ownership; it never removes persistent data.
func (s *walStore) close() error { return s.lock.Close() }

// path keeps each replica’s journal separate inside its owned directory.
func (s *walStore) path(id int) string {
	return filepath.Join(s.dir, fmt.Sprintf("server-%d.wal", id+1))
}

// stateOf excludes volatile fields such as role, timer, and received votes.
// Recovery must start as a follower rather than restoring an old leader label.
func stateOf(n *Node) diskState {
	return diskState{
		Version: 2,
		Term:    n.Term, Vote: n.Vote, Commit: n.Commit,
		Log: n.Log, Snapshot: n.Snapshot,
	}
}

// validate rejects impossible boundaries before recovery can slice the retained
// log. A snapshot index and an array offset are different coordinate systems.
func validate(d diskState) error {
	// Version 1 files predate snapshots. Their implicit snapshot boundary is 0.
	if (d.Version != 1 && d.Version != 2) || d.Term < 0 || d.Vote < -1 || d.Vote > 2 || d.Snapshot.Index < 0 || d.Snapshot.Term < 0 || d.Snapshot.Term > d.Term || d.Commit < d.Snapshot.Index || d.Commit > d.Snapshot.Index+len(d.Log) {
		return fmt.Errorf("invalid persistent state")
	}
	if d.Snapshot.Index == 0 && (d.Snapshot.Term != 0 || len(d.Snapshot.KV) > 0 || len(d.Snapshot.Requests) > 0) {
		return fmt.Errorf("invalid empty snapshot")
	}
	for id, result := range d.Snapshot.Requests {
		if id == "" || result.Key == "" || result.Index < 1 || result.Index > d.Snapshot.Index || result.Term < 0 || result.Term > d.Snapshot.Term {
			return fmt.Errorf("invalid snapshot receipt")
		}
	}
	previous := d.Snapshot.Term
	for _, e := range d.Log {
		if e.Term < previous || e.Term > d.Term || (e.Delete && (e.Key == "" || e.ReadKey != "" || e.Value != "")) {
			return fmt.Errorf("invalid log terms")
		}
		previous = e.Term
	}
	return nil
}

// load replays complete, checksummed state revisions. Only an incomplete FINAL
// record is repairable: a complete record with a bad checksum is corruption.
func (s *walStore) load(id int) (diskState, error) {
	d := diskState{Version: 1, Vote: -1}
	f, err := os.OpenFile(s.path(id), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return d, err
	}
	defer f.Close()
	var offset int64
	for {
		// Each frame begins with 4 bytes of payload length and 4 bytes of CRC32.
		// A checksum detects damaged bytes; it is not cryptographic authentication.
		var header [8]byte
		_, err = io.ReadFull(f, header[:])
		if err == io.EOF {
			break
		}
		if err == io.ErrUnexpectedEOF {
			if err = f.Truncate(offset); err != nil {
				return d, err
			}
			break
		}
		if err != nil {
			return d, err
		}
		size := binary.BigEndian.Uint32(header[:4])
		if size == 0 || size > maxRecord {
			return d, fmt.Errorf("invalid WAL record size at %d", offset)
		}
		payload := make([]byte, size)
		_, err = io.ReadFull(f, payload)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			if err = f.Truncate(offset); err != nil {
				return d, err
			}
			break
		}
		if err != nil {
			return d, err
		}
		if crc32.ChecksumIEEE(payload) != binary.BigEndian.Uint32(header[4:]) {
			return d, fmt.Errorf("WAL checksum mismatch at %d", offset)
		}
		var next diskState
		if err = json.Unmarshal(payload, &next); err != nil {
			return d, err
		}
		if err = validate(next); err != nil {
			return d, err
		}
		d = next
		s.bases[id] = next.Snapshot.Index
		s.last[id] = payload
		offset += 8 + int64(size)
	}
	// Sync file creation / torn-tail repair before admitting new operations.
	if err = f.Sync(); err != nil {
		return d, err
	}
	directory, err := os.Open(s.dir)
	if err != nil {
		return d, err
	}
	defer directory.Close()
	if err = directory.Sync(); err != nil {
		return d, err
	}
	return d, nil
}

// save appends a changed state revision and fsyncs before returning. Callers may
// release a protocol acknowledgment only after this operation succeeds.
func (s *walStore) save(n *Node) error {
	// Installing a newer snapshot also reclaims old WAL revisions. This happens
	// before the follower's acknowledgment is released to the network.
	if n.Snapshot.Index > s.bases[n.ID] {
		return s.compact(n)
	}
	payload, err := json.Marshal(stateOf(n))
	if err != nil {
		return err
	}
	if bytes.Equal(s.last[n.ID], payload) {
		return nil
	}
	if len(payload) > maxRecord {
		return fmt.Errorf("WAL revision exceeds 16 MiB; compact the retained log or reduce state size")
	}
	f, err := os.OpenFile(s.path(n.ID), os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	var header [8]byte
	binary.BigEndian.PutUint32(header[:4], uint32(len(payload)))
	binary.BigEndian.PutUint32(header[4:], crc32.ChecksumIEEE(payload))
	record := append(header[:], payload...)
	written, err := f.Write(record)
	if err == nil && written != len(record) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	s.last[n.ID] = payload
	s.bases[n.ID] = n.Snapshot.Index
	return nil
}

// Snapshot and retained log are installed as ONE checksummed WAL revision.
// Writing two independent files would create a dangerous crash window where a
// new snapshot could be paired with an old/truncated log (or the reverse).
func (s *walStore) compact(n *Node) error {
	state := stateOf(n)
	if err := validate(state); err != nil {
		return err
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if len(payload) > maxRecord {
		return fmt.Errorf("snapshot revision exceeds 16 MiB")
	}
	f, err := os.CreateTemp(s.dir, ".snapshot-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	var header [8]byte
	binary.BigEndian.PutUint32(header[:4], uint32(len(payload)))
	binary.BigEndian.PutUint32(header[4:], crc32.ChecksumIEEE(payload))
	record := append(header[:], payload...)
	written, err := f.Write(record)
	if err == nil && written != len(record) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	// Until rename, a crash leaves the old WAL intact. After rename, recovery
	// sees a complete snapshot + suffix. Sync the directory to persist rename.
	if err = os.Rename(name, s.path(n.ID)); err != nil {
		return err
	}
	dir, err := os.Open(s.dir)
	if err != nil {
		return err
	}
	err = dir.Sync()
	dir.Close()
	if err != nil {
		return err
	}
	s.last[n.ID] = payload
	s.bases[n.ID] = n.Snapshot.Index
	return nil
}

// NewDurableCluster is the disk-backed simulator constructor. Live children
// load only their own node's WAL in runNode, instead of sharing three replicas.
func NewDurableCluster(dir string) (*Cluster, error) {
	s, err := openStore(dir)
	if err != nil {
		return nil, err
	}
	c := NewCluster()
	c.store = s
	c.Durable = true
	for _, n := range c.Nodes {
		d, err := s.load(n.ID)
		if err != nil {
			s.close()
			return nil, fmt.Errorf("server %d recovery: %w", n.ID+1, err)
		}
		n.Term = d.Term
		n.Vote = d.Vote
		n.Log = d.Log
		n.Commit = d.Commit
		n.Snapshot = d.Snapshot
		c.apply(n)
	}
	c.note("Recovered durable logs. Servers start as followers; uncommitted entries remain unapplied.")
	return c, nil
}

// persist latches storage errors and discards outgoing messages. Continuing to
// acknowledge after a failed sync could falsely promise durable replication.
func (c *Cluster) persist(n *Node) bool {
	if c.storageErr != nil {
		return false
	}
	if c.store == nil {
		return true
	}
	if err := c.store.save(n); err != nil {
		c.storageErr = err
		c.Messages = nil
		c.note("Storage failure: experiment halted. Restart after repairing storage.")
		return false
	}
	return true
}
