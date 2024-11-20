package transport

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Command is a byte representing a command
type Command byte

const (
	CmdNonce Command = iota
	CMDSet
	CMDGet
	CMDDel
	CMDJoin
)

// Status is a byte representing the status of a command
type Status byte

// String returns the string representation of the status
func (s Status) String() string {
	switch s {
	case StatusError:
		return "ERR"
	case StatusOK:
		return "OK"
	case StatusKeyNotFound:
		return "NOTFOUND"
	case StatusExpired:
		return "EXPIRED"
	default:
		return "NONE"
	}
}

const (
	StatusNone Status = iota
	StatusOK
	StatusError
	StatusKeyNotFound
	StatusExpired
)

// Response is a response to a command
type ResponseSet struct {
	Status Status
}

// Bytes returns the byte representation of the response
func (r *ResponseSet) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, r.Status)

	return buf.Bytes()
}

// ResponseGet is a response to a get command
type ResponseGet struct {
	Status Status
	Value  []byte
}

// Bytes returns the byte representation of the response
func (r *ResponseGet) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, r.Status)

	valueLen := int32(len(r.Value))
	binary.Write(buf, binary.LittleEndian, valueLen)
	binary.Write(buf, binary.LittleEndian, r.Value)

	return buf.Bytes()
}

// ParseSetResponse
func ParseSetResponse(r io.Reader) (*ResponseSet, error) {
	resp := &ResponseSet{}

	err := binary.Read(r, binary.LittleEndian, &resp.Status)

	return resp, err
}

// ParseGetResponse parses a get response from the reader
func ParseGetResponse(r io.Reader) (*ResponseGet, error) {
	resp := &ResponseGet{}

	binary.Read(r, binary.LittleEndian, &resp.Status)

	var valueLen int32
	binary.Read(r, binary.LittleEndian, &valueLen)
	resp.Value = make([]byte, valueLen)
	binary.Read(r, binary.LittleEndian, &resp.Value)

	return resp, nil
}

// CommandSet is a command to get a set
type CommandSet struct {
	Key   []byte
	Value []byte
	TTL   int
}

// Bytes returns the byte representation of the set command
func (c *CommandSet) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, CMDSet)

	keyLen := int32(len(c.Key))
	binary.Write(buf, binary.LittleEndian, keyLen)
	binary.Write(buf, binary.LittleEndian, c.Key)

	valueLen := int32(len(c.Value))
	binary.Write(buf, binary.LittleEndian, valueLen)
	binary.Write(buf, binary.LittleEndian, c.Value)

	binary.Write(buf, binary.LittleEndian, int32(c.TTL))

	return buf.Bytes()
}

// ParseCommand parses a command from the reader
func ParseCommand(r io.Reader) (any, error) {
	var cmd Command
	binary.Read(r, binary.LittleEndian, &cmd)

	switch cmd {
	case CMDSet:
		return parseSetCommand(r), nil
	case CMDGet:
		return parseGetCommand(r), nil
	default:
		return nil, fmt.Errorf("invalid commnad")

	}
}

// CommandGet is a command to get a key
type CommandGet struct {
	Key []byte
}

// Bytes returns the byte representation of the get command
func (c *CommandGet) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, CMDGet)

	keyLen := int32(len(c.Key))
	binary.Write(buf, binary.LittleEndian, keyLen)
	binary.Write(buf, binary.LittleEndian, c.Key)

	return buf.Bytes()
}

// parseSetCommand
func parseSetCommand(r io.Reader) *CommandSet {
	cmd := &CommandSet{}

	var keyLen int32
	binary.Read(r, binary.LittleEndian, &keyLen)
	cmd.Key = make([]byte, keyLen)
	binary.Read(r, binary.LittleEndian, &cmd.Key)

	var valueLen int32
	binary.Read(r, binary.LittleEndian, &valueLen)
	cmd.Value = make([]byte, valueLen)
	binary.Read(r, binary.LittleEndian, &cmd.Value)

	var ttl int32
	binary.Read(r, binary.LittleEndian, &ttl)
	cmd.TTL = int(ttl)

	return cmd
}

// parseGetCommand
func parseGetCommand(r io.Reader) *CommandGet {
	cmd := &CommandGet{}

	var keyLen int32
	binary.Read(r, binary.LittleEndian, &keyLen)
	cmd.Key = make([]byte, keyLen)
	binary.Read(r, binary.LittleEndian, &cmd.Key)

	return cmd
}
