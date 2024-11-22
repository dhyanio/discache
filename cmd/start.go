package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/dhyanio/discache/cache"
	"github.com/dhyanio/discache/rafter"
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
	Use: "node [nodeName] [nodeEndpoint] [leader [leaderName]]",
	// Use:   "node [name] [leader [leaderPort]]",
	Short: "Start a node server, optionally specifying a leader",
	Long:  "Start a node server on the specified port, optionally specifying a leader name",
	Args:  cobra.MinimumNArgs(1), // At least the node port is required
	Run: func(cmd *cobra.Command, args []string) {
		nodeName := args[0]
		nodeEndpoint := args[1]
		leaderName := ""
		isLeader := true

		if len(args) > 2 {
			// Validate if "leader" keyword is provided with correct format
			if args[2] == "leader" {
				if len(args) == 4 {
					leaderName = args[3]
					isLeader = false
				} else {
					// Print error if "leader" keyword is used but the name is missing
					fmt.Println("Error: 'leader' specified without name. Use 'discache start node [port]' for a standalone node or 'discache start node [nodeName] [nodeEndpoint] leader [leaderName]' for a node with leader.")
					os.Exit(1)
				}
			} else {
				// Print error if invalid argument is provided after the port
				fmt.Printf("Error: Unknown argument '%s'. Use 'leader [leaderName]' after the name to specify a leader.\n", args[1])
				os.Exit(1)
			}
		}

		log, err := gogger.NewLogger(loggerFilePath, gogger.INFO)
		if err != nil {
			fmt.Printf("Error: Unknown argument '%s'. Use 'leader [leaderName]' after the name to specify a leader.\n", args[1])
			os.Exit(1)
		}

		opts := rafter.RaftServerOpts{
			ID:         nodeName,
			ListenAddr: nodeEndpoint,
			IsLeader:   isLeader,
			LeaderAddr: leaderName,
			Log:        log,
		}
		startServer(opts)
	},
}

// startServer starts a server with the specified role, port, and leader port
func startServer(opts rafter.RaftServerOpts) {
	// Initialize cache with capacity 5, TTL 5 seconds, and custom eviction callback
	cc := cache.NewCache(cacheCapabity, cacheTTL, evictFunc)
	raftSever(cc, opts)
}

// raftServer using raft Server and raft's own Transport layer
func raftSever(cc *cache.Cache, opts rafter.RaftServerOpts) {
	raftFSM := rafter.NewRaftFSM(cc)
	rafter.Rafting(raftFSM, opts)
	select {}
}
