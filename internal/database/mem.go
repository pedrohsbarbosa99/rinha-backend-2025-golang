package database

import (
	"encoding/json"
	"fmt"
	"gorinha/internal/models"
	"net"
	"os"
	"time"
)

const defaultSocket = "/tmp/kvstore.sock"

type MemClient struct {
	SocketPath string
	Timeout    time.Duration
	Conn       net.Conn
}

func NewMemClient() *MemClient {
	path := os.Getenv("MEMDB_SOCKET")
	if path == "" {
		path = defaultSocket
	}
	c := MemClient{
		SocketPath: path,
		Timeout:    500 * time.Second,
	}
	conn, err := c.dial()
	if err == nil {
		c.Conn = conn
	}
	return &c
}

type command struct {
	Type    string                `json:"type"`
	Key     int8                  `json:"key"`
	Payment models.PaymentRequest `json:"payment"`
	Data    []byte                `json:"data"`
	FromTs  int64                 `json:"from_ts,omitempty"`
	ToTs    int64                 `json:"to_ts,omitempty"`
}

func (c *MemClient) dial() (net.Conn, error) {
	conn, err := net.Dial("unix", c.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("dial unix socket %s: %w", c.SocketPath, err)
	}
	if c.Timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(c.Timeout))
	}
	return conn, nil
}

func (c *MemClient) Enqueue(body []byte) (err error) {
	cmd := command{
		Type: "enqueue",
		Data: body,
	}

	if err := json.NewEncoder(c.Conn).Encode(cmd); err != nil {
		return fmt.Errorf("encode put cmd: %w", err)
	}
	return
}

func (c *MemClient) Put(key int8, payment models.PaymentRequest) (err error) {
	cmd := command{
		Type:    "put",
		Payment: payment,
	}

	if err := json.NewEncoder(c.Conn).Encode(cmd); err != nil {
		return fmt.Errorf("encode put cmd: %w", err)
	}
	return
}

func (c *MemClient) RangeQuery(key int8, fromTs, toTs int64) (amounts []int64, err error) {
	cmd := command{
		Type:   "range",
		Key:    key,
		FromTs: fromTs,
		ToTs:   toTs,
	}

	if err := json.NewEncoder(c.Conn).Encode(cmd); err != nil {
		return nil, fmt.Errorf("encode range cmd: %w", err)
	}

	if err := json.NewDecoder(c.Conn).Decode(&amounts); err != nil {
		return nil, fmt.Errorf("decode range resp: %w", err)
	}
	return
}
