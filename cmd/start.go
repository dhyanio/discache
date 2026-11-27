package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dhyanio/discache/cluster"
	"github.com/dhyanio/discache/rafter"
	cache "github.com/dhyanio/go-lru"
	"github.com/dhyanio/gogger"
	"github.com/spf13/cobra"
)

const (
	loggerFilePath = "discache.log"
	cacheCapabity  = 10
	cacheTTL       = 5 * time.Second
)

var evictFunc = func(key string, value []byte) {
	fmt.Printf("Evicted: %s -> %s\n", key, value)
}

// startCmd creates the start command
func startCmd() *cobra.Command {
	command := cobra.Command{
		Use:   "start",
		Short: "Start cache server nodes",
		Long:  "Start cache server nodes, either as a leader or a node connected to a leader",
	}
	command.AddCommand(&nodeCmd)
	return &command
}

// nodeCmd creates the node command

var nodeCmd = cobra.Command{
	Use:   "node [nodeName] config [configPath] [leader [leaderName]]",
	Short: "Start a node server, optionally specifying a leader",
	Long:  "Start a node using the cluster configuration file",
	Args:  cobra.MinimumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		nodeName := args[0]

		if args[1] != "config" {
			fmt.Printf("Error: expected 'config [configPath]' after node name\n")
			os.Exit(1)
		}

		configPath := args[2]

		// Load cluster configuration.
		configData, err := os.ReadFile(configPath)
		if err != nil {
			fmt.Printf("Failed to read config file: %v\n", err)
			os.Exit(1)
		}

		var clusterConfig cluster.ClusterConfig
		if err := json.Unmarshal(configData, &clusterConfig); err != nil {
			fmt.Printf("Failed to parse config file: %v\n", err)
			os.Exit(1)
		}

		// Validate that the requested node exists.
		node, ok := clusterConfig.NodeByID(nodeName)
		if !ok {
			fmt.Printf("Node %q not found in cluster configuration\n", nodeName)
			os.Exit(1)
		}

		leaderName := ""
		isLeader := true

		// Optional leader argument.
		if len(args) > 3 {
			if args[3] != "leader" || len(args) != 5 {
				fmt.Println("Expected: leader [leaderName]")
				os.Exit(1)
			}

			leaderName = args[4]
			isLeader = false

			// Validate leader exists in configuration.
			if _, ok := clusterConfig.NodeByID(leaderName); !ok {
				fmt.Printf("Leader %q not found in cluster configuration\n", leaderName)
				os.Exit(1)
			}
		}

		log, err := gogger.NewLogger(loggerFilePath, gogger.INFO)
		if err != nil {
			fmt.Printf("Failed to create logger: %v\n", err)
			os.Exit(1)
		}

		opts := rafter.RaftServerOpts{
			ID:            nodeName,
			ListenAddr:    node.RaftAddr,
			IsLeader:      isLeader,
			LeaderAddr:    leaderName,
			ClusterConfig: clusterConfig,
			Log:           log,
		}

		startServer(opts)
	},
}

// startServer starts a server with the specified role, port, and leader port
func startServer(opts rafter.RaftServerOpts) {
	// Initialize cache with capacity 5, TTL 5 seconds, and custom eviction callback
	cacheOpts := cache.CacheOpts{
		Capacity: cacheCapabity,
		// TTL:      cacheTTL,
		TTL:     0,
		OnEvict: evictFunc,
	}
	cc := cache.NewCache(cacheOpts)
	raftSever(cc, opts)
}

// raftServer using raft Server and raft's own Transport layer
func raftSever(cc *cache.Cache, opts rafter.RaftServerOpts) {
	raftFSM := rafter.NewRaftFSM(cc)
	rafter.Rafting(raftFSM, opts)
	select {}
}
