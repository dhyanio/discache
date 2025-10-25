package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/dhyanio/discache/transport"
	"github.com/dhyanio/discache/util"
	cache "github.com/dhyanio/go-lru"
	"github.com/dhyanio/gogger"
	"github.com/hashicorp/raft"
)

// ServerOpts represents the options for a server
type ServerOpts struct {
	ListenAddr string
	RaftNode   *raft.Raft
	Cache      *cache.Cache
	Log        *gogger.Logger
	PeerMap    map[string]string
}

// Server represents a server
type Server struct {
	ServerOpts
}

// NewServer creates a new cache server
func NewServer(opts ServerOpts) *Server {
	return &Server{
		ServerOpts: opts,
	}
}

// Start starts the server
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen error: %w", err)
	}

	s.Log.Info().Msgf("server starting on port [%s]\n", s.ListenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			s.Log.Error().Msgf("accept error: %s\n", err)
			continue
		}
		go s.handleConn(conn)
	}
}

// handleConn handles the incoming connection
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	s.Log.Info().Msgf("connection made: %s", conn.RemoteAddr())

	for {
		cmd, err := transport.ParseCommand(conn)
		if err != nil {
			if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "connection reset by peer") {
				s.Log.Info().Msg("client disconnected")
				break
			}
			s.Log.Error().Err(err).Msg("parse command error")
			break
		}

		go s.handleCommand(conn, cmd)
	}

	s.Log.Info().Msgf("connection closed: %s", conn.RemoteAddr())
}

// handleCommand handles the incoming command
func (s *Server) handleCommand(conn net.Conn, cmd any) {
	switch v := cmd.(type) {
	case *transport.CommandSet:
		s.handleSetCommand(conn, v)
		fmt.Println("its done right!")
	case *transport.CommandGet:
		s.handleGetCommand(conn, v)
	default:
		s.Log.Error().Msgf("unknown command type: %T", v)
	}
}

// handleGetCommand handles the GET command
func (s *Server) handleGetCommand(conn net.Conn, cmd *transport.CommandGet) {
	resp := transport.ResponseGet{}
	resp.Status = transport.StatusOK

	// readFuture := s.RaftNode.VerifyLeader()
	// if err := readFuture.Error(); err != nil {
	// 	s.Log.Error().Msgf("not the leader: %v", err)
	// 	resp.Status = transport.StatusError
	// 	s.writeResponse(conn, resp.Bytes())
	// 	return
	// }

	value, err := s.Cache.Get(cmd.Key)
	if err != nil {
		var expiredErr *util.ExpiredKeyError
		var notFoundErr *util.KeyNotFoundError
		switch {
		case errors.As(err, &expiredErr):
			resp.Status = transport.StatusExpired
		case errors.As(err, &notFoundErr):
			resp.Status = transport.StatusKeyNotFound
		default:
			resp.Status = transport.StatusError
		}

		s.writeResponse(conn, resp.Bytes())
		return
	}

	resp.Value = value
	s.writeResponse(conn, resp.Bytes())
}

// handleSetCommand handles the SET command
func (s *Server) handleSetCommand(conn net.Conn, cmd *transport.CommandSet) {
	resp := transport.ResponseSet{}
	// Redirect to the leader if this node is not the leader
	leaderAppAddr, err := s.getLeaderAddr()
	if err != nil {
		resp.Status = transport.StatusError
		s.writeResponse(conn, resp.Bytes())
		return
	}

	if raft.ServerAddress(leaderAppAddr) != raft.ServerAddress(s.ListenAddr) {
		if err := s.dialLeader(cmd, leaderAppAddr); err != nil {
			s.Log.Error().Msgf("failed to dial leader: %s", err.Error())
			resp.Status = transport.StatusError
			s.writeResponse(conn, resp.Bytes())
			return
		}
		resp.Status = transport.StatusOK
		s.writeResponse(conn, resp.Bytes())
		return
	}

	future := s.RaftNode.Apply(cmd.Bytes(), 5*time.Second)
	if future.Error() != nil {
		resp.Status = transport.StatusError
		s.writeResponse(conn, resp.Bytes())
		return
	}

	s.Log.Info().Msgf("Response %s", future.Response())
	resp.Status = transport.StatusOK
	s.writeResponse(conn, resp.Bytes())
}

// writeResponse writes the response to the connection
func (s *Server) writeResponse(conn net.Conn, data []byte) {
	if _, err := conn.Write(data); err != nil {
		s.Log.Error().Msgf("failed to write response: %s", err.Error())
	}
}

// dialLeader dials the leader
func (s *Server) dialLeader(cmd *transport.CommandSet, leaderAddr string) error {
	conn, err := net.Dial("tcp", leaderAddr)
	if err != nil {
		return fmt.Errorf("failed to dial leader [%s]: %w", leaderAddr, err)
	}

	s.Log.Info().Msgf("connected to leader: %s", leaderAddr)

	if err := binary.Write(conn, binary.LittleEndian, cmd.Bytes()); err != nil {
		return fmt.Errorf("failed to write command to leader: %w", err)
	}

	s.handleConn(conn)
	return nil
}

func (s *Server) dialLeaderAndForward(origConn net.Conn, cmd *transport.CommandSet, leaderAddr string) error {
	conn, err := net.Dial("tcp", leaderAddr)
	if err != nil {
		return fmt.Errorf("failed to dial leader [%s]: %w", leaderAddr, err)
	}
	defer conn.Close()

	// Send command to leader
	if _, err := conn.Write(cmd.Bytes()); err != nil {
		return fmt.Errorf("failed to write to leader: %w", err)
	}

	// Wait for leader's response
	var resp transport.ResponseSet
	if err := binary.Read(conn, binary.LittleEndian, &resp.Status); err != nil {
		return fmt.Errorf("failed to read from leader: %w", err)
	}

	// Forward response to original client
	if _, err := origConn.Write(resp.Bytes()); err != nil {
		return fmt.Errorf("failed to write back to client: %w", err)
	}

	return nil
}

// getLeaderAddr returns the leader address
func (s *Server) getLeaderAddr() (string, error) {
	raftAddr := string(s.RaftNode.Leader())
	if raftAddr == "" {
		return "", fmt.Errorf("leader not known yet")
	}
	appAddr, ok := s.PeerMap[raftAddr]
	if !ok {
		return "", fmt.Errorf("no app mapping for leader %s", raftAddr)
	}
	return appAddr, nil
}
