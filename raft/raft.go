package rafter

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/dhyanio/discache/cache"
	"github.com/dhyanio/discache/server"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

const raftClusterElectionTimeout = 5 * time.Second

// Command structure for key-value updates
type Command struct {
	Op    string // Operation type, e.g., "set"
	Key   string // Key to set
	Value string // Value to associate with the key
}

// raftFSM with cache
type raftFSM struct {
	// cache *cache.Cache
	kvStore map[string]string
}

// NewDemoFSM initializes the FSM with an empty kvStore.
func NewRaftFSM(cache *cache.Cache) *raftFSM {
	return &raftFSM{
		// cache: cache,
		kvStore: make(map[string]string),
	}
}

func (f *raftFSM) Apply(log *raft.Log) any {
	// Decode the command from the Log entry
	var cmd Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		fmt.Printf("Failed to unmarshal command: %v\n", err)
		return nil
	}

	// Apply the command to the kvStore
	if cmd.Op == "set" {
		f.kvStore[cmd.Key] = cmd.Value
		fmt.Printf("Applied command: set %s = %s\n", cmd.Key, cmd.Value)
	}

	// Print the current state of the kvStore
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

// createRaftNodeWithCluster will create raft node and cluster
func createRaftNodeWithCluster(opts server.ServerOpts, peers []raft.Server) (*raft.Raft, error) {
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

// Rafting
func Rafting(raftFSM *raftFSM, opts server.ServerOpts) {
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

	go startHTTPServer(raftNode, "127.0.0.1:9082", opts.ListenAddr)
	// run tcp
}

func startHTTPServer(raftNode *raft.Raft, address, nodeAddress string) {
	http.HandleFunc("/apply", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is supported", http.StatusMethodNotAllowed)
			return
		}

		fmt.Println("Leader: ", raftNode.Leader())
		fmt.Println("Node: ", raft.ServerAddress(nodeAddress))

		// Decode the command from the request body
		var cmd Command
		commandData, _ := json.Marshal(cmd)

		// Redirect to the leader if this node is not the leader
		if raftNode.Leader() != raft.ServerAddress(nodeAddress) {
			fmt.Println("Calling leader")
			leader := raftNode.Leader()

			if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
				http.Error(w, fmt.Sprintf("Invalid command: %v", err), http.StatusBadRequest)
				return
			}
			// Apply the command to the Raft log
			future := raftNode.Apply(commandData, 5*time.Second)
			if err := future.Error(); err != nil {
				http.Error(w, fmt.Sprintf("Failed to apply command: %v", err), http.StatusInternalServerError)
				return
			}
			return
		}

		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			http.Error(w, fmt.Sprintf("Invalid command: %v", err), http.StatusBadRequest)
			return
		}

		// Apply the command to the Raft log
		future := raftNode.Apply(commandData, 5*time.Second)
		if err := future.Error(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to apply command: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Command applied successfully"))
	})

	log.Printf("Starting HTTP server on %s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

// raftClient sends commands to the cluster leader only
// func raftClient(opts server.ServerOpts) {
// 	go func() {
// 		time.Sleep(40 * time.Second) // Wait for Raft to initialize

// 		// Check if this node is the Leader
// 		if raftNode.Leader() == raft.ServerAddress(address) {
// 			// Example command to set a key-value pair
// 			cmd := Command{
// 				Op:    "set",
// 				Key:   "foo",
// 				Value: "bar",
// 			}
// 			commandData, _ := json.Marshal(cmd)

// 			future := raftNode.Apply(commandData, 5*time.Second)
// 			if err := future.Error(); err != nil {
// 				opts.Log.Info("Error applying command: %v\n", err)
// 			} else {
// 				opts.Log.Info("Command applied successfully")
// 			}
// 		} else {
// 			opts.Log.Info("This node is not the leader. Cannot apply commands.")
// 		}
// 	}()
// }
