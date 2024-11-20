package rafter

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/hashicorp/raft"
)

// startHTTPServer will start the HTTP server for the raft node
func startHTTPServer(raftNode *raft.Raft, addressNodeHTTP, addressNode string) {
	http.HandleFunc("/apply", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is supported", http.StatusMethodNotAllowed)
			return
		}

		fmt.Println("Leader: ", raftNode.Leader())
		fmt.Println("Node: ", raft.ServerAddress(addressNode))

		// Redirect to the leader if this node is not the leader
		if raftNode.Leader() != raft.ServerAddress(addressNode) {
			leaderHost, _, err := net.SplitHostPort(string(raftNode.Leader()))
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to parse leader address: %v", err), http.StatusInternalServerError)
				return
			}
			leaderHTTPAddr := fmt.Sprintf("%s%s", leaderHost, addressNodeHTTP)
			http.Redirect(w, r, fmt.Sprintf("http://%s/apply", leaderHTTPAddr), http.StatusTemporaryRedirect)
			return
		}

		// Decode the command from the request body
		var cmd Command
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			http.Error(w, fmt.Sprintf("Invalid command: %v", err), http.StatusBadRequest)
			return
		}
		commandData, _ := json.Marshal(cmd)

		// Apply the command to the Raft log
		future := raftNode.Apply(commandData, 5*time.Second)
		if err := future.Error(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to apply command: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Command applied successfully"))
	})

	log.Printf("Starting HTTP server on %s", addressNodeHTTP)
	log.Fatal(http.ListenAndServe(addressNodeHTTP, nil))
}
