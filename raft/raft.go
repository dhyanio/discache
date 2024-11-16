package rafter

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/dhyanio/discache/cache"
	"github.com/dhyanio/discache/logger"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

// Command structure for key-value updates
type Command struct {
	Op    string // Operation type, e.g., "set"
	Key   string // Key to set
	Value string // Value to associate with the key
}

// raftFSM with cache
type raftFSM struct {
	cache *cache.Cache
}

// NewDemoFSM initializes the FSM with an empty kvStore.
func NewRaftFSM(cache *cache.Cache) *raftFSM {
	return &raftFSM{
		cache: cache,
	}
}

func (f *raftFSM) Apply(log *raft.Log) any {
	// // Decode the command from the Log entry
	// var cmd Command
	// if err := json.Unmarshal(log.Data, &cmd); err != nil {
	// 	fmt.Printf("Failed to unmarshal command: %v\n", err)
	// 	return nil
	// }

	// // Apply the command to the kvStore
	// if cmd.Op == "set" {
	// 	f.kvStore[cmd.Key] = cmd.Value
	// 	fmt.Printf("Applied command: set %s = %s\n", cmd.Key, cmd.Value)
	// }

	// // Print the current state of the kvStore
	fmt.Printf("Current kvStore state: %v\n", log.Data)

	return nil
}

func (f *raftFSM) Snapshot() (raft.FSMSnapshot, error) {
	return &demoSnapshot{}, nil
}

func (f *raftFSM) Restore(io.ReadCloser) error {
	return nil
}

type demoSnapshot struct{}

func (s *demoSnapshot) Persist(sink raft.SnapshotSink) error {
	return sink.Close()
}

func (s *demoSnapshot) Release() {}

// CreateRaftNode
func createRaftNode(id string, address string, peers []raft.Server) (*raft.Raft, error) {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(id)
	config.ElectionTimeout = 5 * time.Second

	logStore, err := raftboltdb.NewBoltStore(fmt.Sprintf("raft-log-%s.bolt", id))
	if err != nil {
		return nil, fmt.Errorf("failed to create log store: %v", err)
	}

	stableStore, err := raftboltdb.NewBoltStore(fmt.Sprintf("raft-stable-%s.bolt", id))
	if err != nil {
		return nil, fmt.Errorf("failed to create stable store: %v", err)
	}

	snapshotStore := raft.NewDiscardSnapshotStore()

	// Convert the address to Raft's format
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %v", err)
	}

	transport, err := raft.NewTCPTransport(address, addr, 3, 10*time.Second, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %v", err)
	}

	raftNode, err := raft.NewRaft(config, NewRaftFSM(nil), logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("failed to create Raft: %v", err)
	}

	// Bootstrap only the first node
	if id == "node1" {
		raftNode.BootstrapCluster(raft.Configuration{Servers: peers})
	}

	return raftNode, nil
}

// Rafting
func Rafting(rafter *raftFSM, log *logger.Logger) {
	if len(os.Args) < 3 {
		log.Fatal("Usage: %s <node_id> <address>", os.Args[0])
	}
	nodeID := os.Args[1]
	address := os.Args[2]

	// Define the cluster configuration with all nodes
	peers := []raft.Server{
		{ID: raft.ServerID("node1"), Address: raft.ServerAddress("127.0.0.1:8080")},
		{ID: raft.ServerID("node2"), Address: raft.ServerAddress("127.0.0.1:8081")},
		{ID: raft.ServerID("node3"), Address: raft.ServerAddress("127.0.0.1:8082")},
	}

	// Create the Raft node
	raftNode, err := createRaftNode(nodeID, address, peers)
	if err != nil {
		log.Fatal("Error starting node %s: %v", nodeID, err)
	}

	// Display the current leader periodically
	go func() {
		for {
			time.Sleep(20 * time.Second)
			leader := raftNode.Leader()
			log.Info("Current leader: %s\n", leader)
		}
	}()

	// Apply command only on the leader node
	if nodeID == "node1" {
		go func() {
			time.Sleep(40 * time.Second) // Wait for Raft to initialize

			// Check if this node is the Leader
			if raftNode.Leader() == raft.ServerAddress(address) {
				// Example command to set a key-value pair
				cmd := Command{
					Op:    "set",
					Key:   "foo",
					Value: "bar",
				}
				commandData, _ := json.Marshal(cmd)

				future := raftNode.Apply(commandData, 5*time.Second)
				if err := future.Error(); err != nil {
					log.Info("Error applying command: %v\n", err)
				} else {
					log.Info("Command applied successfully")
				}
			} else {
				log.Info("This node is not the leader. Cannot apply commands.")
			}
		}()
	}
}
