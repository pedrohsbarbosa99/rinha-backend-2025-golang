package processor

import (
	"gorinha/external/getway"
	"gorinha/internal/config"
	"gorinha/internal/models"
	"net/http"
)

func processPayment(client *http.Client, p models.PaymentRequest, paymentPending chan models.Payment) (err error) {
	for range 2 {
		err = getway.PostPayment(
			client,
			p,
			config.PROCESSOR_DEFAULT_URL,
		)
		if err != nil {
			err = getway.PostPayment(
				client,
				p,
				config.PROCESSOR_FALLBACK_URL,
			)
			if err == nil {
				paymentPending <- models.Payment{
					CorrelationId: p.CorrelationId,
					Amount:        p.Amount,
					RequestedAt:   p.RequestedAt,
					Processor:     "fallback",
				}
				return
			}
		} else {
			paymentPending <- models.Payment{
				CorrelationId: p.CorrelationId,
				Amount:        p.Amount,
				RequestedAt:   p.RequestedAt,
				Processor:     "default",
			}
			return
		}
	}
	queue <- p
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
