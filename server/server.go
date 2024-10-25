package server

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/dhyanio/discache/cache"
	"github.com/dhyanio/discache/client"
	"github.com/dhyanio/discache/logger"
	"github.com/dhyanio/discache/transport"
)

// ServerOpts represents the options for a cache server
type ServerOpts struct {
	ListenAddr string
	IsLeader   bool
	LeaderAddr string
	Log        *logger.Logger
}

// Server represents a cache server
type Server struct {
	ServerOpts
	members map[*client.Client]struct{}
	cache   cache.Cacher
}

// NewServer creates a new cache server
func NewServer(opts ServerOpts, c cache.Cacher) *Server {
	return &Server{
		ServerOpts: opts,
		cache:      c,
		members:    make(map[*client.Client]struct{}),
	}
}

// Start starts the cache server
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen error: %s", err)
	}

	s.Log.Info("working3")
	if !s.IsLeader && len(s.LeaderAddr) != 0 {
		go func() {
			if err := s.dialLeader(); err != nil {
				s.Log.Error(err.Error())
			}
		}()
	}
	s.Log.Info(fmt.Sprintf("server starting on port [%s]\n", s.ListenAddr))

	for {
		conn, err := ln.Accept()
		if err != nil {
			s.Log.Error(fmt.Sprintf("accept error: %s\n", err))
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *Server) dialLeader() error {
	conn, err := net.Dial("tcp", s.LeaderAddr)
	if err != nil {
		return fmt.Errorf("failed to dial leader [%s]", s.LeaderAddr)
	}
	s.Log.Info(fmt.Sprintf("connected to leader: %s", s.LeaderAddr))

	binary.Write(conn, binary.LittleEndian, transport.CMDJoin)
	s.handleConn(conn)
	return nil
}

// handleConn handles the incoming connection
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	s.Log.Info(fmt.Sprintf("connection made: %s", conn.RemoteAddr()))

	for {
		cmd, err := transport.ParseCommand(conn)
		if err != nil {
			if err == io.EOF {
				break
			}
			s.Log.Error(fmt.Sprintf("parse command error: %s", err.Error()))
			break
		}
		go s.handleCommand(conn, cmd)
	}

	s.Log.Info(fmt.Sprintf("connection closed: %s", conn.RemoteAddr()))
}

// handleCommand handles the incoming command
func (s *Server) handleCommand(conn net.Conn, cmd any) {
	switch v := cmd.(type) {
	case *transport.CommandSet:
		s.handleSetCommand(conn, v)
	case *transport.CommandGet:
		s.handleGetCommand(conn, v)
	case *transport.CommandJoin:
		s.handleJoinCommand(conn)
	}
}

func (s *Server) handleJoinCommand(conn net.Conn) error {
	s.Log.Info(fmt.Sprintf("member joined %s", conn.RemoteAddr()))
	s.members[client.NewFromConn(conn)] = struct{}{}

	return nil
}

func (s *Server) handleGetCommand(conn net.Conn, cmd *transport.CommandGet) error {
	resp := transport.ResponseGet{}
	value, err := s.cache.Get(cmd.Key)
	if err != nil {
		resp.Status = transport.StatusError
		_, err := conn.Write(resp.Bytes())
		return err
	}
	resp.Status = transport.StatusOK
	resp.Value = value

	_, err = conn.Write(resp.Bytes())

	return err
}

func (s *Server) handleSetCommand(conn net.Conn, cmd *transport.CommandSet) error {
	s.Log.Info(fmt.Sprintf("SET %s to %s", cmd.Key, cmd.Value))

	go func() {
		for member := range s.members {
			err := member.Put(context.TODO(), cmd.Key, cmd.Value, cmd.TTL)
			if err != nil {
				s.Log.Info(fmt.Sprintf("forward to member error: %s", err.Error()))
			}
		}
	}()

	resp := transport.ResponseSet{}
	if err := s.cache.Put(cmd.Key, cmd.Value, time.Duration(cmd.TTL)); err != nil {
		resp.Status = transport.StatusError

		_, err := conn.Write(resp.Bytes())
		return err
	}
	resp.Status = transport.StatusOK

	_, err := conn.Write(resp.Bytes())

	return err
}
