package main

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

type Command string

const (
	CMDSet Command = "SET"
	CMDGet Command = "GET"
)

// Message represents a message sent to the server
type (
	Message struct {
		Cmd   Command
		Key   []byte
		Value []byte
		TTL   time.Duration
	}
)

// parseMessage parses the incoming message and returns a Message struct
func parseMessage(data []byte) (*Message, error) {
	var (
		rawStr = string(data)
		parts  = strings.Split(rawStr, " ")
	)
	if len(parts) < 2 {
		return nil, errors.New("invalid message")
	}
	msg := &Message{
		Cmd: Command(parts[0]),
		Key: []byte(parts[1]),
	}

	if msg.Cmd == CMDSet {
		if len(parts) < 4 {
			return nil, errors.New("invalid message")
		}
		msg.Value = []byte(parts[2])

		ttl, err := strconv.Atoi(parts[3])
		if err != nil {
			return nil, errors.New("invalid ttl")
		}
		msg.TTL = time.Duration(ttl)
	}
	return msg, nil
}
