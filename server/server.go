package server

import (
	"encoding/json"
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
func (s *Server) handleGetCommand(conn net.Conn, cmd any) error {
	resp := transport.ResponseGet{}

	commandData, _ := json.Marshal(cmd)

	future := s.RaftNode.Apply(commandData, 5*time.Second)
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
	s.Log.Info("SET %s to %s", cmd.Key, cmd.Value)

	resp := transport.ResponseSet{}

	commandData, _ := json.Marshal(cmd)

	future := s.RaftNode.Apply(commandData, 5*time.Second)
	if future.Error() != nil {
		resp.Status = transport.StatusError
		_, err := conn.Write(resp.Bytes())
		return err
	}

	resp.Status = transport.StatusOK

	_, err := conn.Write(resp.Bytes())

	return err
}
