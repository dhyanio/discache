package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/dhyanio/discache/cache"
)

// ServerOpts represents the options for a cache server
type ServerOpts struct {
	ListenAddr string
	IsLeader   bool
}

// Server represents a cache server
type Server struct {
	ServerOpts
	cache cache.Cacher
}

// NewServer creates a new cache server
func NewServer(opts ServerOpts, c cache.Cacher) *Server {
	return &Server{
		ServerOpts: opts,
		cache:      c,
	}
}

// Start starts the cache server
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen error: %s", err)
	}
	log.Printf("server starting on port [%s]\n", s.ListenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %s\n", err)
			continue
		}
		go s.handleConn(conn)
	}
}

// handleConn handles the incoming connection
func (s *Server) handleConn(conn net.Conn) {
	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("conn read error: %s\n", err)
			break
		}

		go s.handleCommand(conn, buf[:n])
	}
}

// handleCommand handles the incoming command
func (s *Server) handleCommand(conn net.Conn, rawCmd []byte) {
	msg, err := parseMessage(rawCmd)
	if err != nil {
		fmt.Println("failed to parse command", err)
		conn.Write([]byte(fmt.Sprintf("ERR %s\n", err)))
		return
	}
	switch msg.Cmd {
	case CMDSet:
		err = s.handleSetCmd(conn, msg)
	case CMDGet:
		err = s.handleGetCmd(conn, msg)
	}

	if err != nil {
		fmt.Println("failed to handle command", err)
		conn.Write([]byte(fmt.Sprintf("ERR %s\n", err)))
	}
}

// handleGetCmd handles the GET command
func (s *Server) handleGetCmd(conn net.Conn, msg *Message) error {
	val, err := s.cache.Get(msg.Key)
	if err != nil {
		return fmt.Errorf("cache get error: %s", err)
	}
	conn.Write([]byte(fmt.Sprintf("VALUE %s\n", val)))

	return nil
}

// handleSetCmd handles the SET command
func (s *Server) handleSetCmd(conn net.Conn, msg *Message) error {
	if err := s.cache.Set(msg.Key, msg.Value, msg.TTL); err != nil {
		return fmt.Errorf("cache set error: %s", err)
	}
	conn.Write([]byte("OK\n"))
	go s.sendToFollwers(context.TODO(), msg)
	return nil
}

func (s *Server) sendToFollwers(ctx context.Context, msg *Message) error {
	return nil
}
