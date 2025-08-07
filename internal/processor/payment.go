package processor

import (
	"gorinha/external/getway"
	"gorinha/internal/config"
	"gorinha/internal/models"
	"net/http"
)

func processPayment(client *http.Client, p models.PaymentRequest, paymentPending chan models.Payment) (err error) {
	err = getway.PostPayment(
		client,
		p,
		config.PROCESSOR_DEFAULT_URL,
	)
	if err == nil {
		paymentPending <- models.Payment{
			Amount:      p.Amount,
			RequestedAt: p.RequestedAt,
			Processor:   0,
		}
		return
	} else if p.Err {
		err = getway.PostPayment(
			client,
			p,
			config.PROCESSOR_FALLBACK_URL,
		)
		if err == nil {
			paymentPending <- models.Payment{
				Amount:      p.Amount,
				RequestedAt: p.RequestedAt,
				Processor:   1,
			}
			return

		}
	}
	queue <- models.PaymentRequest{
		CorrelationId: p.CorrelationId,
		Amount:        p.Amount,
		RequestedAt:   p.RequestedAt,
		Err:           true,
	}
	return
}

// func processPayments(
// 	client *http.Client,
// 	payments []models.PaymentRequest,
// 	wg *sync.WaitGroup,
// 	paymentPending chan models.Payment,
// ) {
// 	for _, p := range payments {
// 		wg.Add(1)
// 		payment := p
// 		go func(payment models.PaymentRequest) {
// 			defer wg.Done()
// 			processPayment(client, payment, paymentPending)
// 		}(payment)
// 	}
// }
