package models

type PaymentPost struct {
	CorrelationId string  `json:"correlationId"`
	Amount        float32 `json:"amount"`
}
