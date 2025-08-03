package models

import "time"

type PaymentPost struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float32 `json:"amount"`
}

type Payment struct {
	CorrelationId string
	Amount        float32
	RequestedAt   time.Time
	Processor     string
}
