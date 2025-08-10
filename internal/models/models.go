package models

import "time"

type PaymentRequest struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float32 `json:"amount"`
	RequestedAt   time.Time
	Err           bool
}

type Summary struct {
	TotalRequests int     `json:"totalRequests"`
	TotalAmount   float32 `json:"totalAmount"`
}
