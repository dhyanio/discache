package transport

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseGetCommand tests the ParseCommand function with a CommandGet
func TestParseGetCommand(t *testing.T) {
	cmd := &CommandGet{
		Key: []byte("Foo"),
	}

	r := bytes.NewReader(cmd.Bytes())

	pcmd, err := ParseCommand(r)
	assert.Nil(t, err)
	assert.NotNil(t, pcmd)
	assert.IsType(t, &CommandGet{}, pcmd)
	assert.Equal(t, cmd.Key, pcmd.(*CommandGet).Key)
}

// TestParseSetCommand tests the ParseCommand function with a CommandSet
func TestParseSetCommand(t *testing.T) {
	cmd := &CommandSet{
		Key:   []byte("Foo"),
		Value: []byte("Bar"),
		TTL:   2,
	}

	r := bytes.NewReader(cmd.Bytes())

	pcmd, err := ParseCommand(r)
	assert.Nil(t, err)
	assert.NotNil(t, pcmd)
	assert.IsType(t, &CommandSet{}, pcmd)
	assert.Equal(t, cmd.Key, pcmd.(*CommandSet).Key)
	assert.Equal(t, cmd.Value, pcmd.(*CommandSet).Value)
	assert.Equal(t, cmd.TTL, pcmd.(*CommandSet).TTL)
}

// TestParseCommandWithInvalidData tests the ParseCommand function with invalid data
func TestParseCommandWithInvalidData(t *testing.T) {
	invalidData := []byte("invalid data")
	r := bytes.NewReader(invalidData)

	pcmd, err := ParseCommand(r)
	assert.NotNil(t, err)
	assert.Nil(t, pcmd)
}

// BenchmarkParseCommand benchmarks the ParseCommand function
func BenchmarkParseCommand(b *testing.B) {
	cmd := &CommandSet{
		Key:   []byte("Foo"),
		Value: []byte("Bar"),
		TTL:   2,
	}

	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(cmd.Bytes())
		_, _ = ParseCommand(r)
	}
}
