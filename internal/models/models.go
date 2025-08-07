package models

import "time"

type PaymentRequest struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float32 `json:"amount"`
	RequestedAt   time.Time
	Err           bool
}

type Payment struct {
	Amount      float32
	RequestedAt time.Time
	Processor   int8
}

type PaymentPayload struct {
	CorrelationId string    `json:"correlationId"`
	Amount        float32   `json:"amount"`
	RequestedAt   time.Time `json:"requestedAt"`
}
