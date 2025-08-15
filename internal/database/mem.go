package database

import (
	"gorinha/internal/models"
	"math"
	"sync"
)

type Command struct {
	Type    string                `json:"type"`
	Key     int8                  `json:"key"`
	Payment models.PaymentRequest `json:"payment"`
	Body    []byte                `json:"data"`
	FromTs  int64                 `json:"from_ts"`
	ToTs    int64                 `json:"to_ts"`
}

type Response struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Amounts []float32      `json:"amounts,omitempty"`
	Stats   map[string]int `json:"stats,omitempty"`
}

type Store struct {
	mu    sync.RWMutex
	data  map[int8][]models.PaymentRequest
	Queue chan []byte
}

func NewStore() *Store {
	return &Store{
		data:  make(map[int8][]models.PaymentRequest),
		Queue: make(chan []byte, 20_000),
	}
}

func (s *Store) Enqueue(data []byte) {
	s.Queue <- data
}

func (s *Store) Put(processor int8, payment models.PaymentRequest) {
	s.data[processor] = append(s.data[processor], payment)
}

func (s *Store) RangeQuery(key int8, fromTs, toTs int64) ([]int64, error) {
	values := s.data[key]
	var amounts []int64

	for _, p := range values {
		timestamp := p.RequestedAt.UnixNano()

		if timestamp >= fromTs && timestamp <= toTs {
			amounts = append(amounts, int64(math.Round(float64(p.Amount*100))))

		} else if timestamp > toTs {
			break
		}
	}

	return amounts, nil
}
