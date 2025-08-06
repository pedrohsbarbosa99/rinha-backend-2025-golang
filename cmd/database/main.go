package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const (
	SocketPath = "/tmp/kvstore.sock"
)

type Value struct {
	Timestamp int64   `json:"timestamp"`
	Amount    float32 `json:"amount"`
}

type Store struct {
	mu   sync.RWMutex
	data map[int8][][]byte
}

func NewStore() *Store {
	return &Store{
		data: make(map[int8][][]byte),
	}
}

func (s *Store) Put(key int8, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = append(s.data[key], data)
	return nil
}

func (s *Store) RangeQuery(key int8, fromTs, toTs int64) ([]float32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	values := s.data[key]
	var amounts []float32

	for _, v := range values {
		var timestamp int64
		var amount float32
		binary.Read(bytes.NewReader(v[:8]), binary.BigEndian, &timestamp)
		binary.Read(bytes.NewReader(v[8:]), binary.BigEndian, &amount)

		if timestamp >= fromTs && timestamp <= toTs {
			amounts = append(amounts, amount)
		} else if timestamp > toTs {
			break
		}
	}

	return amounts, nil
}

type Command struct {
	Type   string `json:"type"`
	Key    int8   `json:"key"`
	Data   []byte `json:"data"`
	FromTs int64  `json:"from_ts"`
	ToTs   int64  `json:"to_ts"`
}

type Response struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Amounts []float32      `json:"amounts,omitempty"`
	Stats   map[string]int `json:"stats,omitempty"`
}

type Server struct {
	store    *Store
	listener net.Listener
}

func NewServer() *Server {
	return &Server{
		store: NewStore(),
	}
}

func (srv *Server) Start() error {
	os.Remove(SocketPath)

	listener, err := net.Listen("unix", SocketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %v", err)
	}
	srv.listener = listener

	log.Printf("Server listening on %s", SocketPath)

	go srv.handleShutdown()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		go srv.handleConnection(conn)
	}
}

func (srv *Server) handleShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down server...")
	if srv.listener != nil {
		srv.listener.Close()
	}
	os.Remove(SocketPath)
	os.Exit(0)
}

func (srv *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	log.Printf("Client connected: %s", conn.RemoteAddr())

	scanner := bufio.NewScanner(conn)
	encoder := json.NewEncoder(conn)

	for scanner.Scan() {
		line := scanner.Text()

		var cmd Command
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			fmt.Println(err.Error())
			response := Response{Success: false, Message: "Invalid command format"}
			encoder.Encode(response)
			continue
		}

		switch cmd.Type {
		case "put":
			srv.processCommandPut(cmd)
		case "range":
			res := srv.ProcessCommandQuery(cmd)
			encoder.Encode(res)
		}

	}

	log.Printf("Client disconnected: %s", conn.RemoteAddr())
}

func (srv *Server) ProcessCommandQuery(cmd Command) (amounts []float32) {
	amounts, err := srv.store.RangeQuery(cmd.Key, cmd.FromTs, cmd.ToTs)
	if err != nil {
		return amounts
	}

	return
}

func (srv *Server) processCommandPut(cmd Command) {
	err := srv.store.Put(cmd.Key, cmd.Data)
	if err != nil {
		return
	}
}

func main() {
	srv := NewServer()

	if err := srv.Start(); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
