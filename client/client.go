package client

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/dhyanio/discache/transport"
)

type Options struct {
}

type Client struct {
	conn net.Conn
}

func NewFromConn(conn net.Conn) *Client {
	return &Client{
		conn: conn,
	}
}

func New(endpoint string, opts Options) (*Client, error) {
	conn, err := net.Dial("tcp", endpoint)
	if err != nil {
		log.Fatal(err)
	}
	return &Client{
		conn: conn,
	}, nil
}

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
	if resp.Status == transport.StatusKeyNotFound {
		return nil, fmt.Errorf("key [%s] not present", key)
	}

	if resp.Status != transport.StatusOK {
		return nil, fmt.Errorf("server responsed with not OK status [%s]", resp.Status)
	}
	return resp.Value, nil
}

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

func (c *Client) Close() error {
	return c.conn.Close()
}
