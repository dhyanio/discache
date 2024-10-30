package cmd

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dhyanio/discache/cache"
	"github.com/dhyanio/discache/client"
	"github.com/dhyanio/discache/logger"
	"github.com/dhyanio/discache/server"
	"github.com/spf13/cobra"
)

// startCmd
func startCmd() *cobra.Command {
	command := cobra.Command{
		Use:   "start",
		Short: "Start cache server nodes",
		Long:  "Start cache server nodes, either as a leader or a node connected to a leader",
	}
	command.AddCommand(&leaderCmd, &nodeCmd)
	return &command
}

// startServer
func startServer(role, port, leaderPort string) {
	var opts server.ServerOpts

	// Initialize Logger
	logFile, _ := os.Create("discache.log")
	defer logFile.Close()

	log := logger.NewLogger(logger.INFO, nil)

	if role == "leader" {
		fmt.Printf("Starting leader on port %s\n", port)
		opts = server.ServerOpts{
			ListenAddr: port,
			IsLeader:   len(leaderPort) == 0,
			LeaderAddr: leaderPort,
			Log:        log,
		}
	} else {
		if leaderPort != "" {
			fmt.Printf("Starting node on port %s with leader at %s\n", port, leaderPort)
			opts = server.ServerOpts{
				ListenAddr: port,
				IsLeader:   len(leaderPort) == 0,
				LeaderAddr: leaderPort,
				Log:        log,
			}
		} else {
			fmt.Printf("Starting node on port %s\n", port)
			opts = server.ServerOpts{
				ListenAddr: port,
				IsLeader:   len(leaderPort) == 0,
				LeaderAddr: leaderPort,
				Log:        log,
			}
		}
	}

	go func() {
		if opts.IsLeader {
			time.Sleep(time.Second * 10)
			SendStuff(log)
		}
	}()

	// Initialize cache with capacity 3, TTL 5 seconds, and custom eviction callback
	cc := cache.NewCache(5, 5*time.Second, func(key string, value []byte) {
		fmt.Printf("Evicted: %s -> %s\n", key, value)
	})

	server := server.NewServer(opts, cc)

	fmt.Println("IsLeader", opts.IsLeader)

	if err := server.Start(); err != nil {
		log.Fatal(err.Error())
	}
}

func randomByte(n int) []byte {
	buf := make([]byte, n)
	io.ReadFull(rand.Reader, buf)
	return buf
}

func SendStuff(log *logger.Logger) {
	for i := 0; i < 70; i++ {
		go func(i int) {
			client, err := client.New(":3000", client.Options{Log: log})
			if err != nil {
				log.Fatal(err.Error())
			}

			var (
				key   = []byte(fmt.Sprintf("test_key_%d", i))
				value = []byte(fmt.Sprintf("test_value_%d", i))
			)
			err = client.Put(context.Background(), key, value, 0)
			if err != nil {
				log.Fatal(err.Error())
			}

			fmt.Println("get", string(key))
			resp, err := client.Get(context.Background(), key)
			if err != nil {
				log.Fatal(err.Error())
			}
			log.Info(string(resp))
			client.Close()
		}(i)
	}
}

// leaderCmd
var leaderCmd = cobra.Command{
	Use:   "leader [port]",
	Short: "Start a leader server",
	Long:  "Start a leader server with the specified port",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		startServer("leader", args[0], "")
	},
}

// nodeCmd
var nodeCmd = cobra.Command{
	Use:   "node [port] [leader [leaderPort]]",
	Short: "Start a node server, optionally specifying a leader",
	Long:  "Start a node server on the specified port, optionally specifying a leader port",
	Args:  cobra.MinimumNArgs(1), // At least the node port is required
	Run: func(cmd *cobra.Command, args []string) {
		nodePort := args[0]
		leaderPort := ""

		if len(args) > 1 {
			// Validate if "leader" keyword is provided with correct format
			if args[1] == "leader" {
				if len(args) == 3 {
					leaderPort = args[2]
				} else {
					// Print error if "leader" keyword is used but the port is missing
					fmt.Println("Error: 'leader' specified without port. Use 'discache start node [port]' for a standalone node or 'discache start node [port] leader [leaderPort]' for a node with leader.")
					os.Exit(1)
				}
			} else {
				// Print error if invalid argument is provided after the port
				fmt.Printf("Error: Unknown argument '%s'. Use 'leader [leaderPort]' after the port to specify a leader.\n", args[1])
				os.Exit(1)
			}
		}
		startServer("node", nodePort, leaderPort)
	},
}
