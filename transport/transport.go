package transport

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type Command byte

const (
	CmdNonce Command = iota
	CMDSet
	CMDGet
	CMDDel
	CMDJoin
)

type Status byte

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

type ResponseSet struct {
	Status Status
}

func (r *ResponseSet) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, r.Status)

	return buf.Bytes()
}

type ResponseGet struct {
	Status Status
	Value  []byte
}

func (r *ResponseGet) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, r.Status)

	valueLen := int32(len(r.Value))
	binary.Write(buf, binary.LittleEndian, valueLen)
	binary.Write(buf, binary.LittleEndian, r.Value)

	return buf.Bytes()
}

func ParseSetResponse(r io.Reader) (*ResponseSet, error) {
	resp := &ResponseSet{}

	err := binary.Read(r, binary.LittleEndian, &resp.Status)

	return resp, err
}

func ParseGetResponse(r io.Reader) (*ResponseGet, error) {
	resp := &ResponseGet{}

	binary.Read(r, binary.LittleEndian, &resp.Status)

	var valueLen int32
	binary.Read(r, binary.LittleEndian, &valueLen)
	resp.Value = make([]byte, valueLen)
	binary.Read(r, binary.LittleEndian, &resp.Value)

	return resp, nil
}

type CommandSet struct {
	Key   []byte
	Value []byte
	TTL   int
}

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

func ParseCommand(r io.Reader) (any, error) {
	var cmd Command
	binary.Read(r, binary.LittleEndian, &cmd)

	switch cmd {
	case CMDSet:
		return parseSetCommand(r), nil
	case CMDGet:
		return parseGetCommand(r), nil
	case CMDJoin:
		return &CommandJoin{}, nil
	default:
		return nil, fmt.Errorf("invalid commnad")

	}
}

type CommandGet struct {
	Key []byte
}

func (c *CommandGet) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, CMDGet)

	keyLen := int32(len(c.Key))
	binary.Write(buf, binary.LittleEndian, keyLen)
	binary.Write(buf, binary.LittleEndian, c.Key)

	return buf.Bytes()

}

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

func parseGetCommand(r io.Reader) *CommandGet {
	cmd := &CommandGet{}

	var keyLen int32
	binary.Read(r, binary.LittleEndian, &keyLen)
	cmd.Key = make([]byte, keyLen)
	binary.Read(r, binary.LittleEndian, &cmd.Key)

	return cmd
}

type CommandJoin struct {
}
