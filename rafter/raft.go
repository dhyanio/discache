package rafter

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/dhyanio/discache/cluster"
	"github.com/dhyanio/discache/server"
	"github.com/dhyanio/discache/transport"
	cache "github.com/dhyanio/go-lru"
	"github.com/dhyanio/gogger"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

// RaftServerOpts represents the options for a Raft server
type RaftServerOpts struct {
	ID            string
	ListenAddr    string
	IsLeader      bool
	LeaderAddr    string
	Log           *gogger.Logger
	ClusterConfig cluster.ClusterConfig
}

const (
	raftClusterElectionTimeout = 5 * time.Second
	nodeHTTPServer             = ":9080" // HTTP server default port for the node
)

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
	fmt.Println("RAFT FSM APPLY CALLED:", string(log.Data))
	r := bytes.NewReader(log.Data)

	cmd, err := transport.ParseCommand(r)
	if err != nil {
		return fmt.Errorf("parse command error: %s", err.Error())
	}

	switch v := cmd.(type) {
	case *transport.CommandSet:
		if err := f.cache.Put(v.Key, v.Value); err != nil {
			return fmt.Errorf("failed to set value: %s", err.Error())
		}
		// <add logger here>
		return fmt.Sprintf("SET %s to %s", v.Key, v.Value)
	default:
		return fmt.Errorf("unknown operation: %T", cmd)
	}
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
	defer sink.Close()
	return nil
}

// Release releases the snapshot.
func (s *snapshot) Release() {}

// createRaftNodeWithCluster will create raft node and cluster
func createRaftNodeWithCluster(opts RaftServerOpts, peers []raft.Server, fsm *raftFSM) (*raft.Raft, error) {
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

	// Create discardSnapshotStore
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
	raftNode, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("failed to create Raft: %v", err)
	}

	// Bootstrap raft cluster on leader only
	if opts.IsLeader {
		cfg := raft.Configuration{
			Servers: peers,
		}
		f := raftNode.BootstrapCluster(cfg)
		if err := f.Error(); err != nil {
			return nil, fmt.Errorf("raft.Raft.BootstrapCluster: %v", err)
		}
	}

	return raftNode, nil
}

// Rafting will start the raft node
func Rafting(raftFSM *raftFSM, opts RaftServerOpts) {
	// 1. Validate that this node exists in the cluster configuration.
	node, ok := opts.ClusterConfig.NodeByID(opts.ID)
	if !ok {
		opts.Log.Fatal().Msgf(
			"Node %s not found in cluster configuration",
			opts.ID,
		)
		return
	}

	// 2. Validate that the configured Raft address matches this node.
	if node.RaftAddr != opts.ListenAddr {
		opts.Log.Fatal().Msgf(
			"Raft address mismatch for node %s: configured=%s, actual=%s",
			opts.ID,
			node.RaftAddr,
			opts.ListenAddr,
		)
		return
	}

	// 3. Generate Raft peers from cluster configuration.
	peers := opts.ClusterConfig.RaftPeers()

	// 4. Create the Raft node.
	raftNode, err := createRaftNodeWithCluster(opts, peers, raftFSM)
	if err != nil {
		opts.Log.Fatal().Msgf(
			"Error starting node %s: %v",
			opts.ID,
			err,
		)
		return
	}

	opts.Log.Info().Msgf(
		"Raft node %s started at %s",
		opts.ID,
		opts.ListenAddr,
	)

	// 5. Periodically display the current leader.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			leaderAddr, leaderID := raftNode.LeaderWithID()

			opts.Log.Info().Msgf(
				"Node=%s State=%s LeaderID=%s LeaderAddr=%s",
				opts.ID,
				raftNode.State(),
				leaderID,
				leaderAddr,
			)
		}
	}()

	// 6. Start the application server on this node's application address.
	serverOpts := server.ServerOpts{
		ListenAddr:    node.AppAddr,
		Log:           opts.Log,
		RaftNode:      raftNode,
		ClusterConfig: opts.ClusterConfig,
		Cache:         raftFSM.cache,
	}

	appServer := server.NewServer(serverOpts)

	if err := appServer.Start(); err != nil {
		opts.Log.Fatal().Msgf(
			"Failed to start application server on %s: %v",
			node.AppAddr,
			err,
		)
		return
	}
}
