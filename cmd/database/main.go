package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"gorinha/internal/models"
	"gorinha/internal/processor"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var queuePayment = make(chan models.PaymentRequest, 100_000)

var queueBody = make(chan []byte, 100_000)

const (
	SocketPath = "/tmp/kvstore.sock"
)

type Value struct {
	Timestamp int64   `json:"timestamp"`
	Amount    float32 `json:"amount"`
}

type Store struct {
	mu   sync.RWMutex
	data map[int8][]models.PaymentRequest
}

func NewStore() *Store {
	return &Store{
		data: make(map[int8][]models.PaymentRequest),
	}
}

func (*Store) Enqueue(body []byte) {
	queueBody <- body

}

func (s *Store) Put(key int8, data models.PaymentRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = append(s.data[key], data)
}

func (s *Store) RangeQuery(key int8, fromTs, toTs int64) ([]int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	values := s.data[key]
	var amounts []int64

	for _, v := range values {
		timestamp := v.RequestedAt.UnixNano()

		if timestamp >= fromTs && timestamp <= toTs {
			amounts = append(
				amounts, int64(math.Round(float64(v.Amount*100))))
		} else if timestamp > toTs {
			break
		}
	}

	return amounts, nil
}

type Command struct {
	Type    string                `json:"type"`
	Key     int8                  `json:"key"`
	Payment models.PaymentRequest `json:"payment"`
	Data    []byte                `json:"data"`
	FromTs  int64                 `json:"from_ts"`
	ToTs    int64                 `json:"to_ts"`
}

type Response struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Amounts []int64        `json:"amounts,omitempty"`
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
		case "enqueue":
			srv.store.Enqueue(cmd.Data)
		case "put":
			srv.processCommandPut(cmd)
		case "range":
			res := srv.ProcessCommandQuery(cmd)
			encoder.Encode(res)
		}

	}

	log.Printf("Client disconnected: %s", conn.RemoteAddr())
}

func (srv *Server) ProcessCommandQuery(cmd Command) (amounts []int64) {
	amounts, _ = srv.store.RangeQuery(cmd.Key, cmd.FromTs, cmd.ToTs)

	return
}

func (srv *Server) processCommandPut(cmd Command) {
	srv.store.Put(cmd.Key, cmd.Payment)
}

func addToQueue() {
	for {
		var p models.PaymentRequest
		body := <-queueBody
		json.Unmarshal(body, &p)
		p.RequestedAt = time.Now().UTC()
		queuePayment <- p
	}
}

func workerPayments(srv *Server) {
	httpClient := &http.Client{Timeout: 4 * time.Second}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	processorHealth := processor.ChoiceProcessor()

	for {
		go func() {
			for range ticker.C {
				processorHealth = processor.ChoiceProcessor()
				fmt.Println(processorHealth)
			}
		}()

		if processorHealth.Failing || processorHealth.MinResponseTime > 1000 {
			time.Sleep(500 * time.Millisecond)
		} else if processorHealth.MinResponseTime <= 50 {
			go func() {
				payment := <-queuePayment
				pp, err := processor.ProcessPayment(httpClient, payment)

				if err != nil {
					queuePayment <- payment
					return
				}
				srv.store.Put(pp, payment)
			}()

		}
		payment := <-queuePayment
		pp, err := processor.ProcessPayment(httpClient, payment)

		if err != nil {
			queuePayment <- payment
			continue
		}
		srv.store.Put(pp, payment)

	}
}

func main() {
	srv := NewServer()
	go addToQueue()
	go workerPayments(srv)

	if err := srv.Start(); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
