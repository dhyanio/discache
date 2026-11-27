package cluster

import (
	"testing"

	"github.com/hashicorp/raft"
)

func testClusterConfig() ClusterConfig {
	return ClusterConfig{
		Nodes: []Node{
			{
				ID:       "node1",
				RaftAddr: "127.0.0.1:8080",
				AppAddr:  "127.0.0.1:9080",
			},
			{
				ID:       "node2",
				RaftAddr: "127.0.0.1:8081",
				AppAddr:  "127.0.0.1:9081",
			},
			{
				ID:       "node3",
				RaftAddr: "127.0.0.1:8082",
				AppAddr:  "127.0.0.1:9082",
			},
		},
	}
}

func TestNodeByID(t *testing.T) {
	config := testClusterConfig()

	tests := []struct {
		name   string
		id     string
		want   Node
		wantOK bool
	}{
		{
			name: "existing node",
			id:   "node2",
			want: Node{
				ID:       "node2",
				RaftAddr: "127.0.0.1:8081",
				AppAddr:  "127.0.0.1:9081",
			},
			wantOK: true,
		},
		{
			name:   "unknown node",
			id:     "node4",
			wantOK: false,
		},
		{
			name:   "empty node ID",
			id:     "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := config.NodeByID(tt.id)

			if ok != tt.wantOK {
				t.Fatalf("NodeByID() ok = %v, want %v", ok, tt.wantOK)
			}

			if got != tt.want {
				t.Errorf("NodeByID() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAppAddrForRaft(t *testing.T) {
	config := testClusterConfig()

	tests := []struct {
		name   string
		addr   string
		want   string
		wantOK bool
	}{
		{
			name:   "existing raft address",
			addr:   "127.0.0.1:8081",
			want:   "127.0.0.1:9081",
			wantOK: true,
		},
		{
			name:   "unknown raft address",
			addr:   "127.0.0.1:8083",
			wantOK: false,
		},
		{
			name:   "empty raft address",
			addr:   "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := config.AppAddrForRaft(tt.addr)

			if ok != tt.wantOK {
				t.Fatalf("AppAddrForRaft() ok = %v, want %v", ok, tt.wantOK)
			}

			if got != tt.want {
				t.Errorf("AppAddrForRaft() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRaftPeers(t *testing.T) {
	config := testClusterConfig()

	got := config.RaftPeers()

	if len(got) != 3 {
		t.Fatalf("RaftPeers() returned %d peers, want 3", len(got))
	}

	want := []raft.Server{
		{
			ID:      raft.ServerID("node1"),
			Address: raft.ServerAddress("127.0.0.1:8080"),
		},
		{
			ID:      raft.ServerID("node2"),
			Address: raft.ServerAddress("127.0.0.1:8081"),
		},
		{
			ID:      raft.ServerID("node3"),
			Address: raft.ServerAddress("127.0.0.1:8082"),
		},
	}

	for i := range want {
		if got[i].ID != want[i].ID {
			t.Errorf("peer[%d].ID = %q, want %q", i, got[i].ID, want[i].ID)
		}

		if got[i].Address != want[i].Address {
			t.Errorf("peer[%d].Address = %q, want %q", i, got[i].Address, want[i].Address)
		}
	}
}

func TestRaftPeersEmptyConfig(t *testing.T) {
	config := ClusterConfig{}

	got := config.RaftPeers()

	if len(got) != 0 {
		t.Errorf("RaftPeers() returned %d peers, want 0", len(got))
	}
}
