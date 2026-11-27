package cluster

import "github.com/hashicorp/raft"

type Node struct {
	ID       string `json:"id"`
	RaftAddr string `json:"raft_addr"`
	AppAddr  string `json:"app_addr"`
}

type ClusterConfig struct {
	Nodes []Node `json:"nodes"`
}

func (c ClusterConfig) RaftPeers() []raft.Server {
	peers := make([]raft.Server, 0, len(c.Nodes))

	for _, node := range c.Nodes {
		peers = append(peers, raft.Server{
			ID:      raft.ServerID(node.ID),
			Address: raft.ServerAddress(node.RaftAddr),
		})
	}

	return peers
}

func (c ClusterConfig) AppAddrForRaft(raftAddr string) (string, bool) {
	for _, node := range c.Nodes {
		if node.RaftAddr == raftAddr {
			return node.AppAddr, true
		}
	}

	return "", false
}

func (c ClusterConfig) NodeByID(id string) (Node, bool) {
	for _, node := range c.Nodes {
		if node.ID == id {
			return node, true
		}
	}

	return Node{}, false
}
