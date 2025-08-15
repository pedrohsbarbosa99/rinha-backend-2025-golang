package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"gorinha/internal/config"
	"log"
	"net"
	"time"
)

type Client struct {
	encoder *json.Encoder
	decoder *json.Decoder
	scanner *bufio.Scanner
	conn    net.Conn
}

func NewClient() *Client {
	conn, err := net.Dial("unix", config.SOCKET_PATH)
	if err != nil {
		panic(err)
	}
	conn.SetDeadline(time.Now().Add(500 * time.Second))

	return &Client{
		conn:    conn,
		encoder: json.NewEncoder(conn),
		decoder: json.NewDecoder(conn),
		scanner: bufio.NewScanner(conn),
	}
}

func (c *Client) Enqueue(body []byte) {
	putCmd := Command{
		Type: "enqueue",
		Body: body,
	}
	if err := c.encoder.Encode(putCmd); err != nil {
		log.Fatalf("failed to send put command: %v", err)
	}
}

func (c *Client) Query(processor int8, fromTs, toTs int64) {
	rangeCmd := Command{
		Type:   "range",
		Key:    processor,
		FromTs: fromTs,
		ToTs:   toTs,
	}
	if err := c.encoder.Encode(rangeCmd); err != nil {
		log.Fatalf("failed to send range command: %v", err)
	}

	if c.scanner.Scan() {
		var res Response
		if err := json.Unmarshal(c.scanner.Bytes(), &res); err != nil {
			log.Fatalf("failed to parse response: %v", err)
		}
		fmt.Printf("Response: %+v\n", res)
	}
}
