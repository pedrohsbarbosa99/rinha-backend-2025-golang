package processor

import (
	"gorinha/external/getway"
	"gorinha/internal/config"
	"gorinha/internal/models"
	"net/http"
	"sync"
	"time"
)

func processPayment(client *http.Client, p models.PaymentPost, paymentPending chan models.Payment) (err error) {

	requestedAt, err := getway.PostPayment(
		client,
		p.Amount,
		p.CorrelationId,
		config.PROCESSOR_DEFAULT_URL,
	)
	for range 2 {
		if err != nil {
			requestedAt, err = getway.PostPayment(
				client,
				p.Amount,
				p.CorrelationId,
				config.PROCESSOR_FALLBACK_URL,
			)
			if err == nil {
				paymentPending <- models.Payment{
					CorrelationId: p.CorrelationId,
					Amount:        p.Amount,
					RequestedAt:   requestedAt,
					Processor:     "fallback",
				}
				return
			}
		} else {
			paymentPending <- models.Payment{
				CorrelationId: p.CorrelationId,
				Amount:        p.Amount,
				RequestedAt:   requestedAt,
				Processor:     "default",
			}
			return
		}
	}
	time.Sleep(2 * time.Second)
	queue <- p
	return

}

func processPayments(
	client *http.Client,
	payments []models.PaymentPost,
	wg *sync.WaitGroup,
	paymentPending chan models.Payment,
) {
	for _, p := range payments {
		wg.Add(1)
		payment := p
		go func(payment models.PaymentPost) {
			defer wg.Done()
			processPayment(client, payment, paymentPending)
		}(payment)
	}
}
