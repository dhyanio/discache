package rafter

import (
	"testing"

	"github.com/dhyanio/discache/transport"
	cache "github.com/dhyanio/go-lru"
	"github.com/hashicorp/raft"
)

func newTestFSM() *raftFSM {
	c := cache.NewCache(cache.CacheOpts{
		Capacity: 10,
		TTL:      0,
	})

	return NewRaftFSM(c)
}

func TestRaftFSMAppliedIndex(t *testing.T) {
	fsm := newTestFSM()

	if got := fsm.AppliedIndex(); got != 0 {
		t.Fatalf("initial applied index = %d, want 0", got)
	}

	cmd := &transport.CommandSet{
		Key:   []byte("user:1"),
		Value: []byte("Ajay"),
	}

	result := fsm.Apply(&raft.Log{
		Index: 101,
		Data:  cmd.Bytes(),
	})

	if _, ok := result.(error); ok {
		t.Fatalf("Apply() returned error: %v", result)
	}

	if got := fsm.AppliedIndex(); got != 101 {
		t.Errorf("applied index = %d, want 101", got)
	}

	result = fsm.Apply(&raft.Log{
		Index: 102,
		Data: (&transport.CommandSet{
			Key:   []byte("user:2"),
			Value: []byte("Rahul"),
		}).Bytes(),
	})

	if _, ok := result.(error); ok {
		t.Fatalf("Apply() returned error: %v", result)
	}

	if got := fsm.AppliedIndex(); got != 102 {
		t.Errorf("applied index = %d, want 102", got)
	}
}

func TestRaftFSMAppliedIndexUnchangedOnInvalidCommand(t *testing.T) {
	fsm := newTestFSM()

	result := fsm.Apply(&raft.Log{
		Index: 101,
		Data:  []byte("invalid-command"),
	})

	if _, ok := result.(error); !ok {
		t.Fatalf("Apply() result = %v, want error", result)
	}

	if got := fsm.AppliedIndex(); got != 0 {
		t.Errorf("applied index = %d, want 0", got)
	}
}

func TestRaftFSMAppliedIndexUnchangedOnUnknownCommand(t *testing.T) {
	fsm := newTestFSM()

	result := fsm.Apply(&raft.Log{
		Index: 101,
		Data:  []byte{byte(transport.CmdNonce)},
	})

	if _, ok := result.(error); !ok {
		t.Fatalf("Apply() result = %v, want error", result)
	}

	if got := fsm.AppliedIndex(); got != 0 {
		t.Errorf("applied index = %d, want 0", got)
	}
}
