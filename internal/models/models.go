package models

import "time"

type PaymentRequest struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float32 `json:"amount"`
	RequestedAt   time.Time
	Err           bool
	Processor     int8
}

type Payment struct {
	CorrelationId string
	Amount        float32
	RequestedAt   time.Time
	Processor     int8
}

type Summary struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float32 `json:"totalAmount"`
}
