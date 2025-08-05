package models

import "time"

type PaymentRequest struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float32 `json:"amount"`
	RequestedAt   time.Time
	Err           bool
}

type Payment struct {
	CorrelationId string
	Amount        float32
	RequestedAt   time.Time
	Processor     string
}
