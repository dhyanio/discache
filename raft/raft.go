package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

type demoFSM struct{}

func (f *demoFSM) Apply(log *raft.Log) interface{} {
	fmt.Printf("Applying command: %s\n", string(log.Data))
	return nil
}

func (f *demoFSM) Snapshot() (raft.FSMSnapshot, error) {
	return &demoSnapshot{}, nil
}

func (f *demoFSM) Restore(io.ReadCloser) error {
	return nil
}

type demoSnapshot struct{}

func (s *demoSnapshot) Persist(sink raft.SnapshotSink) error {
	return sink.Close()
}

func (s *demoSnapshot) Release() {}

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

	raftNode, err := raft.NewRaft(config, &demoFSM{}, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("failed to create Raft: %v", err)
	}

	// Bootstrap only the first node
	if id == "node1" {
		raftNode.BootstrapCluster(raft.Configuration{Servers: peers})
	}

	return raftNode, nil
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <node_id> <address>", os.Args[0])
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
		log.Fatalf("Error starting node %s: %v", nodeID, err)
	}

	// Apply command only on the leader node
	if nodeID == "node1" {
		go func() {
			time.Sleep(30 * time.Second)
			command := []byte("Hello, Raft!")
			future := raftNode.Apply(command, 5*time.Second)
			if err := future.Error(); err != nil {
				fmt.Printf("Error applying command: %v\n", err)
			} else {
				fmt.Println("Command applied successfully")
			}
		}()
	}

	select {}
}
