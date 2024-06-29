package store

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	tuplespace "tuplespaceCD/pkg/tuplespace"

	"github.com/hashicorp/raft"
	opt "github.com/micutio/goptional"
)

const (
	retainSnapshotCount = 2
	raftTimeout         = 10 * time.Second
)

type command struct {
	Op    string            `json:"op,omitempty"`
	Tuple []tuplespace.Elem `json:"tuple,omitempty"`
}

// Store is a distributed tuple space store, where all changes are made via Raft consensus.
type Store struct {
	RaftDir  string
	RaftBind string

	mu         sync.Mutex
	tupleSpace tuplespace.Store // The tuple space for the system.

	raft   *raft.Raft // The consensus mechanism
	logger *log.Logger
}

// New returns a new Store.
func New() *Store {
	return &Store{
		tupleSpace: tuplespace.NewSimpleStore(), // Initialize the tuple space
		logger:     log.New(os.Stderr, "[store] ", log.LstdFlags),
	}
}

// Open opens the store. If enableSingle is set, and there are no existing peers,
// then this node becomes the first node, and therefore leader, of the cluster.
// localID should be the server identifier for this node.
func (s *Store) Open(enableSingle bool, localID string) error {
	// Setup Raft configuration.
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(localID)

	// Setup Raft communication.
	addr, err := net.ResolveTCPAddr("tcp", s.RaftBind)
	if err != nil {
		return err
	}
	transport, err := raft.NewTCPTransport(s.RaftBind, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return err
	}

	// Create the snapshot store. This allows the Raft to truncate the log.
	snapshots, err := raft.NewFileSnapshotStore(s.RaftDir, retainSnapshotCount, os.Stderr)
	if err != nil {
		return fmt.Errorf("file snapshot store: %s", err)
	}

	// Create the log store and stable store.
	logStore := raft.NewInmemStore()
	stableStore := raft.NewInmemStore()

	// Instantiate the Raft systems.
	ra, err := raft.NewRaft(config, (*fsm)(s), logStore, stableStore, snapshots, transport)
	if err != nil {
		return fmt.Errorf("new raft: %s", err)
	}
	s.raft = ra

	if enableSingle {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		ra.BootstrapCluster(configuration)
	}

	return nil
}

// Write writes a tuple to the tuple space.
func (s *Store) Write(tuple tuplespace.Tuple) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}
	fmt.Printf("Write: %s\n", tuple)
	c := &command{
		Op:    "write",
		Tuple: tuple.GetElements(),
	}
	b, err := json.Marshal(c)
	fmt.Printf("Write JSON: %s\n", b)
	if err != nil {
		return err
	}

	f := s.raft.Apply(b, raftTimeout)
	return f.Error()
}

// Get retrieves and removes a tuple matching the query from the tuple space.
func (s *Store) Get(query tuplespace.Tuple) (opt.Maybe[tuplespace.Tuple], error) {
	if s.raft.State() != raft.Leader {
		return opt.NewNothing[tuplespace.Tuple](), fmt.Errorf("not leader")
	}

	c := &command{
		Op:    "get",
		Tuple: query.GetElements(),
	}
	b, err := json.Marshal(c)
	if err != nil {
		return opt.NewNothing[tuplespace.Tuple](), err
	}

	f := s.raft.Apply(b, raftTimeout)
	if err := f.Error(); err != nil {
		return opt.NewNothing[tuplespace.Tuple](), err
	}

	result, ok := f.Response().(opt.Maybe[tuplespace.Tuple])
	if !ok {
		return opt.NewNothing[tuplespace.Tuple](), fmt.Errorf("unexpected response type")
	}
	return result, nil
}

