package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"gorinha/internal/config"
	"gorinha/internal/database"
	"gorinha/internal/models"
	"gorinha/internal/processor"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	db       *database.Store
	listener net.Listener
}

func NewServer() *Server {

	return &Server{
		db: database.NewStore(),
	}

}

func (srv *Server) Start() error {
	os.Remove(config.SOCKET_PATH)
	listener, err := net.Listen("unix", config.SOCKET_PATH)
	if err != nil {
		return fmt.Errorf("failed to create socket: %v", err)
	}

	srv.listener = listener
	log.Printf("Server listening on %s", config.SOCKET_PATH)
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
	os.Remove(config.SOCKET_PATH)
	os.Exit(0)

}

func (srv *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	log.Printf("Client connected: %s", conn.RemoteAddr())
	scanner := bufio.NewScanner(conn)
	encoder := json.NewEncoder(conn)
	for scanner.Scan() {
		line := scanner.Text()
		var cmd database.Command
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			fmt.Println(err.Error())
			response := database.Response{Success: false, Message: "Invalid command format"}
			encoder.Encode(response)
			continue

		}

		switch cmd.Type {
		case "put":
			srv.processCommandPut(cmd)
		case "range":
			res := srv.ProcessCommandQuery(cmd)
			encoder.Encode(res)
		case "enqueue":
			srv.processCommandEnqueue(cmd)
		}

	}

	log.Printf("Client disconnected: %s", conn.RemoteAddr())

}

func (srv *Server) ProcessCommandQuery(cmd database.Command) (amounts []int64) {
	amounts, err := srv.db.RangeQuery(cmd.Key, cmd.FromTs, cmd.ToTs)
	if err != nil {
		return amounts
	}
	return

}

func (srv *Server) processCommandPut(cmd database.Command) {
	srv.db.Put(cmd.Key, cmd.Payment)
}

func (srv *Server) processCommandEnqueue(cmd database.Command) {
	srv.db.Enqueue(cmd.Body)
}

var queue = make(chan *models.PaymentRequest, 20000)

func addToQueue(srv *Server) {
	var p *models.PaymentRequest
	data := <-srv.db.Queue
	json.Unmarshal(data, p)
	queue <- p
}

func workerPayments(srv *Server) {
	httpClient := &http.Client{Timeout: 4 * time.Second}

	for {
		payment := <-queue
		processor, err := processor.ProcessPayment(httpClient, payment)
		if err != nil {
			payment.Err = true
			queue <- payment
			continue
		}
		srv.db.Put(processor, *payment)
	}
}

func main() {
	srv := NewServer()
	if err := srv.Start(); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
	go workerPayments(srv)
	go addToQueue(srv)
}
