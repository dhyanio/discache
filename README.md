<img src="./doc/discache.png" alt="Discache" width="600"/>

[![release](https://img.shields.io/github/release/dhyanio/discache.svg)](https://github.com/dhyanio/discache/releases)
[![tests](https://github.com/dhyanio/discache/actions/workflows/test.yaml/badge.svg)](https://github.com/dhyanio/discache/actions/workflows/test.yaml)
[![linter](https://github.com/dhyanio/discache/actions/workflows/linter.yaml/badge.svg)](https://github.com/dhyanio/discache/actions/workflows/linter.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/dhyanio/discache)](https://goreportcard.com/report/github.com/dhyanio/discache)
![Go Version](https://img.shields.io/badge/go%20version-%3E=1.23-61CFDD.svg?style=flat-square)
[![release](https://godoc.org/github.com/dhyanio/discache?status.svg)](https://pkg.go.dev/github.com/dhyanio/discache?tab=doc)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](https://opensource.org/licenses/MIT)

# Discache

**A distributed in-memory cache built with Go and HashiCorp Raft.**

Discache is an experimental distributed key-value cache that uses TCP for client communication and the Raft consensus algorithm to replicate write operations across multiple nodes.

The project explores how distributed systems work internally: leader election, log replication, follower forwarding, state machine application, and failure handling.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Why Two Different Ports?](#why-two-different-ports)
- [How Discache Works](#how-discache-works)
- [SET Request Flow](#set-request-flow)
- [GET Request Flow](#get-request-flow)
- [Writing to a Follower](#writing-to-a-follower)
- [Leader Failure and Failover](#leader-failure-and-failover)
- [Local Setup](#local-setup)
- [Testing](#testing)
- [Project Structure](#project-structure)
- [Current Limitations](#current-limitations)

---

## Overview

Discache maintains a cache on multiple nodes.

Each node has its own local in-memory cache. Raft is used to coordinate write operations and replicate them across the cluster.

For example, consider a cluster with three nodes:

| Node | Raft Port | Application Port | Initial Role |
|---|---:|---:|---|
| node1 | 8080 | 9080 | Leader |
| node2 | 8081 | 9081 | Follower |
| node3 | 8082 | 9082 | Follower |

When a client writes a key-value pair:

```text
SET user:1001 Ajay
```

The write is submitted to the Raft leader. Once the operation is committed, the nodes apply it to their local cache.

The goal is to keep the cache state replicated across the cluster rather than maintaining three unrelated caches.

## Architecture

Discache separates client-facing application traffic from internal Raft communication.

```text
                         CLIENT
                            |
                            |
                     TCP / Binary Protocol
                            |
                            v
                 Application Layer (9080-9082)
                            |
             +--------------+--------------+
             |              |              |
             v              v              v
         +--------+     +--------+     +--------+
         | Node 1 |     | Node 2 |     | Node 3 |
         | Leader |     |Follower|     |Follower|
         +--------+     +--------+     +--------+
             |              |              |
             +--------------+--------------+
                            |
                      Raft Consensus
                      TCP (8080-8082)
                            |
                            v
                    Raft Log Replication
                            |
                            v
                     Raft FSM Apply()
                            |
             +--------------+--------------+
             |              |              |
             v              v              v
          Cache 1        Cache 2        Cache 3
```

Each node has two important components:

### 1. Application Layer

The application layer is responsible for:

- Accepting client TCP connections.
- Decoding binary commands.
- Processing `SET` and `GET` requests.
- Forwarding writes to the current leader when necessary.
- Sending responses back to clients.
- Reading values from the local cache.

It exposes the client-facing TCP endpoint.

### 2. Raft Layer

The Raft layer is responsible for:

- Leader election.
- Maintaining cluster membership.
- Replicating write logs.
- Establishing consensus.
- Applying committed log entries to the state machine.

Discache uses HashiCorp Raft for this functionality.

The Raft FSM (Finite State Machine) is responsible for applying committed commands to the local cache.

---

## Why Two Different Ports?

This is an important architectural decision.

Each node exposes two independent TCP endpoints because application traffic and consensus traffic serve different purposes.

### Application ports: 9080, 9081, 9082

These ports are used by clients and by application-level forwarding between nodes.

For example:

```text
Client
  |
  | SET user:1001 Ajay
  |
  v
127.0.0.1:9081
```

The client connects to the application port of node2.

The application server understands the Discache binary protocol, decodes commands and generates application responses.

### Raft ports: 8080, 8081, 8082

These ports are used internally by HashiCorp Raft.

Nodes communicate over these ports to exchange Raft messages, including:

- RequestVote
- PreVote
- AppendEntries
- Heartbeats
- Replicated log entries

For example:

```text
Node 1 (8080)
     |
     | Raft RPCs
     |
     +------------> Node 2 (8081)
     |
     +------------> Node 3 (8082)
```

### Why not use the same port?

The two layers speak different protocols and have different responsibilities.

| Application Layer | Raft Layer |
|---|---|
| Client-facing | Internal cluster communication |
| Custom binary protocol | HashiCorp Raft transport protocol |
| SET and GET operations | Voting, heartbeats and log replication |
| Returns cache responses | Maintains consensus |
| Ports 9080–9082 | Ports 8080–8082 |

Separating the ports makes the system easier to reason about and keeps client request handling independent from consensus transport.

**Important:** Port 8080 is not an HTTP API endpoint. It is the TCP transport endpoint used by Raft.

---

## How Discache Works

The core idea is that a client-facing request is not necessarily handled by the node that receives it.

For writes, the current Raft leader is responsible for coordinating the operation.

For reads, the current implementation reads from the receiving node's local cache.

This distinction is important.

### SET Request Flow

Suppose a client sends:

```text
SET user:1001 Ajay
```

The normal flow when the client connects directly to the leader is:

```text
Client
  |
  | SET user:1001 Ajay
  |
  v
Node 1 (Leader)
  |
  | 1. Decode command
  |
  | 2. Submit command to Raft
  |
  v
Raft Log
  |
  | 3. Replicate log entry
  |
  +-----------> Node 2 (Follower)
  |
  +-----------> Node 3 (Follower)
  |
  | 4. Obtain majority agreement
  |
  v
Commit Log Entry
  |
  | 5. Apply committed entry
  |
  +-----------> FSM on Node 1
  |
  +-----------> FSM on Node 2
  |
  +-----------> FSM on Node 3
  |
  v
Local Cache Updated
  |
  v
Client receives response
```

Step by step:

1. The application server receives the client's SET command.
2. The command is submitted to the Raft leader.
3. Raft replicates the log entry to the followers.
4. Raft commits the entry after obtaining the required quorum.
5. Each node applies the committed entry through its FSM.
6. The FSM decodes the command and updates that node's local cache.
7. The application returns a response to the client.

The important part is that the write is coordinated through Raft rather than independently updating each node's cache.

### What is the FSM?

FSM stands for Finite State Machine.

In Discache, the FSM is the component that converts a committed Raft log entry into an actual cache operation.

For example, the replicated command contains the serialized SET operation.

The FSM decodes it:

```go
cmd, err := transport.ParseCommand(r)
```

Then it identifies the command:

```go
switch v := cmd.(type) {
case *transport.CommandSet:
    if err := f.cache.Put(v.Key, v.Value); err != nil {
        return fmt.Errorf("failed to set value: %w", err)
    }
}
```

The same committed command is applied on each participating node.

This is how the local caches are updated consistently through the Raft log.

---

## GET Request Flow

GET is different from SET.

In the current implementation, GET reads directly from the local cache of the node that receives the request.

For example:

```text
Client
  |
  | GET user:1001
  |
  v
Node 2 (Follower)
  |
  | Read local cache
  |
  v
Cache 2
  |
  | Return value
  |
  v
Client receives Ajay
```

The GET operation does not currently go through Raft consensus.

This makes reads straightforward and avoids sending every read through the leader.

However, there is an important trade-off:

A follower can potentially return stale data if it has not yet applied the latest committed write.

The current GET implementation should therefore be understood as a local cache read, not a linearizable read.

---

## Writing to a Follower

What happens if a client sends SET to node2 even though node1 is the leader?

For example:

```text
Client
  |
  | SET user:1001 Ajay
  |
  v
Node 2 (Follower)
  |
  | Check current Raft leader
  |
  v
Leader = 127.0.0.1:8080
  |
  | Map Raft address to application address
  |
  v
127.0.0.1:9080
  |
  | Forward SET command
  |
  v
Node 1 (Leader)
  |
  | Raft replication and commit
  |
  v
All nodes apply command
  |
  v
Node 1 sends response
  |
  v
Node 2 forwards response
  |
  v
Client receives response
```

The follower does not independently commit the write.

Instead, the application server:

1. Receives the SET request.
2. Checks the current Raft leader using the Raft node.
3. Gets the leader's Raft address.
4. Uses the peer mapping to find that node's application address.
5. Opens a TCP connection to the leader's application port.
6. Sends the serialized SET command.
7. Reads the leader's response.
8. Forwards the response to the original client.

Example mapping:

```go
peerMap := map[string]string{
    "127.0.0.1:8080": "127.0.0.1:9080",
    "127.0.0.1:8081": "127.0.0.1:9081",
    "127.0.0.1:8082": "127.0.0.1:9082",
}
```

The first address is the Raft endpoint; the second is the application endpoint.

This mapping is important because Raft knows about Raft addresses, while the application server needs to connect to application addresses.

---

## Leader Failure and Failover

Raft maintains a leader among the cluster members.

In a three-node cluster, a majority requires two votes.

For example:

```text
Before failure:

       Node 1
       LEADER
       /    \
      /      \
  Node 2    Node 3
 FOLLOWER  FOLLOWER
```

Now suppose Node 1 stops unexpectedly.

```text
After Node 1 failure:

       Node 1
        DOWN

  Node 2    Node 3
 FOLLOWER  CANDIDATE
              |
              | Election
              |
              v
           LEADER
```

Node 2 and Node 3 can still communicate.

If one of them receives the required majority, it becomes the new leader.

Once the election completes, the remaining nodes can continue accepting writes.

### What we tested locally

The following behavior was observed during the local three-node test:

- Node 1 initially acted as leader.
- Node 2 and Node 3 acted as followers.
- Node 1 was stopped.
- Node 2 attempted an election but did not obtain a quorum in that attempt.
- Node 3 subsequently became the leader.
- Node 2 reported the new leader address as `127.0.0.1:8082`.

This demonstrates leader failover in the local test cluster.

---

## Local Setup

### Prerequisites

- Go installed
- Git installed
- Linux, macOS, or WSL
- Three available pairs of TCP ports

### Clone the repository

```bash
git clone https://github.com/dhyanio/discache.git
cd discache
```

### Start Node 1 (Initial Leader)

Open terminal 1:

```bash
make run NAME=node1 CONFIG=config.json
```

Node 1 uses:

- Raft: `127.0.0.1:8080`
- Application: `127.0.0.1:9080`

### Start Node 2 (Follower)

Open terminal 2:

```bash
make follower NAME=node2 Leader_NAME=node1 CONFIG=config.json
```

Node 2 uses:

- Raft: `127.0.0.1:8081`
- Application: `127.0.0.1:9081`

### Start Node 3 (Follower)

Open terminal 3:

```bash
make follower NAME=node3 Leader_NAME=node1 CONFIG=config.json
```

Node 3 uses:

- Raft: `127.0.0.1:8082`
- Application: `127.0.0.1:9082`

Your cluster should now look like this:

| Node | Raft | Application | Role |
|---|---|---|---|
| node1 | 8080 | 9080 | Leader |
| node2 | 8081 | 9081 | Follower |
| node3 | 8082 | 9082 | Follower |

---

## Testing

### 1. Test SET and GET on the leader

```bash
go run ./example/simple.go 127.0.0.1:9080
```

This example exercises the client PUT/GET flow against node1.

### 2. Test SET through a follower

```bash
go run ./example/simple.go 127.0.0.1:9081
```

The request reaches node2, which should forward the SET to the current leader.

### 3. Read from Node 2

```bash
go run ./example/read.go 127.0.0.1:9081
```

### 4. Read from Node 3

```bash
go run ./example/read.go 127.0.0.1:9082
```

If the write has been applied on both followers, the same value should be available on both nodes.

### 5. Test leader failover

First, start all three nodes.

Then stop Node 1 using Ctrl+C.

Wait for Raft to detect the failure and elect a new leader.

Check the logs for the new leader. For example:

```text
Current leader: 127.0.0.1:8082
```

This indicates that Node 3 is now the leader.

Now test a write through Node 2:

```bash
go run ./example/simple.go 127.0.0.1:9081
```

Then read from the remaining nodes:

```bash
go run ./example/read.go 127.0.0.1:9081
go run ./example/read.go 127.0.0.1:9082
```

The write should be coordinated by the new leader.

**Note:** A three-voter cluster can continue operating with two nodes available. If two nodes are unavailable, the remaining node cannot form a majority and cannot commit new writes.

---

## Project Structure

```text
discache/
|
+-- client/
|   +-- client.go
|       Client-side API for PUT and GET
|
+-- cmd/
|   +-- start.go
|       Node initialization and startup
|
+-- example/
|   +-- simple.go
|   +-- read.go
|       Example client programs
|
+-- rafter/
|   +-- raft.go
|       Raft configuration, cluster setup and FSM
|
+-- server/
|   +-- server.go
|       TCP server, request handling and forwarding
|
+-- transport/
|   +-- transport.go
|       Binary command and response serialization
|
+-- util/
|       Utility functions and errors
|
+-- Makefile
+-- main.go
+-- go.mod
```

### Main Components

| Component | Responsibility |
|---|---|
| `client` | Connects to a Discache node and sends commands |
| `transport` | Encodes and decodes binary commands and responses |
| `server` | Handles TCP connections and application requests |
| `rafter` | Creates and manages Raft nodes and the FSM |
| `cmd` | Initializes the cache and starts the node |
| `example` | Demonstrates client operations |

---

## Current Limitations

Discache is an experimental distributed cache and is not yet a production-ready distributed database or cache service.

Some important limitations to keep in mind:

1. **Local reads:** GET reads from the receiving node's local cache. It does not currently use a Raft read barrier or leader verification.
2. **Snapshot and restore:** The current FSM snapshot and restore implementations are placeholders. They do not yet serialize and restore the cache contents.
3. **TTL:** The FSM currently applies SET using the cache's `Put` operation. The TTL field in the serialized SET command is not yet applied as a per-key TTL. Cache-wide TTL is a separate configuration.
4. **Recovery testing:** Node restart, snapshot recovery, and rejoining a node after a period offline need additional testing.
5. **Dynamic membership:** Adding and removing nodes dynamically is not yet documented as a supported workflow.

These are useful areas for future development.

---