// Read retrieves a tuple matching the query from the tuple space.
func (s *Store) Read(query tuplespace.Tuple) (opt.Maybe[tuplespace.Tuple], error) {
	if s.raft.State() != raft.Leader {
		return opt.NewNothing[tuplespace.Tuple](), fmt.Errorf("not leader")
	}

	c := &command{
		Op:    "read",
		Tuple: query.GetElements(),
	}
	b, err := json.Marshal(c)
	if err != nil {
		return opt.NewNothing[tuplespace.Tuple](), err
	}

	f := s.raft.Apply(b, raftTimeout)
	if err := f.Error(); err != nil {
		return opt.NewNothing[tuplespace.Tuple](), err
	}

	result, ok := f.Response().(opt.Maybe[tuplespace.Tuple])
	if !ok {
		return opt.NewNothing[tuplespace.Tuple](), fmt.Errorf("unexpected response type")
	}
	fmt.Printf("Read: %s\n", result)
	return result, nil
}

// Join joins a node, identified by nodeID and located at addr, to this store.
// The node must be ready to respond to Raft communications at that address.
func (s *Store) Join(nodeID, addr string) error {
	s.logger.Printf("received join request for remote node %s at %s", nodeID, addr)

	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		s.logger.Printf("failed to get raft configuration: %v", err)
		return err
	}

	for _, srv := range configFuture.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if srv.Address == raft.ServerAddress(addr) && srv.ID == raft.ServerID(nodeID) {
				s.logger.Printf("node %s at %s already member of cluster, ignoring join request", nodeID, addr)
				return nil
			}

			future := s.raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, addr, err)
			}
		}
	}

	f := s.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if f.Error() != nil {
		return f.Error()
	}
	s.logger.Printf("node %s at %s joined successfully", nodeID, addr)
	return nil
}

type fsm Store

// Apply applies a Raft log entry to the tuple space store.
func (f *fsm) Apply(l *raft.Log) interface{} {
	var c command
	if err := json.Unmarshal(l.Data, &c); err != nil {
		panic(fmt.Sprintf("failed to unmarshal command: %s", err.Error()))
	}

	elements := c.Tuple
	tuple := tuplespace.MakeTuple(elements...)

	switch c.Op {
	case "write":
		return f.applyWrite(tuple)
	case "get":
		return f.applyGet(tuple)
	case "read":
		return f.applyRead(tuple)
	default:
		panic(fmt.Sprintf("unrecognized command op: %s", c.Op))
	}
}

// Snapshot returns a snapshot of the tuple space store.
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Clone the tuple space store.
	tupleSnapshot := f.cloneTupleSpace()
	return &fsmSnapshot{tuples: tupleSnapshot}, nil
}

// Restore restores the tuple space store to a previous state.
func (f *fsm) Restore(rc io.ReadCloser) error {
	var snapshot tuplespace.BTreeStore
	if err := json.NewDecoder(rc).Decode(&snapshot); err != nil {
		return err
	}

	// Restore the state from the snapshot.
	f.mu.Lock()
	f.tupleSpace = &snapshot
	f.mu.Unlock()

	return nil
}

func (f *fsm) applyWrite(tuple tuplespace.Tuple) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tupleSpace.Write(tuple)
}

func (f *fsm) applyGet(query tuplespace.Tuple) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tupleSpace.Get(query)
}

func (f *fsm) applyRead(query tuplespace.Tuple) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tupleSpace.Read(query)
}

// cloneTupleSpace creates a deep copy of the tuple space store.
func (f *fsm) cloneTupleSpace() *tuplespace.BTreeStore {
	clone := f.tupleSpace.(*tuplespace.BTreeStore).Copy()
	return clone
}

type fsmSnapshot struct {
	tuples *tuplespace.BTreeStore
}

func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		// Encode data.
		b, err := json.Marshal(f.tuples)
		if err != nil {
			return err
		}

		// Write data to sink.
		if _, err := sink.Write(b); err != nil {
			return err
		}

		// Close the sink.
		return sink.Close()
	}()

	if err != nil {
		sink.Cancel()
	}

	return err
}

func (f *fsmSnapshot) Release() {}
