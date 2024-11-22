package client

import (
	"context"
	"fmt"
	"net"

	"github.com/dhyanio/discache/transport"
	"github.com/dhyanio/gogger"
)

// Options is the configuration for the client
type Options struct {
	Log *gogger.Logger
}

// Client is the client to interact with the server
type Client struct {
	Options
	conn net.Conn
}

// NewFromConn creates a new client from an existing connection
func NewFromConn(conn net.Conn) *Client {
	return &Client{
		conn: conn,
	}
}

// New creates a new client
func New(endpoint string, opts Options) (*Client, error) {
	conn, err := net.Dial("tcp", endpoint)
	if err != nil {
		opts.Log.Fatal().Msgf("failed to create discache client: %s", err.Error())
	}
	return &Client{
		Options: opts,
		conn:    conn,
	}, nil
}

// Get gets the value for the key from the server
func (c *Client) Get(ctx context.Context, key []byte) ([]byte, error) {
	cmd := &transport.CommandGet{
		Key: key,
	}

	_, err := c.conn.Write(cmd.Bytes())
	if err != nil {
		return nil, err
	}

	resp, err := transport.ParseGetResponse(c.conn)
	if err != nil {
		return nil, err
	}

	if resp.Status == transport.StatusExpired {
		c.Log.Warn().Msgf("key [%s] expired", key)
		return nil, nil
	}

	if resp.Status == transport.StatusKeyNotFound {
		c.Log.Warn().Msgf("key [%s] not present", key)
		return nil, nil
	}

	if resp.Status != transport.StatusOK {
		return nil, fmt.Errorf("server responsed with not OK status [%s]", resp.Status)
	}

	return resp.Value, nil
}

// Put puts the key value pair in the server
func (c *Client) Put(ctx context.Context, key, value []byte, ttl int) error {
	cmd := &transport.CommandSet{
		Key:   key,
		Value: value,
		TTL:   ttl,
	}

	_, err := c.conn.Write(cmd.Bytes())
	if err != nil {
		return err
	}

	resp, err := transport.ParseSetResponse(c.conn)
	if err != nil {
		return err
	}
	if resp.Status != transport.StatusOK {
		return fmt.Errorf("server responsed with noe OK status [%s]", resp.Status)
	}
	return nil
}

// Close closes the connection
func (c *Client) Close() error {
	return c.conn.Close()
}
