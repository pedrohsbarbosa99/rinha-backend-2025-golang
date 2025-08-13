package database

import (
	"gorinha/internal/models"
	"math"
	"sync"
)

type Store struct {
	mu   sync.RWMutex
	data map[int8][]models.PaymentRequest
}

func NewStore() *Store {
	return &Store{
		data: make(map[int8][]models.PaymentRequest),
	}
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
