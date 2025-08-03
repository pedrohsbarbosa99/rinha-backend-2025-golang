package processor

import (
	"gorinha/external/getway"
	"gorinha/internal/models"
	"sync"
)

func processPayment(p models.PaymentPost, paymentPending chan models.Payment) (err error) {
	requestedAt, err := getway.PostPayment(
		p.Amount,
		p.CorrelationId,
		env.PROCESSOR_DEFAULT_URL,
	)
	if err != nil {
		requestedAt, err = getway.PostPayment(
			p.Amount,
			p.CorrelationId,
			env.PROCESSOR_FALLBACK_URL,
		)
		if err == nil {
			paymentPending <- models.Payment{
				CorrelationId: p.CorrelationId,
				Amount:        p.Amount,
				RequestedAt:   requestedAt,
				Processor:     "fallback",
			}
		}
	} else {
		paymentPending <- models.Payment{
			CorrelationId: p.CorrelationId,
			Amount:        p.Amount,
			RequestedAt:   requestedAt,
			Processor:     "default",
		}
	}
	return

}

func processPayments(
	payments []models.PaymentPost,
	wg *sync.WaitGroup,
	paymentPending chan models.Payment,
) {
	for _, p := range payments {
		wg.Add(1)
		payment := p
		go func(payment models.PaymentPost) {
			defer wg.Done()
			processPayment(payment, paymentPending)
		}(payment)
	}
}
