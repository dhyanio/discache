package server

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/dhyanio/discache/logger"
	"github.com/dhyanio/discache/transport"
	"github.com/hashicorp/raft"
)

// ServerOpts represents the options for a server
type ServerOpts struct {
	ListenAddr string
	RaftNode   *raft.Raft
	Log        *logger.Logger
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
		return fmt.Errorf("listen error: %s", err)
	}

	s.Log.Info("server starting on port [%s]\n", s.ListenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			s.Log.Error("accept error: %s\n", err)
			continue
		}
		go s.handleConn(conn)
	}
}

// handleConn handles the incoming connection
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	s.Log.Info("connection made: %s", conn.RemoteAddr())

	for {
		cmd, err := transport.ParseCommand(conn)
		if err != nil {
			if err == io.EOF {
				break
			}
			s.Log.Error("parse command error: %s", err.Error())
			break
		}
		go s.handleCommand(conn, cmd)
	}

	s.Log.Info("connection closed: %s", conn.RemoteAddr())
}

// dialLeader dials the leader
func (s *Server) dialLeader(cmd *transport.CommandSet) error {
	leaderHost, leaderPort, err := net.SplitHostPort(string(s.RaftNode.Leader()))
	if err != nil {
		return fmt.Errorf("Failed to parse leader address: %v", err)
	}
	leaderHTTPAddr := fmt.Sprintf("%s%s", leaderHost, leaderPort)

	conn, err := net.Dial("tcp", leaderHTTPAddr)
	if err != nil {
		return fmt.Errorf("failed to dial leader [%s]", leaderHTTPAddr)
	}
	s.Log.Info("connected to leader: %s", leaderHTTPAddr)

	// create set comamnd
	binary.Write(conn, binary.LittleEndian, cmd.Bytes())
	s.handleConn(conn)
	return nil
}

// handleCommand handles the incoming command
func (s *Server) handleCommand(conn net.Conn, cmd any) {
	switch v := cmd.(type) {
	case *transport.CommandSet:
		s.handleSetCommand(conn, v)
	case *transport.CommandGet:
		s.handleGetCommand(conn, v)
	}
}

// handleGetCommand handles the GET command
func (s *Server) handleGetCommand(conn net.Conn, cmd *transport.CommandGet) error {
	resp := transport.ResponseGet{}

	future := s.RaftNode.Apply(cmd.Bytes(), 5*time.Second)
	if future.Error() != nil {
		resp.Status = transport.StatusError
		_, err := conn.Write(resp.Bytes())
		return err
	}

	value := future.Response()
	resp.Status = transport.StatusOK

	readFuture := s.RaftNode.VerifyLeader()
	if err := readFuture.Error(); err != nil {
		return fmt.Errorf("not the leader: %v", err)
	}

	resp.Value = value.([]byte)
	_, err := conn.Write(resp.Bytes())

	return err
}

// handleSetCommand handles the SET command
func (s *Server) handleSetCommand(conn net.Conn, cmd *transport.CommandSet) error {
	resp := transport.ResponseSet{}

	// Redirect to the leader if this node is not the leader
	if s.RaftNode.Leader() != raft.ServerAddress(s.ListenAddr) {
		if err := s.dialLeader(cmd); err != nil {
			s.Log.Error("failed to start server : %s", err.Error())
		}
		resp.Status = transport.StatusError
		_, err := conn.Write(resp.Bytes())
		return err
	}

	s.Log.Info("SET %s to %s", cmd.Key, cmd.Value)

	future := s.RaftNode.Apply(cmd.Bytes(), 5*time.Second)
	if future.Error() != nil {
		resp.Status = transport.StatusError
		_, err := conn.Write(resp.Bytes())
		return err
	}

	resp.Status = transport.StatusOK

	_, err := conn.Write(resp.Bytes())

	return err
}
