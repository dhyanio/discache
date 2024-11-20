package rafter

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/dhyanio/discache/cache"
	"github.com/dhyanio/discache/logger"
	"github.com/dhyanio/discache/server"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

// RaftServerOpts represents the options for a Raft server
type RaftServerOpts struct {
	ID         string
	ListenAddr string
	IsLeader   bool
	LeaderAddr string
	Log        *logger.Logger
}

const (
	raftClusterElectionTimeout = 5 * time.Second
	nodeHTTPServer             = ":9080" // HTTP server default port for the node
)

// Command structure for key-value updates
type Command struct {
	Op    string // Operation type, e.g., "set"
	Key   string // Key to set
	Value string // Value to associate with the key
}

// raftFSM is a finite state machine that applies log entries to the key-value store.
type raftFSM struct {
	cache *cache.Cache
}

// NewRaftFSM creates a new Raft finite state machine.
func NewRaftFSM(cache *cache.Cache) *raftFSM {
	return &raftFSM{
		cache: cache,
	}
}

// Apply applies a Raft log entry to the Cache.
func (f *raftFSM) Apply(log *raft.Log) any {
	// Decode the command from the Log entry
	var cmd Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		fmt.Printf("Failed to unmarshal command: %v\n", err)
		return nil
	}

	// Apply the command to the Cache
	if cmd.Op == "set" {
	} else if cmd.Op == "get" {
	}

	// Print the current state of the kvStore
	fmt.Printf("Current kvStore state: %v\n", log.Data)

	return nil
}

// Snapshot returns a snapshot of the key-value store.
func (f *raftFSM) Snapshot() (raft.FSMSnapshot, error) {
	return &snapshot{}, nil
}

// Restore restores the key-value store to a previous state.
func (f *raftFSM) Restore(io.ReadCloser) error {
	return nil
}

// snapshot is a structure that represents a snapshot of the key-value store.
type snapshot struct{}

// Persist persists the snapshot to a sink.
func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	return sink.Close()
}

// Release releases the snapshot.
func (s *snapshot) Release() {}

// createRaftNodeWithCluster will create raft node and cluster
func createRaftNodeWithCluster(opts RaftServerOpts, peers []raft.Server) (*raft.Raft, error) {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(opts.ID)
	config.ElectionTimeout = raftClusterElectionTimeout

	// Create logStore
	logStore, err := raftboltdb.NewBoltStore(fmt.Sprintf("raft-log-%s.bolt", opts.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to create log store: %v", err)
	}

	// Create stableStore
	stableStore, err := raftboltdb.NewBoltStore(fmt.Sprintf("raft-stable-%s.bolt", opts.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to create stable store: %v", err)
	}

	// Create discardSnapshortStore
	snapshotStore := raft.NewDiscardSnapshotStore()

	// Convert the address to Raft's format
	addr, err := net.ResolveTCPAddr("tcp", opts.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %v", err)
	}

	// Create transporter
	transport, err := raft.NewTCPTransport(opts.ListenAddr, addr, 3, 10*time.Second, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %v", err)
	}

	// Construct a new Raft node
	raftNode, err := raft.NewRaft(config, NewRaftFSM(nil), logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("failed to create Raft: %v", err)
	}

	// Bootstrap raft cluster on leader only
	if opts.IsLeader {
		raftNode.BootstrapCluster(raft.Configuration{Servers: peers})
	}

	return raftNode, nil
}

// Rafting will start the raft node
func Rafting(raftFSM *raftFSM, opts RaftServerOpts) {
	// Define the cluster configuration with all nodes
	peers := []raft.Server{
		{ID: raft.ServerID("node1"), Address: raft.ServerAddress("127.0.0.1:8080")},
		{ID: raft.ServerID("node2"), Address: raft.ServerAddress("127.0.0.1:8081")},
		{ID: raft.ServerID("node3"), Address: raft.ServerAddress("127.0.0.1:8082")},
	}

	// Create the Raft node
	raftNode, err := createRaftNodeWithCluster(opts, peers)
	if err != nil {
		opts.Log.Fatal("Error starting node %s: %v", opts.ID, err)
	}

	// Display the current leader periodically
	go func() {
		for {
			time.Sleep(20 * time.Second)
			leader := raftNode.Leader()
			opts.Log.Info("Current leader: %s\n", leader)
		}
	}()

	// Start the HTTP server.
	// This will be for the HTTP server that will be used to communicate with the Raft node.
	// startHTTPServer(raftNode, nodeHTTPServer, opts.ListenAddr)

	// Start the Raft node server
	nodeListenHost, _, err := net.SplitHostPort(opts.ListenAddr)
	if err != nil {
		opts.Log.Error("Failed to parse node address: %v", err)
		return
	}
	nodeServerAddr := fmt.Sprintf("%s%s", nodeListenHost, nodeHTTPServer)

	serverOpts := server.ServerOpts{
		ListenAddr: nodeServerAddr,
		Log:        opts.Log,
		RaftNode:   raftNode,
	}
	server := server.NewServer(serverOpts)
	if err := server.Start(); err != nil {
		opts.Log.Fatal(err.Error())
	}
}
