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

// ResponseSet is a response to a set command
type ResponseSet struct {
	Status Status
}

// Bytes returns the byte representation of the response
func (r *ResponseSet) Bytes() []byte {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, r.Status); err != nil {
		return nil
	}
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
	if err := binary.Write(buf, binary.LittleEndian, r.Status); err != nil {
		return nil
	}

	valueLen := int32(len(r.Value))
	if err := binary.Write(buf, binary.LittleEndian, valueLen); err != nil {
		return nil
	}
	if err := binary.Write(buf, binary.LittleEndian, r.Value); err != nil {
		return nil
	}

	return buf.Bytes()
}

// ParseSetResponse parses a set response from the reader
func ParseSetResponse(r io.Reader) (*ResponseSet, error) {
	resp := &ResponseSet{}
	if err := binary.Read(r, binary.LittleEndian, &resp.Status); err != nil {
		return nil, err
	}
	return resp, nil
}

// ParseGetResponse parses a get response from the reader
func ParseGetResponse(r io.Reader) (*ResponseGet, error) {
	resp := &ResponseGet{}
	if err := binary.Read(r, binary.LittleEndian, &resp.Status); err != nil {
		return nil, err
	}

	var valueLen int32
	if err := binary.Read(r, binary.LittleEndian, &valueLen); err != nil {
		return nil, err
	}
	resp.Value = make([]byte, valueLen)
	if _, err := io.ReadFull(r, resp.Value); err != nil {
		return nil, err
	}

	return resp, nil
}

// CommandSet is a command to set a key-value pair with a TTL
type CommandSet struct {
	Key   []byte
	Value []byte
	TTL   int
}

// Bytes returns the byte representation of the set command
func (c *CommandSet) Bytes() []byte {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, CMDSet); err != nil {
		return nil
	}

	keyLen := int32(len(c.Key))
	if err := binary.Write(buf, binary.LittleEndian, keyLen); err != nil {
		return nil
	}
	if err := binary.Write(buf, binary.LittleEndian, c.Key); err != nil {
		return nil
	}

	valueLen := int32(len(c.Value))
	if err := binary.Write(buf, binary.LittleEndian, valueLen); err != nil {
		return nil
	}
	if err := binary.Write(buf, binary.LittleEndian, c.Value); err != nil {
		return nil
	}

	if err := binary.Write(buf, binary.LittleEndian, int32(c.TTL)); err != nil {
		return nil
	}

	return buf.Bytes()
}

// ParseCommand parses a command from the reader
func ParseCommand(r io.Reader) (any, error) {
	var cmd Command
	if err := binary.Read(r, binary.LittleEndian, &cmd); err != nil {
		return nil, err
	}

	switch cmd {
	case CMDSet:
		return parseSetCommand(r)
	case CMDGet:
		return parseGetCommand(r)
	default:
		return nil, fmt.Errorf("invalid command")
	}
}

// CommandGet is a command to get a key
type CommandGet struct {
	Key []byte
}

// Bytes returns the byte representation of the get command
func (c *CommandGet) Bytes() []byte {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, CMDGet); err != nil {
		return nil
	}

	keyLen := int32(len(c.Key))
	if err := binary.Write(buf, binary.LittleEndian, keyLen); err != nil {
		return nil
	}
	if err := binary.Write(buf, binary.LittleEndian, c.Key); err != nil {
		return nil
	}

	return buf.Bytes()
}

// parseSetCommand parses a set command from the reader
func parseSetCommand(r io.Reader) (*CommandSet, error) {
	cmd := &CommandSet{}

	var keyLen int32
	if err := binary.Read(r, binary.LittleEndian, &keyLen); err != nil {
		return nil, err
	}
	cmd.Key = make([]byte, keyLen)
	if _, err := io.ReadFull(r, cmd.Key); err != nil {
		return nil, err
	}

	var valueLen int32
	if err := binary.Read(r, binary.LittleEndian, &valueLen); err != nil {
		return nil, err
	}
	cmd.Value = make([]byte, valueLen)
	if _, err := io.ReadFull(r, cmd.Value); err != nil {
		return nil, err
	}

	var ttl int32
	if err := binary.Read(r, binary.LittleEndian, &ttl); err != nil {
		return nil, err
	}
	cmd.TTL = int(ttl)

	return cmd, nil
}

// parseGetCommand parses a get command from the reader
func parseGetCommand(r io.Reader) (*CommandGet, error) {
	cmd := &CommandGet{}

	var keyLen int32
	if err := binary.Read(r, binary.LittleEndian, &keyLen); err != nil {
		return nil, err
	}
	cmd.Key = make([]byte, keyLen)
	if _, err := io.ReadFull(r, cmd.Key); err != nil {
		return nil, err
	}

	return cmd, nil
}
